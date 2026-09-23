package label

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// QRCodeMatrix 与具体二维码库隔离，渲染后端只消费稳定的模块矩阵。
type QRCodeMatrix struct {
	Modules [][]bool
}

type QRCodeGenerator interface {
	Generate(content string, options QRCodeOptions) (*QRCodeMatrix, error)
}

type goQRCodeGenerator struct{}

func newQRCodeGenerator() QRCodeGenerator { return goQRCodeGenerator{} }

func (goQRCodeGenerator) Generate(content string, options QRCodeOptions) (*QRCodeMatrix, error) {
	level := qrcode.Medium
	switch options.ErrorCorrection {
	case "L":
		level = qrcode.Low
	case "Q":
		level = qrcode.High
	case "H":
		level = qrcode.Highest
	}
	code, err := qrcode.New(content, level)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQRGenerate, err)
	}
	// 静区由 Schema 显式控制，关闭库边框避免输出后端重复留白。
	code.DisableBorder = true
	return &QRCodeMatrix{Modules: code.Bitmap()}, nil
}
