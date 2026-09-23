package label

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// ErrQRGenerate 供平台层稳定映射二维码生成错误，避免依赖具体库文案。
var ErrQRGenerate = errors.New("label QR generation failed")

// ExpressionResolver 由平台公式引擎适配；内核禁止 eval。
type ExpressionResolver interface {
	Evaluate(ctx context.Context, expression string, data RenderData) (any, error)
}

// Renderer 从受控 Schema 生成 SVG；所有 XML 节点均由服务端构造。
type Renderer struct {
	expressions ExpressionResolver
}

func NewRenderer(expressions ExpressionResolver) *Renderer {
	return &Renderer{expressions: expressions}
}

func (r *Renderer) Render(ctx context.Context, request RenderRequest) (*RenderResult, error) {
	if request.Format == "" {
		request.Format = "svg"
	}
	if request.Format != "svg" {
		return nil, fmt.Errorf("unsupported label format %q", request.Format)
	}
	if issues := Validate(&request.Schema); len(issues) > 0 {
		return nil, fmt.Errorf("invalid label schema at %s: %s", issues[0].Path, issues[0].Message)
	}
	content, err := r.renderSVG(ctx, &request.Schema, request.Data)
	if err != nil {
		return nil, err
	}
	return &RenderResult{
		Content: content, MIMEType: "image/svg+xml; charset=utf-8",
		Width: request.Schema.Page.Width, Height: request.Schema.Page.Height,
	}, nil
}

func (r *Renderer) renderSVG(ctx context.Context, schema *Schema, data RenderData) ([]byte, error) {
	elements := append([]Element(nil), schema.Elements...)
	sort.SliceStable(elements, func(i, j int) bool { return elements[i].ZIndex < elements[j].ZIndex })
	var body bytes.Buffer
	for index := range elements {
		if !elements[index].Visible {
			continue
		}
		node, err := r.renderElement(ctx, &elements[index], data)
		if err != nil {
			return nil, fmt.Errorf("render element %s: %w", elements[index].ID, err)
		}
		body.WriteString(node)
	}
	unit := schema.Page.Unit
	return []byte(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%s%s" height="%s%s" viewBox="0 0 %s %s" role="img" aria-label="%s"><title>%s</title><rect width="100%%" height="100%%" fill="%s"/>%s</svg>`,
		number(schema.Page.Width), unit, number(schema.Page.Height), unit,
		number(schema.Page.Width), number(schema.Page.Height), escape(schema.Name), escape(schema.Name),
		escape(schema.Page.Background), body.String(),
	)), nil
}

func (r *Renderer) renderElement(ctx context.Context, element *Element, data RenderData) (string, error) {
	switch element.Type {
	case "text", "field":
		value, err := r.resolve(ctx, element.Value, data)
		if err != nil {
			return "", err
		}
		if element.Type == "field" && element.Label != "" {
			separator := element.Separator
			if separator == "" {
				separator = ":"
			}
			value = element.Label + separator + value
		}
		return renderText(element, value), nil
	case "qrcode":
		value, err := r.resolve(ctx, element.Value, data)
		if err != nil {
			return "", err
		}
		return renderQRCode(element, value)
	case "image":
		value, err := r.resolve(ctx, element.Value, data)
		if err != nil {
			return "", err
		}
		if !safeImageURL(value) {
			return "", nil
		}
		fit := "meet"
		if element.ObjectFit == "cover" {
			fit = "slice"
		}
		return fmt.Sprintf(`<image x="%s" y="%s" width="%s" height="%s" href="%s" preserveAspectRatio="xMidYMid %s"%s%s/>`,
			number(element.X), number(element.Y), number(element.Width), number(element.Height), escape(value), fit,
			rotation(element), opacity(element)), nil
	case "rect":
		stroke := ""
		if element.Stroke != "" {
			stroke = fmt.Sprintf(` stroke="%s"`, escape(element.Stroke))
		}
		strokeWidth := ""
		if element.StrokeWidth > 0 {
			strokeWidth = fmt.Sprintf(` stroke-width="%s"`, number(element.StrokeWidth))
		}
		return fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"%s%s%s%s/>`,
			number(element.X), number(element.Y), number(element.Width), number(element.Height), number(element.Radius),
			escape(element.Fill), stroke, strokeWidth, rotation(element), opacity(element)), nil
	case "line":
		return fmt.Sprintf(`<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s"%s%s/>`,
			number(element.X), number(element.Y), number(element.X+element.Width), number(element.Y+element.Height),
			escape(element.Color), number(element.StrokeWidth), rotation(element), opacity(element)), nil
	default:
		return "", fmt.Errorf("unsupported element type %q", element.Type)
	}
}

func (r *Renderer) resolve(ctx context.Context, source *ValueSource, data RenderData) (string, error) {
	if source == nil {
		return "", nil
	}
	var value any
	switch source.Type {
	case "static":
		value = source.Value
	case "field":
		value = data.Fields[source.FieldID]
	case "system":
		value = data.System[source.Key]
	case "expression":
		if r.expressions == nil {
			return "", fmt.Errorf("expression resolver is not configured")
		}
		resolved, err := r.expressions.Evaluate(ctx, source.Expression, data)
		if err != nil {
			return "", err
		}
		value = resolved
	}
	return stringify(value), nil
}

func renderText(element *Element, value string) string {
	style := element.Style
	anchor, x := "start", element.X
	if style.TextAlign == "center" {
		anchor, x = "middle", element.X+element.Width/2
	} else if style.TextAlign == "right" {
		anchor, x = "end", element.X+element.Width
	}
	y := element.Y + style.FontSize
	if style.VerticalAlign == "middle" {
		y = element.Y + element.Height/2 + style.FontSize*0.35
	} else if style.VerticalAlign == "bottom" {
		y = element.Y + element.Height
	}
	clipID := "label-clip-" + safeID(element.ID)
	return fmt.Sprintf(`<g%s%s><clipPath id="%s"><rect x="%s" y="%s" width="%s" height="%s"/></clipPath><text x="%s" y="%s" clip-path="url(#%s)" fill="%s" font-family="%s" font-size="%s" font-weight="%d" text-anchor="%s">%s</text></g>`,
		rotation(element), opacity(element), clipID, number(element.X), number(element.Y), number(element.Width), number(element.Height),
		number(x), number(y), clipID, escape(style.Color), escape(style.FontFamily), number(style.FontSize), style.FontWeight, anchor, escape(value))
}

func renderQRCode(element *Element, value string) (string, error) {
	level := qrcode.Medium
	switch element.Options.ErrorCorrection {
	case "L":
		level = qrcode.Low
	case "Q":
		level = qrcode.High
	case "H":
		level = qrcode.Highest
	}
	code, err := qrcode.New(value, level)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrQRGenerate, err)
	}
	// 静区由 LabelSchema 显式控制，关闭库自带边框避免重复留白。
	code.DisableBorder = true
	bitmap := code.Bitmap()
	quiet := element.Options.QuietZone
	size := len(bitmap) + quiet*2
	if size <= 0 {
		return "", fmt.Errorf("empty QR bitmap")
	}
	var paths strings.Builder
	for row := range bitmap {
		for column, dark := range bitmap[row] {
			if dark {
				fmt.Fprintf(&paths, "M%d %dh1v1h-1z", column+quiet, row+quiet)
			}
		}
	}
	return fmt.Sprintf(`<svg x="%s" y="%s" width="%s" height="%s" viewBox="0 0 %d %d" preserveAspectRatio="xMidYMid meet"%s%s><rect width="%d" height="%d" fill="%s"/><path fill="%s" d="%s"/></svg>`,
		number(element.X), number(element.Y), number(element.Width), number(element.Height), size, size,
		rotation(element), opacity(element), size, size, escape(element.Options.Background), escape(element.Options.Foreground), paths.String()), nil
}

func stringify(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	default:
		encoded, _ := json.Marshal(value)
		return string(encoded)
	}
}

func number(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }
func escape(value string) string  { return html.EscapeString(value) }

func safeID(value string) string {
	var result strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			result.WriteRune(char)
		} else {
			result.WriteByte('-')
		}
	}
	return result.String()
}

func safeImageURL(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "data:image/png;base64,") || strings.HasPrefix(trimmed, "data:image/jpeg;base64,") || strings.HasPrefix(trimmed, "data:image/webp;base64,")
}

func rotation(element *Element) string {
	if element.Rotation == 0 {
		return ""
	}
	return fmt.Sprintf(` transform="rotate(%s %s %s)"`, number(element.Rotation), number(element.X+element.Width/2), number(element.Y+element.Height/2))
}

func opacity(element *Element) string {
	if element.Opacity == nil {
		return ""
	}
	return fmt.Sprintf(` opacity="%s"`, number(*element.Opacity))
}
