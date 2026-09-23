package label

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
)

// renderPDF 使用实际物理页尺寸，并将同一绘制指令生成的确定性位图嵌入单页 PDF。
// 标签 PDF 的 MediaBox 不使用 A4，避免打印驱动二次缩放标签。
func renderPDF(document *ResolvedDocument) ([]byte, int, int, error) {
	canvas, pixelWidth, pixelHeight, err := renderRaster(document)
	if err != nil {
		return nil, 0, 0, err
	}
	pageWidth, pageHeight := pdfPageSize(document.Page)
	imageData, err := compressPDFImage(canvas)
	if err != nil {
		return nil, 0, 0, err
	}
	content := []byte(fmt.Sprintf("q\n%s 0 0 %s 0 0 cm\n/Im0 Do\nQ\n", number(pageWidth), number(pageHeight)))
	contentData, err := flate(content)
	if err != nil {
		return nil, 0, 0, err
	}

	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] /Resources << /XObject << /Im0 4 0 R >> >> /Contents 5 0 R >>", number(pageWidth), number(pageHeight))),
		pdfStream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode", pixelWidth, pixelHeight), imageData),
		pdfStream("/Filter /FlateDecode", contentData),
	}
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n", index+1)
		output.Write(object)
		output.WriteString("\nendobj\n")
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n", len(offsets))
	output.WriteString("0000000000 65535 f \n")
	for index := 1; index < len(offsets); index++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return output.Bytes(), pixelWidth, pixelHeight, nil
}

func pdfPageSize(page Page) (float64, float64) {
	if page.Unit == "mm" {
		return page.Width / 25.4 * 72, page.Height / 25.4 * 72
	}
	return page.Width / float64(page.DPI) * 72, page.Height / float64(page.DPI) * 72
}

func compressPDFImage(source *image.RGBA) ([]byte, error) {
	raw := make([]byte, 0, source.Bounds().Dx()*source.Bounds().Dy()*3)
	for y := source.Bounds().Min.Y; y < source.Bounds().Max.Y; y++ {
		for x := source.Bounds().Min.X; x < source.Bounds().Max.X; x++ {
			r, g, b, a := source.At(x, y).RGBA()
			// PDF DeviceRGB 没有透明通道；透明像素按白底合成，输出保持可打印。
			r = r + (0xffff-r)*(0xffff-a)/0xffff
			g = g + (0xffff-g)*(0xffff-a)/0xffff
			b = b + (0xffff-b)*(0xffff-a)/0xffff
			raw = append(raw, byte(r>>8), byte(g>>8), byte(b>>8))
		}
	}
	return flate(raw)
}

func flate(raw []byte) ([]byte, error) {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(raw); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func pdfStream(dictionary string, content []byte) []byte {
	var stream bytes.Buffer
	fmt.Fprintf(&stream, "<< %s /Length %d >>\nstream\n", dictionary, len(content))
	stream.Write(content)
	stream.WriteString("\nendstream")
	return stream.Bytes()
}
