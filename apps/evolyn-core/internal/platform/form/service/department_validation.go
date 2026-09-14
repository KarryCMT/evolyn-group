package service

import (
	"context"
	"fmt"
	"strings"
)

// validateDepartmentReferences 在值协议终审完成后复核部门目录。值校验只负责
// string 形状；目录校验才是租户隔离边界，不能由前端部门树或物理 BIGINT 转换替代。
// 单选与多选均在此逐项复核；多选字段以 BIGINT[] 物理列保存，但目录校验仍
// 必须发生在写入前，避免浏览器构造跨租户或已停用的部门 ID。
func (s *formService) validateDepartmentReferences(
	ctx context.Context,
	content map[string]any,
	values map[string]any,
) (RecordFieldErrors, error) {
	fields, err := buildSnapshotFields(content)
	if err != nil {
		return nil, fmt.Errorf("published schema fields: %w", err)
	}
	references := make([]string, 0)
	for name, field := range fields {
		if field.widgetType != "dept" && field.widgetType != "deptgroup" {
			continue
		}
		if reference, ok := values[name].(string); ok && strings.TrimSpace(reference) != "" {
			references = append(references, reference)
		}
		if selections, ok := values[name].([]any); ok {
			for _, selection := range selections {
				if reference, ok := selection.(string); ok && strings.TrimSpace(reference) != "" {
					references = append(references, reference)
				}
			}
		}
	}
	if len(references) == 0 {
		return nil, nil
	}
	if s.departments == nil {
		// 目录校验是安全边界而非可选展示增强。生产装配遗漏时拒绝提交，避免
		// 伪造或跨租户 ID 被物理值层静默接纳。
		return nil, fmt.Errorf("department directory is not configured")
	}
	valid, err := s.departments.ResolveActiveDepartmentIDs(ctx, references)
	if err != nil {
		return nil, err
	}
	fieldErrors := RecordFieldErrors{}
	for name, field := range fields {
		if field.widgetType != "dept" && field.widgetType != "deptgroup" {
			continue
		}
		invalid := false
		switch references := values[name].(type) {
		case string:
			invalid = strings.TrimSpace(references) != "" && !valid[references]
		case []any:
			for _, selection := range references {
				reference, ok := selection.(string)
				if ok && strings.TrimSpace(reference) != "" && !valid[reference] {
					invalid = true
					break
				}
			}
		}
		if invalid {
			fieldErrors[name] = []string{fmt.Sprintf("%s不存在、已停用或无权选择", field.label)}
		}
	}
	return fieldErrors, nil
}
