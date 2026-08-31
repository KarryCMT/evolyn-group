// 权限感知提交管线（表单权限 P1，设计 §4/S9 定版）：edit 提交不复用
// ValidateSubmittedRecordValues，而是按三阶段执行：
//
//	① 信封解包——与既有规则一致：每个数据字段必须携带 visible 且等于发布
//	   快照可见性（资产权限隐藏不改变信封 visible，前端对权限隐藏字段照常
//	   携带快照可见性 + 空 data）；隐藏/布局字段不得携带数据；
//	② 权限合并——逐字段按 FieldsFor(op, record) 结果判定：editable=false
//	   且请求携带非空 data → 整体拒绝（越权，按 widgetName 回填错误）；
//	   editable=false 字段进入 rawValues 前以 previous 旧值回填（edit）或
//	   置 null（add，配置期规则已保证必填可编辑）；
//	③ 终审复用——以合并后的 rawValues 调用既有 ValidateRecordValues 完成
//	   类型/范围/必填终审（必填由旧值自然满足）。
//
// 越权拒绝口径对齐流程引擎 UpdateData：整体拒绝，不做静默裁剪。
package service

import (
	"encoding/json"

	"evolyn/internal/platform/form/model"
)

// permissionDeniedFieldMessage 越权字段错误文案（按 widgetName 回填）
const permissionDeniedFieldMessage = "没有编辑该字段的权限"

// ValidateSubmittedRecordValuesWithPermission 权限感知提交管线。
//
// content 为发布快照文档解码视图；submitted 为字段包装协议提交值；fields 为
// FieldsFor/FieldsForNew 的判定矩阵（nil 视为全字段不可编辑——deny-by-default
// 的防御默认，正常调用方必传完整矩阵）；previous 为编辑前的记录旧值
// （edit 场景；add 场景传 nil，非可编辑字段一律置 null）。
func ValidateSubmittedRecordValuesWithPermission(
	content map[string]any,
	submitted map[string]model.SubmitFieldValue,
	fields map[string]FieldPermission,
	previous map[string]any,
) (map[string]any, RecordFieldErrors) {
	snapshotFields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, RecordFieldErrors{"": {"表单快照异常，请刷新后重试"}}
	}
	rawValues := make(map[string]json.RawMessage, len(snapshotFields))
	fieldErrors := RecordFieldErrors{}

	for name, field := range snapshotFields {
		wrapped, exists := submitted[name]
		if field.widgetType == "separator" || field.widgetType == "button" {
			// ① 布局字段无值语义：不得进入提交值
			if exists {
				fieldErrors[name] = []string{"分割线等布局字段不能进入提交值"}
			}
			continue
		}
		if !exists || wrapped.Visible == nil {
			fieldErrors[name] = []string{"缺少字段可见状态"}
			continue
		}
		// ① 信封 visible 语义保持「发布快照可见性」，与资产权限隐藏解耦：
		// 权限隐藏字段照常携带快照可见性 + 空 data
		if *wrapped.Visible != field.visible {
			fieldErrors[name] = []string{"字段可见状态与发布快照不一致"}
			continue
		}
		if !field.visible {
			if len(wrapped.Data) > 0 && !isNullJSON(json.RawMessage(wrapped.Data)) {
				fieldErrors[name] = []string{"隐藏字段不能提交值"}
			}
			rawValues[name] = json.RawMessage(`null`)
			continue
		}

		// ② 权限合并：矩阵缺失键按 deny-by-default（不可编辑）
		editable := false
		if fields != nil {
			if permission, ok := fields[name]; ok {
				editable = permission.Editable
			}
		}
		carriesData := len(wrapped.Data) > 0 && !isNullJSON(json.RawMessage(wrapped.Data))
		if !editable {
			if carriesData {
				fieldErrors[name] = []string{permissionDeniedFieldMessage}
				continue
			}
			// 不可编辑字段不采纳提交值：edit 以旧值回填（必填由旧值自然满足），
			// add 置 null（配置期规则已保证必填字段在含 add 组中可编辑）
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

		if len(wrapped.Data) == 0 {
			rawValues[name] = json.RawMessage(`null`)
			continue
		}
		rawValues[name] = json.RawMessage(wrapped.Data)
	}

	for name := range submitted {
		if _, known := snapshotFields[name]; !known {
			fieldErrors[name] = []string{"提交了表单中不存在的字段"}
		}
	}

	// ③ 终审复用：合并结果按同一套快照校验完成类型/范围/必填终审
	cleaned, valueErrors := ValidateRecordValues(content, rawValues)
	for name, messages := range valueErrors {
		if _, exists := fieldErrors[name]; !exists {
			fieldErrors[name] = messages
		}
	}
	return cleaned, fieldErrors
}
