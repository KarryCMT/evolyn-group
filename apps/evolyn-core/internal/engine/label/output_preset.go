package label

import (
	"errors"
	"math"
)

var ErrOutputPresetNotFound = errors.New("label output preset not found")

// OutputPresets 返回协议中的发布预设。存量快照没有该字段时投影 original，
// 使历史模板保持原始画布输出且无需改写不可变版本。
func OutputPresets(schema *Schema) []OutputPreset {
	if schema == nil {
		return nil
	}
	if len(schema.Settings.OutputPresets) > 0 {
		return append([]OutputPreset(nil), schema.Settings.OutputPresets...)
	}
	return []OutputPreset{{
		ID: "original", Name: "原始尺寸", Width: schema.Page.Width, Height: schema.Page.Height,
		Unit: schema.Page.Unit, DPI: schema.Page.DPI,
	}}
}

// StandardOutputPresets 是快速设置页与服务端新建草稿共用的标准纸张规格。
// 仅在 3:2/2:3 画布上提供大小尺寸；其他比例保守回退原始尺寸。
func StandardOutputPresets(page Page) []OutputPreset {
	ratio := page.Width / page.Height
	if math.Abs(ratio-1.5) <= 0.000001 {
		return []OutputPreset{
			{ID: "large", Name: "大尺寸", Width: 150, Height: 100, Unit: page.Unit, DPI: page.DPI},
			{ID: "small", Name: "小尺寸", Width: 60, Height: 40, Unit: page.Unit, DPI: page.DPI},
		}
	}
	if math.Abs(ratio-2.0/3.0) <= 0.000001 {
		return []OutputPreset{
			{ID: "large", Name: "大尺寸", Width: 100, Height: 150, Unit: page.Unit, DPI: page.DPI},
			{ID: "small", Name: "小尺寸", Width: 40, Height: 60, Unit: page.Unit, DPI: page.DPI},
		}
	}
	return []OutputPreset{{ID: "original", Name: "原始尺寸", Width: page.Width, Height: page.Height, Unit: page.Unit, DPI: page.DPI}}
}

// EnsureOutputPresets 把存量草稿的隐式 original 显式写入后续发布快照。
func EnsureOutputPresets(schema *Schema) {
	if schema != nil && len(schema.Settings.OutputPresets) == 0 {
		schema.Settings.OutputPresets = OutputPresets(schema)
	}
}

// PixelSize 使用唯一舍入规则换算前端展示和位图输出尺寸。
func PixelSize(preset OutputPreset) (int, int) {
	if preset.Unit == "px" {
		return int(math.Round(preset.Width)), int(math.Round(preset.Height))
	}
	return int(math.Round(preset.Width / 25.4 * float64(preset.DPI))),
		int(math.Round(preset.Height / 25.4 * float64(preset.DPI)))
}

// WithOutputPreset 复制并等比缩放 Schema，不修改不可变发布快照。
func WithOutputPreset(schema Schema, presetID string) (Schema, OutputPreset, error) {
	var selected *OutputPreset
	for _, preset := range OutputPresets(&schema) {
		if preset.ID == presetID {
			copy := preset
			selected = &copy
			break
		}
	}
	if selected == nil {
		return Schema{}, OutputPreset{}, ErrOutputPresetNotFound
	}
	scale := selected.Width / schema.Page.Width
	scaled := schema
	scaled.Page.Width = selected.Width
	scaled.Page.Height = selected.Height
	scaled.Page.Unit = selected.Unit
	scaled.Page.DPI = selected.DPI
	scaled.Settings.OutputPresets = append([]OutputPreset(nil), schema.Settings.OutputPresets...)
	scaled.Elements = make([]Element, len(schema.Elements))
	for index := range schema.Elements {
		element := schema.Elements[index]
		element.X *= scale
		element.Y *= scale
		element.Width *= scale
		element.Height *= scale
		element.StrokeWidth *= scale
		element.Radius *= scale
		if element.Style != nil {
			style := *element.Style
			style.FontSize *= scale
			element.Style = &style
		}
		scaled.Elements[index] = element
	}
	return scaled, *selected, nil
}
