// Package dashboard 定义业务仪表盘的纯协议模型与规范化器。
// 本包不依赖 Gin、GORM、Redis 或 HTTP，保存与发布都必须复用这里的校验结论。
package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	CurrentVersion = 1
	DesktopColumns = 12
)

// Issue 是可稳定定位到 JSON 路径的协议问题。
type Issue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GridRect 是 GridStack 无关的持久化布局语义。
type GridRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type CanvasSettings struct {
	Columns   int `json:"columns"`
	RowHeight int `json:"rowHeight"`
}

type Settings struct {
	Desktop CanvasSettings `json:"desktop"`
}

// IdentifiedConfig 为后续 Dataset、筛选器和联动协议预留稳定身份。
// 阶段一只固化身份与扩展配置容器，具体业务字段由后续协议版本扩展。
type IdentifiedConfig struct {
	ID string `json:"id"`
}

type Widget struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Title    string          `json:"title,omitempty"`
	Layout   GridRect        `json:"layout"`
	Dataset  string          `json:"datasetId,omitempty"`
	Settings json.RawMessage `json:"settings,omitempty"`
}

type PublishScope struct {
	Type string `json:"type"`
}

// Document 是 dashboard v1 的最小完整文档；空集合必须显式存在，避免不同
// 客户端用缺省键表达不同语义。
type Document struct {
	Version      int                `json:"version"`
	Settings     Settings           `json:"settings"`
	Datasets     []IdentifiedConfig `json:"datasets"`
	Widgets      []Widget           `json:"widgets"`
	Filters      []IdentifiedConfig `json:"filters"`
	Interactions []IdentifiedConfig `json:"interactions"`
	PublishScope PublishScope       `json:"publishScope"`
}

type NormalizeResult struct {
	Document Document        `json:"document"`
	Content  json.RawMessage `json:"content"`
	Issues   []Issue         `json:"issues"`
}

// EmptyDocument 返回当前版本的最小合法空草稿。
func EmptyDocument() Document {
	return Document{
		Version:  CurrentVersion,
		Settings: Settings{Desktop: CanvasSettings{Columns: DesktopColumns, RowHeight: 80}},
		Datasets: []IdentifiedConfig{}, Widgets: []Widget{}, Filters: []IdentifiedConfig{},
		Interactions: []IdentifiedConfig{}, PublishScope: PublishScope{Type: "all"},
	}
}

func EmptyContent() json.RawMessage {
	raw, _ := json.Marshal(EmptyDocument())
	return raw
}

// Normalize 严格解码并输出确定性的 JSON。任何 issue 都表示结果不可持久化。
func Normalize(raw []byte) NormalizeResult {
	result := NormalizeResult{Issues: []Issue{}}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result.Document); err != nil {
		result.Issues = append(result.Issues, Issue{Path: "$", Code: "DASHBOARD_DOCUMENT_INVALID", Message: "仪表盘文档必须是合法且受支持的 JSON 对象"})
		return result
	}
	if err := ensureEOF(decoder); err != nil {
		result.Issues = append(result.Issues, Issue{Path: "$", Code: "DASHBOARD_DOCUMENT_INVALID", Message: "仪表盘文档只能包含一个 JSON 对象"})
		return result
	}
	validate(&result.Document, &result.Issues)
	if len(result.Issues) == 0 {
		result.Content, _ = json.Marshal(result.Document)
	}
	return result
}

func ensureEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing json value")
		}
		return err
	}
	return nil
}

func validate(doc *Document, issues *[]Issue) {
	if doc.Version != CurrentVersion {
		*issues = append(*issues, Issue{Path: "version", Code: "DASHBOARD_VERSION_UNSUPPORTED", Message: fmt.Sprintf("不支持的仪表盘协议版本：%d", doc.Version)})
		return
	}
	if doc.Settings.Desktop.Columns != DesktopColumns || doc.Settings.Desktop.RowHeight <= 0 {
		*issues = append(*issues, Issue{Path: "settings.desktop", Code: "DASHBOARD_LAYOUT_INVALID", Message: "桌面画布必须使用 12 列且行高为正数"})
	}
	validateIDs("datasets", idsOf(doc.Datasets), issues)
	validateIDs("filters", idsOf(doc.Filters), issues)
	validateIDs("interactions", idsOf(doc.Interactions), issues)
	widgetIDs := make([]string, 0, len(doc.Widgets))
	for i := range doc.Widgets {
		widget := &doc.Widgets[i]
		widgetIDs = append(widgetIDs, widget.ID)
		if strings.TrimSpace(widget.Type) == "" {
			*issues = append(*issues, Issue{Path: fmt.Sprintf("widgets[%d].type", i), Code: "DASHBOARD_WIDGET_TYPE_REQUIRED", Message: "组件类型不能为空"})
		}
		layout := widget.Layout
		if layout.X < 0 || layout.Y < 0 || layout.W <= 0 || layout.H <= 0 || layout.X+layout.W > DesktopColumns {
			*issues = append(*issues, Issue{Path: fmt.Sprintf("widgets[%d].layout", i), Code: "DASHBOARD_LAYOUT_INVALID", Message: "组件布局必须位于 12 列桌面画布内且宽高为正数"})
		}
	}
	validateIDs("widgets", widgetIDs, issues)
	if doc.PublishScope.Type != "all" {
		*issues = append(*issues, Issue{Path: "publishScope.type", Code: "DASHBOARD_PUBLISH_SCOPE_INVALID", Message: "当前协议仅支持 all 发布范围"})
	}
}

func idsOf(items []IdentifiedConfig) []string {
	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	return ids
}

func validateIDs(path string, ids []string, issues *[]Issue) {
	seen := make(map[string]bool, len(ids))
	for i, rawID := range ids {
		id := strings.TrimSpace(rawID)
		itemPath := fmt.Sprintf("%s[%d].id", path, i)
		if id == "" {
			*issues = append(*issues, Issue{Path: itemPath, Code: "DASHBOARD_ID_REQUIRED", Message: "标识不能为空"})
			continue
		}
		if seen[id] {
			*issues = append(*issues, Issue{Path: itemPath, Code: "DASHBOARD_ID_DUPLICATED", Message: fmt.Sprintf("标识 %q 重复", id)})
			continue
		}
		seen[id] = true
	}
}
