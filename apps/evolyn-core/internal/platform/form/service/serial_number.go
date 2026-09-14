package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func serialNumberKeys(content map[string]any) map[string]bool {
	keys := map[string]bool{}
	items, _ := documentItems(content)
	for _, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		widget, _ := item["widget"].(map[string]any)
		if widget["type"] != "sn" {
			continue
		}
		if name, ok := widget["widgetName"].(string); ok {
			keys[name] = true
		}
	}
	return keys
}

func valuesWithoutSerialNumbers(content map[string]any, values map[string]any) map[string]any {
	keys := serialNumberKeys(content)
	copy := make(map[string]any, len(values))
	for name, value := range values {
		if !keys[name] {
			copy[name] = value
		}
	}
	return copy
}

func sameJSONIgnoringSerialNumbers(existing []byte, expected []byte, content map[string]any) bool {
	var values map[string]any
	if err := json.Unmarshal(existing, &values); err != nil {
		return false
	}
	actual, err := json.Marshal(valuesWithoutSerialNumbers(content, values))
	return err == nil && sameJSON(actual, expected)
}

// applySerialNumbers 在记录持久化前为所有流水号字段覆写服务端值。调用方已处于
// 同一提交事务：计数分配、物理/JSONB 写入和流程发起任一步失败都会整体回滚。
func (s *formService) applySerialNumbers(
	ctx context.Context, tenantID, formID uint, content map[string]any, values map[string]any, now time.Time,
) error {
	items, ok := documentItems(content)
	if !ok {
		return fmt.Errorf("snapshot missing items")
	}
	type fieldMeta struct{ name, fieldType string }
	fields := make(map[string]fieldMeta, len(items))
	for _, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		widget, _ := item["widget"].(map[string]any)
		fieldID, _ := widget["fieldId"].(string)
		name, _ := widget["widgetName"].(string)
		fieldType, _ := widget["type"].(string)
		if fieldID != "" && name != "" {
			fields[fieldID] = fieldMeta{name: name, fieldType: fieldType}
		}
	}
	for itemIndex, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		widget, _ := item["widget"].(map[string]any)
		if widget["type"] != "sn" {
			continue
		}
		fieldID, _ := widget["fieldId"].(string)
		name, _ := widget["widgetName"].(string)
		parts, _ := widget["rules"].([]any)
		if fieldID == "" || name == "" || len(parts) == 0 {
			return fmt.Errorf("serial field %d has invalid published rule", itemIndex)
		}
		if s.serialCounters == nil {
			return fmt.Errorf("serial counter repository is not configured")
		}
		var output strings.Builder
		for _, rawPart := range parts {
			part, _ := rawPart.(map[string]any)
			typeName, _ := part["type"].(string)
			switch typeName {
			case "literal":
				text, _ := part["value"].(string)
				output.WriteString(text)
			case "submittedAt":
				formatted, err := formatSerialDate(now, part["format"])
				if err != nil {
					return err
				}
				output.WriteString(formatted)
			case "counter":
				cycle, _ := part["resetCycle"].(string)
				cycleKey, err := serialCycleKey(now, cycle)
				if err != nil {
					return err
				}
				initial, _ := asInteger(part["initialValue"])
				sequence, err := s.serialCounters.Allocate(ctx, tenantID, formID, fieldID, cycleKey, int64(initial))
				if err != nil {
					return err
				}
				digits, _ := asInteger(part["digits"])
				fixed, _ := part["fixedWidth"].(bool)
				text := strconv.FormatInt(sequence, 10)
				if fixed {
					if len(text) > digits {
						return fmt.Errorf("流水号计数已超过 %d 位上限", digits)
					}
					text = fmt.Sprintf("%0*d", digits, sequence)
				}
				output.WriteString(text)
			case "field":
				referencedID, _ := part["fieldId"].(string)
				referenced, found := fields[referencedID]
				if !found {
					return fmt.Errorf("流水号引用字段不存在")
				}
				text, err := serialFieldText(values[referenced.name], referenced.fieldType)
				if err != nil {
					return fmt.Errorf("流水号引用字段「%s」%w", referenced.name, err)
				}
				output.WriteString(text)
			default:
				return fmt.Errorf("流水号包含未知规则片段")
			}
		}
		if output.Len() == 0 || output.Len() > 256 {
			return fmt.Errorf("流水号长度必须在 1–256 个字符之间")
		}
		values[name] = output.String()
	}
	return nil
}

func formatSerialDate(now time.Time, raw any) (string, error) {
	format, _ := raw.(string)
	if !validSerialDateFormat(format) {
		return "", fmt.Errorf("invalid serial submittedAt format")
	}
	var output strings.Builder
	for len(format) > 0 {
		switch {
		case strings.HasPrefix(format, "yyyy"):
			output.WriteString(now.Format("2006"))
			format = format[4:]
		case strings.HasPrefix(format, "MM"):
			output.WriteString(now.Format("01"))
			format = format[2:]
		case strings.HasPrefix(format, "dd"):
			output.WriteString(now.Format("02"))
			format = format[2:]
		default:
			output.WriteByte(format[0]) // 前置校验已限制为 - 或 /
			format = format[1:]
		}
	}
	return output.String(), nil
}

func serialCycleKey(now time.Time, cycle string) (string, error) {
	switch cycle {
	case "none":
		return "all", nil
	case "daily":
		return now.Format("20060102"), nil
	case "monthly":
		return now.Format("200601"), nil
	case "yearly":
		return now.Format("2006"), nil
	default:
		return "", fmt.Errorf("invalid serial reset cycle")
	}
}

func serialFieldText(value any, fieldType string) (string, error) {
	if value == nil {
		return "", fmt.Errorf("不能为空")
	}
	switch fieldType {
	case "text", "textarea", "decimal", "money", "percent", "datetime", "radiogroup", "combo":
		if text, ok := value.(string); ok && text != "" {
			return text, nil
		}
	case "number":
		if number, ok := value.(float64); ok {
			return strconv.FormatFloat(number, 'f', -1, 64), nil
		}
	}
	return "", fmt.Errorf("没有可拼接的标量值")
}
