package service

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"evolyn/internal/contextx"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/sirupsen/logrus"
)

var errFrontendEventTargetNotWritable = errors.New("frontend event target not writable")

// DebugFrontendEvent 从当前草稿按 eventId 读取配置并执行一次安全调试。
// 客户端不能提交事件定义，避免绕过草稿校验临时调用任意地址。
func (s *formService) DebugFrontendEvent(
	ctx context.Context,
	member *iammodel.User,
	code, eventID string,
	req *model.FrontendEventExecuteRequest,
) (*model.FrontendEventExecuteResult, error) {
	if !s.access.Permissions(ctx, member)["forms:update"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot debug form %s frontend event", code))
	}
	if s.frontendEvents == nil {
		return nil, httpx.Wrap(apperrors.ErrFrontendEventRequestFailed, errors.New("frontend event invoker not configured"))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	event, content, err := frontendEventFromDocument(form.DraftContent, eventID)
	if err != nil {
		return nil, err
	}
	if !event.Enabled {
		return nil, httpx.Wrap(apperrors.ErrFrontendEventDisabled, fmt.Errorf("frontend event %s disabled", eventID))
	}
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrFrontendEventInvalid, err)
	}
	values := req.Values
	if values == nil {
		values = map[string]any{}
	}
	invocation, summary, err := buildFrontendEventInvocation(event, values)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrFrontendEventTemplateInvalid, err)
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, httpx.Wrap(apperrors.ErrFrontendEventRequestFailed, errors.New("tenant context missing"))
	}
	invocation.TenantID = tenantID
	response, err := s.frontendEvents.Invoke(ctx, invocation)
	if err != nil {
		if errors.Is(err, ErrFrontendEventURLBlocked) {
			return nil, httpx.Wrap(apperrors.ErrFrontendEventRequestBlocked, err)
		}
		return nil, httpx.Wrap(apperrors.ErrFrontendEventRequestFailed, err)
	}
	result := &model.FrontendEventExecuteResult{
		Sequence:       req.Sequence,
		Writes:         map[string]any{},
		RequestSummary: summary,
		ResponseSummary: model.FrontendEventResponseSummary{
			StatusCode: response.StatusCode,
			DurationMS: response.DurationMS,
			Format:     event.Request.Format,
			Body:       string(response.Body),
		},
	}
	// 调试接口保留非 2xx 的响应摘要供定位，但不执行任何字段回填。
	if response.StatusCode < 200 || response.StatusCode > 299 {
		result.ErrorCode = apperrors.ErrFrontendEventRequestFailed.Code
		logFrontendEventResult(tenantID, event, result)
		return result, nil
	}
	root, err := parseFrontendEventResponse(event.Request.Format, response.Body)
	if err != nil {
		result.ErrorCode = apperrors.ErrFrontendEventResponseInvalid.Code
		logFrontendEventResult(tenantID, event, result)
		return result, nil
	}
	writes, err := mapFrontendEventWrites(event, root, values, fields)
	if err != nil {
		if errors.Is(err, errFrontendEventTargetNotWritable) {
			result.ErrorCode = apperrors.ErrFrontendEventTargetNotWritable.Code
		} else {
			result.ErrorCode = apperrors.ErrFrontendEventMappingFailed.Code
		}
		logFrontendEventResult(tenantID, event, result)
		return result, nil
	}
	result.Writes = writes
	logFrontendEventResult(tenantID, event, result)
	return result, nil
}

// logFrontendEventResult 只记录受控元数据；表单值、URL query、Header、请求体和
// 响应正文都不得进入日志。
func logFrontendEventResult(tenantID uint, event model.FrontendEvent, result *model.FrontendEventExecuteResult) {
	targetFields := make([]string, 0, len(event.Action))
	for _, action := range event.Action {
		targetFields = append(targetFields, action.Field)
	}
	logrus.WithFields(logrus.Fields{
		"tenantId":     tenantID,
		"eventId":      event.ID,
		"status":       result.ResponseSummary.StatusCode,
		"durationMs":   result.ResponseSummary.DurationMS,
		"targetFields": targetFields,
		"errorCode":    result.ErrorCode,
	}).Info("form frontend event result")
}

func frontendEventFromDocument(raw model.JSONContent, eventID string) (model.FrontendEvent, map[string]any, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return model.FrontendEvent{}, nil, httpx.Wrap(apperrors.ErrFrontendEventInvalid, err)
	}
	content, ok := root["content"].(map[string]any)
	if !ok {
		return model.FrontendEvent{}, nil, httpx.Wrap(apperrors.ErrFrontendEventInvalid, errors.New("content missing"))
	}
	events, _ := content["formEvents"].([]any)
	for _, rawEvent := range events {
		eventMap, _ := rawEvent.(map[string]any)
		if frontendEventString(eventMap["id"]) != eventID {
			continue
		}
		encoded, err := json.Marshal(eventMap)
		if err != nil {
			return model.FrontendEvent{}, nil, httpx.Wrap(apperrors.ErrFrontendEventInvalid, err)
		}
		var event model.FrontendEvent
		if err := json.Unmarshal(encoded, &event); err != nil {
			return model.FrontendEvent{}, nil, httpx.Wrap(apperrors.ErrFrontendEventInvalid, err)
		}
		return event, root, nil
	}
	return model.FrontendEvent{}, nil, httpx.Wrap(apperrors.ErrFrontendEventNotFound, fmt.Errorf("frontend event %s not found", eventID))
}

func buildFrontendEventInvocation(event model.FrontendEvent, values map[string]any) (FrontendEventInvokeRequest, model.FrontendEventRequestSummary, error) {
	resolve := func(template string) (string, error) {
		var resolveErr error
		resolved := frontendEventTokenPattern.ReplaceAllStringFunc(template, func(token string) string {
			match := frontendEventTokenPattern.FindStringSubmatch(token)
			if len(match) < 2 {
				resolveErr = fmt.Errorf("invalid token %q", token)
				return ""
			}
			value, exists := values[match[1]]
			if !exists || value == nil {
				return ""
			}
			switch typed := value.(type) {
			case string:
				return typed
			case float64, bool, json.Number:
				return fmt.Sprint(typed)
			default:
				encoded, err := json.Marshal(typed)
				if err != nil {
					resolveErr = err
					return ""
				}
				return string(encoded)
			}
		})
		if resolveErr != nil || strings.Contains(resolved, "${") {
			return "", fmt.Errorf("template contains unresolved token")
		}
		return resolved, nil
	}
	resolvedURL, err := resolve(event.Request.URL)
	if err != nil {
		return FrontendEventInvokeRequest{}, model.FrontendEventRequestSummary{}, err
	}
	headers := make(map[string]string, len(event.Request.Header)+1)
	headerNames := make([]string, 0, len(event.Request.Header)+1)
	for _, entry := range event.Request.Header {
		key, keyErr := resolve(entry.Key)
		value, valueErr := resolve(entry.Value)
		if keyErr != nil || valueErr != nil {
			return FrontendEventInvokeRequest{}, model.FrontendEventRequestSummary{}, fmt.Errorf("header template invalid")
		}
		if !frontendEventHeaderPattern.MatchString(key) || frontendEventRestrictedHeaders[strings.ToLower(strings.TrimSpace(key))] {
			return FrontendEventInvokeRequest{}, model.FrontendEventRequestSummary{}, fmt.Errorf("resolved header name is invalid or forbidden")
		}
		headers[key] = value
		headerNames = append(headerNames, key)
	}
	body := ""
	if event.Request.Method == "post" {
		bodyValues := make(map[string]string, len(event.Request.Body))
		for _, entry := range event.Request.Body {
			key, keyErr := resolve(entry.Key)
			value, valueErr := resolve(entry.Value)
			if keyErr != nil || valueErr != nil {
				return FrontendEventInvokeRequest{}, model.FrontendEventRequestSummary{}, fmt.Errorf("body template invalid")
			}
			bodyValues[key] = value
		}
		encoded, err := json.Marshal(bodyValues)
		if err != nil {
			return FrontendEventInvokeRequest{}, model.FrontendEventRequestSummary{}, err
		}
		body = string(encoded)
		if !frontendEventHeaderExists(headers, "Content-Type") {
			headers["Content-Type"] = "application/json"
			headerNames = append(headerNames, "Content-Type")
		}
	}
	sort.Strings(headerNames)
	return FrontendEventInvokeRequest{
			EventID: event.ID,
			Method:  strings.ToUpper(event.Request.Method),
			URL:     resolvedURL,
			Headers: headers,
			Body:    body,
		}, model.FrontendEventRequestSummary{
			Method:      strings.ToUpper(event.Request.Method),
			URL:         resolvedURL,
			HeaderNames: headerNames,
			Body:        body,
		}, nil
}

func frontendEventHeaderExists(headers map[string]string, target string) bool {
	for name := range headers {
		if strings.EqualFold(name, target) {
			return true
		}
	}
	return false
}

func parseFrontendEventResponse(format string, body []byte) (any, error) {
	if format == "xml" {
		return parseFrontendEventXML(body)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("response contains multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

func mapFrontendEventWrites(event model.FrontendEvent, root any, values map[string]any, fields map[string]snapshotField) (map[string]any, error) {
	writes := make(map[string]any, len(event.Action))
	for _, action := range event.Action {
		field, exists := fields[action.Field]
		if !exists || !frontendEventValueField(field.widgetType) || action.Field == event.Trigger {
			return nil, fmt.Errorf("%w: %q", errFrontendEventTargetNotWritable, action.Field)
		}
		mapped, err := resolveFrontendEventAction(action.Value, event.Request.Format, root, values)
		if err != nil {
			return nil, fmt.Errorf("map target %s: %w", action.Field, err)
		}
		if fieldErrors := validateFieldValue(field, mapped); len(fieldErrors) > 0 {
			return nil, fmt.Errorf("mapped value for %s invalid: %s", action.Field, strings.Join(fieldErrors, "; "))
		}
		writes[action.Field] = mapped
	}
	return writes, nil
}

func resolveFrontendEventAction(expression, format string, root any, values map[string]any) (any, error) {
	if format == "xml" && frontendEventXPathPattern.MatchString(expression) {
		return resolveFrontendEventXMLPath(root, expression)
	}
	if format != "xml" && frontendEventJSONPathPattern.MatchString(expression) {
		return resolveFrontendEventJSONPath(root, expression)
	}
	resolved := expression
	for _, name := range frontendEventDependencies(expression) {
		resolved = strings.ReplaceAll(resolved, "${"+name+"}", fmt.Sprint(values[name]))
	}
	return resolved, nil
}

func resolveFrontendEventJSONPath(root any, path string) (any, error) {
	if path == "$response" {
		return root, nil
	}
	cursor := root
	remainder := strings.TrimPrefix(path, "$response")
	for remainder != "" {
		if strings.HasPrefix(remainder, ".") {
			remainder = remainder[1:]
			end := strings.IndexAny(remainder, ".[")
			if end < 0 {
				end = len(remainder)
			}
			key := remainder[:end]
			object, ok := cursor.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%q is not an object", key)
			}
			cursor, ok = object[key]
			if !ok {
				return nil, fmt.Errorf("path key %q not found", key)
			}
			remainder = remainder[end:]
			continue
		}
		if strings.HasPrefix(remainder, "[") {
			end := strings.IndexByte(remainder, ']')
			if end < 2 {
				return nil, fmt.Errorf("invalid array index")
			}
			index, err := strconv.Atoi(remainder[1:end])
			array, ok := cursor.([]any)
			if err != nil || !ok || index < 0 || index >= len(array) {
				return nil, fmt.Errorf("array index out of range")
			}
			cursor = array[index]
			remainder = remainder[end+1:]
			continue
		}
		return nil, fmt.Errorf("invalid json path")
	}
	return cursor, nil
}

type frontendEventXMLNode struct {
	Name     string
	Text     string
	Children []*frontendEventXMLNode
}

func parseFrontendEventXML(body []byte) (*frontendEventXMLNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	var stack []*frontendEventXMLNode
	var root *frontendEventXMLNode
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			node := &frontendEventXMLNode{Name: typed.Name.Local}
			if len(stack) == 0 {
				root = node
			} else {
				stack[len(stack)-1].Children = append(stack[len(stack)-1].Children, node)
			}
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(typed)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		return nil, errors.New("empty xml response")
	}
	return root, nil
}

func resolveFrontendEventXMLPath(root any, path string) (any, error) {
	node, ok := root.(*frontendEventXMLNode)
	if !ok {
		return nil, errors.New("xml root invalid")
	}
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segments) == 0 || segments[0] != node.Name {
		return nil, fmt.Errorf("xml root %q not found", segments[0])
	}
	for _, segment := range segments[1:] {
		var child *frontendEventXMLNode
		for _, candidate := range node.Children {
			if candidate.Name == segment {
				child = candidate
				break
			}
		}
		if child == nil {
			return nil, fmt.Errorf("xml path %q not found", segment)
		}
		node = child
	}
	return strings.TrimSpace(node.Text), nil
}
