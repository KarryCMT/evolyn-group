package service

import (
	"context"
	"fmt"
	"strings"
)

// validateDepartmentReferences 在值协议终审完成后复核部门目录。值校验只负责
// string 形状；目录校验才是租户隔离边界，不能由前端部门树或物理 BIGINT 转换替代。
// 当前仅开放 dept 单选，deptgroup 在具备数组物理存储前仍不会进入已发布快照。
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
		if field.widgetType != "dept" {
			continue
		}
		if reference, ok := values[name].(string); ok && strings.TrimSpace(reference) != "" {
			references = append(references, reference)
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
		if field.widgetType != "dept" {
			continue
		}
		reference, ok := values[name].(string)
		if !ok || strings.TrimSpace(reference) == "" || valid[reference] {
			continue
		}
		fieldErrors[name] = []string{fmt.Sprintf("%s不存在、已停用或无权选择", field.label)}
	}
	return fieldErrors, nil
}
