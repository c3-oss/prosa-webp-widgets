package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	prosav1 "github.com/c3-oss/prosa/gen/go/prosa/v1"
)

func TestFromReportsAggregatesWidgetSnapshot(t *testing.T) {
	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	until := since.Add(30 * 24 * time.Hour)
	reports := map[string]*prosav1.GetReportResponse{
		"usage": {
			Headers: []string{"AGENT", "SESSIONS", "MEASURED", "TOTAL", "INPUT", "OUTPUT", "CACHED", "EST_COST_USD"},
			Rows: []*prosav1.AnalyticsRow{
				row("codex", "10", "10", "1000", "700", "200", "100", "1.2500"),
				row("claude-code", "4", "4", "800", "500", "200", "100", "2.0000"),
			},
		},
		"usage_by_model": {
			Headers: []string{"MODEL", "SESSIONS", "TOTAL", "INPUT", "OUTPUT", "EST_COST_USD"},
			Rows: []*prosav1.AnalyticsRow{
				row("gpt-5-codex", "10", "1000", "700", "200", "1.2500"),
				row("claude-opus-4-5", "4", "800", "500", "200", "2.0000"),
			},
		},
		"projects": {
			Headers: []string{"PROJECT", "AGENT", "SESSIONS"},
			Rows: []*prosav1.AnalyticsRow{
				row("c3-oss/prosa", "codex", "7"),
				row("c3-oss/prosa", "claude-code", "2"),
				row("c3-oss/tocaia", "codex", "5"),
			},
		},
		"tools": {
			Headers: []string{"TOOL", "USES", "SESSIONS"},
			Rows:    []*prosav1.AnalyticsRow{row("exec_command", "42", "8")},
		},
		"subagents": {
			Headers: []string{"AGENT", "PARENTS", "CHILDREN", "MAX_FANOUT"},
			Rows:    []*prosav1.AnalyticsRow{row("codex", "2", "5", "3")},
		},
		"subagent_usage_by_day": {
			Headers: []string{"DAY", "KIND", "MODEL", "SESSIONS", "MEASURED", "TOTAL", "INPUT", "OUTPUT", "CACHED", "CACHE_READ", "CACHE_CREATION"},
			Rows: []*prosav1.AnalyticsRow{
				row("2026-06-01", "direct", "gpt-5-codex", "11", "11", "1300", "900", "300", "100", "0", "0"),
				row("2026-06-01", "subagent", "gpt-5-codex", "3", "3", "500", "300", "100", "100", "0", "0"),
			},
		},
		"duration_stats": {
			Headers: []string{"MEDIAN_S", "P90_S", "AVG_S", "MAX_S"},
			Rows:    []*prosav1.AnalyticsRow{row("600", "1800", "900", "3600")},
		},
	}

	snap, err := FromReports(reports, since, until)
	require.NoError(t, err)
	require.Equal(t, int64(14), snap.TotalSessions)
	require.Equal(t, int64(1800), snap.TotalTokens)
	require.Equal(t, 3.25, snap.EstCostUSD)
	require.Equal(t, "codex", snap.Agents[0].Name)
	require.Equal(t, "claude-opus-4-5", snap.Models[0].Name)
	require.Equal(t, "c3-oss/prosa", snap.Projects[0].Name)
	require.Equal(t, int64(3), snap.SubagentSessions)
	require.Equal(t, 30*time.Minute, snap.P90Duration)
}

func row(values ...string) *prosav1.AnalyticsRow {
	return &prosav1.AnalyticsRow{Values: values}
}
