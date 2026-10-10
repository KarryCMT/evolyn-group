package label

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutputPresetProjectionAndScaling(t *testing.T) {
	schema := Schema{
		SchemaVersion: "1.0", Name: "标签",
		Page: Page{Width: 90, Height: 60, Unit: "mm", DPI: 300, Background: "#ffffff"},
		Settings: Settings{GridSize: 1, OutputPresets: []OutputPreset{
			{ID: "large", Name: "大尺寸", Width: 150, Height: 100, Unit: "mm", DPI: 300},
			{ID: "small", Name: "小尺寸", Width: 60, Height: 40, Unit: "mm", DPI: 300},
		}},
		Elements: []Element{{
			ID: "title", Type: "text", X: 6, Y: 3, Width: 30, Height: 6, Visible: true,
			Value: &ValueSource{Type: "static", Value: "测试"},
			Style: &TextStyle{FontFamily: "sans-serif", FontSize: 3, FontWeight: 400, Color: "#111111", LineHeight: 1.2, TextAlign: "left", VerticalAlign: "top", Overflow: "clip"},
		}},
	}

	scaled, preset, err := WithOutputPreset(schema, "large")
	require.NoError(t, err)
	require.Equal(t, "large", preset.ID)
	require.Equal(t, 150.0, scaled.Page.Width)
	require.Equal(t, 10.0, scaled.Elements[0].X)
	require.Equal(t, 5.0, scaled.Elements[0].Style.FontSize)
	require.Equal(t, 1772, func() int { width, _ := PixelSize(preset); return width }())
	require.Equal(t, 1181, func() int { _, height := PixelSize(preset); return height }())
	// 缩放使用深拷贝，发布快照几何不被任务修改。
	require.Equal(t, 6.0, schema.Elements[0].X)
	require.Equal(t, 3.0, schema.Elements[0].Style.FontSize)

	svg, err := NewRenderer(nil).Render(context.Background(), RenderRequest{Schema: scaled, Format: "svg"})
	require.NoError(t, err)
	require.Contains(t, string(svg.Content), `width="150mm"`)
	pdf, err := NewRenderer(nil).RenderBatch(context.Background(), scaled, []RenderData{{}, {}})
	require.NoError(t, err)
	require.Contains(t, string(pdf.Content), "/MediaBox [0 0 425.1968503937008 283.46456692913387]")
	require.Equal(t, 2, strings.Count(string(pdf.Content), "/Type /Page "))
}

func TestOutputPresetValidationAndLegacyProjection(t *testing.T) {
	schema := Schema{
		SchemaVersion: "1.0", Name: "标签",
		Page:     Page{Width: 90, Height: 60, Unit: "mm", DPI: 300, Background: "#ffffff"},
		Settings: Settings{GridSize: 1},
	}
	require.Equal(t, "original", OutputPresets(&schema)[0].ID)
	EnsureOutputPresets(&schema)
	require.Len(t, schema.Settings.OutputPresets, 1)

	schema.Settings.OutputPresets = []OutputPreset{{
		ID: "bad", Name: "错误比例", Width: 100, Height: 100, Unit: "mm", DPI: 300,
	}}
	issues := Validate(&schema)
	require.Contains(t, issues, Issue{Path: "$.settings.outputPresets[0]", Code: "OUTPUT_PRESET_RATIO_INVALID", Message: "输出尺寸预设必须与设计画布保持相同宽高比"})
}
