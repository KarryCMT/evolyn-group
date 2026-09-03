// 不可见字段赋值值决议器（v6 纯领域服务，设计方案 §4.3/§6.2）。
//
// 统一事务内顺序：解析发布快照与字段权限 → 校验提交键与信封 → 合并允许
// 写入的原始值 → 编译/求值显隐规则得到 effectiveVisible → 对不可见字段按
// clear / preserve / recompute 决议 → 对最终值执行类型、必填、范围、选项
// 终审。新建提交、记录编辑与流程写回共用同一入口（ResolveSubmittedValues /
// ResolveMergedRecordValues），不得各自实现策略；任何客户端提交的隐藏字段
// data 都不能成为 preserve 或 recompute 的数据来源——preserve 只取事务内
// 锁定的基线，recompute 只取服务端执行器结果。
package service

import (
	"encoding/json"
	"fmt"

	"evolyn/internal/platform/form/model"
)

// ResolveSubmittedValues v6 信封提交决议（新建/编辑统一入口）。
//
// content 为发布快照文档解码视图；submitted 为字段包装协议提交值
// （values[widgetName] = {data?, visible}）；permissions 为 FieldsFor/
// FieldsForNew 判定矩阵（nil 表示无权限组基线，全量放行；提供后缺失键
// deny-by-default）；previous 为受理前锁定的记录基线（编辑/流程场景；
// 新建传 nil）；currentMemberID 为提交人（显隐规则 includeCurrentMember
// 注入源）。返回策略决议后的完整字段值（所有可处理字段均保留键）与按
// widgetName 回填的错误集合。
func ResolveSubmittedValues(
	content map[string]any,
	submitted map[string]model.SubmitFieldValue,
	permissions map[string]FieldPermission,
	previous map[string]any,
	currentMemberID string,
) (map[string]any, RecordFieldErrors) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, RecordFieldErrors{"": {"表单快照异常，请刷新后重试"}}
	}
	policy := parseInvisibleValuePolicy(content)
	permissionVisible, permissionEditable := permissionLookups(permissions)

	// 有效可见性 = 静态 ∧ 权限 ∧ 显隐规则；条件值只读客户端提交的可写 data。
	visibility := effectiveFieldVisibility(fields, content, permissionVisible, func(name string) any {
		wrapped, ok := submitted[name]
		if !ok || len(wrapped.Data) == 0 {
			return nil
		}
		return decodeShowValue(json.RawMessage(wrapped.Data))
	}, currentMemberID)

	rawValues := make(map[string]json.RawMessage, len(fields))
	fieldErrors := RecordFieldErrors{}

	for name, field := range fields {
		wrapped, exists := submitted[name]
		if field.widgetType == "separator" || field.widgetType == "button" {
			// 布局项无值语义：不得进入提交值。
			if exists {
				fieldErrors[name] = []string{"分割线等布局字段不能进入提交值"}
			}
			continue
		}
		if !exists || wrapped.Visible == nil {
			fieldErrors[name] = []string{"缺少字段可见状态"}
			continue
		}
		effective := visibility[name]
		// 信封 visible 必须等于服务端算出的有效可见性（§4.3）。
		if *wrapped.Visible != effective {
			fieldErrors[name] = []string{"字段可见状态与发布快照不一致"}
			continue
		}
		carriesData := len(wrapped.Data) > 0 && !isNullJSON(json.RawMessage(wrapped.Data))
		if !effective {
			// 有效不可见字段一律不得携带 data——伪造隐藏值直接拒绝。
			if carriesData {
				fieldErrors[name] = []string{"隐藏字段不能提交值"}
				continue
			}
			resolveInvisibleField(name, field, policy, previous, rawValues, fieldErrors)
			continue
		}

		// 可见但不可编辑：沿用既有权限管线，只允许服务端从基线合并。
		if !permissionEditable(name) {
			if carriesData {
				fieldErrors[name] = []string{permissionDeniedFieldMessage}
				continue
			}
			if backfill, ok := previous[name]; ok && backfill != nil {
				raw, marshalErr := json.Marshal(backfill)
				if marshalErr != nil {
					fieldErrors[name] = []string{"表单快照异常，请刷新后重试"}
					continue
				}
				rawValues[name] = raw
			} else {
				rawValues[name] = json.RawMessage(`null`)
			}
			continue
		}

		if !carriesData {
			rawValues[name] = json.RawMessage(`null`)
			continue
		}
		rawValues[name] = json.RawMessage(wrapped.Data)
	}

	for name := range submitted {
		if _, known := fields[name]; !known {
			fieldErrors[name] = []string{"提交了表单中不存在的字段"}
		}
	}
	return finalizeResolvedValues(fields, visibility, rawValues, fieldErrors)
}

// ResolveMergedRecordValues v6 受信合并决议（流程写回路径，WorkflowRecordStore）：
// merged 为「记录基线 ∪ 服务端受信 patch」的全量值（无信封），以合并值重算
// 有效可见性后执行同一套策略决议。写回路径无提交人身份上下文，显隐规则按
// 匿名口径求值（includeCurrentMember 不注入任何成员）。
func ResolveMergedRecordValues(
	content map[string]any,
	merged map[string]any,
	baseline map[string]any,
) (map[string]any, RecordFieldErrors) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, RecordFieldErrors{"": {"表单快照异常，请刷新后重试"}}
	}
	policy := parseInvisibleValuePolicy(content)
	visibility := effectiveFieldVisibility(fields, content, nil, func(name string) any {
		return merged[name]
	}, "")

	rawValues := make(map[string]json.RawMessage, len(fields))
	fieldErrors := RecordFieldErrors{}
	for name, field := range fields {
		if field.widgetType == "separator" || field.widgetType == "button" {
			// 布局项无值：合并值中的残留静默丢弃，不进入落库值。
			continue
		}
		if !visibility[name] {
			resolveInvisibleField(name, field, policy, baseline, rawValues, fieldErrors)
			continue
		}
		value := merged[name]
		if value == nil {
			rawValues[name] = json.RawMessage(`null`)
			continue
		}
		raw, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			fieldErrors[name] = []string{"表单快照异常，请刷新后重试"}
			continue
		}
		rawValues[name] = raw
	}
	return finalizeResolvedValues(fields, visibility, rawValues, fieldErrors)
}

// resolveInvisibleField 对有效不可见字段按策略决议值（§4.1）：
// clear 写类型空值；preserve 保留受理前基线（新建无基线即为类型空值）；
// recompute 需派生执行器——发布校验已拒绝该配置，防御路径按字段错误收口
// （fail-closed，不降级为相信前端旧值）。
func resolveInvisibleField(
	name string,
	field snapshotField,
	policy invisibleValuePolicy,
	baseline map[string]any,
	rawValues map[string]json.RawMessage,
	fieldErrors RecordFieldErrors,
) {
	switch policy.strategyOf(name) {
	case submitRulePreserve:
		if value, ok := baseline[name]; ok && value != nil {
			raw, err := json.Marshal(value)
			if err != nil {
				fieldErrors[name] = []string{"表单快照异常，请刷新后重试"}
				return
			}
			rawValues[name] = raw
			return
		}
		writeTypedEmpty(name, field.widgetType, rawValues)
	case submitRuleRecompute:
		fieldErrors[name] = []string{"该字段的重算能力尚未开放"}
	default:
		writeTypedEmpty(name, field.widgetType, rawValues)
	}
}

// writeTypedEmpty 类型化空值经 JSON 原文进入终审（多选 []，其余 null）。
func writeTypedEmpty(name, widgetType string, rawValues map[string]json.RawMessage) {
	if multiValueWidgetTypes[widgetType] {
		rawValues[name] = json.RawMessage(`[]`)
		return
	}
	rawValues[name] = json.RawMessage(`null`)
}

// finalizeResolvedValues 终审（§4.3）：可见字段执行类型、必填、范围与选项
// 校验；不可见字段的策略决议值（类型空值/锁定基线）是服务端权威结果，不再
// 复核——必填约束只对本次操作者可见字段生效（§3.2）。输出保留全部可处理
// 字段键（§4.1——区分「快照中存在但本次为空」与「未知字段」）。
func finalizeResolvedValues(
	fields map[string]snapshotField,
	visibility map[string]bool,
	rawValues map[string]json.RawMessage,
	fieldErrors RecordFieldErrors,
) (map[string]any, RecordFieldErrors) {
	cleaned := make(map[string]any, len(fields))
	for name, field := range fields {
		if field.widgetType == "separator" || field.widgetType == "button" {
			continue
		}
		if _, rejected := fieldErrors[name]; rejected {
			continue
		}
		raw, submitted := rawValues[name]
		var value any
		if submitted && !isNullJSON(raw) {
			if err := json.Unmarshal(raw, &value); err != nil {
				fieldErrors[name] = []string{fmt.Sprintf("%s的值类型不正确", field.label)}
				continue
			}
		}
		if visibility[name] {
			if errs := validateFieldValue(field, value); len(errs) > 0 {
				fieldErrors[name] = errs
				continue
			}
		}
		if isEmptyValue(value) {
			cleaned[name] = emptyValueForType(field.widgetType)
			continue
		}
		cleaned[name] = value
	}
	return cleaned, fieldErrors
}

// permissionLookups 权限矩阵读取闭包：nil 矩阵 = 无权限组基线全量放行；
// 提供后缺失键 deny-by-default（与 FieldsFor 投影同口径）。
func permissionLookups(
	permissions map[string]FieldPermission,
) (visible func(name string) bool, editable func(name string) bool) {
	if permissions == nil {
		return func(string) bool { return true }, func(string) bool { return true }
	}
	return func(name string) bool {
			permission, ok := permissions[name]
			return ok && permission.Visible
		}, func(name string) bool {
			permission, ok := permissions[name]
			return ok && permission.Editable
		}
}
