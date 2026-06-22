package widgets

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"

	"github.com/c3-oss/prosa-webp-widgets/internal/metrics"
)

const (
	Width      = 1130
	Height     = 348
	PixelRatio = 2
)

// Widget is one rendered output.
type Widget struct {
	Name string
	HTML string
}

type view struct {
	Kind      string
	Title     string
	Eyebrow   string
	Metric    string
	Submetric string
	Footer    string
	Stats     []stat
	Bars      []bar
	Split     []bar
}

type stat struct {
	Label string
	Value string
}

type bar struct {
	Label string
	Value string
	Share int
	Tone  string
}

// Build returns all GitHub-profile widgets with stable file names.
func Build(s metrics.Snapshot) ([]Widget, error) {
	views := []view{
		overview(s),
		agentMix(s),
		modelSpend(s),
		projectFocus(s),
		delegation(s),
	}
	out := make([]Widget, 0, len(views))
	for _, v := range views {
		html, err := render(v)
		if err != nil {
			return nil, err
		}
		out = append(out, Widget{Name: v.Kind + ".webp", HTML: html})
	}
	return out, nil
}

func overview(s metrics.Snapshot) view {
	return view{
		Kind:      "overview",
		Title:     "Prosa activity",
		Eyebrow:   windowLabel(s),
		Metric:    compact(s.TotalSessions),
		Submetric: "sessions captured",
		Footer:    fmt.Sprintf("generated %s", s.GeneratedAt.Local().Format("2006-01-02 15:04")),
		Stats: []stat{
			{Label: "tokens", Value: compact(s.TotalTokens)},
			{Label: "est. spend", Value: money(s.EstCostUSD)},
			{Label: "p90 duration", Value: duration(s.P90Duration)},
		},
		Bars: agentBars(s, 4),
	}
}

func agentMix(s metrics.Snapshot) view {
	return view{
		Kind:      "agent-mix",
		Title:     "Agent mix",
		Eyebrow:   windowLabel(s),
		Metric:    compact(s.TotalTokens),
		Submetric: "tokens across agents",
		Footer:    "sessions, tokens, and estimated spend by agent",
		Stats: []stat{
			{Label: "agents", Value: compact(int64(len(s.Agents)))},
			{Label: "input", Value: compact(s.InputTokens)},
			{Label: "output", Value: compact(s.OutputTokens)},
		},
		Bars: agentBars(s, 5),
	}
}

func modelSpend(s metrics.Snapshot) view {
	return view{
		Kind:      "model-spend",
		Title:     "Model spend",
		Eyebrow:   windowLabel(s),
		Metric:    money(s.EstCostUSD),
		Submetric: "estimated model cost",
		Footer:    "costs use the pricing table embedded in prosa",
		Stats: []stat{
			{Label: "models", Value: compact(int64(len(s.Models)))},
			{Label: "cached", Value: compact(s.CachedTokens)},
			{Label: "avg duration", Value: duration(s.AvgDuration)},
		},
		Bars: modelBars(s, 5),
	}
}

func projectFocus(s metrics.Snapshot) view {
	top := "unscoped"
	if len(s.Projects) > 0 {
		top = s.Projects[0].Name
	}
	return view{
		Kind:      "project-focus",
		Title:     "Project focus",
		Eyebrow:   windowLabel(s),
		Metric:    trimProject(top),
		Submetric: "most active project",
		Footer:    "project identity follows prosa remote / marker / path priority",
		Stats: []stat{
			{Label: "projects", Value: compact(int64(len(s.Projects)))},
			{Label: "sessions", Value: compact(s.TotalSessions)},
			{Label: "median", Value: duration(s.MedianDuration)},
		},
		Bars: projectBars(s, 5),
	}
}

func delegation(s metrics.Snapshot) view {
	totalChildren := int64(0)
	maxFanout := int64(0)
	for _, row := range s.Subagents {
		totalChildren += row.Children
		if row.MaxFanout > maxFanout {
			maxFanout = row.MaxFanout
		}
	}
	return view{
		Kind:      "delegation",
		Title:     "Delegation",
		Eyebrow:   windowLabel(s),
		Metric:    percent(s.SubagentSessions, s.DirectSessions+s.SubagentSessions),
		Submetric: "sessions run as subagents",
		Footer:    "direct and delegated work split by parent-session edges",
		Stats: []stat{
			{Label: "children", Value: compact(totalChildren)},
			{Label: "max fanout", Value: compact(maxFanout)},
			{Label: "sub tokens", Value: compact(s.SubagentTokens)},
		},
		Split: delegationSplit(s),
		Bars:  subagentBars(s, 5),
	}
}

func render(v view) (string, error) {
	var buf bytes.Buffer
	if err := widgetTemplate.Execute(&buf, v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func agentBars(s metrics.Snapshot, limit int) []bar {
	total := int64(0)
	for _, row := range s.Agents {
		total += row.Sessions
	}
	out := make([]bar, 0, min(limit, len(s.Agents)))
	for i, row := range s.Agents {
		if i >= limit {
			break
		}
		out = append(out, bar{
			Label: label(row.Name),
			Value: fmt.Sprintf("%s sessions · %s", compact(row.Sessions), compact(row.Tokens)),
			Share: share(row.Sessions, total),
			Tone:  tone(i),
		})
	}
	return out
}

func modelBars(s metrics.Snapshot, limit int) []bar {
	total := 0.0
	for _, row := range s.Models {
		total += row.CostUSD
	}
	out := make([]bar, 0, min(limit, len(s.Models)))
	for i, row := range s.Models {
		if i >= limit {
			break
		}
		out = append(out, bar{
			Label: label(row.Name),
			Value: fmt.Sprintf("%s · %s tokens", money(row.CostUSD), compact(row.Tokens)),
			Share: shareFloat(row.CostUSD, total),
			Tone:  tone(i),
		})
	}
	return out
}

func projectBars(s metrics.Snapshot, limit int) []bar {
	total := int64(0)
	for _, row := range s.Projects {
		total += row.Sessions
	}
	out := make([]bar, 0, min(limit, len(s.Projects)))
	for i, row := range s.Projects {
		if i >= limit {
			break
		}
		out = append(out, bar{
			Label: trimProject(row.Name),
			Value: fmt.Sprintf("%s sessions", compact(row.Sessions)),
			Share: share(row.Sessions, total),
			Tone:  tone(i),
		})
	}
	return out
}

func subagentBars(s metrics.Snapshot, limit int) []bar {
	total := int64(0)
	for _, row := range s.Subagents {
		total += row.Children
	}
	out := make([]bar, 0, min(limit, len(s.Subagents)))
	for i, row := range s.Subagents {
		if i >= limit {
			break
		}
		out = append(out, bar{
			Label: label(row.Agent),
			Value: fmt.Sprintf("%s children · %s parents", compact(row.Children), compact(row.Parents)),
			Share: share(row.Children, total),
			Tone:  tone(i),
		})
	}
	return out
}

func delegationSplit(s metrics.Snapshot) []bar {
	total := s.DirectSessions + s.SubagentSessions
	return []bar{
		{Label: "direct", Value: compact(s.DirectSessions), Share: share(s.DirectSessions, total), Tone: "blue"},
		{Label: "subagent", Value: compact(s.SubagentSessions), Share: share(s.SubagentSessions, total), Tone: "orange"},
	}
}

func windowLabel(s metrics.Snapshot) string {
	days := int(math.Round(s.Until.Sub(s.Since).Hours() / 24))
	if days <= 0 {
		return "latest activity"
	}
	return fmt.Sprintf("last %dd", days)
}

func compact(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	switch {
	case n >= 1_000_000_000:
		return sign + compactFloat(float64(n)/1_000_000_000) + "b"
	case n >= 1_000_000:
		return sign + compactFloat(float64(n)/1_000_000) + "m"
	case n >= 1_000:
		return sign + compactFloat(float64(n)/1_000) + "k"
	default:
		return sign + fmt.Sprintf("%d", n)
	}
}

func compactFloat(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	return strings.TrimSuffix(s, ".0")
}

func money(v float64) string {
	if v == 0 {
		return "$0"
	}
	return "$" + compactFloat(v)
}

func duration(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(math.Round(d.Minutes())))
	}
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%02d", h, m)
}

func percent(part, total int64) string {
	if total <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%d%%", share(part, total))
}

func share(part, total int64) int {
	if total <= 0 || part <= 0 {
		return 0
	}
	pct := int(math.Round(float64(part) / float64(total) * 100))
	if pct == 0 {
		return 1
	}
	return pct
}

func shareFloat(part, total float64) int {
	if total <= 0 || part <= 0 {
		return 0
	}
	pct := int(math.Round(part / total * 100))
	if pct == 0 {
		return 1
	}
	return pct
}

func label(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(none)"
	}
	return s
}

func trimProject(s string) string {
	s = label(s)
	s = strings.TrimSuffix(s, ".git")
	parts := strings.Split(s, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return s
}

func tone(i int) string {
	tones := []string{"green", "blue", "orange", "pink", "violet"}
	return tones[i%len(tones)]
}

var widgetTemplate = template.Must(template.New("widget").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<style>
* { box-sizing: border-box; }
html, body { width: 1130px; height: 348px; margin: 0; overflow: hidden; }
body { font-family: Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f5f7fb; color: #17202a; }
.widget { width: 1130px; height: 348px; display: grid; grid-template-columns: 210px 1fr; background: #f5f7fb; }
.rail { padding: 34px 28px; color: #edf5f1; background: linear-gradient(160deg, #18342f 0%, #16233d 58%, #321d34 100%); }
.brand { font-size: 34px; line-height: 1; letter-spacing: 0; font-weight: 680; }
.eyebrow { margin-top: 14px; color: #b7d6cc; font-size: 15px; }
.rail-footer { position: absolute; bottom: 28px; width: 150px; color: #9db6ca; font-size: 12px; line-height: 1.35; }
.main { padding: 30px 34px 26px; display: grid; grid-template-columns: 330px 1fr; gap: 30px; }
.hero { min-width: 0; }
.title { margin: 0; color: #53606d; text-transform: uppercase; font-size: 14px; font-weight: 720; letter-spacing: 0.06em; }
.metric { margin-top: 20px; font-size: 64px; line-height: .92; font-weight: 760; letter-spacing: 0; color: #17202a; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.submetric { margin-top: 10px; color: #667382; font-size: 20px; }
.stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-top: 30px; }
.stat { border-top: 1px solid #d8dee8; padding-top: 12px; min-width: 0; }
.stat-value { font-size: 25px; font-weight: 720; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.stat-label { margin-top: 4px; color: #718091; font-size: 12px; text-transform: uppercase; letter-spacing: .05em; }
.bars { display: grid; gap: 13px; align-content: center; }
.bar-row { display: grid; grid-template-columns: 180px 1fr 150px; gap: 14px; align-items: center; min-width: 0; }
.bar-label { font-size: 17px; font-weight: 680; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bar-track { height: 13px; border-radius: 999px; background: #dde4ec; overflow: hidden; }
.bar-fill { height: 100%; min-width: 4px; border-radius: 999px; }
.bar-value { color: #5e6b7a; font-size: 14px; text-align: right; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.green { background: #1b8a6b; }
.blue { background: #3468c7; }
.orange { background: #d27623; }
.pink { background: #bd4f74; }
.violet { background: #7d5cc7; }
.split { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin-bottom: 8px; }
.split .bar-row { grid-template-columns: 90px 1fr 54px; }
</style>
</head>
<body>
<div class="widget">
  <aside class="rail">
    <div class="brand">prosa</div>
    <div class="eyebrow">{{.Eyebrow}}</div>
    <div class="rail-footer">{{.Footer}}</div>
  </aside>
  <main class="main">
    <section class="hero">
      <h1 class="title">{{.Title}}</h1>
      <div class="metric">{{.Metric}}</div>
      <div class="submetric">{{.Submetric}}</div>
      <div class="stats">
        {{range .Stats}}
        <div class="stat">
          <div class="stat-value">{{.Value}}</div>
          <div class="stat-label">{{.Label}}</div>
        </div>
        {{end}}
      </div>
    </section>
    <section class="bars">
      {{if .Split}}
      <div class="split">
      {{range .Split}}
        <div class="bar-row">
          <div class="bar-label">{{.Label}}</div>
          <div class="bar-track"><div class="bar-fill {{.Tone}}" style="width: {{.Share}}%"></div></div>
          <div class="bar-value">{{.Value}}</div>
        </div>
      {{end}}
      </div>
      {{end}}
      {{range .Bars}}
      <div class="bar-row">
        <div class="bar-label">{{.Label}}</div>
        <div class="bar-track"><div class="bar-fill {{.Tone}}" style="width: {{.Share}}%"></div></div>
        <div class="bar-value">{{.Value}}</div>
      </div>
      {{end}}
    </section>
  </main>
</div>
</body>
</html>`))
