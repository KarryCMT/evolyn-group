package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

type linkageDocument struct {
	Content struct {
		Linkages []linkageRule `json:"linkages"`
	} `json:"content"`
}

type linkageRule struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	Source  struct {
		Type     string `json:"type"`
		AppID    uint   `json:"appId"`
		SourceID string `json:"sourceId"`
	} `json:"source"`
	Filter struct {
		Logic      string             `json:"logic"`
		Conditions []linkageCondition `json:"conditions"`
	} `json:"filter"`
	Mappings []linkageMapping `json:"mappings"`
	Runtime  struct {
		EmptyStrategy string `json:"emptyStrategy"`
	} `json:"runtime"`
}

type linkageCondition struct {
	ID            string `json:"id"`
	SourceFieldID string `json:"sourceFieldId"`
	Operator      string `json:"operator"`
	Value         *struct {
		Type    string `json:"type"`
		FieldID string `json:"fieldId"`
		Value   any    `json:"value"`
	} `json:"value,omitempty"`
}

type linkageMapping struct {
	SourceFieldID string `json:"sourceFieldId"`
	TargetFieldID string `json:"targetFieldId"`
}

// ListLinkageFields 读取数据源最新发布快照，并按当前成员 view 字段矩阵裁剪。
// 返回的全部标识均为逻辑 widgetName，不暴露物理列。
func (s *formService) ListLinkageFields(ctx context.Context, member *iammodel.User, sourceCode string) ([]model.LinkageSourceField, error) {
	if member == nil || !s.access.Permissions(ctx, member)["forms:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot inspect linkage source %s", sourceCode))
	}
	form, version, content, fields, err := s.loadPublishedLinkageSource(ctx, sourceCode)
	if err != nil {
		return nil, err
	}
	_ = version
	_ = content
	resolved, err := s.evaluatePermissions(ctx, member, form.ID)
	if err != nil {
		return nil, err
	}
	if resolved != nil && !resolved.EntranceAllowed() {
		return nil, httpx.Wrap(apperrors.ErrLinkagePermissionDenied, fmt.Errorf("member cannot view source form %s", sourceCode))
	}
	var permissions map[string]FieldPermission
	if resolved != nil {
		permissions = resolved.FieldsForNew(model.PermissionOpView)
	}
	result := make([]model.LinkageSourceField, 0, len(fields))
	for _, field := range fields {
		if permissions != nil && !permissions[field.Key].Visible {
			continue
		}
		operators := linkageOperatorsForType(field.WidgetType)
		if len(operators) == 0 {
			continue
		}
		result = append(result, model.LinkageSourceField{
			ID: field.Key, Name: field.Label, Type: field.WidgetType, Operators: operators,
		})
	}
	return result, nil
}

// ExecuteLinkage 只信任当前表单发布快照中的规则。浏览器上传的 values 仅作为
// 条件参数；源表、字段、操作符和映射全部由服务端规则决定。
func (s *formService) ExecuteLinkage(ctx context.Context, member *iammodel.User, code, ruleID string, req *model.ExecuteLinkageRequest) (*model.ExecuteLinkageResult, error) {
	if member == nil {
		return nil, httpx.Wrap(apperrors.ErrLinkagePermissionDenied, fmt.Errorf("anonymous linkage request"))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %s not published", code))
	}
	targetPermissions, err := s.evaluatePermissions(ctx, member, form.ID)
	if err != nil {
		return nil, err
	}
	if targetPermissions != nil && !targetPermissions.EntranceAllowed() {
		return nil, httpx.Wrap(apperrors.ErrLinkagePermissionDenied, fmt.Errorf("member cannot view target form %s", code))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		return nil, err
	}
	if req.SchemaVersion != version.VersionNo {
		return nil, httpx.Wrap(apperrors.ErrLinkageVersionMismatch, fmt.Errorf("linkage schema version %d != %d", req.SchemaVersion, version.VersionNo))
	}
	document, err := decodeLinkageDocument(version.Content)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrLinkageQueryFailed, err)
	}
	rule, ok := linkageRuleByID(document.Content.Linkages, ruleID)
	if !ok {
		return nil, httpx.Wrap(apperrors.ErrLinkageRuleNotFound, fmt.Errorf("rule %s not found", ruleID))
	}
	if !rule.Enabled {
		return nil, httpx.Wrap(apperrors.ErrLinkageRuleDisabled, fmt.Errorf("rule %s disabled", ruleID))
	}
	if rule.Source.Type != "form" || rule.Source.AppID != form.AppID {
		return nil, httpx.Wrap(apperrors.ErrLinkageSourceNotFound, fmt.Errorf("rule %s source is outside current app", ruleID))
	}

	children := make([]model.RecordQueryExpression, 0, len(rule.Filter.Conditions))
	for _, condition := range rule.Filter.Conditions {
		value, hasValue, emptyDependency := resolveLinkageConditionValue(condition, req.Values)
		if emptyDependency {
			return &model.ExecuteLinkageResult{RuleID: ruleID, RequestVersion: req.RequestVersion, Matched: false, Values: map[string]any{}}, nil
		}
		expression := model.RecordQueryExpression{Type: "condition", Field: condition.SourceFieldID, Operator: condition.Operator}
		if hasValue {
			expression.Value = value
		}
		children = append(children, expression)
	}
	query := model.RecordQueryDocument{
		Version: 1,
		Filter:  &model.RecordQueryExpression{Type: "group", Conjunction: rule.Filter.Logic, Children: children},
		Paging:  model.RecordQueryPaging{Page: 1, PageSize: 2},
	}
	page, err := s.ListRecords(ctx, member, rule.Source.SourceID, query)
	if err != nil {
		return nil, err
	}
	result := &model.ExecuteLinkageResult{RuleID: ruleID, RequestVersion: req.RequestVersion, Values: map[string]any{}}
	if len(page.Items) == 0 {
		return result, nil
	}
	result.Matched = true
	row := page.Items[0].Values
	for _, mapping := range rule.Mappings {
		value, readable := row[mapping.SourceFieldID]
		if !readable {
			return nil, httpx.Wrap(apperrors.ErrLinkagePermissionDenied,
				fmt.Errorf("source field %s is not readable", mapping.SourceFieldID))
		}
		result.Values[mapping.TargetFieldID] = value
	}
	return result, nil
}

func (s *formService) loadPublishedLinkageSource(ctx context.Context, code string) (*model.Form, *model.FormVersion, map[string]any, []permissionFieldMeta, error) {
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil, nil, httpx.Wrap(apperrors.ErrLinkageSourceNotFound, err)
		}
		return nil, nil, nil, nil, err
	}
	if form.LatestVersionID == nil {
		return nil, nil, nil, nil, httpx.Wrap(apperrors.ErrLinkageSourceNotFound, fmt.Errorf("source %s not published", code))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	content := make(map[string]any)
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return nil, nil, nil, nil, err
	}
	fields, err := buildPermissionFieldList(content)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return form, version, content, fields, nil
}

func decodeLinkageDocument(content model.JSONContent) (*linkageDocument, error) {
	document := new(linkageDocument)
	if err := json.Unmarshal(content, document); err != nil {
		return nil, fmt.Errorf("decode published linkage schema: %w", err)
	}
	return document, nil
}

func linkageRuleByID(rules []linkageRule, id string) (linkageRule, bool) {
	for _, rule := range rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return linkageRule{}, false
}

func resolveLinkageConditionValue(condition linkageCondition, values map[string]any) (any, bool, bool) {
	if condition.Operator == "empty" || condition.Operator == "not_empty" {
		return nil, false, false
	}
	if condition.Value == nil {
		return nil, false, true
	}
	if condition.Value.Type == "constant" {
		return condition.Value.Value, true, false
	}
	value, ok := values[condition.Value.FieldID]
	if !ok || linkageValueEmpty(value) {
		return nil, false, true
	}
	return value, true, false
}

func linkageValueEmpty(value any) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	if values, ok := value.([]any); ok {
		return len(values) == 0
	}
	return false
}

func linkageOperatorsForType(widgetType string) []string {
	switch widgetType {
	case "text", "textarea":
		return []string{"eq", "neq", "contains", "notContains", "empty", "not_empty"}
	case "number", "decimal", "money", "percent", "datetime":
		return []string{"eq", "neq", "gt", "gte", "lt", "lte", "empty", "not_empty"}
	case "radiogroup", "combo":
		return []string{"eq", "neq", "in", "not_in", "empty", "not_empty"}
	case "checkboxgroup", "combocheck":
		return []string{"contains", "in", "not_in", "empty", "not_empty"}
	case "user", "dept":
		return []string{"eq", "neq", "contains", "empty", "not_empty"}
	case "usergroup", "deptgroup":
		return []string{"contains", "empty", "not_empty"}
	default:
		return nil
	}
}

func linkageTypesCompatible(sourceType, targetType string) bool {
	if sourceType == targetType {
		return true
	}
	return permissionClassOfWidget(sourceType) == permFieldClassNumber && permissionClassOfWidget(targetType) == permFieldClassNumber
}

// validatePublishedLinkageReferences 在发布目标表单前读取每个源表最新发布快照，
// 终审源字段存在性、操作符能力和映射类型。规则仍只写入目标版本 Schema。
func (s *formService) validatePublishedLinkageReferences(ctx context.Context, form *model.Form) error {
	document, err := decodeLinkageDocument(form.DraftContent)
	if err != nil {
		return err
	}
	currentContent := make(map[string]any)
	if err := json.Unmarshal(form.DraftContent, &currentContent); err != nil {
		return err
	}
	currentFields, err := buildPermissionFieldList(currentContent)
	if err != nil {
		return err
	}
	currentIndex := permissionFieldIndex(currentFields)
	for _, rule := range document.Content.Linkages {
		if rule.Source.Type != "form" || rule.Source.AppID != form.AppID {
			return httpx.Wrap(apperrors.ErrLinkageSourceNotFound, fmt.Errorf("rule %s source outside current app", rule.ID))
		}
		sourceForm, _, _, sourceFields, err := s.loadPublishedLinkageSource(ctx, rule.Source.SourceID)
		if err != nil {
			return err
		}
		if sourceForm.AppID != form.AppID {
			return httpx.Wrap(apperrors.ErrLinkageSourceNotFound, fmt.Errorf("source %s app mismatch", rule.Source.SourceID))
		}
		sourceIndex := permissionFieldIndex(sourceFields)
		for _, condition := range rule.Filter.Conditions {
			source, ok := sourceIndex[condition.SourceFieldID]
			if !ok {
				return httpx.Wrap(apperrors.ErrLinkageFieldNotFound, fmt.Errorf("source field %s not found", condition.SourceFieldID))
			}
			if !containsLinkageOperator(linkageOperatorsForType(source.WidgetType), condition.Operator) {
				return httpx.Wrap(apperrors.ErrLinkageOperatorUnsupported, fmt.Errorf("operator %s not allowed for %s", condition.Operator, source.WidgetType))
			}
			if condition.Value != nil && condition.Value.Type == "field" {
				current, ok := currentIndex[condition.Value.FieldID]
				if !ok {
					return httpx.Wrap(apperrors.ErrLinkageFieldNotFound, fmt.Errorf("dependency field %s not found", condition.Value.FieldID))
				}
				if !linkageTypesCompatible(source.WidgetType, current.WidgetType) {
					return httpx.Wrap(apperrors.ErrLinkageFieldTypeMismatch, fmt.Errorf("condition %s type mismatch", condition.ID))
				}
			}
		}
		for _, mapping := range rule.Mappings {
			source, sourceOK := sourceIndex[mapping.SourceFieldID]
			target, targetOK := currentIndex[mapping.TargetFieldID]
			if !sourceOK || !targetOK {
				return httpx.Wrap(apperrors.ErrLinkageFieldNotFound, fmt.Errorf("mapping %s -> %s not found", mapping.SourceFieldID, mapping.TargetFieldID))
			}
			if !linkageTypesCompatible(source.WidgetType, target.WidgetType) {
				return httpx.Wrap(apperrors.ErrLinkageFieldTypeMismatch, fmt.Errorf("mapping %s -> %s type mismatch", mapping.SourceFieldID, mapping.TargetFieldID))
			}
		}
	}
	return nil
}

func containsLinkageOperator(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
