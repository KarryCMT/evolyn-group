package service

import (
	"encoding/json"
	"testing"

	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func frontendEventTestDefinition() model.FrontendEvent {
	return model.FrontendEvent{
		ID: "evt_contact01", Enabled: true, Name: "补全联系信息", Trigger: "_widget_a1",
		TriggerType: "widget", RequestType: 0,
		Request: model.FrontendEventRequest{
			Method: "post", URL: "https://api.lingyanyun.com/member?id=${_widget_a1}",
			Header: []model.FrontendEventRequestEntry{{Key: "X-Source", Value: "${_widget_a1}"}},
			Body:   []model.FrontendEventRequestEntry{{Key: "member", Value: "${_widget_a1}"}},
			Format: "json",
		},
		RequestRely:     []string{"_widget_a1"},
		Action:          []model.FrontendEventAction{{Field: "_widget_b1", Value: "$response.data.phone"}},
		ActionRely:      []string{},
		SubformFillRule: "merge",
	}
}

func TestValidateFrontendEventSchema(t *testing.T) {
	second := validTextItem()
	second["widget"].(map[string]any)["widgetName"] = "_widget_b1"
	raw := doc(validTextItem(), second)
	var document map[string]any
	require.NoError(t, json.Unmarshal(raw, &document))
	content := document["content"].(map[string]any)
	eventBytes, err := json.Marshal(frontendEventTestDefinition())
	require.NoError(t, err)
	var event map[string]any
	require.NoError(t, json.Unmarshal(eventBytes, &event))
	content["formEvents"] = []any{event}
	raw, err = json.Marshal(document)
	require.NoError(t, err)
	assert.Empty(t, ValidateFormSchema(raw, model.CurrentProtocolVersion))

	event["request"].(map[string]any)["header"] = []any{map[string]any{"key": "Authorization", "value": "Bearer token"}}
	raw, err = json.Marshal(document)
	require.NoError(t, err)
	assert.Empty(t, ValidateFormSchema(raw, model.CurrentProtocolVersion))

	event["request"].(map[string]any)["header"] = []any{map[string]any{"key": "Cookie", "value": "session=secret"}}
	raw, err = json.Marshal(document)
	require.NoError(t, err)
	issues := ValidateFormSchema(raw, model.CurrentProtocolVersion)
	assert.Contains(t, issues, SchemaIssue{
		Path: "content.formEvents[0].request.header[0].key", Message: "Header 名称无效或属于禁止的敏感 Header",
	})
}

func TestBuildFrontendEventInvocationAndMapWrites(t *testing.T) {
	event := frontendEventTestDefinition()
	event.Request.Header = append(event.Request.Header, model.FrontendEventRequestEntry{
		Key: "Authorization", Value: "Bearer event-token",
	})
	invocation, summary, err := buildFrontendEventInvocation(event, map[string]any{"_widget_a1": "member-001"})
	require.NoError(t, err)
	assert.Equal(t, "POST", invocation.Method)
	assert.Equal(t, "https://api.lingyanyun.com/member?id=member-001", invocation.URL)
	assert.JSONEq(t, `{"member":"member-001"}`, invocation.Body)
	assert.Equal(t, "Bearer event-token", invocation.Headers["Authorization"])
	assert.Equal(t, []string{"Authorization", "Content-Type", "X-Source"}, summary.HeaderNames)

	root, err := parseFrontendEventResponse("json", []byte(`{"data":{"phone":"13800000000"}}`))
	require.NoError(t, err)
	writes, err := mapFrontendEventWrites(event, root, map[string]any{"_widget_a1": "member-001"}, map[string]snapshotField{
		"_widget_a1": {widgetName: "_widget_a1", widgetType: "text", allowBlank: true, visible: true},
		"_widget_b1": {widgetName: "_widget_b1", widgetType: "text", allowBlank: true, visible: true},
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"_widget_b1": "13800000000"}, writes)
}

func TestNormalizeFrontendEventDependencies(t *testing.T) {
	event := frontendEventTestDefinition()
	event.RequestRely = []string{"_widget_stale"}
	event.Action[0].Value = "${_widget_a1}:$response.data.phone"
	event.ActionRely = []string{"_widget_stale"}
	raw, err := json.Marshal(map[string]any{"content": map[string]any{"formEvents": []model.FrontendEvent{event}}})
	require.NoError(t, err)
	normalized, err := normalizeFrontendEventDependencies(raw)
	require.NoError(t, err)
	var document struct {
		Content struct {
			Events []model.FrontendEvent `json:"formEvents"`
		} `json:"content"`
	}
	require.NoError(t, json.Unmarshal(normalized, &document))
	require.Len(t, document.Content.Events, 1)
	assert.Equal(t, []string{"_widget_a1"}, document.Content.Events[0].RequestRely)
	assert.Equal(t, []string{"_widget_a1"}, document.Content.Events[0].ActionRely)
}

func TestParseFrontendEventXMLPath(t *testing.T) {
	root, err := parseFrontendEventResponse("xml", []byte(`<response><data><phone>13800000000</phone></data></response>`))
	require.NoError(t, err)
	value, err := resolveFrontendEventXMLPath(root, "/response/data/phone")
	require.NoError(t, err)
	assert.Equal(t, "13800000000", value)
}
