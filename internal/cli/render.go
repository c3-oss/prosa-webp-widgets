package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/c3-oss/prosa-webp-widgets/internal/metrics"
	"github.com/c3-oss/prosa-webp-widgets/internal/render"
	"github.com/c3-oss/prosa-webp-widgets/internal/storage"
	"github.com/c3-oss/prosa-webp-widgets/internal/widgets"
)

type renderOptions struct {
	mock     bool
	outDir   string
	upload   bool
	last     string
	server   string
	token    string
	timeout  string
	excludes []string
}

func newRenderCmd() *cobra.Command {
	opts := renderOptions{
		outDir:  "out",
		last:    "30d",
		server:  os.Getenv("PROSA_SERVER_URL"),
		token:   os.Getenv("PROSA_APP_TOKEN"),
		timeout: defaultHTTPTimeout(),
	}
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render all widgets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRender(cmd.Context(), cmd, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.mock, "mock", false, "render deterministic mock data")
	cmd.Flags().StringVar(&opts.outDir, "out", opts.outDir, "output directory; set empty with --out '' to disable disk writes")
	cmd.Flags().BoolVar(&opts.upload, "upload", false, "upload rendered widgets to S3")
	cmd.Flags().StringVar(&opts.last, "last", opts.last, "lookback window, such as 7d, 30d, 12h")
	cmd.Flags().StringVar(&opts.server, "server", opts.server, "prosa server URL (default $PROSA_SERVER_URL)")
	cmd.Flags().StringVar(&opts.token, "token", opts.token, "prosa app token (default $PROSA_APP_TOKEN)")
	cmd.Flags().StringVar(&opts.timeout, "timeout", opts.timeout, "remote HTTP timeout (default $PROSA_HTTP_TIMEOUT or 30s)")
	cmd.Flags().StringSliceVar(&opts.excludes, "exclude-project", nil, "project to hide from project-focus (repeatable or comma-separated; matches owner/repo)")
	return cmd
}

func runRender(ctx context.Context, cmd *cobra.Command, opts renderOptions) error {
	since, until, err := resolveWindow(opts.last, time.Now())
	if err != nil {
		return err
	}
	provider, err := providerFor(opts)
	if err != nil {
		return err
	}
	snap, err := provider.Snapshot(ctx, since, until)
	if err != nil {
		return err
	}
	snap.Projects = filterProjects(snap.Projects, opts.excludes)
	built, err := widgets.Build(snap)
	if err != nil {
		return err
	}
	browser, err := render.NewBrowser()
	if err != nil {
		return err
	}
	defer browser.Close()

	sinks, err := sinksFor(ctx, opts)
	if err != nil {
		return err
	}
	if len(sinks) == 0 {
		return fmt.Errorf("no output sinks configured")
	}

	out := cmd.OutOrStdout()
	for _, widget := range built {
		slog.Info("rendering widget", "name", widget.Name)
		data, err := browser.CaptureWebP(ctx, widget.HTML)
		if err != nil {
			return fmt.Errorf("render %s: %w", widget.Name, err)
		}
		for _, sink := range sinks {
			location, err := sink.Put(ctx, widget.Name, data)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "%s\t%s\n", widget.Name, location)
		}
	}
	return nil
}

func providerFor(opts renderOptions) (metrics.Provider, error) {
	if opts.mock {
		return metrics.MockProvider{}, nil
	}
	timeout, err := parseHTTPTimeout(opts.timeout)
	if err != nil {
		return nil, err
	}
	return metrics.NewRemoteProviderWithTimeout(opts.server, opts.token, timeout)
}

func sinksFor(ctx context.Context, opts renderOptions) ([]storage.Sink, error) {
	var out []storage.Sink
	if strings.TrimSpace(opts.outDir) != "" {
		out = append(out, storage.DiskSink{Dir: opts.outDir})
	}
	if opts.upload {
		s3sink, err := storage.NewS3Sink(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, s3sink)
	}
	return out, nil
}

func resolveWindow(last string, now time.Time) (time.Time, time.Time, error) {
	last = strings.TrimSpace(last)
	if last == "" {
		last = "30d"
	}
	d, err := parseLookback(last)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	until := now.UTC()
	return until.Add(-d), until, nil
}

func parseLookback(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n, err := parsePositiveInt(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --last %q", s)
	}
	if d <= 0 {
		return 0, fmt.Errorf("--last must be positive")
	}
	return d, nil
}

func parsePositiveInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("--last must be a positive duration")
	}
	return n, nil
}

// filterProjects drops projects named in exclude, matching either the full
// identity or its owner/repo tail (case-insensitive, .git-insensitive).
func filterProjects(projects []metrics.ProjectUsage, exclude []string) []metrics.ProjectUsage {
	if len(exclude) == 0 {
		return projects
	}
	hidden := make(map[string]bool, len(exclude)*2)
	for _, e := range exclude {
		if full := normalizeProject(e); full != "" {
			hidden[full] = true
			hidden[projectTail(full)] = true
		}
	}
	out := make([]metrics.ProjectUsage, 0, len(projects))
	for _, p := range projects {
		full := normalizeProject(p.Name)
		if hidden[full] || hidden[projectTail(full)] {
			continue
		}
		out = append(out, p)
	}
	return out
}

func normalizeProject(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.TrimSuffix(s, ".git")
}

func projectTail(s string) string {
	parts := strings.Split(s, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return s
}

func defaultHTTPTimeout() string {
	if v := strings.TrimSpace(os.Getenv("PROSA_HTTP_TIMEOUT")); v != "" {
		return v
	}
	return metrics.DefaultHTTPTimeout.String()
}

func parseHTTPTimeout(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("--timeout/PROSA_HTTP_TIMEOUT must be a positive duration, such as 30s, 2m, or 500ms")
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --timeout/PROSA_HTTP_TIMEOUT %q: use a duration such as 30s, 2m, or 500ms", s)
	}
	if d <= 0 {
		return 0, fmt.Errorf("--timeout/PROSA_HTTP_TIMEOUT must be positive")
	}
	return d, nil
}
