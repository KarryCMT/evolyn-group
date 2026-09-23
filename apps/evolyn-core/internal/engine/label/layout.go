package label

import (
	"context"
	"fmt"
	"sort"
)

// ResolvedDocument 是所有输出后端共享的不可变绘制事实源。
type ResolvedDocument struct {
	Name     string
	Page     Page
	Commands []DrawCommand
}

// DrawCommand 只保存已解析值和绘制参数，不再含字段、系统值或表达式绑定。
type DrawCommand struct {
	ID          string
	Type        string
	X           float64
	Y           float64
	Width       float64
	Height      float64
	Rotation    float64
	Opacity     float64
	Text        string
	Style       *TextStyle
	QRCode      *QRCodeMatrix
	QRCodeStyle *QRCodeOptions
	ImageSource string
	ObjectFit   string
	Fill        string
	Stroke      string
	StrokeWidth float64
	Radius      float64
	Color       string
}

func (r *Renderer) resolve(ctx context.Context, schema *Schema, data RenderData) (*ResolvedDocument, error) {
	elements := append([]Element(nil), schema.Elements...)
	sort.SliceStable(elements, func(i, j int) bool { return elements[i].ZIndex < elements[j].ZIndex })
	document := &ResolvedDocument{Name: schema.Name, Page: schema.Page, Commands: make([]DrawCommand, 0, len(elements))}
	for index := range elements {
		element := &elements[index]
		if !element.Visible {
			continue
		}
		command := DrawCommand{
			ID: element.ID, Type: element.Type, X: element.X, Y: element.Y,
			Width: element.Width, Height: element.Height, Rotation: element.Rotation,
			Opacity: 1, Style: element.Style, ObjectFit: element.ObjectFit,
			Fill: element.Fill, Stroke: element.Stroke, StrokeWidth: element.StrokeWidth,
			Radius: element.Radius, Color: element.Color,
		}
		if element.Opacity != nil {
			command.Opacity = *element.Opacity
		}
		switch element.Type {
		case "text", "field", "image", "qrcode":
			value, err := r.resolveValue(ctx, element.Value, data)
			if err != nil {
				return nil, fmt.Errorf("resolve element %s: %w", element.ID, err)
			}
			if element.Type == "field" && element.Label != "" {
				separator := element.Separator
				if separator == "" {
					separator = ":"
				}
				value = element.Label + separator + value
			}
			if element.Type == "image" {
				command.ImageSource = value
			} else if element.Type == "qrcode" {
				matrix, err := r.qr.Generate(value, *element.Options)
				if err != nil {
					return nil, fmt.Errorf("render element %s: %w", element.ID, err)
				}
				command.QRCode, command.QRCodeStyle = matrix, element.Options
			} else {
				command.Text = value
			}
		}
		document.Commands = append(document.Commands, command)
	}
	return document, nil
}
