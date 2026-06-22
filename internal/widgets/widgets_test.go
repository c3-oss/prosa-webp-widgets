package widgets

import (
	"fmt"
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
	require.Len(t, got, 5)

	expected := []struct {
		Name  string
		Title string
	}{
		{Name: "overview.webp", Title: "Prosa activity"},
		{Name: "agent-mix.webp", Title: "Agent mix"},
		{Name: "model-spend.webp", Title: "Model spend"},
		{Name: "project-focus.webp", Title: "Project focus"},
		{Name: "delegation.webp", Title: "Delegation"},
	}
	sizeRule := fmt.Sprintf("width: %dpx; height: %dpx", Width, Height)

	for i, widget := range got {
		require.Equal(t, expected[i].Name, widget.Name)
		require.Contains(t, widget.HTML, "<!doctype html>")
		require.Contains(t, widget.HTML, `<meta charset="utf-8">`)
		require.Contains(t, widget.HTML, `<div class="brand">prosa</div>`)
		require.Contains(t, widget.HTML, expected[i].Title)
		require.Contains(t, widget.HTML, "last 7d")
		require.Contains(t, widget.HTML, sizeRule)
		require.Contains(t, widget.HTML, `.widget { `+sizeRule)
		require.Contains(t, widget.HTML, `</html>`)
		require.NotContains(t, widget.HTML, "<no value>")
	}
}
