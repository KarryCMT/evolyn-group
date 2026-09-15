package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- 工作台文档校验器单测：与前端 @evolyn.do/dashboard schema/lifecycle.ts
// 语义镜像（迁移/结构/白名单/坐标约束/未知字段剔除/服务端护栏）----

func TestValidateWorkbenchDocumentValid(t *testing.T) {
	// 与前端默认工作台同构的合法文档：含可选约束、布尔锁定、presetKey、config
	raw := `{
		"version": 1,
		"widgets": [
			{"id":"onboarding-0-0","type":"onboarding","title":"新手引导","x":0,"y":0,"w":12,"h":2,"noResize":true},
			{"id":"greeting-0-2","type":"greeting","title":"问候语","x":0,"y":2,"w":3,"h":1,"minW":3,"maxH":1,"presetKey":"greeting"},
			{"id":"favorites-3-2","type":"favorites","title":"最近使用","x":3,"y":2,"w":9,"h":2,"minW":4,"minH":2,"presetKey":"recent","config":{"max":8}}
		]
	}`
	normalized, issue := ValidateWorkbenchDocument([]byte(raw))
	require.Nil(t, issue)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(normalized, &doc))
	assert.Equal(t, float64(1), doc["version"])
	widgets, ok := doc["widgets"].([]any)
	require.True(t, ok)
	require.Len(t, widgets, 3)

	greeting := widgets[1].(map[string]any)
	assert.Equal(t, "greeting", greeting["type"])
	assert.Equal(t, float64(3), greeting["minW"])
	assert.Equal(t, float64(1), greeting["maxH"])
	assert.Equal(t, "greeting", greeting["presetKey"])
	favorites := widgets[2].(map[string]any)
	assert.Equal(t, map[string]any{"max": float64(8)}, favorites["config"])
}

func TestValidateWorkbenchDocumentMigratesMissingVersion(t *testing.T) {
	// 早期无 version 文档：补齐 v1（前端 migrateDashboardSchema 同语义）
	raw := `{"widgets":[{"id":"a","type":"greeting","title":"问候语","x":0,"y":0,"w":3,"h":1}]}`
	normalized, issue := ValidateWorkbenchDocument([]byte(raw))
	require.Nil(t, issue)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(normalized, &doc))
	assert.Equal(t, float64(1), doc["version"])
}

func TestValidateWorkbenchDocumentStripsUnknownFields(t *testing.T) {
	// 未知根键与未知卡片键剔除，防未知结构入库
	raw := `{"version":1,"future":true,"widgets":[{"id":"a","type":"todo","title":"流程中心","x":0,"y":0,"w":3,"h":4,"evil":"drop"}]}`
	normalized, issue := ValidateWorkbenchDocument([]byte(raw))
	require.Nil(t, issue)
	assert.NotContains(t, string(normalized), "future")
	assert.NotContains(t, string(normalized), "evil")
}

func TestValidateWorkbenchDocumentIssues(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		code string
	}{
		{"根不是对象", `[]`, "invalid-root"},
		{"非法 JSON", `{oops`, "invalid-root"},
		{"版本不支持", `{"version":2,"widgets":[]}`, "unsupported-version"},
		{"version 非数字", `{"version":"1","widgets":[]}`, "unsupported-version"},
		{"widgets 非数组", `{"version":1,"widgets":{}}`, "invalid-widgets"},
		{"卡片非对象", `{"version":1,"widgets":[1]}`, "invalid-widget"},
		{"卡片 id 重复", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1},{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1}]}`, "duplicate-widget-id"},
		{"未知卡片类型", `{"version":1,"widgets":[{"id":"a","type":"unknown","title":"t","x":0,"y":0,"w":1,"h":1}]}`, "unknown-widget-type"},
		{"id 缺失", `{"version":1,"widgets":[{"type":"todo","title":"t","x":0,"y":0,"w":1,"h":1}]}`, "invalid-text"},
		{"title 空白", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"  ","x":0,"y":0,"w":1,"h":1}]}`, "invalid-text"},
		{"x 负数", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":-1,"y":0,"w":1,"h":1}]}`, "invalid-layout-value"},
		{"w 零值", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":0,"h":1}]}`, "invalid-layout-value"},
		{"h 非整数", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1.5}]}`, "invalid-layout-value"},
		{"minW 大于 maxW", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":4,"h":1,"minW":5,"maxW":4}]}`, "invalid-width-range"},
		{"minH 大于 maxH", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":4,"minH":5,"maxH":4}]}`, "invalid-height-range"},
		{"w 小于 minW", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":2,"h":1,"minW":3}]}`, "width-below-minimum"},
		{"w 大于 maxW", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":5,"h":1,"maxW":4}]}`, "width-above-maximum"},
		{"h 小于 minH", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":2,"minH":3}]}`, "height-below-minimum"},
		{"h 大于 maxH", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":5,"maxH":4}]}`, "height-above-maximum"},
		{"noMove 非布尔", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1,"noMove":"yes"}]}`, "invalid-no-move"},
		{"noResize 非布尔", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1,"noResize":1}]}`, "invalid-no-resize"},
		{"presetKey 非字符串", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1,"presetKey":3}]}`, "invalid-preset-key"},
		{"config 非对象", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1,"config":[1,2]}]}`, "invalid-config"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, issue := ValidateWorkbenchDocument([]byte(tc.raw))
			require.NotNil(t, issue)
			assert.Equal(t, tc.code, issue.Code)
		})
	}
}

func TestValidateWorkbenchDocumentGuards(t *testing.T) {
	t.Run("卡片数量上限", func(t *testing.T) {
		widgets := make([]string, 0, maxWorkbenchWidgets+1)
		for i := 0; i <= maxWorkbenchWidgets; i++ {
			widgets = append(widgets, fmt.Sprintf(
				`{"id":"w%03d","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1}`, i))
		}
		raw := `{"version":1,"widgets":[` + strings.Join(widgets, ",") + `]}`
		_, issue := ValidateWorkbenchDocument([]byte(raw))
		require.NotNil(t, issue)
		assert.Equal(t, "too-many-widgets", issue.Code)
	})

	t.Run("文档体积上限", func(t *testing.T) {
		raw := `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1,"config":{"padding":"` +
			strings.Repeat("x", maxWorkbenchDocBytes) + `"}}]}`
		_, issue := ValidateWorkbenchDocument([]byte(raw))
		require.NotNil(t, issue)
		assert.Equal(t, "document-too-large", issue.Code)
	})

	t.Run("文本长度上限", func(t *testing.T) {
		raw := `{"version":1,"widgets":[{"id":"a","type":"todo","title":"` +
			strings.Repeat("标", maxWorkbenchTextLen+1) + `","x":0,"y":0,"w":1,"h":1}]}`
		_, issue := ValidateWorkbenchDocument([]byte(raw))
		require.NotNil(t, issue)
		assert.Equal(t, "invalid-text", issue.Code)
	})
}
