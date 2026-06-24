package widgets

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"regexp"
	"strings"

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
	Metric    string
	Submetric string
	FootLeft  []string
	FootRight []string
	Bars      []bar
	Theme     string
	Fonts     template.CSS
}

// seg is one value fragment: a number with an optional trailing unit.
type seg struct {
	Num  string
	Unit string
}

type bar struct {
	Label     string
	Share     int
	Primary   seg
	Secondary seg
}

// Build returns all GitHub-profile widgets with stable file names, in both
// light and dark themes.
func Build(s metrics.Snapshot) ([]Widget, error) {
	views := []view{overview(s), agentMix(s), modelSpend(s), projectFocus(s), delegation(s)}
	themes := []struct{ class, suffix string }{{"", ""}, {"dark-mode", "-dark"}}
	out := make([]Widget, 0, len(views)*len(themes))
	for _, v := range views {
		for _, th := range themes {
			v.Theme = th.class
			v.Fonts = fontFaces
			html, err := render(v)
			if err != nil {
				return nil, err
			}
			out = append(out, Widget{Name: v.Kind + th.suffix + ".webp", HTML: html})
		}
	}
	return out, nil
}

func overview(s metrics.Snapshot) view {
	return view{
		Kind:      "overview",
		Title:     "Agent activity",
		Metric:    compact(s.TotalSessions),
		Submetric: "sessions captured",
		// est appears only here, on the one financial value.
		FootLeft:  []string{compact(s.TotalTokens) + " tokens", "est " + money(s.EstCostUSD) + " spend"},
		FootRight: footRight(s),
		Bars:      agentBars(s, 3),
	}
}

func agentMix(s metrics.Snapshot) view {
	return view{
		Kind:      "agent-mix",
		Title:     "Agent mix",
		Metric:    compact(s.TotalTokens),
		Submetric: "tokens across agents",
		FootLeft:  []string{compact(int64(len(s.Agents))) + " agents", compact(s.InputTokens) + " input", compact(s.OutputTokens) + " output"},
		FootRight: footRight(s),
		Bars:      agentBars(s, 3),
	}
}

func modelSpend(s metrics.Snapshot) view {
	return view{
		Kind:      "model-spend",
		Title:     "Model spend",
		Metric:    money(s.EstCostUSD),
		Submetric: "estimated model cost",
		FootLeft:  []string{compact(int64(len(s.Models))) + " models", compact(s.CachedTokens) + " cached"},
		FootRight: footRight(s),
		Bars:      modelBars(s, 3),
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
		Metric:    trimProject(top),
		Submetric: "most active project",
		FootLeft:  []string{compact(int64(len(s.Projects))) + " projects", compact(s.TotalSessions) + " sessions"},
		FootRight: footRight(s),
		Bars:      projectBars(s, 3),
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
		Metric:    percent(s.SubagentSessions, s.DirectSessions+s.SubagentSessions),
		Submetric: "sessions run as subagents",
		FootLeft:  []string{compact(totalChildren) + " children", compact(maxFanout) + " max fanout"},
		FootRight: footRight(s),
		Bars:      subagentBars(s, 3),
	}
}

func footRight(s metrics.Snapshot) []string {
	return []string{"prosa", windowLabel(s)}
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
			Label:     agentName(row.Name),
			Share:     share(row.Sessions, total),
			Primary:   seg{compact(row.Sessions), "sessions"},
			Secondary: seg{compact(row.Tokens), ""},
		})
	}
	return normalize(out)
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
			Label:     modelName(row.Name),
			Share:     shareFloat(row.CostUSD, total),
			Primary:   seg{money(row.CostUSD), ""},
			Secondary: seg{compact(row.Tokens), ""},
		})
	}
	return normalize(out)
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
			Label:   trimProject(row.Name),
			Share:   share(row.Sessions, total),
			Primary: seg{compact(row.Sessions), "sessions"},
		})
	}
	return normalize(out)
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
			Label:     agentName(row.Agent),
			Share:     share(row.Children, total),
			Primary:   seg{compact(row.Children), "children"},
			Secondary: seg{compact(row.Parents), "parents"},
		})
	}
	return normalize(out)
}

// normalize rescales bar widths so the leader fills the track.
func normalize(bars []bar) []bar {
	maxShare := 0
	for _, b := range bars {
		if b.Share > maxShare {
			maxShare = b.Share
		}
	}
	if maxShare <= 0 {
		return bars
	}
	for i := range bars {
		w := int(math.Round(float64(bars[i].Share) / float64(maxShare) * 100))
		if bars[i].Share > 0 && w < 8 {
			w = 8
		}
		bars[i].Share = w
	}
	return bars
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

// money formats a dollar amount. The "est" qualifier is added by callers only
// where an estimate needs flagging (e.g. the overview footer legend).
func money(v float64) string {
	if v == 0 {
		return "$0"
	}
	return "$" + compactFloat(v)
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

var acronyms = map[string]string{"gpt": "GPT", "ai": "AI", "cli": "CLI"}

// agentName turns a raw agent slug into a display name: "claude-code" ->
// "Claude Code", "hermes" -> "Hermes".
func agentName(s string) string {
	s = label(s)
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, p := range parts {
		parts[i] = word(p)
	}
	if len(parts) == 0 {
		return s
	}
	return strings.Join(parts, " ")
}

var versionDash = regexp.MustCompile(`(\d)-(\d)`)

// modelName turns a raw model id into a display name: "claude-opus-4-8" ->
// "Opus 4.8", "gpt-5-codex" -> "GPT-5 Codex", "gemini-2.5-pro" -> "Gemini 2.5 Pro".
func modelName(s string) string {
	s = strings.ToLower(label(s))
	s = strings.TrimPrefix(s, "claude-")
	s = strings.TrimPrefix(s, "claude_")
	s = versionDash.ReplaceAllString(s, "$1.$2")
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	if len(parts) == 0 {
		return label(s)
	}
	var b strings.Builder
	for i, p := range parts {
		token := word(p)
		if i > 0 {
			if isNumber(p) && b.String() == "GPT" {
				b.WriteByte('-')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(token)
	}
	return b.String()
}

func word(p string) string {
	if p == "" {
		return p
	}
	if a, ok := acronyms[p]; ok {
		return a
	}
	if isNumber(p) {
		return p
	}
	return strings.ToUpper(p[:1]) + p[1:]
}

func isNumber(p string) bool {
	return p != "" && p[0] >= '0' && p[0] <= '9'
}

var widgetTemplate = template.Must(template.New("widget").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<style>
{{.Fonts}}
*{box-sizing:border-box;margin:0;padding:0;}
:root{
  --bg:#ffffff; --card:#f6f8fa; --border:#e6eaef;
  --text:#1f2329; --muted:#636c76; --soft:#838e9a; --softer:#aab8c4;
  --track:#e7ebf0; --divider:#1f2329; --bar:#0969da;
}
.dark-mode{
  --bg:#0d1117; --card:#161b22; --border:#272d36;
  --text:#e6edf3; --muted:#8b949e; --soft:#717880; --softer:#3d4246;
  --track:#262c36; --divider:#e6edf3; --bar:#4493f8;
}
html,body{width:1130px;height:348px;overflow:hidden;}
body{font-family:'roboto';background:var(--bg);color:var(--text);}
.card{position:relative;margin:25px;width:1080px;height:296px;background:var(--card);
  border:1px solid var(--border);border-radius:30px;padding:36px 44px;}
.eyebrow{font-size:15px;letter-spacing:3px;text-transform:uppercase;color:var(--soft);}
.hero{font-family:'roboto-medium';font-size:62px;line-height:1.04;color:var(--text);
  white-space:nowrap;overflow:hidden;text-overflow:ellipsis;}
.subtitle{font-family:'roboto-light';font-size:29px;color:var(--muted);
  white-space:nowrap;overflow:hidden;text-overflow:ellipsis;}
.divider{border:none;border-top:1px solid var(--divider);}
.dot{color:var(--bar);}
.foot-left .dot,.foot-right .dot{margin:0 9px;}
.foot-left{font-size:16px;letter-spacing:1.5px;text-transform:uppercase;color:var(--muted);
  white-space:nowrap;overflow:hidden;text-overflow:ellipsis;}
.foot-right{font-family:'roboto-mono-light';font-size:17px;letter-spacing:1px;
  color:var(--muted);white-space:nowrap;}
.powered-by{position:absolute;right:26px;bottom:5px;font-size:10px;letter-spacing:1px;
  text-transform:uppercase;color:var(--softer);}
.bars{display:grid;grid-template-columns:150px 130px 1fr auto auto auto;
  column-gap:16px;row-gap:24px;align-content:center;align-items:center;min-width:0;}
.b-name{font-family:'roboto-medium';font-size:18px;color:var(--text);
  white-space:nowrap;overflow:hidden;text-overflow:ellipsis;min-width:0;}
.bar-track{height:2px;border-radius:2px;background:var(--track);}
.bar-fill{height:2px;border-radius:2px;background:var(--bar);}
.b-val{font-family:'roboto-mono-light';font-size:15px;text-align:right;white-space:nowrap;color:var(--muted);}
.b-val .num{color:var(--muted);}
.b-val .unit{color:var(--soft);}
.b-dot{font-family:'roboto-mono-light';font-size:15px;text-align:center;}
</style>
</head>
<body class="{{.Theme}}">
  <div class="card">
    <div style="display:grid;grid-template-columns:0.82fr 1.18fr;gap:40px;height:178px;">
      <div style="display:flex;flex-direction:column;justify-content:center;min-width:0;">
        <div class="eyebrow">{{.Title}}</div>
        <div class="hero" style="margin-top:14px;">{{.Metric}}</div>
        <div class="subtitle" style="margin-top:10px;">{{.Submetric}}</div>
      </div>
      <div class="bars">
        {{range .Bars}}
        <div class="b-name">{{.Label}}</div>
        <div class="bar-track"><div class="bar-fill" style="width:{{.Share}}%"></div></div>
        <div></div>
        <div class="b-val"><span class="num">{{.Primary.Num}}</span>{{if .Primary.Unit}} <span class="unit">{{.Primary.Unit}}</span>{{end}}</div>
        <div class="b-dot">{{if .Secondary.Num}}<span class="dot">·</span>{{end}}</div>
        <div class="b-val">{{if .Secondary.Num}}<span class="num">{{.Secondary.Num}}</span>{{if .Secondary.Unit}} <span class="unit">{{.Secondary.Unit}}</span>{{end}}{{end}}</div>
        {{end}}
      </div>
    </div>
    <hr class="divider" style="margin-top:6px;">
    <div style="display:flex;justify-content:space-between;align-items:flex-end;margin-top:16px;">
      <div class="foot-left">{{range $i, $s := .FootLeft}}{{if $i}}<span class="dot">·</span>{{end}}{{$s}}{{end}}</div>
      <div class="foot-right">{{range $i, $s := .FootRight}}{{if $i}}<span class="dot">·</span>{{end}}{{$s}}{{end}}</div>
    </div>
  </div>
  <div class="powered-by">powered by github.com/c3-oss/prosa-webp-widgets</div>
</body>
</html>`))
