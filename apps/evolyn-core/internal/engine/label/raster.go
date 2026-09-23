package label

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"math"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

//go:embed assets/NotoSansSC-VF.ttf
var notoSansSC []byte

var (
	embeddedFontOnce sync.Once
	embeddedFont     *opentype.Font
	embeddedFontErr  error
)

const maxRasterPixels = 20_000_000

func loadEmbeddedFont() (*opentype.Font, error) {
	embeddedFontOnce.Do(func() {
		embeddedFont, embeddedFontErr = opentype.Parse(notoSansSC)
	})
	if embeddedFontErr != nil {
		return nil, fmt.Errorf("parse embedded label font: %w", embeddedFontErr)
	}
	return embeddedFont, nil
}

func pagePixelSize(page Page) (int, int, float64, error) {
	scale := 1.0
	if page.Unit == "mm" {
		scale = float64(page.DPI) / 25.4
	}
	width := int(math.Round(page.Width * scale))
	height := int(math.Round(page.Height * scale))
	if width <= 0 || height <= 0 || width > maxRasterPixels/height {
		return 0, 0, 0, fmt.Errorf("label raster size exceeds %d pixels", maxRasterPixels)
	}
	return width, height, scale, nil
}

func renderPNG(document *ResolvedDocument) ([]byte, int, int, error) {
	canvas, width, height, err := renderRaster(document)
	if err != nil {
		return nil, 0, 0, err
	}
	var output bytes.Buffer
	if err := encodePNGWithDPI(&output, canvas, document.Page.DPI); err != nil {
		return nil, 0, 0, err
	}
	return output.Bytes(), width, height, nil
}

func renderRaster(document *ResolvedDocument) (*image.RGBA, int, int, error) {
	width, height, scale, err := pagePixelSize(document.Page)
	if err != nil {
		return nil, 0, 0, err
	}
	background, err := parseColor(document.Page.Background)
	if err != nil {
		return nil, 0, 0, err
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	fontFile, err := loadEmbeddedFont()
	if err != nil {
		return nil, 0, 0, err
	}
	for index := range document.Commands {
		command := &document.Commands[index]
		layer := image.NewRGBA(canvas.Bounds())
		if err := drawRasterCommand(layer, command, scale, fontFile); err != nil {
			return nil, 0, 0, fmt.Errorf("render element %s: %w", command.ID, err)
		}
		compositeLayer(canvas, layer, command, scale)
	}
	return canvas, width, height, nil
}

func drawRasterCommand(layer *image.RGBA, command *DrawCommand, scale float64, fontFile *opentype.Font) error {
	switch command.Type {
	case "text", "field":
		return drawRasterText(layer, command, scale, fontFile)
	case "qrcode":
		return drawRasterQR(layer, command, scale)
	case "image":
		return drawRasterImage(layer, command, scale)
	case "rect":
		return drawRasterRect(layer, command, scale)
	case "line":
		stroke, err := parseColor(command.Color)
		if err != nil {
			return err
		}
		drawLine(layer, px(command.X, scale), px(command.Y, scale), px(command.X+command.Width, scale), px(command.Y+command.Height, scale), max(1, px(command.StrokeWidth, scale)), stroke)
		return nil
	default:
		return fmt.Errorf("unsupported drawing command %q", command.Type)
	}
}

func drawRasterText(layer *image.RGBA, command *DrawCommand, scale float64, fontFile *opentype.Font) error {
	style := command.Style
	face, err := opentype.NewFace(fontFile, &opentype.FaceOptions{
		Size: style.FontSize * scale, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return fmt.Errorf("create label font face: %w", err)
	}
	defer face.Close()
	foreground, err := parseColor(style.Color)
	if err != nil {
		return err
	}
	x, y := px(command.X, scale), px(command.Y, scale)
	w, h := px(command.Width, scale), px(command.Height, scale)
	clip := image.Rect(x, y, x+w, y+h).Intersect(layer.Bounds())
	if clip.Empty() {
		return nil
	}
	drawer := &font.Drawer{Dst: layer.SubImage(clip).(draw.Image), Src: &image.Uniform{C: foreground}, Face: face}
	textWidth := drawer.MeasureString(command.Text).Round()
	textX := x
	if style.TextAlign == "center" {
		textX = x + (w-textWidth)/2
	} else if style.TextAlign == "right" {
		textX = x + w - textWidth
	}
	metrics := face.Metrics()
	ascent := metrics.Ascent.Round()
	baseline := y + ascent
	if style.VerticalAlign == "middle" {
		baseline = y + (h-(metrics.Ascent+metrics.Descent).Round())/2 + ascent
	} else if style.VerticalAlign == "bottom" {
		baseline = y + h - metrics.Descent.Round()
	}
	drawer.Dot = fixedPoint(textX, baseline)
	drawer.DrawString(command.Text)
	// x/image 当前不解析可变字体 weight 轴；用亚像素级横向叠绘保持 600+
	// 字重在 PNG/PDF 中可见，同时仍由同一嵌入字体提供确定性字形。
	if style.FontWeight >= 600 {
		drawer.Dot = fixedPoint(textX+max(1, int(math.Round(scale*0.12))), baseline)
		drawer.DrawString(command.Text)
	}
	return nil
}

func drawRasterQR(layer *image.RGBA, command *DrawCommand, scale float64) error {
	foreground, err := parseColor(command.QRCodeStyle.Foreground)
	if err != nil {
		return err
	}
	background, err := parseColor(command.QRCodeStyle.Background)
	if err != nil {
		return err
	}
	x, y := px(command.X, scale), px(command.Y, scale)
	w, h := px(command.Width, scale), px(command.Height, scale)
	side := min(w, h)
	x += (w - side) / 2
	y += (h - side) / 2
	draw.Draw(layer, image.Rect(x, y, x+side, y+side), &image.Uniform{C: background}, image.Point{}, draw.Src)
	quiet := command.QRCodeStyle.QuietZone
	modules := len(command.QRCode.Modules) + quiet*2
	for row := range command.QRCode.Modules {
		for column, dark := range command.QRCode.Modules[row] {
			if !dark {
				continue
			}
			x0 := x + int(math.Round(float64(column+quiet)*float64(side)/float64(modules)))
			y0 := y + int(math.Round(float64(row+quiet)*float64(side)/float64(modules)))
			x1 := x + int(math.Round(float64(column+quiet+1)*float64(side)/float64(modules)))
			y1 := y + int(math.Round(float64(row+quiet+1)*float64(side)/float64(modules)))
			draw.Draw(layer, image.Rect(x0, y0, x1, y1), &image.Uniform{C: foreground}, image.Point{}, draw.Src)
		}
	}
	return nil
}

func drawRasterImage(layer *image.RGBA, command *DrawCommand, scale float64) error {
	if strings.TrimSpace(command.ImageSource) == "" {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(command.ImageSource), "https://") {
		return fmt.Errorf("remote label images must be resolved through File Service before raster rendering")
	}
	comma := strings.IndexByte(command.ImageSource, ',')
	if comma < 0 || !strings.Contains(command.ImageSource[:comma], ";base64") {
		return fmt.Errorf("label image must be a supported base64 data URL")
	}
	raw, err := base64.StdEncoding.DecodeString(command.ImageSource[comma+1:])
	if err != nil {
		return fmt.Errorf("decode label image: %w", err)
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("decode label image pixels: %w", err)
	}
	target := image.Rect(px(command.X, scale), px(command.Y, scale), px(command.X+command.Width, scale), px(command.Y+command.Height, scale))
	if target.Empty() {
		return nil
	}
	sourceRect, destination := fitRectangles(source.Bounds(), target, command.ObjectFit)
	xdraw.CatmullRom.Scale(layer, destination, source, sourceRect, draw.Over, nil)
	return nil
}

func fitRectangles(source, target image.Rectangle, objectFit string) (image.Rectangle, image.Rectangle) {
	scaleX := float64(target.Dx()) / float64(source.Dx())
	scaleY := float64(target.Dy()) / float64(source.Dy())
	if objectFit == "cover" {
		targetRatio := float64(target.Dx()) / float64(target.Dy())
		sourceRatio := float64(source.Dx()) / float64(source.Dy())
		cropped := source
		if sourceRatio > targetRatio {
			width := int(math.Round(float64(source.Dy()) * targetRatio))
			cropped.Min.X += (source.Dx() - width) / 2
			cropped.Max.X = cropped.Min.X + width
		} else if sourceRatio < targetRatio {
			height := int(math.Round(float64(source.Dx()) / targetRatio))
			cropped.Min.Y += (source.Dy() - height) / 2
			cropped.Max.Y = cropped.Min.Y + height
		}
		return cropped, target
	}
	scale := math.Min(scaleX, scaleY)
	width, height := int(math.Round(float64(source.Dx())*scale)), int(math.Round(float64(source.Dy())*scale))
	x, y := target.Min.X+(target.Dx()-width)/2, target.Min.Y+(target.Dy()-height)/2
	return source, image.Rect(x, y, x+width, y+height)
}

func drawRasterRect(layer *image.RGBA, command *DrawCommand, scale float64) error {
	fill, err := parseColor(command.Fill)
	if err != nil {
		return err
	}
	rectangle := image.Rect(px(command.X, scale), px(command.Y, scale), px(command.X+command.Width, scale), px(command.Y+command.Height, scale))
	radius := max(0, px(command.Radius, scale))
	if command.Stroke != "" && command.StrokeWidth > 0 {
		stroke, err := parseColor(command.Stroke)
		if err != nil {
			return err
		}
		width := max(1, px(command.StrokeWidth, scale))
		drawRoundedRectangle(layer, rectangle, radius, stroke)
		inner := rectangle.Inset(width)
		if !inner.Empty() {
			drawRoundedRectangle(layer, inner, max(0, radius-width), fill)
		}
		return nil
	}
	drawRoundedRectangle(layer, rectangle, radius, fill)
	return nil
}

func drawRoundedRectangle(destination draw.Image, rectangle image.Rectangle, radius int, fill color.Color) {
	if radius <= 0 {
		draw.Draw(destination, rectangle, &image.Uniform{C: fill}, image.Point{}, draw.Over)
		return
	}
	radius = min(radius, min(rectangle.Dx(), rectangle.Dy())/2)
	centerLeft, centerRight := rectangle.Min.X+radius, rectangle.Max.X-radius-1
	centerTop, centerBottom := rectangle.Min.Y+radius, rectangle.Max.Y-radius-1
	for y := rectangle.Min.Y; y < rectangle.Max.Y; y++ {
		for x := rectangle.Min.X; x < rectangle.Max.X; x++ {
			cx := min(max(x, centerLeft), centerRight)
			cy := min(max(y, centerTop), centerBottom)
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= radius*radius {
				destination.Set(x, y, fill)
			}
		}
	}
}

func drawLine(destination draw.Image, x0, y0, x1, y1, width int, value color.Color) {
	dx, dy := math.Abs(float64(x1-x0)), math.Abs(float64(y1-y0))
	steps := int(math.Max(dx, dy))
	if steps == 0 {
		steps = 1
	}
	radius := width / 2
	for step := 0; step <= steps; step++ {
		t := float64(step) / float64(steps)
		x := int(math.Round(float64(x0) + float64(x1-x0)*t))
		y := int(math.Round(float64(y0) + float64(y1-y0)*t))
		draw.Draw(destination, image.Rect(x-radius, y-radius, x-radius+width, y-radius+width), &image.Uniform{C: value}, image.Point{}, draw.Over)
	}
}

func compositeLayer(canvas, layer *image.RGBA, command *DrawCommand, scale float64) {
	if command.Opacity < 1 {
		for index := 0; index < len(layer.Pix); index += 4 {
			layer.Pix[index] = uint8(math.Round(float64(layer.Pix[index]) * command.Opacity))
			layer.Pix[index+1] = uint8(math.Round(float64(layer.Pix[index+1]) * command.Opacity))
			layer.Pix[index+2] = uint8(math.Round(float64(layer.Pix[index+2]) * command.Opacity))
			layer.Pix[index+3] = uint8(math.Round(float64(layer.Pix[index+3]) * command.Opacity))
		}
	}
	if command.Rotation == 0 {
		draw.Draw(canvas, canvas.Bounds(), layer, image.Point{}, draw.Over)
		return
	}
	angle := command.Rotation * math.Pi / 180
	cosine, sine := math.Cos(angle), math.Sin(angle)
	cx := (command.X + command.Width/2) * scale
	cy := (command.Y + command.Height/2) * scale
	matrix := f64.Aff3{
		cosine, -sine, cx - cosine*cx + sine*cy,
		sine, cosine, cy - sine*cx - cosine*cy,
	}
	xdraw.BiLinear.Transform(canvas, matrix, layer, layer.Bounds(), draw.Over, nil)
}

func parseColor(value string) (color.NRGBA, error) {
	trimmed := strings.TrimPrefix(value, "#")
	if len(trimmed) != 6 && len(trimmed) != 8 {
		return color.NRGBA{}, fmt.Errorf("invalid label color %q", value)
	}
	var rgba uint64
	if _, err := fmt.Sscanf(trimmed, "%x", &rgba); err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid label color %q", value)
	}
	if len(trimmed) == 6 {
		return color.NRGBA{R: uint8(rgba >> 16), G: uint8(rgba >> 8), B: uint8(rgba), A: 255}, nil
	}
	return color.NRGBA{R: uint8(rgba >> 24), G: uint8(rgba >> 16), B: uint8(rgba >> 8), A: uint8(rgba)}, nil
}

func px(value, scale float64) int { return int(math.Round(value * scale)) }

func fixedPoint(x, y int) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
}

func encodePNGWithDPI(output *bytes.Buffer, source image.Image, dpi int) error {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		return err
	}
	raw := encoded.Bytes()
	if len(raw) < 33 {
		return fmt.Errorf("encoded PNG is incomplete")
	}
	pixelsPerMeter := uint32(math.Round(float64(dpi) / 0.0254))
	payload := make([]byte, 9)
	binary.BigEndian.PutUint32(payload[0:4], pixelsPerMeter)
	binary.BigEndian.PutUint32(payload[4:8], pixelsPerMeter)
	payload[8] = 1
	chunk := make([]byte, 4+4+len(payload)+4)
	binary.BigEndian.PutUint32(chunk[0:4], uint32(len(payload)))
	copy(chunk[4:8], "pHYs")
	copy(chunk[8:17], payload)
	binary.BigEndian.PutUint32(chunk[17:21], crc32.ChecksumIEEE(chunk[4:17]))
	output.Write(raw[:33])
	output.Write(chunk)
	output.Write(raw[33:])
	return nil
}
