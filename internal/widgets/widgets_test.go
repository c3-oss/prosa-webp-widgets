package widgets

import (
	"testing"
	"time"

	"github.com/c3-oss/prosa-webp-widgets/internal/metrics"
	"github.com/stretchr/testify/require"
)

func TestBuildEmptySnapshotReturnsStableWidgets(t *testing.T) {
	snapshot := metrics.Snapshot{
		GeneratedAt: time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
		Since:       time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Until:       time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
	}

	got, err := Build(snapshot)
	require.NoError(t, err)
	require.Len(t, got, 10)

	expected := []struct {
		Name  string
		Title string
		Dark  bool
	}{
		{Name: "overview.webp", Title: "Agent activity"},
		{Name: "overview-dark.webp", Title: "Agent activity", Dark: true},
		{Name: "agent-mix.webp", Title: "Agent mix"},
		{Name: "agent-mix-dark.webp", Title: "Agent mix", Dark: true},
		{Name: "model-spend.webp", Title: "Model spend"},
		{Name: "model-spend-dark.webp", Title: "Model spend", Dark: true},
		{Name: "project-focus.webp", Title: "Project focus"},
		{Name: "project-focus-dark.webp", Title: "Project focus", Dark: true},
		{Name: "delegation.webp", Title: "Delegation"},
		{Name: "delegation-dark.webp", Title: "Delegation", Dark: true},
	}

	for i, widget := range got {
		require.Equal(t, expected[i].Name, widget.Name)
		require.Contains(t, widget.HTML, "<!doctype html>")
		require.Contains(t, widget.HTML, `<meta charset="utf-8">`)
		require.Contains(t, widget.HTML, "width:1130px;height:348px")
		require.Contains(t, widget.HTML, expected[i].Title)
		require.Contains(t, widget.HTML, "last 7d")
		require.Contains(t, widget.HTML, "powered by github.com/c3-oss/prosa-webp-widgets")
		require.Contains(t, widget.HTML, "</html>")
		require.NotContains(t, widget.HTML, "<no value>")
		if expected[i].Dark {
			require.Contains(t, widget.HTML, `<body class="dark-mode">`)
		} else {
			require.Contains(t, widget.HTML, `<body class="">`)
		}
	}
}

func TestAgentName(t *testing.T) {
	cases := map[string]string{
		"claude-code": "Claude Code",
		"codex":       "Codex",
		"hermes":      "Hermes",
		"gemini":      "Gemini",
		"":            "(none)",
	}
	for in, want := range cases {
		require.Equal(t, want, agentName(in), "agentName(%q)", in)
	}
}

func TestModelName(t *testing.T) {
	cases := map[string]string{
		"claude-opus-4-8":   "Opus 4.8",
		"claude-opus-4-5":   "Opus 4.5",
		"claude-sonnet-4-5": "Sonnet 4.5",
		"gpt-5-codex":       "GPT-5 Codex",
		"gemini-2.5-pro":    "Gemini 2.5 Pro",
	}
	for in, want := range cases {
		require.Equal(t, want, modelName(in), "modelName(%q)", in)
	}
}

func TestNormalizeScalesToLeader(t *testing.T) {
	got := normalize([]bar{
		{Share: 50},
		{Share: 25},
		{Share: 5},
	})
	require.Equal(t, 100, got[0].Share)
	require.Equal(t, 50, got[1].Share)
	require.Equal(t, 10, got[2].Share) // 5/50 of the leader, rescaled
}

func TestMoneyHasNoEstPrefix(t *testing.T) {
	require.Equal(t, "$142.4", money(142.37))
	require.Equal(t, "$0", money(0))
}

func TestEstAppearsOnlyOnOverviewFinancialFooter(t *testing.T) {
	got, err := Build(metrics.Snapshot{
		Since:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Until:      time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
		EstCostUSD: 142.37,
	})
	require.NoError(t, err)
	byName := map[string]string{}
	for _, w := range got {
		byName[w.Name] = w.HTML
	}
	require.Contains(t, byName["overview.webp"], "est $")
	require.NotContains(t, byName["agent-mix.webp"], "est $")
	require.NotContains(t, byName["model-spend.webp"], "est $")
	require.NotContains(t, byName["delegation.webp"], "est $")
}
