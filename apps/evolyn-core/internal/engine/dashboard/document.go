// Package dashboard 定义业务仪表盘的纯协议模型与规范化器。
// 本包不依赖 Gin、GORM、Redis 或 HTTP，保存与发布都必须复用这里的校验结论。
package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	queryengine "evolyn/internal/engine/query"
)

const (
	CurrentVersion = 1
	DesktopColumns = 12
	MaxDatasets    = 50
	MaxWidgets     = 100
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
type IdentifiedConfig struct {
	ID string `json:"id"`
}

type DatasetSource struct {
	Type     string `json:"type"`
	FormCode string `json:"formCode"`
}
type Dataset struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Source DatasetSource        `json:"source"`
	Query  queryengine.Document `json:"query"`
}

type FieldRef struct {
	FieldID string `json:"fieldId"`
}
type ChartDimension struct {
	Field FieldRef `json:"field"`
	Label string   `json:"label,omitempty"`
}
type ChartMetric struct {
	AggregateAlias string `json:"aggregateAlias"`
	Label          string `json:"label,omitempty"`
}
type ChartEncoding struct {
	Dimensions []ChartDimension `json:"dimensions"`
	Metrics    []ChartMetric    `json:"metrics"`
}
type ChartLegend struct {
	Visible  bool   `json:"visible"`
	Position string `json:"position"`
}
type ChartLabels struct {
	Visible bool `json:"visible"`
}
type ChartDisplay struct {
	Variant     string      `json:"variant"`
	Orientation string      `json:"orientation"`
	Stack       string      `json:"stack"`
	Legend      ChartLegend `json:"legend"`
	Labels      ChartLabels `json:"labels"`
}
type ChartSettings struct {
	Encoding ChartEncoding `json:"encoding"`
	Display  ChartDisplay  `json:"display"`
}

type TableColumn struct {
	ID     string   `json:"id"`
	Field  FieldRef `json:"field"`
	Label  string   `json:"label,omitempty"`
	Width  int      `json:"width,omitempty"`
	Align  string   `json:"align"`
	Format string   `json:"format"`
}
type TableSort struct {
	Field     FieldRef `json:"field"`
	Direction string   `json:"direction"`
}
type TablePagination struct {
	PageSize int `json:"pageSize"`
}
type TableDisplay struct {
	Density    string `json:"density"`
	Striped    bool   `json:"striped"`
	Bordered   bool   `json:"bordered"`
	ShowHeader bool   `json:"showHeader"`
	EmptyText  string `json:"emptyText"`
}
type TableSettings struct {
	Columns    []TableColumn   `json:"columns"`
	Sorts      []TableSort     `json:"sorts"`
	Pagination TablePagination `json:"pagination"`
	Display    TableDisplay    `json:"display"`
}

type Widget struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title,omitempty"`
	Layout    GridRect        `json:"layout"`
	DatasetID string          `json:"datasetId,omitempty"`
	Settings  json.RawMessage `json:"settings"`
}
type PublishScope struct {
	Type string `json:"type"`
}

// Document 是 dashboard v1 完整文档。渲染器原生 option/spec/实例不属于协议。
type Document struct {
	Version      int                `json:"version"`
	Settings     Settings           `json:"settings"`
	Datasets     []Dataset          `json:"datasets"`
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
		Datasets: []Dataset{}, Widgets: []Widget{}, Filters: []IdentifiedConfig{},
		Interactions: []IdentifiedConfig{}, PublishScope: PublishScope{Type: "all"},
	}
}

func EmptyContent() json.RawMessage { raw, _ := json.Marshal(EmptyDocument()); return raw }

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

func validate(document *Document, issues *[]Issue) {
	if document.Version != CurrentVersion {
		*issues = append(*issues, Issue{Path: "version", Code: "DASHBOARD_VERSION_UNSUPPORTED", Message: fmt.Sprintf("不支持的仪表盘协议版本：%d", document.Version)})
		return
	}
	if document.Settings.Desktop.Columns != DesktopColumns || document.Settings.Desktop.RowHeight <= 0 {
		addIssue(issues, "settings.desktop", "DASHBOARD_LAYOUT_INVALID", "桌面画布必须使用 12 列且行高为正数")
	}
	validateDatasets(document, issues)
	validateIDs("filters", idsOf(document.Filters), issues)
	validateIDs("interactions", idsOf(document.Interactions), issues)
	validateWidgets(document, issues)
	if document.PublishScope.Type != "all" {
		addIssue(issues, "publishScope.type", "DASHBOARD_PUBLISH_SCOPE_INVALID", "当前协议仅支持 all 发布范围")
	}
}

func validateDatasets(document *Document, issues *[]Issue) {
	if len(document.Datasets) > MaxDatasets {
		addIssue(issues, "datasets", "DASHBOARD_DATASET_LIMIT_EXCEEDED", fmt.Sprintf("单个仪表盘最多允许 %d 个 Dataset", MaxDatasets))
	}
	ids := make([]string, 0, len(document.Datasets))
	for i := range document.Datasets {
		dataset := &document.Datasets[i]
		path := fmt.Sprintf("datasets[%d]", i)
		ids = append(ids, dataset.ID)
		dataset.ID = strings.TrimSpace(dataset.ID)
		dataset.Name = strings.TrimSpace(dataset.Name)
		dataset.Source.FormCode = strings.TrimSpace(dataset.Source.FormCode)
		if dataset.Name == "" {
			addIssue(issues, path+".name", "DASHBOARD_DATASET_NAME_REQUIRED", "Dataset 名称不能为空")
		}
		if dataset.Source.Type != "form" || dataset.Source.FormCode == "" {
			addIssue(issues, path+".source", "DASHBOARD_DATA_SOURCE_INVALID", "当前仅支持具有公开编码的表单数据源")
		}
		queryResult := queryengine.Validate(dataset.Query, nil, queryengine.DefaultBudget())
		for _, issue := range queryResult.Issues {
			addIssue(issues, path+".query."+issue.Path, issue.Code, issue.Message)
		}
		if queryResult.Document != nil {
			dataset.Query = *queryResult.Document
		}
	}
	validateIDs("datasets", ids, issues)
}

func validateWidgets(document *Document, issues *[]Issue) {
	if len(document.Widgets) > MaxWidgets {
		addIssue(issues, "widgets", "DASHBOARD_WIDGET_LIMIT_EXCEEDED", fmt.Sprintf("单个仪表盘最多允许 %d 个组件", MaxWidgets))
	}
	datasetByID := make(map[string]Dataset, len(document.Datasets))
	for _, dataset := range document.Datasets {
		datasetByID[strings.TrimSpace(dataset.ID)] = dataset
	}
	widgetIDs := make([]string, 0, len(document.Widgets))
	for i := range document.Widgets {
		widget := &document.Widgets[i]
		path := fmt.Sprintf("widgets[%d]", i)
		widget.ID = strings.TrimSpace(widget.ID)
		widget.Title = strings.TrimSpace(widget.Title)
		widget.DatasetID = strings.TrimSpace(widget.DatasetID)
		widgetIDs = append(widgetIDs, widget.ID)
		if widget.Type != "chart" && widget.Type != "table" {
			addIssue(issues, path+".type", "DASHBOARD_WIDGET_TYPE_UNSUPPORTED", "不支持的组件类型")
		}
		layout := widget.Layout
		if layout.X < 0 || layout.Y < 0 || layout.W <= 0 || layout.H <= 0 || layout.X+layout.W > DesktopColumns {
			addIssue(issues, path+".layout", "DASHBOARD_LAYOUT_INVALID", "组件布局必须位于 12 列桌面画布内且宽高为正数")
		}
		switch widget.Type {
		case "chart":
			settings, ok := normalizeChartSettings(widget.Settings, path+".settings", issues)
			if ok {
				widget.Settings = settings
				validateChartBindings(settings, widget.DatasetID, datasetByID, path, issues)
			}
		case "table":
			settings, ok := normalizeTableSettings(widget.Settings, path+".settings", issues)
			if ok {
				widget.Settings = settings
				validateTableBindings(settings, widget.DatasetID, datasetByID, path, issues)
			}
		}
		if widget.DatasetID != "" {
			if _, ok := datasetByID[widget.DatasetID]; !ok {
				addIssue(issues, path+".datasetId", "DASHBOARD_DATASET_NOT_FOUND", "组件引用的 Dataset 不存在")
			}
		}
	}
	validateIDs("widgets", widgetIDs, issues)
}

func normalizeChartSettings(raw json.RawMessage, path string, issues *[]Issue) (json.RawMessage, bool) {
	var settings ChartSettings
	if err := strictDecode(raw, &settings); err != nil {
		addIssue(issues, path, "DASHBOARD_CHART_CONFIG_INVALID", "统计图配置必须是受控对象")
		return nil, false
	}
	if !containsString([]string{"bar", "line", "pie"}, settings.Display.Variant) || !containsString([]string{"vertical", "horizontal"}, settings.Display.Orientation) || !containsString([]string{"none", "normal"}, settings.Display.Stack) || !containsString([]string{"top", "right", "bottom", "left"}, settings.Display.Legend.Position) {
		addIssue(issues, path+".display", "DASHBOARD_CHART_CONFIG_INVALID", "统计图展示配置无效")
		return nil, false
	}
	for i := range settings.Encoding.Dimensions {
		settings.Encoding.Dimensions[i].Field.FieldID = strings.TrimSpace(settings.Encoding.Dimensions[i].Field.FieldID)
		if settings.Encoding.Dimensions[i].Field.FieldID == "" {
			addIssue(issues, fmt.Sprintf("%s.encoding.dimensions[%d].field.fieldId", path, i), "DASHBOARD_FIELD_REF_INVALID", "字段引用不能为空")
		}
	}
	for i := range settings.Encoding.Metrics {
		settings.Encoding.Metrics[i].AggregateAlias = strings.TrimSpace(settings.Encoding.Metrics[i].AggregateAlias)
		if settings.Encoding.Metrics[i].AggregateAlias == "" {
			addIssue(issues, fmt.Sprintf("%s.encoding.metrics[%d].aggregateAlias", path, i), "DASHBOARD_CHART_ENCODING_INVALID", "聚合别名不能为空")
		}
	}
	normalized, _ := json.Marshal(settings)
	return normalized, true
}

func normalizeTableSettings(raw json.RawMessage, path string, issues *[]Issue) (json.RawMessage, bool) {
	var settings TableSettings
	if err := strictDecode(raw, &settings); err != nil {
		addIssue(issues, path, "DASHBOARD_TABLE_CONFIG_INVALID", "明细表配置必须是受控对象")
		return nil, false
	}
	if settings.Pagination.PageSize <= 0 || settings.Pagination.PageSize > 100 {
		addIssue(issues, path+".pagination.pageSize", "DASHBOARD_TABLE_PAGE_SIZE_INVALID", "明细表每页数量必须为 1 到 100")
	}
	if !containsString([]string{"compact", "default", "comfortable"}, settings.Display.Density) {
		addIssue(issues, path+".display", "DASHBOARD_TABLE_CONFIG_INVALID", "明细表展示配置无效")
	}
	columnIDs := make([]string, 0, len(settings.Columns))
	for i := range settings.Columns {
		column := &settings.Columns[i]
		column.ID = strings.TrimSpace(column.ID)
		column.Field.FieldID = strings.TrimSpace(column.Field.FieldID)
		columnIDs = append(columnIDs, column.ID)
		if column.Field.FieldID == "" || !containsString([]string{"left", "center", "right"}, column.Align) || !containsString([]string{"auto", "text", "decimal", "money", "percent", "date", "datetime"}, column.Format) || column.Width < 0 || column.Width > 1200 {
			addIssue(issues, fmt.Sprintf("%s.columns[%d]", path, i), "DASHBOARD_TABLE_COLUMN_INVALID", "明细表列配置无效")
		}
	}
	validateIDs(path+".columns", columnIDs, issues)
	for i := range settings.Sorts {
		settings.Sorts[i].Field.FieldID = strings.TrimSpace(settings.Sorts[i].Field.FieldID)
		if settings.Sorts[i].Field.FieldID == "" || (settings.Sorts[i].Direction != "asc" && settings.Sorts[i].Direction != "desc") {
			addIssue(issues, fmt.Sprintf("%s.sorts[%d]", path, i), "DASHBOARD_TABLE_SORT_INVALID", "明细表排序配置无效")
		}
	}
	normalized, _ := json.Marshal(settings)
	return normalized, true
}

func validateChartBindings(raw json.RawMessage, datasetID string, datasets map[string]Dataset, path string, issues *[]Issue) {
	if datasetID == "" {
		return
	}
	dataset, ok := datasets[datasetID]
	if !ok {
		return
	}
	var settings ChartSettings
	if json.Unmarshal(raw, &settings) != nil {
		return
	}
	projected, grouped, aliases := stringSet(dataset.Query.Projection), stringSet(dataset.Query.GroupBy), aggregateAliases(dataset.Query.Aggregates)
	for i, dimension := range settings.Encoding.Dimensions {
		if !projected[dimension.Field.FieldID] && !grouped[dimension.Field.FieldID] {
			addIssue(issues, fmt.Sprintf("%s.settings.encoding.dimensions[%d].field.fieldId", path, i), "DASHBOARD_FIELD_BINDING_INVALID", "维度字段不在 Dataset 输出中")
		}
	}
	for i, metric := range settings.Encoding.Metrics {
		if !aliases[metric.AggregateAlias] {
			addIssue(issues, fmt.Sprintf("%s.settings.encoding.metrics[%d].aggregateAlias", path, i), "DASHBOARD_AGGREGATE_BINDING_INVALID", "指标未引用 Dataset 聚合结果")
		}
	}
}

func validateTableBindings(raw json.RawMessage, datasetID string, datasets map[string]Dataset, path string, issues *[]Issue) {
	if datasetID == "" {
		return
	}
	dataset, ok := datasets[datasetID]
	if !ok {
		return
	}
	var settings TableSettings
	if json.Unmarshal(raw, &settings) != nil {
		return
	}
	projected := stringSet(dataset.Query.Projection)
	for i, column := range settings.Columns {
		if !projected[column.Field.FieldID] {
			addIssue(issues, fmt.Sprintf("%s.settings.columns[%d].field.fieldId", path, i), "DASHBOARD_FIELD_BINDING_INVALID", "明细列字段不在 Dataset 投影中")
		}
	}
}

func strictDecode(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureEOF(decoder)
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
			addIssue(issues, itemPath, "DASHBOARD_ID_REQUIRED", "标识不能为空")
			continue
		}
		if seen[id] {
			addIssue(issues, itemPath, "DASHBOARD_ID_DUPLICATED", fmt.Sprintf("标识 %q 重复", id))
			continue
		}
		seen[id] = true
	}
}
func addIssue(issues *[]Issue, path, code, message string) {
	*issues = append(*issues, Issue{Path: path, Code: code, Message: message})
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
func aggregateAliases(values []queryengine.Aggregate) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value.Alias] = true
	}
	return out
}
