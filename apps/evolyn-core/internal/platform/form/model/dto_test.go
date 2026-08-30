package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubmitRecordRequestEnvelope(t *testing.T) {
	raw := []byte(`{
		"appCode":"app_demo",
		"entryCode":"menu_demo",
		"formCode":"form_demo",
		"publishedVersion":3,
		"schemaRevision":"77",
		"values":{
			"_widget_text":{"data":"测试","visible":true},
			"_widget_empty":{"visible":true},
			"_widget_hidden":{"visible":false}
		},
		"hasResult":true,
		"dataOpId":"8f85ff13-a326-45a6-8d09-12b93cc789b0"
	}`)
	var req SubmitRecordRequest
	assert.NoError(t, json.Unmarshal(raw, &req))
	assert.Equal(t, "app_demo", req.AppCode)
	assert.Equal(t, "menu_demo", req.EntryCode)
	assert.Equal(t, JSONContent(`"测试"`), req.Values["_widget_text"].Data)
	assert.Empty(t, req.Values["_widget_empty"].Data)
	assert.True(t, *req.Values["_widget_empty"].Visible)
	assert.False(t, *req.Values["_widget_hidden"].Visible)
	assert.True(t, *req.HasResult)
}
