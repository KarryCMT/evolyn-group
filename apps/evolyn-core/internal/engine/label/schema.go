// Package label 提供与平台、HTTP、数据库无关的标签协议校验和值解析/SVG 渲染内核。
package label

// Schema 是 LabelSchema V1 的服务端镜像；它是设计、发布与渲染的唯一事实源。
type Schema struct {
	SchemaVersion string    `json:"schemaVersion"`
	ID            string    `json:"id,omitempty"`
	Name          string    `json:"name"`
	Page          Page      `json:"page"`
	Source        *Source   `json:"source,omitempty"`
	Elements      []Element `json:"elements"`
	Settings      Settings  `json:"settings"`
}

type Page struct {
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Unit       string  `json:"unit"`
	DPI        int     `json:"dpi"`
	Background string  `json:"background"`
}

type Source struct {
	Type   string `json:"type"`
	AppID  string `json:"appId,omitempty"`
	FormID string `json:"formId,omitempty"`
}

type Settings struct {
	SnapToGrid bool    `json:"snapToGrid"`
	GridSize   float64 `json:"gridSize"`
	ShowGrid   bool    `json:"showGrid"`
}

// Element 使用单结构承载受控元素并按 Type 解释对应字段，避免任意 SVG/XML 注入。
type Element struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Name     string   `json:"name,omitempty"`
	X        float64  `json:"x"`
	Y        float64  `json:"y"`
	Width    float64  `json:"width"`
	Height   float64  `json:"height"`
	Rotation float64  `json:"rotation"`
	ZIndex   int      `json:"zIndex"`
	Visible  bool     `json:"visible"`
	Locked   bool     `json:"locked"`
	Opacity  *float64 `json:"opacity,omitempty"`

	Value       *ValueSource   `json:"value,omitempty"`
	Style       *TextStyle     `json:"style,omitempty"`
	Label       string         `json:"label,omitempty"`
	Separator   string         `json:"separator,omitempty"`
	Formatter   *Formatter     `json:"formatter,omitempty"`
	Options     *QRCodeOptions `json:"options,omitempty"`
	ObjectFit   string         `json:"objectFit,omitempty"`
	Fill        string         `json:"fill,omitempty"`
	Stroke      string         `json:"stroke,omitempty"`
	StrokeWidth float64        `json:"strokeWidth,omitempty"`
	Radius      float64        `json:"radius,omitempty"`
	Color       string         `json:"color,omitempty"`
}

type ValueSource struct {
	Type       string `json:"type"`
	Value      string `json:"value,omitempty"`
	FieldID    string `json:"fieldId,omitempty"`
	Key        string `json:"key,omitempty"`
	Expression string `json:"expression,omitempty"`
}

type TextStyle struct {
	FontFamily    string  `json:"fontFamily"`
	FontSize      float64 `json:"fontSize"`
	FontWeight    int     `json:"fontWeight"`
	Color         string  `json:"color"`
	LineHeight    float64 `json:"lineHeight"`
	TextAlign     string  `json:"textAlign"`
	VerticalAlign string  `json:"verticalAlign"`
	Overflow      string  `json:"overflow"`
}

type Formatter struct {
	Type    string `json:"type,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

type QRCodeOptions struct {
	ErrorCorrection string `json:"errorCorrection"`
	QuietZone       int    `json:"quietZone"`
	Foreground      string `json:"foreground"`
	Background      string `json:"background"`
}

// RenderData 是已经经过平台权限裁剪和字段身份映射后的可信渲染数据。
type RenderData struct {
	Fields map[string]any
	System map[string]any
}

type RenderRequest struct {
	Schema Schema
	Data   RenderData
	Format string
}

type RenderResult struct {
	Content     []byte
	MIMEType    string
	Width       float64
	Height      float64
	PixelWidth  int
	PixelHeight int
	DPI         int
}

// Issue 使用稳定路径和代码返回协议问题，供前端定位到具体元素。
type Issue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
