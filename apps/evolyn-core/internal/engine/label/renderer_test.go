package label

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
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

func TestRendererOutputsPhysicalPNGAndPDF(t *testing.T) {
	schema := Schema{
		SchemaVersion: "1.0", Name: "资产标签",
		Page:     Page{Width: 80, Height: 50, Unit: "mm", DPI: 300, Background: "#ffffff"},
		Settings: Settings{GridSize: 1},
		Elements: []Element{
			{ID: "title", Type: "text", X: 4, Y: 3, Width: 45, Height: 8, Visible: true,
				Value: &ValueSource{Type: "static", Value: "资产标签"},
				Style: &TextStyle{FontFamily: "Noto Sans CJK SC", FontSize: 5, FontWeight: 700, Color: "#111111", LineHeight: 1.2, TextAlign: "left", VerticalAlign: "top", Overflow: "clip"}},
			{ID: "qr", Type: "qrcode", X: 52, Y: 14, Width: 24, Height: 24, Visible: true,
				Value:   &ValueSource{Type: "static", Value: "https://lingyanyun.com/records/1"},
				Options: &QRCodeOptions{ErrorCorrection: "M", QuietZone: 4, Foreground: "#000000", Background: "#ffffff"}},
		},
	}
	renderer := NewRenderer(nil)
	pngResult, err := renderer.Render(context.Background(), RenderRequest{Schema: schema, Format: "png"})
	require.NoError(t, err)
	require.Equal(t, "image/png", pngResult.MIMEType)
	require.Equal(t, 945, pngResult.PixelWidth)
	require.Equal(t, 591, pngResult.PixelHeight)
	decoded, err := png.Decode(bytes.NewReader(pngResult.Content))
	require.NoError(t, err)
	require.Equal(t, 945, decoded.Bounds().Dx())
	require.Equal(t, 591, decoded.Bounds().Dy())
	physicalChunk := bytes.Index(pngResult.Content, []byte("pHYs"))
	require.Greater(t, physicalChunk, 0)
	require.Equal(t, uint32(11811), binary.BigEndian.Uint32(pngResult.Content[physicalChunk+4:physicalChunk+8]))

	pdfResult, err := renderer.Render(context.Background(), RenderRequest{Schema: schema, Format: "pdf"})
	require.NoError(t, err)
	require.Equal(t, "application/pdf", pdfResult.MIMEType)
	require.True(t, bytes.HasPrefix(pdfResult.Content, []byte("%PDF-1.4")))
	pageWidth, pageHeight := pdfPageSize(schema.Page)
	require.Contains(t, string(pdfResult.Content), fmt.Sprintf("/MediaBox [0 0 %s %s]", number(pageWidth), number(pageHeight)))
	require.True(t, bytes.HasSuffix(pdfResult.Content, []byte("%%EOF\n")))
}

func TestRendererBuildsMultiPageBatchPDF(t *testing.T) {
	schema := Schema{
		SchemaVersion: "1.0", Name: "批量标签",
		Page:     Page{Width: 40, Height: 20, Unit: "mm", DPI: 203, Background: "#ffffff"},
		Settings: Settings{GridSize: 1},
		Elements: []Element{{
			ID: "code", Type: "field", X: 2, Y: 2, Width: 36, Height: 8, Visible: true,
			Value: &ValueSource{Type: "field", FieldID: "code"},
			Style: &TextStyle{FontFamily: "sans-serif", FontSize: 4, FontWeight: 400, Color: "#111111", LineHeight: 1.2, TextAlign: "left", VerticalAlign: "top", Overflow: "clip"},
		}},
	}
	result, err := NewRenderer(nil).RenderBatch(context.Background(), schema, []RenderData{
		{Fields: map[string]any{"code": "A-001"}},
		{Fields: map[string]any{"code": "A-002"}},
		{Fields: map[string]any{"code": "A-003"}},
	})
	require.NoError(t, err)
	require.Equal(t, "application/pdf", result.MIMEType)
	require.Equal(t, 3, bytes.Count(result.Content, []byte("/Type /Page ")))
	require.Contains(t, string(result.Content), "/Count 3")
}

func TestRendererSVGGolden(t *testing.T) {
	schema := Schema{
		SchemaVersion: "1.0", Name: "设备标签",
		Page:     Page{Width: 40, Height: 20, Unit: "mm", DPI: 203, Background: "#ffffff"},
		Settings: Settings{GridSize: 1},
		Elements: []Element{
			{ID: "frame", Type: "rect", X: 1, Y: 1, Width: 38, Height: 18, Visible: true, ZIndex: 1, Fill: "#ffffff", Stroke: "#111111", StrokeWidth: 0.5, Radius: 1},
			{ID: "title", Type: "text", X: 3, Y: 3, Width: 34, Height: 6, Visible: true, ZIndex: 2,
				Value: &ValueSource{Type: "static", Value: "设备 A-01"},
				Style: &TextStyle{FontFamily: "Noto Sans CJK SC", FontSize: 4, FontWeight: 500, Color: "#111111", LineHeight: 1.2, TextAlign: "center", VerticalAlign: "middle", Overflow: "clip"}},
		},
	}
	result, err := NewRenderer(nil).Render(context.Background(), RenderRequest{Schema: schema, Format: "svg"})
	require.NoError(t, err)
	expected, err := os.ReadFile(filepath.Join("testdata", "basic.golden.svg"))
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(string(expected)), string(result.Content))
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
