package dashboard

import (
	"encoding/json"
	"testing"

	queryengine "evolyn/internal/engine/query"

	"github.com/stretchr/testify/require"
)

func TestEmptyDocumentNormalizes(t *testing.T) {
	result := Normalize(EmptyContent())
	require.Empty(t, result.Issues)
	require.JSONEq(t, string(EmptyContent()), string(result.Content))
}

func TestNormalizeRejectsUnknownVersion(t *testing.T) {
	doc := EmptyDocument()
	doc.Version = 99
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	result := Normalize(raw)
	require.Equal(t, "DASHBOARD_VERSION_UNSUPPORTED", result.Issues[0].Code)
	require.Equal(t, "version", result.Issues[0].Path)
}

func TestNormalizeRejectsDuplicateIdentifiers(t *testing.T) {
	doc := EmptyDocument()
	doc.Widgets = []Widget{
		{ID: "widget_1", Type: "chart", Layout: GridRect{W: 6, H: 4}},
		{ID: "widget_1", Type: "table", Layout: GridRect{X: 6, W: 6, H: 4}},
	}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	result := Normalize(raw)
	require.Contains(t, result.Issues, Issue{Path: "widgets[1].id", Code: "DASHBOARD_ID_DUPLICATED", Message: `标识 "widget_1" 重复`})
}

func TestNormalizeRejectsInvalidLayout(t *testing.T) {
	doc := EmptyDocument()
	doc.Widgets = []Widget{{ID: "widget_1", Type: "chart", Layout: GridRect{X: 10, W: 4, H: 0}}}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	result := Normalize(raw)
	require.Contains(t, result.Issues, Issue{Path: "widgets[0].layout", Code: "DASHBOARD_LAYOUT_INVALID", Message: "组件布局必须位于 12 列桌面画布内且宽高为正数"})
}

func TestNormalizeDatasetAndChartUsesControlledStableBindings(t *testing.T) {
	doc := EmptyDocument()
	doc.Datasets = []Dataset{{
		ID: "dataset_sales", Name: "销售数据", Source: DatasetSource{Type: "form", FormCode: "form_sales"},
		Query: queryengine.Document{
			Version: queryengine.Version, Projection: []string{"field_region"}, GroupBy: []string{"field_region"},
			Aggregates: []queryengine.Aggregate{{Field: "field_amount", Operator: queryengine.AggregateSum, Alias: "total_amount"}},
			Sorts:      []queryengine.Sort{{Field: "total_amount", Direction: "desc"}}, Paging: queryengine.Paging{Page: 1, PageSize: 20},
		},
	}}
	settings, err := json.Marshal(ChartSettings{
		Encoding: ChartEncoding{
			Dimensions: []ChartDimension{{Field: FieldRef{FieldID: "field_region"}}},
			Metrics:    []ChartMetric{{AggregateAlias: "total_amount"}},
		},
		Display: ChartDisplay{
			Variant: "bar", Orientation: "vertical", Stack: "none",
			Legend: ChartLegend{Visible: true, Position: "top"}, Labels: ChartLabels{},
		},
	})
	require.NoError(t, err)
	doc.Widgets = []Widget{{
		ID: "widget_sales", Type: "chart", DatasetID: "dataset_sales",
		Layout: GridRect{W: 6, H: 4}, Settings: settings,
	}}
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	result := Normalize(raw)
	require.Empty(t, result.Issues)
	require.Equal(t, "field_region", result.Document.Datasets[0].Query.GroupBy[0])

	var private map[string]any
	require.NoError(t, json.Unmarshal(settings, &private))
	private["spec"] = map[string]any{"type": "bar"}
	doc.Widgets[0].Settings, err = json.Marshal(private)
	require.NoError(t, err)
	result = Normalize(mustJSON(t, doc))
	require.Contains(t, result.Issues, Issue{Path: "widgets[0].settings", Code: "DASHBOARD_CHART_CONFIG_INVALID", Message: "统计图配置必须是受控对象"})
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return raw
}
