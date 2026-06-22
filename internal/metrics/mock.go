package metrics

import (
	"context"
	"time"
)

// MockProvider returns stable sample data for visual review.
type MockProvider struct{}

func (MockProvider) Snapshot(_ context.Context, since, until time.Time) (Snapshot, error) {
	return Snapshot{
		GeneratedAt:      time.Date(2026, 6, 22, 15, 0, 0, 0, time.UTC),
		Since:            since,
		Until:            until,
		TotalSessions:    184,
		TotalTokens:      43_820_000,
		InputTokens:      31_410_000,
		OutputTokens:     7_960_000,
		CachedTokens:     4_450_000,
		EstCostUSD:       142.37,
		MedianDuration:   18 * time.Minute,
		P90Duration:      74 * time.Minute,
		AvgDuration:      31 * time.Minute,
		MaxDuration:      3*time.Hour + 12*time.Minute,
		DirectSessions:   139,
		SubagentSessions: 45,
		DirectTokens:     29_240_000,
		SubagentTokens:   14_580_000,
		Agents: []AgentUsage{
			{Name: "codex", Sessions: 96, Tokens: 22_500_000, CostUSD: 68.24},
			{Name: "claude-code", Sessions: 61, Tokens: 17_200_000, CostUSD: 61.75},
			{Name: "hermes", Sessions: 18, Tokens: 3_100_000, CostUSD: 9.12},
			{Name: "gemini", Sessions: 9, Tokens: 1_020_000, CostUSD: 3.26},
		},
		Models: []ModelUsage{
			{Name: "gpt-5-codex", Sessions: 92, Tokens: 21_900_000, CostUSD: 66.18},
			{Name: "claude-opus-4-5", Sessions: 44, Tokens: 13_800_000, CostUSD: 52.46},
			{Name: "claude-sonnet-4-5", Sessions: 31, Tokens: 6_400_000, CostUSD: 18.27},
			{Name: "gemini-2.5-pro", Sessions: 17, Tokens: 1_720_000, CostUSD: 5.46},
		},
		Projects: []ProjectUsage{
			{Name: "c3-oss/prosa", Sessions: 64},
			{Name: "c3-oss/prosa-webp-widgets", Sessions: 37},
			{Name: "caian-org/lastfm-webp-widgets", Sessions: 25},
			{Name: "upsetbit/upsetbit", Sessions: 19},
			{Name: "c3-oss/tocaia", Sessions: 14},
		},
		Tools: []ToolUsage{
			{Name: "exec_command", Uses: 1180, Sessions: 113},
			{Name: "apply_patch", Uses: 176, Sessions: 44},
			{Name: "web.run", Uses: 39, Sessions: 18},
			{Name: "view_image", Uses: 12, Sessions: 5},
		},
		Subagents: []SubagentUsage{
			{Agent: "claude-code", Parents: 14, Children: 31, MaxFanout: 5},
			{Agent: "codex", Parents: 6, Children: 14, MaxFanout: 4},
		},
	}, nil
}
