package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	prosav1 "github.com/c3-oss/prosa/gen/go/prosa/v1"
	"github.com/c3-oss/prosa/gen/go/prosa/v1/prosav1connect"
	"github.com/stretchr/testify/require"
)

func TestRemoteProviderRequestsExpectedReportsWithAppToken(t *testing.T) {
	service := &recordingAnalyticsService{}
	path, handler := prosav1connect.NewAnalyticsServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	provider, err := NewRemoteProviderWithTimeout(server.URL+"/", "secret-token", time.Second)
	require.NoError(t, err)

	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	snapshot, err := provider.Snapshot(context.Background(), since, until)
	require.NoError(t, err)
	require.Equal(t, int64(1), snapshot.TotalSessions)

	calls := service.Calls()
	require.Len(t, calls, len(reportNames))
	for i, call := range calls {
		require.Equal(t, reportNames[i], call.Report)
		require.Equal(t, since, call.Since)
		require.Equal(t, until, call.Until)
		require.Equal(t, "App secret-token", call.Authorization)
	}
}

func TestRemoteProviderTimesOutHungServer(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	}))
	defer server.Close()
	defer once.Do(func() {
		close(release)
	})

	provider, err := NewRemoteProviderWithTimeout(server.URL, "token", 20*time.Millisecond)
	require.NoError(t, err)

	start := time.Now()
	_, err = provider.Snapshot(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
	require.ErrorContains(t, err, "fetch usage")
	require.Less(t, time.Since(start), 2*time.Second)
}

type analyticsCall struct {
	Report        string
	Since         time.Time
	Until         time.Time
	Authorization string
}

type recordingAnalyticsService struct {
	mu    sync.Mutex
	calls []analyticsCall
}

func (s *recordingAnalyticsService) GetReport(_ context.Context, req *connect.Request[prosav1.GetReportRequest]) (*connect.Response[prosav1.GetReportResponse], error) {
	msg := req.Msg
	s.mu.Lock()
	s.calls = append(s.calls, analyticsCall{
		Report:        msg.GetReport(),
		Since:         msg.GetSince().AsTime(),
		Until:         msg.GetUntil().AsTime(),
		Authorization: req.Header().Get("Authorization"),
	})
	s.mu.Unlock()

	return connect.NewResponse(remoteReport(msg.GetReport())), nil
}

func (s *recordingAnalyticsService) Calls() []analyticsCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]analyticsCall(nil), s.calls...)
}

func remoteReport(name string) *prosav1.GetReportResponse {
	switch name {
	case "usage":
		return &prosav1.GetReportResponse{
			Headers: []string{"AGENT", "SESSIONS", "TOTAL", "INPUT", "OUTPUT", "CACHED", "EST_COST_USD"},
			Rows:    []*prosav1.AnalyticsRow{remoteRow("codex", "1", "100", "60", "30", "10", "0.0500")},
		}
	case "usage_by_model":
		return &prosav1.GetReportResponse{
			Headers: []string{"MODEL", "SESSIONS", "TOTAL", "EST_COST_USD"},
			Rows:    []*prosav1.AnalyticsRow{remoteRow("gpt-5-codex", "1", "100", "0.0500")},
		}
	case "projects":
		return &prosav1.GetReportResponse{
			Headers: []string{"PROJECT", "AGENT", "SESSIONS"},
			Rows:    []*prosav1.AnalyticsRow{remoteRow("c3-oss/prosa-webp-widgets", "codex", "1")},
		}
	case "tools":
		return &prosav1.GetReportResponse{
			Headers: []string{"TOOL", "USES", "SESSIONS"},
			Rows:    []*prosav1.AnalyticsRow{remoteRow("exec_command", "3", "1")},
		}
	case "subagents":
		return &prosav1.GetReportResponse{
			Headers: []string{"AGENT", "PARENTS", "CHILDREN", "MAX_FANOUT"},
			Rows:    []*prosav1.AnalyticsRow{remoteRow("codex", "1", "2", "2")},
		}
	case "subagent_usage_by_day":
		return &prosav1.GetReportResponse{
			Headers: []string{"DAY", "KIND", "MODEL", "SESSIONS", "TOTAL"},
			Rows:    []*prosav1.AnalyticsRow{remoteRow("2026-06-01", "direct", "gpt-5-codex", "1", "100")},
		}
	case "duration_stats":
		return &prosav1.GetReportResponse{
			Headers: []string{"MEDIAN_S", "P90_S", "AVG_S", "MAX_S"},
			Rows:    []*prosav1.AnalyticsRow{remoteRow("60", "120", "90", "180")},
		}
	default:
		return &prosav1.GetReportResponse{}
	}
}

func remoteRow(values ...string) *prosav1.AnalyticsRow {
	return &prosav1.AnalyticsRow{Values: values}
}
