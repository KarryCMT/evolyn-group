// 权限组配置期校验器（表单权限 P1，设计 §3/§4/§5）：
//   - 操作键按 tn_forms.form_type 分派合法集（standard 出现 workflow_* 整体拒绝）；
//   - 字段矩阵按字段清单逐项校验（deny-by-default 语义见判定器，配置期只管
//     「键在清单内 + visible/editable 不矛盾 + 必填协调两规则」）；
//   - 数据范围按字段 widget.type（datetime 系再按 props.format）分派类型类，
//     operator 白名单与比较值形状在配置期终审，运行期（SQL 编译/内存匹配）
//     不再做配置判定。
//
// 字段清单事实源：最新发布版本 content schema（未发布回落草稿，同一提取器）；
// 权限域清单只含值字段（排除 separator/button 布局项——无值字段不进矩阵）。
package service

import (
	"fmt"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
)

// permissionFieldMeta 权限域字段清单条目（快照 schema 的最小投影）
type permissionFieldMeta struct {
	Key        string
	Label      string
	WidgetType string
	Format     string // datetime 系的 props.format（date/datetime/month/time）
	Required   bool   // 快照 allowBlank 反相
	Visible    bool   // 快照 visible（信封 visible 语义的唯一事实源）
}

// buildPermissionFieldList 从快照文档提取权限域字段清单（顶层有序值字段）。
// 复用提交校验的字段视图提取器（buildSnapshotFields），过滤布局项。
func buildPermissionFieldList(content map[string]any) ([]permissionFieldMeta, error) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, err
	}
	list := make([]permissionFieldMeta, 0, len(fields))
	for _, field := range fields {
		if field.widgetType == "separator" || field.widgetType == "button" {
			continue // 布局项无值，不进权限矩阵
		}
		format, _ := field.widget["format"].(string)
		list = append(list, permissionFieldMeta{
			Key:        field.widgetName,
			Label:      field.label,
			WidgetType: field.widgetType,
			Format:     format,
			Required:   !field.allowBlank,
			Visible:    field.visible,
		})
	}
	return list, nil
}

// permissionFieldIndex 清单索引：键 → 元数据
func permissionFieldIndex(list []permissionFieldMeta) map[string]permissionFieldMeta {
	index := make(map[string]permissionFieldMeta, len(list))
	for _, field := range list {
		index[field.Key] = field
	}
	return index
}

// ValidatePermissionOperations 校验操作键集合（设计 §3）：
// 未知键一律拒绝；standard 表单出现 workflow_* 键整体拒绝；重复键按去重收敛
// （幂等配置而非错误）。返回去重后的稳定键集。
func ValidatePermissionOperations(formType model.FormType, operations []string) ([]string, error) {
	legalSet := make(map[string]bool)
	for _, key := range model.LegalPermissionOperations(formType) {
		legalSet[key] = true
	}
	seen := make(map[string]bool, len(operations))
	deduped := make([]string, 0, len(operations))
	for _, op := range operations {
		if !legalSet[op] {
			return nil, httpx.Wrap(apperrors.ErrPermissionOperationInvalid,
				fmt.Errorf("operation %q is not legal for form type %s", op, formType))
		}
		if seen[op] {
			continue
		}
		seen[op] = true
		deduped = append(deduped, op)
	}
	return deduped, nil
}

// ValidatePermissionFieldRules 校验字段矩阵（设计 §4 配置期规则）：
//   - 字段键必须存在于字段清单（清单外键拒绝，防配置漂移）；
//   - visible=false 时 editable 必须为 false（校验器拒绝矛盾组合——值不出网
//     的字段无可编辑语义）；
//   - 必填协调一：visible=false 仅允许非必填字段（可见性裁剪会让必填字段
//     永远拿不到值）；
//   - 必填协调二：operations 含 add 时必填字段必须 editable=true（添加路径
//     无值可填，提交必败）。
func ValidatePermissionFieldRules(
	fieldList []permissionFieldMeta, rules []model.PermissionFieldRule, operations []string,
) error {
	index := permissionFieldIndex(fieldList)
	containsAdd := false
	for _, op := range operations {
		if op == model.PermissionOpAdd {
			containsAdd = true
			break
		}
	}
	seen := make(map[string]bool, len(rules))
	for _, rule := range rules {
		field, ok := index[rule.Field]
		if !ok {
			return httpx.Wrap(apperrors.ErrPermissionFieldInvalid,
				fmt.Errorf("field %q is not in the form field list", rule.Field))
		}
		if seen[rule.Field] {
			return httpx.Wrap(apperrors.ErrPermissionFieldInvalid,
				fmt.Errorf("field %q is configured more than once", rule.Field))
		}
		seen[rule.Field] = true
		if !rule.Visible && rule.Editable {
			return httpx.Wrap(apperrors.ErrPermissionFieldInvalid,
				fmt.Errorf("field %q: invisible field cannot be editable", rule.Field))
		}
		if !rule.Visible && field.Required {
			return httpx.Wrap(apperrors.ErrPermissionFieldInvalid,
				fmt.Errorf("field %q: required field cannot be invisible", rule.Field))
		}
		if containsAdd && field.Required && !rule.Editable {
			return httpx.Wrap(apperrors.ErrPermissionFieldInvalid,
				fmt.Errorf("field %q: required field must be editable when the group grants add", rule.Field))
		}
	}
	return nil
}

// ---- 数据范围类型分派（§5.1 operator 字典 × 字段类型白名单） ----

// permissionFieldClass 数据条件类型类：决定 operator 白名单与比较值形状
type permissionFieldClass string

const (
	permFieldClassText         permissionFieldClass = "text"         // 单/多行文本
	permFieldClassNumber       permissionFieldClass = "number"       // 数字
	permFieldClassDateTime     permissionFieldClass = "datetime"     // 日期时间（四形状定宽文本）
	permFieldClassSingleOption permissionFieldClass = "singleOption" // 单选/成员/部门
	permFieldClassMultiOption  permissionFieldClass = "multiOption"  // 多选/成员多选/部门多选
)

// permissionClassOfWidget 字段 widget.type → 类型类分派；返回空串表示该类型
// 不支持数据条件（无确定值形状的字段：图片/附件/定位/签名/关联/子表单等）
func permissionClassOfWidget(widgetType string) permissionFieldClass {
	switch widgetType {
	case "text", "textarea":
		return permFieldClassText
	case "number":
		return permFieldClassNumber
	case "datetime":
		return permFieldClassDateTime
	case "radiogroup", "combo", "user", "dept":
		return permFieldClassSingleOption
	case "checkboxgroup", "combocheck", "usergroup", "deptgroup":
		return permFieldClassMultiOption
	default:
		return ""
	}
}

// permissionClassOperators 类型类 → operator 白名单（§5.1 字典）
var permissionClassOperators = map[permissionFieldClass]map[string]bool{
	permFieldClassText:         {"eq": true, "ne": true, "contains": true, "empty": true, "not_empty": true},
	permFieldClassNumber:       {"eq": true, "ne": true, "gt": true, "gte": true, "lt": true, "lte": true, "empty": true, "not_empty": true},
	permFieldClassDateTime:     {"eq": true, "ne": true, "gt": true, "gte": true, "lt": true, "lte": true, "empty": true, "not_empty": true},
	permFieldClassSingleOption: {"eq": true, "ne": true, "in": true, "not_in": true, "empty": true, "not_empty": true},
	permFieldClassMultiOption:  {"contains": true, "in": true, "not_in": true, "empty": true, "not_empty": true},
}

// maxPermissionScopeConditions 单组数据条件数上限（防御性收敛配置体积；
// 组数 ≤ 50/表、条件逐条参与判定，超限配置无业务意义）
const maxPermissionScopeConditions = 50

// ValidatePermissionDataScope 校验数据范围（设计 §5 配置期规则）：
// match ∈ {all, any}（空回落 all）；逐条件按字段类型类校验 operator 白名单
// 与比较值形状（datetime 系按 props.format 分形状）。空条件 = 全部数据（S6）。
func ValidatePermissionDataScope(fieldList []permissionFieldMeta, scope *model.PermissionDataScopeSpec) error {
	if scope == nil {
		return nil
	}
	if scope.Match != "" && scope.Match != model.PermissionScopeMatchAll && scope.Match != model.PermissionScopeMatchAny {
		return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
			fmt.Errorf("data scope match %q must be all/any", scope.Match))
	}
	if len(scope.Conditions) > maxPermissionScopeConditions {
		return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
			fmt.Errorf("data scope conditions %d exceed limit %d", len(scope.Conditions), maxPermissionScopeConditions))
	}
	index := permissionFieldIndex(fieldList)
	for _, condition := range scope.Conditions {
		field, ok := index[condition.Field]
		if !ok {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("data scope field %q is not in the form field list", condition.Field))
		}
		class := permissionClassOfWidget(field.WidgetType)
		if class == "" {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("data scope field %q (type %s) does not support conditions", field.Key, field.WidgetType))
		}
		operators, ok := permissionClassOperators[class]
		if !ok || !operators[condition.Operator] {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("operator %q is not applicable to field %q (type %s)", condition.Operator, field.Key, field.WidgetType))
		}
		if err := validateScopeConditionValue(field, class, condition); err != nil {
			return err
		}
	}
	return nil
}

// validateScopeConditionValue 比较值形状校验：empty/not_empty 须为空数组；
// 标量 operator 逐类校验元素个数与元素形状。
func validateScopeConditionValue(field permissionFieldMeta, class permissionFieldClass, condition model.PermissionDataCondition) error {
	switch condition.Operator {
	case "empty", "not_empty":
		if len(condition.Value) != 0 {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("operator %q requires an empty value array", condition.Operator))
		}
		return nil
	case "eq", "ne", "contains":
		if len(condition.Value) != 1 {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("operator %q requires exactly one value", condition.Operator))
		}
		return validateScopeScalarValue(field, class, condition.Value[0])
	case "gt", "gte", "lt", "lte":
		if len(condition.Value) != 1 {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("operator %q requires exactly one value", condition.Operator))
		}
		// 比较 operator 的比较值仅允许数字/日期两类
		if class == permFieldClassNumber {
			if _, ok := jsonFloat(condition.Value[0]); !ok {
				return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
					fmt.Errorf("field %q comparison value must be a number", field.Key))
			}
			return nil
		}
		return validateScopeDateTimeShape(field, condition.Value[0])
	case "in", "not_in":
		for _, element := range condition.Value {
			if _, ok := element.(string); !ok {
				return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
					fmt.Errorf("field %q set values must be strings", field.Key))
			}
		}
		return nil
	default:
		return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
			fmt.Errorf("unknown operator %q", condition.Operator))
	}
}

// validateScopeScalarValue 标量比较值形状：文本/单选类须字符串；数字类须数字；
// 日期类按 format 形状正则。
func validateScopeScalarValue(field permissionFieldMeta, class permissionFieldClass, value any) error {
	switch class {
	case permFieldClassText, permFieldClassSingleOption:
		if _, ok := value.(string); !ok {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("field %q value must be a string", field.Key))
		}
	case permFieldClassNumber:
		if _, ok := jsonFloat(value); !ok {
			return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
				fmt.Errorf("field %q value must be a number", field.Key))
		}
	case permFieldClassDateTime:
		return validateScopeDateTimeShape(field, value)
	}
	return nil
}

// validateScopeDateTimeShape 日期比较值形状校验（定宽零填充形状，与 §5.2
// 运行期守卫同正则；真实日历校验不在此处——形状守卫只保证可比较性）
func validateScopeDateTimeShape(field permissionFieldMeta, value any) error {
	text, ok := value.(string)
	if !ok {
		return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
			fmt.Errorf("field %q datetime value must be a string", field.Key))
	}
	format := field.Format
	if format == "" {
		format = "datetime"
	}
	pattern, ok := timeShapePatterns[format]
	if !ok || !pattern.MatchString(text) {
		return httpx.Wrap(apperrors.ErrPermissionDataScopeInvalid,
			fmt.Errorf("field %q datetime value does not match format %s", field.Key, format))
	}
	return nil
}
