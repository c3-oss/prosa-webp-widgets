package metrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	prosav1 "github.com/c3-oss/prosa/gen/go/prosa/v1"
)

// Provider returns the aggregate view used by the widget renderer.
type Provider interface {
	Snapshot(ctx context.Context, since, until time.Time) (Snapshot, error)
}

// Snapshot is the normalized analytics shape shared by real and mock data.
type Snapshot struct {
	GeneratedAt time.Time
	Since       time.Time
	Until       time.Time

	TotalSessions int64
	TotalTokens   int64
	InputTokens   int64
	OutputTokens  int64
	CachedTokens  int64
	EstCostUSD    float64

	Agents   []AgentUsage
	Models   []ModelUsage
	Projects []ProjectUsage
	Tools    []ToolUsage

	DirectSessions   int64
	SubagentSessions int64
	DirectTokens     int64
	SubagentTokens   int64
	Subagents        []SubagentUsage

	MedianDuration time.Duration
	P90Duration    time.Duration
	AvgDuration    time.Duration
	MaxDuration    time.Duration
}

type AgentUsage struct {
	Name     string
	Sessions int64
	Tokens   int64
	CostUSD  float64
}

type ModelUsage struct {
	Name     string
	Sessions int64
	Tokens   int64
	CostUSD  float64
}

type ProjectUsage struct {
	Name     string
	Sessions int64
}

type ToolUsage struct {
	Name     string
	Uses     int64
	Sessions int64
}

type SubagentUsage struct {
	Agent     string
	Parents   int64
	Children  int64
	MaxFanout int64
}

// FromReports converts prosa AnalyticsService reports into a widget snapshot.
func FromReports(reports map[string]*prosav1.GetReportResponse, since, until time.Time) (Snapshot, error) {
	s := Snapshot{
		GeneratedAt: time.Now().UTC(),
		Since:       since,
		Until:       until,
	}
	if err := applyUsage(&s, reports["usage"]); err != nil {
		return Snapshot{}, err
	}
	if err := applyModels(&s, reports["usage_by_model"]); err != nil {
		return Snapshot{}, err
	}
	if err := applyProjects(&s, reports["projects"]); err != nil {
		return Snapshot{}, err
	}
	if err := applyTools(&s, reports["tools"]); err != nil {
		return Snapshot{}, err
	}
	if err := applySubagents(&s, reports["subagents"]); err != nil {
		return Snapshot{}, err
	}
	if err := applySubagentUsage(&s, reports["subagent_usage_by_day"]); err != nil {
		return Snapshot{}, err
	}
	if err := applyDurationStats(&s, reports["duration_stats"]); err != nil {
		return Snapshot{}, err
	}
	if s.DirectSessions == 0 && s.SubagentSessions == 0 {
		s.DirectSessions = s.TotalSessions
	}
	if s.DirectTokens == 0 && s.SubagentTokens == 0 {
		s.DirectTokens = s.TotalTokens
	}
	return s, nil
}

func applyUsage(s *Snapshot, r *prosav1.GetReportResponse) error {
	if r == nil {
		return fmt.Errorf("missing usage report")
	}
	cols := columns(r.Headers)
	for _, row := range r.Rows {
		agent := cell(row, cols, "AGENT")
		u := AgentUsage{
			Name:     agent,
			Sessions: intCell(row, cols, "SESSIONS"),
			Tokens:   intCell(row, cols, "TOTAL"),
			CostUSD:  floatCell(row, cols, "EST_COST_USD"),
		}
		s.Agents = append(s.Agents, u)
		s.TotalSessions += u.Sessions
		s.TotalTokens += u.Tokens
		s.InputTokens += intCell(row, cols, "INPUT")
		s.OutputTokens += intCell(row, cols, "OUTPUT")
		s.CachedTokens += intCell(row, cols, "CACHED")
		s.EstCostUSD += u.CostUSD
	}
	sort.Slice(s.Agents, func(i, j int) bool {
		if s.Agents[i].Sessions == s.Agents[j].Sessions {
			return s.Agents[i].Name < s.Agents[j].Name
		}
		return s.Agents[i].Sessions > s.Agents[j].Sessions
	})
	return nil
}

func applyModels(s *Snapshot, r *prosav1.GetReportResponse) error {
	if r == nil {
		return fmt.Errorf("missing usage_by_model report")
	}
	cols := columns(r.Headers)
	for _, row := range r.Rows {
		s.Models = append(s.Models, ModelUsage{
			Name:     cell(row, cols, "MODEL"),
			Sessions: intCell(row, cols, "SESSIONS"),
			Tokens:   intCell(row, cols, "TOTAL"),
			CostUSD:  floatCell(row, cols, "EST_COST_USD"),
		})
	}
	sort.Slice(s.Models, func(i, j int) bool {
		if s.Models[i].CostUSD == s.Models[j].CostUSD {
			return s.Models[i].Tokens > s.Models[j].Tokens
		}
		return s.Models[i].CostUSD > s.Models[j].CostUSD
	})
	return nil
}

func applyProjects(s *Snapshot, r *prosav1.GetReportResponse) error {
	if r == nil {
		return fmt.Errorf("missing projects report")
	}
	cols := columns(r.Headers)
	byProject := map[string]int64{}
	for _, row := range r.Rows {
		byProject[cell(row, cols, "PROJECT")] += intCell(row, cols, "SESSIONS")
	}
	for name, sessions := range byProject {
		s.Projects = append(s.Projects, ProjectUsage{Name: name, Sessions: sessions})
	}
	sort.Slice(s.Projects, func(i, j int) bool {
		if s.Projects[i].Sessions == s.Projects[j].Sessions {
			return s.Projects[i].Name < s.Projects[j].Name
		}
		return s.Projects[i].Sessions > s.Projects[j].Sessions
	})
	return nil
}

func applyTools(s *Snapshot, r *prosav1.GetReportResponse) error {
	if r == nil {
		return fmt.Errorf("missing tools report")
	}
	cols := columns(r.Headers)
	for _, row := range r.Rows {
		s.Tools = append(s.Tools, ToolUsage{
			Name:     cell(row, cols, "TOOL"),
			Uses:     intCell(row, cols, "USES"),
			Sessions: intCell(row, cols, "SESSIONS"),
		})
	}
	return nil
}

func applySubagents(s *Snapshot, r *prosav1.GetReportResponse) error {
	if r == nil {
		return fmt.Errorf("missing subagents report")
	}
	cols := columns(r.Headers)
	for _, row := range r.Rows {
		u := SubagentUsage{
			Agent:     cell(row, cols, "AGENT"),
			Parents:   intCell(row, cols, "PARENTS"),
			Children:  intCell(row, cols, "CHILDREN"),
			MaxFanout: intCell(row, cols, "MAX_FANOUT"),
		}
		s.Subagents = append(s.Subagents, u)
	}
	sort.Slice(s.Subagents, func(i, j int) bool {
		return s.Subagents[i].Children > s.Subagents[j].Children
	})
	return nil
}

func applySubagentUsage(s *Snapshot, r *prosav1.GetReportResponse) error {
	if r == nil {
		return fmt.Errorf("missing subagent_usage_by_day report")
	}
	cols := columns(r.Headers)
	for _, row := range r.Rows {
		sessions := intCell(row, cols, "SESSIONS")
		tokens := intCell(row, cols, "TOTAL")
		switch cell(row, cols, "KIND") {
		case "subagent":
			s.SubagentSessions += sessions
			s.SubagentTokens += tokens
		default:
			s.DirectSessions += sessions
			s.DirectTokens += tokens
		}
	}
	return nil
}

func applyDurationStats(s *Snapshot, r *prosav1.GetReportResponse) error {
	if r == nil || len(r.Rows) == 0 {
		return fmt.Errorf("missing duration_stats report")
	}
	cols := columns(r.Headers)
	row := r.Rows[0]
	s.MedianDuration = time.Duration(intCell(row, cols, "MEDIAN_S")) * time.Second
	s.P90Duration = time.Duration(intCell(row, cols, "P90_S")) * time.Second
	s.AvgDuration = time.Duration(intCell(row, cols, "AVG_S")) * time.Second
	s.MaxDuration = time.Duration(intCell(row, cols, "MAX_S")) * time.Second
	return nil
}

func columns(headers []string) map[string]int {
	out := make(map[string]int, len(headers))
	for i, h := range headers {
		out[strings.ToUpper(strings.TrimSpace(h))] = i
	}
	return out
}

func cell(row *prosav1.AnalyticsRow, cols map[string]int, name string) string {
	idx, ok := cols[name]
	if !ok || idx < 0 || idx >= len(row.Values) {
		return ""
	}
	return row.Values[idx]
}

func intCell(row *prosav1.AnalyticsRow, cols map[string]int, name string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(cell(row, cols, name)), 10, 64)
	return n
}

func floatCell(row *prosav1.AnalyticsRow, cols map[string]int, name string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(cell(row, cols, name)), 64)
	return n
}
