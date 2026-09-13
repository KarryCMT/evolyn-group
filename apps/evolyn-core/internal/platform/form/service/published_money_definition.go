package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"evolyn/internal/platform/form/model"

	"gorm.io/gorm"
)

// moneyDefinition 是金额字段发布后不可改写的值解释：币种决定金额单位，
// 有效 precision/scale 决定 NUMERIC 的取值边界。币种缺省是历史快照的 CNY
// 兼容语义，精度缺省则由 effectiveNumericPrecisionScale 统一解析。
type moneyDefinition struct {
	WidgetName   string
	CurrencyCode string
	Precision    int
	Scale        int
}

// publishedMoneyDefinitionChanges 比对草稿与当前生效快照。该校验不依赖物理
// 存储绑定，因此老 JSONB 表单也不能绕过币种/精度的发布后不可变约束。
func (s *formService) publishedMoneyDefinitionChanges(
	ctx context.Context,
	form *model.Form,
	draft map[string]any,
) ([]string, error) {
	if form.LatestVersionID == nil {
		return nil, nil
	}
	current, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var published map[string]any
	if err := json.Unmarshal([]byte(current.Content), &published); err != nil {
		return nil, fmt.Errorf("decode published money definitions: %w", err)
	}
	previous := collectMoneyDefinitions(published)
	if len(previous) == 0 {
		return nil, nil
	}
	next := collectMoneyDefinitions(draft)
	changed := make([]string, 0)
	for identity, oldDefinition := range previous {
		newDefinition, exists := next[identity]
		if !exists || oldDefinition.CurrencyCode != newDefinition.CurrencyCode ||
			oldDefinition.Precision != newDefinition.Precision || oldDefinition.Scale != newDefinition.Scale {
			changed = append(changed, oldDefinition.WidgetName)
		}
	}
	sort.Strings(changed)
	return changed, nil
}

// collectMoneyDefinitions 递归收集顶层与子表单中的金额字段。fieldId 是 v8
// 起全局稳定的身份；旧快照没有 fieldId 时退回「作用域 + widgetName」，避免
// 把不同子表单的同名历史字段误判为同一字段。
func collectMoneyDefinitions(document map[string]any) map[string]moneyDefinition {
	definitions := map[string]moneyDefinition{}
	items, ok := documentItems(document)
	if !ok {
		return definitions
	}
	var walk func([]any, string)
	walk = func(scopeItems []any, scope string) {
		for index, rawItem := range scopeItems {
			item, ok := rawItem.(map[string]any)
			if !ok {
				continue
			}
			widget, ok := item["widget"].(map[string]any)
			if !ok {
				continue
			}
			widgetType, _ := widget["type"].(string)
			if widgetType == "money" {
				widgetName, _ := widget["widgetName"].(string)
				identity := moneyDefinitionIdentity(scope, widget, index)
				precision, scale := effectiveNumericPrecisionScale(widgetType, widget)
				currencyCode, _ := widget["currencyCode"].(string)
				if currencyCode == "" {
					currencyCode = "CNY"
				}
				definitions[identity] = moneyDefinition{
					WidgetName: widgetName, CurrencyCode: currencyCode, Precision: precision, Scale: scale,
				}
			}
			if widgetType == "subform" {
				children, _ := widget["items"].([]any)
				walk(children, fmt.Sprintf("%s/%d", scope, index))
			}
		}
	}
	walk(items, "root")
	return definitions
}

func moneyDefinitionIdentity(scope string, widget map[string]any, index int) string {
	if fieldID, _ := widget["fieldId"].(string); fieldID != "" {
		return "id:" + fieldID
	}
	if widgetName, _ := widget["widgetName"].(string); widgetName != "" {
		return "legacy:" + scope + ":" + widgetName
	}
	return fmt.Sprintf("legacy:%s:#%d", scope, index)
}
