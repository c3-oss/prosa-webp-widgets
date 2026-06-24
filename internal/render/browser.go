package render

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"github.com/c3-oss/prosa-webp-widgets/internal/widgets"
)

// Browser renders widget HTML to static WebP screenshots.
type Browser struct {
	launcher *launcher.Launcher
	browser  *rod.Browser
}

func NewBrowser() (*Browser, error) {
	l := launcher.New().Headless(true).Set("hide-scrollbars").Set("no-sandbox")
	if bin := os.Getenv("CHROMIUM_BROWSER_BINARY_PATH"); bin != "" {
		l = l.Bin(bin)
	}
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch chrome: %w", err)
	}
	b := rod.New().ControlURL(controlURL)
	if err := b.Connect(); err != nil {
		l.Cleanup()
		return nil, fmt.Errorf("connect chrome: %w", err)
	}
	return &Browser{launcher: l, browser: b}, nil
}

func (b *Browser) Close() {
	if b.browser != nil {
		_ = b.browser.Close()
	}
	if b.launcher != nil {
		b.launcher.Cleanup()
	}
}

func (b *Browser) CaptureWebP(ctx context.Context, html string) ([]byte, error) {
	page, err := b.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("new page: %w", err)
	}
	defer page.MustClose()

	page = page.Context(ctx)
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             widgets.Width,
		Height:            widgets.Height,
		DeviceScaleFactor: widgets.PixelRatio,
		Mobile:            false,
	}); err != nil {
		return nil, fmt.Errorf("set viewport: %w", err)
	}
	if err := page.SetDocumentContent(html); err != nil {
		return nil, fmt.Errorf("set document: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load: %w", err)
	}
	if _, err := page.Eval(`() => document.fonts.ready`); err != nil {
		return nil, fmt.Errorf("await fonts: %w", err)
	}
	page.MustWaitStable()
	res, err := proto.PageCaptureScreenshot{
		Format: proto.PageCaptureScreenshotFormatWebp,
		Clip: &proto.PageViewport{
			X:      0,
			Y:      0,
			Width:  widgets.Width,
			Height: widgets.Height,
			Scale:  1,
		},
		FromSurface: true,
	}.Call(page)
	if err != nil {
		return nil, fmt.Errorf("capture webp: %w", err)
	}
	if len(res.Data) == 0 {
		return nil, fmt.Errorf("capture webp returned no data")
	}
	if res.Data[0] == 'U' {
		decoded, err := base64.StdEncoding.DecodeString(string(res.Data))
		if err == nil {
			return decoded, nil
		}
	}
	return res.Data, nil
}
