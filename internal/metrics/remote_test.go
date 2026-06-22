package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
