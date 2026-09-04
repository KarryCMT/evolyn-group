// Package formula 提供表单公式的纯协议能力。它不依赖 HTTP、GORM 或运行时表单服务，
// 以便草稿校验、发布编译和填写提交共用同一字段类型语义。
package formula

import (
	"encoding/json"
	"fmt"
)

// ValueType 公式类型系统中的字段值形态。成员、部门和行集合不被降级为文本，
// 后续函数目录必须显式声明能否消费这些类型。
type ValueType string

const (
	ValueTypeText        ValueType = "text"
	ValueTypeNumber      ValueType = "number"
	ValueTypeDate        ValueType = "date"
	ValueTypeArray       ValueType = "array"
	ValueTypeMember      ValueType = "member"
	ValueTypeMembers     ValueType = "members"
	ValueTypeDepartment  ValueType = "department"
	ValueTypeDepartments ValueType = "departments"
	ValueTypeLocation    ValueType = "location"
	ValueTypeRows        ValueType = "rows"
	ValueTypeUnknown     ValueType = "unknown"
)

// FieldMeta 是控件类型到公式字段的冻结映射。FormulaAllowed 仅表示当前 DSL
// 能否把字段直接作为操作数插入；不代表成员/部门字段没有业务值。
type FieldMeta struct {
	ValueType      ValueType
	DisplayType    string
	FormulaAllowed bool
}

var widgetVariableTypes = map[string]FieldMeta{
	"text":          {ValueType: ValueTypeText, DisplayType: "文本", FormulaAllowed: true},
	"textarea":      {ValueType: ValueTypeText, DisplayType: "文本", FormulaAllowed: true},
	"phone":         {ValueType: ValueTypeText, DisplayType: "文本", FormulaAllowed: true},
	"number":        {ValueType: ValueTypeNumber, DisplayType: "数字", FormulaAllowed: true},
	"datetime":      {ValueType: ValueTypeDate, DisplayType: "时间戳", FormulaAllowed: true},
	"radiogroup":    {ValueType: ValueTypeText, DisplayType: "文本", FormulaAllowed: true},
	"combo":         {ValueType: ValueTypeText, DisplayType: "文本", FormulaAllowed: true},
	"checkboxgroup": {ValueType: ValueTypeArray, DisplayType: "数组", FormulaAllowed: true},
	"combocheck":    {ValueType: ValueTypeArray, DisplayType: "数组", FormulaAllowed: true},
	"user":          {ValueType: ValueTypeMember, DisplayType: "成员", FormulaAllowed: false},
	"usergroup":     {ValueType: ValueTypeMembers, DisplayType: "成员数组", FormulaAllowed: false},
	"dept":          {ValueType: ValueTypeDepartment, DisplayType: "部门", FormulaAllowed: false},
	"deptgroup":     {ValueType: ValueTypeDepartments, DisplayType: "部门数组", FormulaAllowed: false},
	"location":      {ValueType: ValueTypeLocation, DisplayType: "位置", FormulaAllowed: false},
	"subform":       {ValueType: ValueTypeRows, DisplayType: "行集合", FormulaAllowed: false},
}

var unknownFieldMeta = FieldMeta{ValueType: ValueTypeUnknown, DisplayType: "未支持"}

// Field 表单草稿投影出的公式变量。Key 使用 widgetName，绝不使用可变的 label。
type Field struct {
	Key            string    `json:"key"`
	Label          string    `json:"label"`
	WidgetType     string    `json:"widgetType"`
	ValueType      ValueType `json:"valueType"`
	DisplayType    string    `json:"displayType"`
	FormulaAllowed bool      `json:"formulaAllowed"`
}

// ProjectFields 从合法表单协议的顶层字段生成公式上下文。布局控件没有用户值，
// 必须排除；未纳入当前公式 DSL 的字段保留在结果中但 FormulaAllowed=false，
// 这样 API 消费方不会把它们误判为普通文本字段。
func ProjectFields(raw []byte) ([]Field, error) {
	var document struct {
		Content struct {
			Items []struct {
				Label  string `json:"label"`
				Widget struct {
					Type       string `json:"type"`
					WidgetName string `json:"widgetName"`
				} `json:"widget"`
			} `json:"items"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("decode form formula context: %w", err)
	}

	fields := make([]Field, 0, len(document.Content.Items))
	for _, item := range document.Content.Items {
		if item.Widget.Type == "separator" || item.Widget.Type == "button" {
			continue
		}
		if item.Widget.Type == "" || item.Widget.WidgetName == "" {
			return nil, fmt.Errorf("form item has no widget type or widgetName")
		}
		meta, ok := widgetVariableTypes[item.Widget.Type]
		if !ok {
			meta = unknownFieldMeta
		}
		label := item.Label
		if label == "" {
			label = item.Widget.WidgetName
		}
		fields = append(fields, Field{
			Key:            item.Widget.WidgetName,
			Label:          label,
			WidgetType:     item.Widget.Type,
			ValueType:      meta.ValueType,
			DisplayType:    meta.DisplayType,
			FormulaAllowed: meta.FormulaAllowed,
		})
	}
	return fields, nil
}
