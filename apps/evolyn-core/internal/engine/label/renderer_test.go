package label

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRendererBuildsSafeSVG(t *testing.T) {
	schema := Schema{
		SchemaVersion: "1.0", Name: "物料<&>",
		Page:     Page{Width: 80, Height: 50, Unit: "mm", DPI: 300, Background: "#ffffff"},
		Settings: Settings{GridSize: 1},
		Elements: []Element{
			{ID: "title", Type: "text", X: 4, Y: 3, Width: 50, Height: 8, Visible: true,
				Value: &ValueSource{Type: "static", Value: "物料<&>"},
				Style: &TextStyle{FontFamily: "sans-serif", FontSize: 5, FontWeight: 700, Color: "#111111", LineHeight: 1.2, TextAlign: "left", VerticalAlign: "top", Overflow: "clip"}},
			{ID: "owner", Type: "field", X: 4, Y: 14, Width: 45, Height: 6, Visible: true, Label: "负责人", Separator: ": ",
				Value: &ValueSource{Type: "field", FieldID: "field_owner"},
				Style: &TextStyle{FontFamily: "sans-serif", FontSize: 4, FontWeight: 400, Color: "#111111", LineHeight: 1.2, TextAlign: "left", VerticalAlign: "top", Overflow: "clip"}},
			{ID: "qr", Type: "qrcode", X: 52, Y: 14, Width: 24, Height: 24, Visible: true,
				Value:   &ValueSource{Type: "system", Key: "recordId"},
				Options: &QRCodeOptions{ErrorCorrection: "M", QuietZone: 1, Foreground: "#000000", Background: "#ffffff"}},
		},
	}
	result, err := NewRenderer(nil).Render(context.Background(), RenderRequest{
		Schema: schema, Format: "svg",
		Data: RenderData{Fields: map[string]any{"field_owner": "张三"}, System: map[string]any{"recordId": "record_1"}},
	})
	require.NoError(t, err)
	svg := string(result.Content)
	require.Contains(t, svg, `width="80mm"`)
	require.Contains(t, svg, "负责人: 张三")
	require.Contains(t, svg, "物料&lt;&amp;&gt;")
	require.Contains(t, svg, "<path fill=\"#000000\"")
	require.False(t, strings.Contains(svg, "<script"))
	require.False(t, strings.Contains(svg, "foreignObject"))
}

func TestRendererReturnsStableQRCodeError(t *testing.T) {
	schema := Schema{
		SchemaVersion: "1.0", Name: "空二维码",
		Page:     Page{Width: 40, Height: 40, Unit: "mm", DPI: 300, Background: "#ffffff"},
		Settings: Settings{GridSize: 1},
		Elements: []Element{{
			ID: "qr", Type: "qrcode", X: 2, Y: 2, Width: 36, Height: 36, Visible: true,
			Value: &ValueSource{Type: "static"},
			Options: &QRCodeOptions{
				ErrorCorrection: "M", QuietZone: 1, Foreground: "#000000", Background: "#ffffff",
			},
		}},
	}
	_, err := NewRenderer(nil).Render(context.Background(), RenderRequest{Schema: schema, Format: "svg"})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrQRGenerate))
}
