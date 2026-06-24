package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/c3-oss/prosa-webp-widgets/internal/metrics"
)

func TestResolveWindowParsesDaysAndDurations(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	since, until, err := resolveWindow("7d", now)
	require.NoError(t, err)
	require.Equal(t, now.UTC(), until)
	require.Equal(t, now.Add(-7*24*time.Hour).UTC(), since)

	since, _, err = resolveWindow("12h", now)
	require.NoError(t, err)
	require.Equal(t, now.Add(-12*time.Hour).UTC(), since)
}

func TestResolveWindowRejectsInvalidValues(t *testing.T) {
	_, _, err := resolveWindow("12x", time.Now())
	require.Error(t, err)
	_, _, err = resolveWindow("0d", time.Now())
	require.Error(t, err)
}

func TestFilterProjectsExcludesByNameAndTail(t *testing.T) {
	projects := []metrics.ProjectUsage{
		{Name: "c3-oss/prosa", Sessions: 64},
		{Name: "c3-oss/prosa-webp-widgets", Sessions: 37},
		{Name: "upsetbit/upsetbit", Sessions: 19},
	}

	require.Equal(t, projects, filterProjects(projects, nil))

	got := filterProjects(projects, []string{"c3-oss/prosa-webp-widgets", "UPSETBIT/upsetbit.git"})
	require.Len(t, got, 1)
	require.Equal(t, "c3-oss/prosa", got[0].Name)
}
