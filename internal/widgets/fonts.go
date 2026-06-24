package widgets

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
)

//go:embed fonts/roboto-regular.ttf
var fontRegular []byte

//go:embed fonts/roboto-light.ttf
var fontLight []byte

//go:embed fonts/roboto-medium.ttf
var fontMedium []byte

//go:embed fonts/roboto-mono-light.ttf
var fontMonoLight []byte

func face(family string, ttf []byte) string {
	return fmt.Sprintf(
		"@font-face{font-family:'%s';src:url(data:font/ttf;base64,%s) format('truetype');}",
		family, base64.StdEncoding.EncodeToString(ttf),
	)
}

// fontFaces is the inlined @font-face block so rendering needs no external assets.
var fontFaces = buildFontFaces()

func buildFontFaces() template.CSS {
	css := face("roboto", fontRegular) +
		face("roboto-light", fontLight) +
		face("roboto-medium", fontMedium) +
		face("roboto-mono-light", fontMonoLight)
	// Fonts are embedded at build time via go:embed; nothing here is user input.
	return template.CSS(css) // #nosec G203
}
