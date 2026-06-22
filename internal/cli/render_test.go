package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
