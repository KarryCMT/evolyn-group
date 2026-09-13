package service

import (
	"context"
	"fmt"
	"strings"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
)

const subformMemberReferenceFieldPrefix = "__evolyn_subform"

// hydrateMemberReferences 从字段可见性裁剪后的值中提取成员引用，一次查询目录后
// 写回仅供展示的映射。原始 Values 从不替换，筛选、导出及历史兼容性不受影响。
func (s *formService) hydrateMemberReferences(ctx context.Context, items []model.FormRecordDTO, content map[string]any) error {
	if s.memberRefs == nil || len(items) == 0 {
		return nil
	}
	fields, ok := documentItems(content)
	if !ok {
		return fmt.Errorf("published schema missing items")
	}
	references := make([]string, 0)
	keysByItem := make([]map[string][]string, len(items))
	for index := range items {
		keys := make(map[string][]string)
		collectMemberReferences(fields, items[index].Values, "", keys)
		keysByItem[index] = keys
		for _, entries := range keys {
			references = append(references, entries...)
		}
	}
	resolved, err := s.memberRefs.ResolveMemberReferences(ctx, references)
	if err != nil {
		return err
	}
	byReference := make(map[string]model.MemberReference, len(resolved))
	for _, entry := range resolved {
		if entry.Reference != "" {
			byReference[entry.Reference] = entry
		}
	}
	for index := range items {
		presentations := make(map[string][]model.MemberReference)
		for field, references := range keysByItem[index] {
			for _, reference := range references {
				if entry, exists := byReference[reference]; exists {
					presentations[field] = append(presentations[field], entry)
				}
			}
		}
		if len(presentations) > 0 {
			items[index].MemberReferences = presentations
		}
	}
	return nil
}

// collectMemberReferences 镜像前端子表单扁平字段名：顶层成员字段使用自身键，
// 子表单成员字段使用 __evolyn_subform:<parent>:<child>，与表格列精确对齐。
func collectMemberReferences(items []any, values map[string]any, prefix string, out map[string][]string) {
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		widget, _ := item["widget"].(map[string]any)
		name, _ := widget["widgetName"].(string)
		kind, _ := widget["type"].(string)
		if name == "" {
			continue
		}
		switch kind {
		case "user", "usergroup":
			field := name
			if prefix != "" {
				field = prefix + ":" + name
			}
			if refs := memberReferenceValues(values[name]); len(refs) > 0 {
				// 子表单的同一列会在多行出现，按表格字段键累积为一次批量解析输入。
				out[field] = append(out[field], refs...)
			}
		case "subform":
			children, _ := widget["items"].([]any)
			rows, _ := values[name].([]any)
			for _, rawRow := range rows {
				row, ok := rawRow.(map[string]any)
				if !ok {
					continue
				}
				collectMemberReferences(children, row, subformMemberReferenceFieldPrefix+":"+name, out)
			}
		}
	}
}

func memberReferenceValues(value any) []string {
	var values []any
	switch typed := value.(type) {
	case string:
		values = []any{typed}
	case []any:
		values = typed
	case []string:
		values = make([]any, len(typed))
		for index := range typed {
			values[index] = typed[index]
		}
	default:
		return nil
	}
	result := make([]string, 0, len(values))
	for _, raw := range values {
		if value, ok := raw.(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				result = append(result, value)
			}
		}
	}
	return result
}

// GetRecordMemberCard 复用记录数据面的访问门：只有拥有该表单记录查看权限的
// 当前租户成员可以读取卡片，并且响应仅含允许公开的系统编号、姓名与部门。
func (s *formService) GetRecordMemberCard(ctx context.Context, member *iammodel.User, code, reference string) (*model.FormRecordMemberCard, error) {
	if s.memberRefs == nil {
		return nil, fmt.Errorf("member reference directory is not configured")
	}
	if member == nil || member.ID == 0 || !s.access.Permissions(ctx, member)["form-records:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot view form record members"))
	}
	if _, err := s.loadByCode(ctx, code); err != nil {
		return nil, err
	}
	entries, err := s.memberRefs.ResolveMemberReferences(ctx, []string{reference})
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Reference == reference {
			return &model.FormRecordMemberCard{MemberCode: entry.MemberCode, Name: entry.Name, Avatar: entry.Avatar, Status: entry.Status, Departments: entry.DepartmentNames}, nil
		}
	}
	return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member reference not found"))
}
