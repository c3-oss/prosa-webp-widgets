package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	prosav1 "github.com/c3-oss/prosa/gen/go/prosa/v1"
	"github.com/c3-oss/prosa/gen/go/prosa/v1/prosav1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var reportNames = []string{
	"usage",
	"usage_by_model",
	"projects",
	"tools",
	"subagents",
	"subagent_usage_by_day",
	"duration_stats",
}

const DefaultHTTPTimeout = 30 * time.Second

// RemoteProvider fetches analytics reports from prosa-server.
type RemoteProvider struct {
	client prosav1connect.AnalyticsServiceClient
}

func NewRemoteProvider(serverURL, token string) (*RemoteProvider, error) {
	return NewRemoteProviderWithTimeout(serverURL, token, DefaultHTTPTimeout)
}

func NewRemoteProviderWithTimeout(serverURL, token string, timeout time.Duration) (*RemoteProvider, error) {
	serverURL = strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if serverURL == "" {
		return nil, fmt.Errorf("PROSA_SERVER_URL is required")
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("PROSA_APP_TOKEN is required")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("HTTP timeout must be positive")
	}
	hc := &http.Client{Transport: appTokenTransport{
		token: token,
		base:  http.DefaultTransport,
	}, Timeout: timeout}
	return &RemoteProvider{
		client: prosav1connect.NewAnalyticsServiceClient(hc, serverURL),
	}, nil
}

func (p *RemoteProvider) Snapshot(ctx context.Context, since, until time.Time) (Snapshot, error) {
	reports := make(map[string]*prosav1.GetReportResponse, len(reportNames))
	for _, name := range reportNames {
		req := connect.NewRequest(&prosav1.GetReportRequest{
			Report: name,
			Since:  timestamppb.New(since),
			Until:  timestamppb.New(until),
		})
		resp, err := p.client.GetReport(ctx, req)
		if err != nil {
			return Snapshot{}, fmt.Errorf("fetch %s: %w", name, err)
		}
		reports[name] = resp.Msg
	}
	return FromReports(reports, since, until)
}

type appTokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t appTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "App "+t.token)
	return t.base.RoundTrip(req)
}
