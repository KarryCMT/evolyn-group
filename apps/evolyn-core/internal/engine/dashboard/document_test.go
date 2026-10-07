package dashboard

import (
	"encoding/json"
	"testing"

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
