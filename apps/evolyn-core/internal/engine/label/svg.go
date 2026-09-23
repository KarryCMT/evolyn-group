package label

import (
	"bytes"
	"fmt"
	"html"
	"strings"
)

func renderSVG(document *ResolvedDocument) ([]byte, error) {
	var body bytes.Buffer
	for index := range document.Commands {
		node, err := renderSVGCommand(&document.Commands[index])
		if err != nil {
			return nil, err
		}
		body.WriteString(node)
	}
	unit := document.Page.Unit
	return []byte(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%s%s" height="%s%s" viewBox="0 0 %s %s" role="img" aria-label="%s"><title>%s</title><rect width="100%%" height="100%%" fill="%s"/>%s</svg>`,
		number(document.Page.Width), unit, number(document.Page.Height), unit,
		number(document.Page.Width), number(document.Page.Height), escape(document.Name), escape(document.Name),
		escape(document.Page.Background), body.String(),
	)), nil
}

func renderSVGCommand(command *DrawCommand) (string, error) {
	switch command.Type {
	case "text", "field":
		return renderSVGText(command), nil
	case "qrcode":
		return renderSVGQRCode(command), nil
	case "image":
		if !safeImageURL(command.ImageSource) {
			return "", nil
		}
		fit := "meet"
		if command.ObjectFit == "cover" {
			fit = "slice"
		}
		return fmt.Sprintf(`<image x="%s" y="%s" width="%s" height="%s" href="%s" preserveAspectRatio="xMidYMid %s"%s%s/>`,
			number(command.X), number(command.Y), number(command.Width), number(command.Height), escape(command.ImageSource), fit,
			svgRotation(command), svgOpacity(command)), nil
	case "rect":
		stroke := ""
		if command.Stroke != "" {
			stroke = fmt.Sprintf(` stroke="%s"`, escape(command.Stroke))
		}
		strokeWidth := ""
		if command.StrokeWidth > 0 {
			strokeWidth = fmt.Sprintf(` stroke-width="%s"`, number(command.StrokeWidth))
		}
		return fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"%s%s%s%s/>`,
			number(command.X), number(command.Y), number(command.Width), number(command.Height), number(command.Radius),
			escape(command.Fill), stroke, strokeWidth, svgRotation(command), svgOpacity(command)), nil
	case "line":
		return fmt.Sprintf(`<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s"%s%s/>`,
			number(command.X), number(command.Y), number(command.X+command.Width), number(command.Y+command.Height),
			escape(command.Color), number(command.StrokeWidth), svgRotation(command), svgOpacity(command)), nil
	default:
		return "", fmt.Errorf("unsupported drawing command %q", command.Type)
	}
}

func renderSVGText(command *DrawCommand) string {
	style := command.Style
	anchor, x := "start", command.X
	if style.TextAlign == "center" {
		anchor, x = "middle", command.X+command.Width/2
	} else if style.TextAlign == "right" {
		anchor, x = "end", command.X+command.Width
	}
	y := command.Y + style.FontSize
	if style.VerticalAlign == "middle" {
		y = command.Y + command.Height/2 + style.FontSize*0.35
	} else if style.VerticalAlign == "bottom" {
		y = command.Y + command.Height
	}
	clipID := "label-clip-" + safeID(command.ID)
	return fmt.Sprintf(`<g%s%s><clipPath id="%s"><rect x="%s" y="%s" width="%s" height="%s"/></clipPath><text x="%s" y="%s" clip-path="url(#%s)" fill="%s" font-family="%s" font-size="%s" font-weight="%d" text-anchor="%s">%s</text></g>`,
		svgRotation(command), svgOpacity(command), clipID, number(command.X), number(command.Y), number(command.Width), number(command.Height),
		number(x), number(y), clipID, escape(style.Color), escape(style.FontFamily), number(style.FontSize), style.FontWeight, anchor, escape(command.Text))
}

func renderSVGQRCode(command *DrawCommand) string {
	quiet := command.QRCodeStyle.QuietZone
	size := len(command.QRCode.Modules) + quiet*2
	var paths strings.Builder
	for row := range command.QRCode.Modules {
		for column, dark := range command.QRCode.Modules[row] {
			if dark {
				fmt.Fprintf(&paths, "M%d %dh1v1h-1z", column+quiet, row+quiet)
			}
		}
	}
	return fmt.Sprintf(`<svg x="%s" y="%s" width="%s" height="%s" viewBox="0 0 %d %d" preserveAspectRatio="xMidYMid meet"%s%s><rect width="%d" height="%d" fill="%s"/><path fill="%s" d="%s"/></svg>`,
		number(command.X), number(command.Y), number(command.Width), number(command.Height), size, size,
		svgRotation(command), svgOpacity(command), size, size, escape(command.QRCodeStyle.Background), escape(command.QRCodeStyle.Foreground), paths.String())
}

func escape(value string) string { return html.EscapeString(value) }

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
	return strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "data:image/png;base64,") ||
		strings.HasPrefix(trimmed, "data:image/jpeg;base64,") || strings.HasPrefix(trimmed, "data:image/webp;base64,")
}

func svgRotation(command *DrawCommand) string {
	if command.Rotation == 0 {
		return ""
	}
	return fmt.Sprintf(` transform="rotate(%s %s %s)"`, number(command.Rotation), number(command.X+command.Width/2), number(command.Y+command.Height/2))
}

func svgOpacity(command *DrawCommand) string {
	if command.Opacity == 1 {
		return ""
	}
	return fmt.Sprintf(` opacity="%s"`, number(command.Opacity))
}
