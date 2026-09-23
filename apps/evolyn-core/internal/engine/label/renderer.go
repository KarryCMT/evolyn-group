package label

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// ErrQRGenerate 供平台层稳定映射二维码生成错误，避免依赖具体库文案。
var ErrQRGenerate = errors.New("label QR generation failed")

// ExpressionResolver 由平台公式引擎适配；内核禁止 eval。
type ExpressionResolver interface {
	Evaluate(ctx context.Context, expression string, data RenderData) (any, error)
}

// Renderer 先把 Schema 和可信数据解析为统一绘制指令，再交给各格式后端输出。
// 该边界保证 SVG、PNG 与 PDF 不会分别解释业务 Schema。
type Renderer struct {
	expressions ExpressionResolver
	qr          QRCodeGenerator
}

func NewRenderer(expressions ExpressionResolver) *Renderer {
	return &Renderer{expressions: expressions, qr: newQRCodeGenerator()}
}

func (r *Renderer) Render(ctx context.Context, request RenderRequest) (*RenderResult, error) {
	if request.Format == "" {
		request.Format = "svg"
	}
	if issues := Validate(&request.Schema); len(issues) > 0 {
		return nil, fmt.Errorf("invalid label schema at %s: %s", issues[0].Path, issues[0].Message)
	}
	document, err := r.resolve(ctx, &request.Schema, request.Data)
	if err != nil {
		return nil, err
	}

	result := &RenderResult{
		Width: request.Schema.Page.Width, Height: request.Schema.Page.Height,
		DPI: request.Schema.Page.DPI,
	}
	switch request.Format {
	case "svg":
		result.Content, err = renderSVG(document)
		result.MIMEType = "image/svg+xml; charset=utf-8"
	case "png":
		result.Content, result.PixelWidth, result.PixelHeight, err = renderPNG(document)
		result.MIMEType = "image/png"
	case "pdf":
		result.Content, result.PixelWidth, result.PixelHeight, err = renderPDF(document)
		result.MIMEType = "application/pdf"
	default:
		return nil, fmt.Errorf("unsupported label format %q", request.Format)
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Renderer) resolveValue(ctx context.Context, source *ValueSource, data RenderData) (string, error) {
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
