package render

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/c3-oss/prosa-webp-widgets/internal/metrics"
	"github.com/c3-oss/prosa-webp-widgets/internal/widgets"
	"github.com/stretchr/testify/require"
)

func TestCaptureWebPOptInRequiresChromium(t *testing.T) {
	// Chromium availability varies across CI runners; run this check explicitly.
	if os.Getenv("PROSA_WEBP_WIDGETS_RENDER_TEST") != "1" {
		t.Skip("set PROSA_WEBP_WIDGETS_RENDER_TEST=1 to run Chromium-backed WebP capture regression")
	}

	built, err := widgets.Build(metrics.Snapshot{
		GeneratedAt: time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC),
		Since:       time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Until:       time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.NotEmpty(t, built)

	browser, err := NewBrowser()
	require.NoError(t, err)
	defer browser.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	data, err := browser.CaptureWebP(ctx, built[0].HTML)
	require.NoError(t, err)

	width, height, err := webpSize(data)
	require.NoError(t, err)
	require.Equal(t, widgets.Width*widgets.PixelRatio, width)
	require.Equal(t, widgets.Height*widgets.PixelRatio, height)
}

func webpSize(data []byte) (int, int, error) {
	if len(data) < 20 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, fmt.Errorf("not a WebP RIFF payload")
	}
	for offset := 12; offset+8 <= len(data); {
		chunkType := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunkStart := offset + 8
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(data) {
			return 0, 0, fmt.Errorf("truncated %s chunk", chunkType)
		}
		chunk := data[chunkStart:chunkEnd]
		switch chunkType {
		case "VP8X":
			if len(chunk) < 10 {
				return 0, 0, fmt.Errorf("short VP8X chunk")
			}
			width := 1 + int(chunk[4]) + int(chunk[5])<<8 + int(chunk[6])<<16
			height := 1 + int(chunk[7]) + int(chunk[8])<<8 + int(chunk[9])<<16
			return width, height, nil
		case "VP8 ":
			if len(chunk) < 10 || chunk[3] != 0x9d || chunk[4] != 0x01 || chunk[5] != 0x2a {
				return 0, 0, fmt.Errorf("invalid VP8 frame header")
			}
			width := int(binary.LittleEndian.Uint16(chunk[6:8]) & 0x3fff)
			height := int(binary.LittleEndian.Uint16(chunk[8:10]) & 0x3fff)
			return width, height, nil
		case "VP8L":
			if len(chunk) < 5 || chunk[0] != 0x2f {
				return 0, 0, fmt.Errorf("invalid VP8L frame header")
			}
			width := 1 + int(chunk[1]) + int(chunk[2]&0x3f)<<8
			height := 1 + int(chunk[2]&0xc0)>>6 + int(chunk[3])<<2 + int(chunk[4]&0x0f)<<10
			return width, height, nil
		}
		offset = chunkEnd + chunkSize%2
	}
	return 0, 0, fmt.Errorf("missing WebP size chunk")
}
