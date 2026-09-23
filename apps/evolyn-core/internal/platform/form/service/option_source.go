package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
)

var relatedOptionFieldTypes = map[string]bool{
	"text": true, "number": true, "datetime": true, "radiogroup": true,
	"checkboxgroup": true, "combo": true, "combocheck": true, "sn": true,
}

type optionSourceDocument struct {
	Content struct {
		Items []struct {
			Widget struct {
				Type         string              `json:"type"`
				WidgetName   string              `json:"widgetName"`
				OptionSource *optionSourceConfig `json:"optionSource"`
			} `json:"widget"`
		} `json:"items"`
	} `json:"content"`
}

type optionSourceConfig struct {
	Mode    string                     `json:"mode"`
	Related *relatedOptionSourceConfig `json:"related"`
}

type relatedOptionSourceConfig struct {
	Source struct {
		Type     string `json:"type"`
		AppID    uint   `json:"appId"`
		SourceID string `json:"sourceId"`
		FieldID  string `json:"fieldId"`
	} `json:"source"`
	Sort struct {
		FieldID   string `json:"fieldId"`
		Direction string `json:"direction"`
	} `json:"sort"`
	Filter struct {
		Logic      string             `json:"logic"`
		Conditions []linkageCondition `json:"conditions"`
	} `json:"filter"`
}

func decodeOptionSourceDocument(content model.JSONContent) (*optionSourceDocument, error) {
	document := new(optionSourceDocument)
	if err := json.Unmarshal(content, document); err != nil {
		return nil, fmt.Errorf("decode option source schema: %w", err)
	}
	return document, nil
}

func optionSourceByField(document *optionSourceDocument, fieldID string) (*relatedOptionSourceConfig, bool) {
	for _, item := range document.Content.Items {
		if item.Widget.WidgetName != fieldID || (item.Widget.Type != "combo" && item.Widget.Type != "combocheck") {
			continue
		}
		config := item.Widget.OptionSource
		if config != nil && config.Mode == "related" && config.Related != nil {
			return config.Related, true
		}
		return nil, false
	}
	return nil, false
}

// QueryRelatedOptions 从目标表单不可变发布快照读取可信配置，再复用记录查询的
// 行级权限、字段权限和参数化谓词。浏览器不能指定源表、源字段或排序表达式。
func (s *formService) QueryRelatedOptions(ctx context.Context, member *iammodel.User, code, fieldID string, req *model.QueryRelatedOptionsRequest) (*model.RelatedOptionPage, error) {
	if member == nil {
		return nil, httpx.Wrap(apperrors.ErrOptionSourcePermission, fmt.Errorf("anonymous option query"))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %s not published", code))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		return nil, err
	}
	if req.SchemaVersion != version.VersionNo {
		return nil, httpx.Wrap(apperrors.ErrOptionSourceVersionConflict, fmt.Errorf("option schema version mismatch"))
	}
	document, err := decodeOptionSourceDocument(version.Content)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrOptionSourceInvalid, err)
	}
	config, ok := optionSourceByField(document, fieldID)
	if !ok || config.Source.Type != "form" || config.Source.AppID != form.AppID {
		return nil, httpx.Wrap(apperrors.ErrOptionSourceNotFound, fmt.Errorf("option source for %s not found", fieldID))
	}
	return s.queryRelatedOptionConfig(ctx, member, config, req)
}

// PreviewRelatedOptions 从服务端已保存草稿读取关联配置。设计器先保存再预览，
// 因而既能展示未发布配置，又不需要信任浏览器上传的数据源/字段/排序定义。
func (s *formService) PreviewRelatedOptions(ctx context.Context, member *iammodel.User, code, fieldID string, req *model.PreviewRelatedOptionsRequest) (*model.RelatedOptionPage, error) {
	if member == nil || !s.access.Permissions(ctx, member)["forms:update"] {
		return nil, httpx.Wrap(apperrors.ErrOptionSourcePermission, fmt.Errorf("member cannot preview option source"))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if form.DraftRevision != req.DraftRevision {
		return nil, httpx.Wrap(apperrors.ErrOptionSourceVersionConflict,
			fmt.Errorf("option draft revision %d != %d", req.DraftRevision, form.DraftRevision))
	}
	document, err := decodeOptionSourceDocument(form.DraftContent)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrOptionSourceInvalid, err)
	}
	config, ok := optionSourceByField(document, fieldID)
	if !ok || config.Source.Type != "form" || config.Source.AppID != form.AppID {
		return nil, httpx.Wrap(apperrors.ErrOptionSourceNotFound, fmt.Errorf("draft option source for %s not found", fieldID))
	}
	// 源字段目录本身已按当前成员的表单入口与字段可见权限裁剪；预览不能借助
	// 草稿中尚未发布终审的配置扩大读取范围。
	fields, err := s.ListLinkageFields(ctx, member, config.Source.SourceID)
	if err != nil {
		return nil, err
	}
	fieldAllowed := false
	for _, field := range fields {
		if field.ID == config.Source.FieldID && relatedOptionFieldTypes[field.Type] {
			fieldAllowed = true
			break
		}
	}
	if !fieldAllowed {
		return nil, httpx.Wrap(apperrors.ErrOptionSourcePermission,
			fmt.Errorf("draft option source field is not visible"))
	}
	return s.queryRelatedOptionConfig(ctx, member, config, &model.QueryRelatedOptionsRequest{
		Values: req.Values, Keyword: req.Keyword, PageSize: req.PageSize,
	})
}

// queryRelatedOptionConfig 是运行时查询与提交终审的共用执行器；
// config 只能来自服务端已发布快照，不接受浏览器组装。
func (s *formService) queryRelatedOptionConfig(ctx context.Context, member *iammodel.User, config *relatedOptionSourceConfig, req *model.QueryRelatedOptionsRequest) (*model.RelatedOptionPage, error) {
	children := make([]model.RecordQueryExpression, 0, len(config.Filter.Conditions))
	for _, condition := range config.Filter.Conditions {
		value, hasValue, emptyDependency := resolveLinkageConditionValue(condition, req.Values)
		if emptyDependency {
			return &model.RelatedOptionPage{Items: []model.RelatedOptionItem{}}, nil
		}
		expression := model.RecordQueryExpression{Type: "condition", Field: condition.SourceFieldID, Operator: condition.Operator}
		if hasValue {
			expression.Value = value
		}
		children = append(children, expression)
	}
	var filter *model.RecordQueryExpression
	if len(children) > 0 {
		filter = &model.RecordQueryExpression{Type: "group", Conjunction: config.Filter.Logic, Children: children}
	}

	rows := make([]model.FormRecordDTO, 0, 100)
	for pageNo := 1; pageNo <= 10; pageNo++ {
		page, queryErr := s.ListRecords(ctx, member, config.Source.SourceID, model.RecordQueryDocument{
			Version: 1, Filter: filter, Paging: model.RecordQueryPaging{Page: pageNo, PageSize: 100},
		})
		if queryErr != nil {
			return nil, queryErr
		}
		rows = append(rows, page.Items...)
		if len(rows) >= int(page.Total) || len(page.Items) == 0 {
			break
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left := optionSortValue(rows[i], config)
		right := optionSortValue(rows[j], config)
		less := compareOptionValue(left, right) < 0
		if config.Sort.Direction == "desc" {
			return !less && compareOptionValue(left, right) != 0
		}
		return less
	})

	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	seen := make(map[string]bool)
	items := make([]model.RelatedOptionItem, 0, len(rows))
	for _, row := range rows {
		for _, value := range flattenOptionValues(row.Values[config.Source.FieldID]) {
			if value == "" || seen[value] || (keyword != "" && !strings.Contains(strings.ToLower(value), keyword)) {
				continue
			}
			seen[value] = true
			items = append(items, model.RelatedOptionItem{Label: value, Value: value})
		}
	}
	total := len(items)
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}
	if len(items) > pageSize {
		items = items[:pageSize]
	}
	return &model.RelatedOptionPage{Items: items, Total: total}, nil
}

// validateSubmittedRelatedOptions 终审关联下拉值是否仍在当前成员可见的
// 记录范围内。这一步防止绕过浏览器直接提交伪造选项。
func (s *formService) validateSubmittedRelatedOptions(ctx context.Context, member *iammodel.User, content model.JSONContent, values map[string]any) error {
	document, err := decodeOptionSourceDocument(content)
	if err != nil {
		return httpx.Wrap(apperrors.ErrOptionSourceInvalid, err)
	}
	fieldErrors := RecordFieldErrors{}
	for _, item := range document.Content.Items {
		config := item.Widget.OptionSource
		if config == nil || config.Mode != "related" || config.Related == nil {
			continue
		}
		selected := flattenOptionValues(values[item.Widget.WidgetName])
		if len(selected) == 0 {
			continue
		}
		page, queryErr := s.queryRelatedOptionConfig(ctx, member, config.Related, &model.QueryRelatedOptionsRequest{
			Values: values, PageSize: 100,
		})
		if queryErr != nil {
			return queryErr
		}
		allowed := make(map[string]bool, len(page.Items))
		for _, option := range page.Items {
			allowed[option.Value] = true
		}
		for _, value := range selected {
			if !allowed[value] {
				fieldErrors[item.Widget.WidgetName] = []string{"当前值不在可选的关联数据范围内"}
				break
			}
		}
	}
	if len(fieldErrors) > 0 {
		return httpx.Wrap(apperrors.ErrRecordInvalid.WithData(map[string]any{"fieldErrors": fieldErrors}),
			fmt.Errorf("submitted related options are not selectable"))
	}
	return nil
}

func optionSortValue(row model.FormRecordDTO, config *relatedOptionSourceConfig) any {
	field := config.Sort.FieldID
	if field == "__value__" {
		field = config.Source.FieldID
	}
	switch field {
	case SysFieldSubmittedAt:
		return row.SubmittedAt.String()
	case SysFieldUpdatedAt:
		return row.UpdatedAt.String()
	default:
		return row.Values[field]
	}
}

func compareOptionValue(left, right any) int {
	leftText, rightText := fmt.Sprint(left), fmt.Sprint(right)
	leftNumber, leftErr := strconv.ParseFloat(leftText, 64)
	rightNumber, rightErr := strconv.ParseFloat(rightText, 64)
	if leftErr == nil && rightErr == nil {
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
		return 0
	}
	return strings.Compare(leftText, rightText)
}

func flattenOptionValues(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{strings.TrimSpace(typed)}
	case float64:
		return []string{strconv.FormatFloat(typed, 'f', -1, 64)}
	case []any:
		result := make([]string, 0, len(typed))
		for _, entry := range typed {
			result = append(result, flattenOptionValues(entry)...)
		}
		return result
	default:
		if typed == nil {
			return nil
		}
		return []string{strings.TrimSpace(fmt.Sprint(typed))}
	}
}

// validatePublishedOptionSources 在发布前终审源表、字段、操作符、排序与当前表单
// 依赖字段，避免只靠设计器候选列表形成越权或悬空配置。
func (s *formService) validatePublishedOptionSources(ctx context.Context, form *model.Form) error {
	document, err := decodeOptionSourceDocument(form.DraftContent)
	if err != nil {
		return err
	}
	currentContent := make(map[string]any)
	if err = json.Unmarshal(form.DraftContent, &currentContent); err != nil {
		return err
	}
	currentFields, err := buildPermissionFieldList(currentContent)
	if err != nil {
		return err
	}
	currentIndex := permissionFieldIndex(currentFields)
	for _, item := range document.Content.Items {
		config := item.Widget.OptionSource
		if config == nil || config.Mode != "related" || config.Related == nil {
			continue
		}
		related := config.Related
		if (item.Widget.Type != "combo" && item.Widget.Type != "combocheck") || related.Source.Type != "form" || related.Source.AppID != form.AppID {
			return httpx.Wrap(apperrors.ErrOptionSourceInvalid, fmt.Errorf("invalid related option target %s", item.Widget.WidgetName))
		}
		sourceForm, _, _, sourceFields, sourceErr := s.loadPublishedLinkageSource(ctx, related.Source.SourceID)
		if sourceErr != nil {
			return httpx.Wrap(apperrors.ErrOptionSourceNotFound, sourceErr)
		}
		if sourceForm.AppID != form.AppID {
			return httpx.Wrap(apperrors.ErrOptionSourceNotFound, fmt.Errorf("source app mismatch"))
		}
		sourceIndex := permissionFieldIndex(sourceFields)
		selected, exists := sourceIndex[related.Source.FieldID]
		if !exists || !relatedOptionFieldTypes[selected.WidgetType] {
			return httpx.Wrap(apperrors.ErrOptionSourceInvalid, fmt.Errorf("option field %s is unsupported", related.Source.FieldID))
		}
		if related.Sort.FieldID != "__value__" && related.Sort.FieldID != SysFieldSubmittedAt && related.Sort.FieldID != SysFieldUpdatedAt {
			if _, exists = sourceIndex[related.Sort.FieldID]; !exists {
				return httpx.Wrap(apperrors.ErrOptionSourceInvalid, fmt.Errorf("sort field not found"))
			}
		}
		for _, condition := range related.Filter.Conditions {
			source, exists := sourceIndex[condition.SourceFieldID]
			if !exists {
				return httpx.Wrap(apperrors.ErrOptionSourceInvalid, fmt.Errorf("filter source field not found"))
			}
			if !containsLinkageOperator(linkageOperatorsForType(source.WidgetType), condition.Operator) {
				return httpx.Wrap(apperrors.ErrOptionSourceInvalid, fmt.Errorf("filter operator unsupported"))
			}
			if condition.Value != nil && condition.Value.Type == "field" {
				current, currentOK := currentIndex[condition.Value.FieldID]
				if !currentOK || !linkageTypesCompatible(source.WidgetType, current.WidgetType) {
					return httpx.Wrap(apperrors.ErrOptionSourceInvalid, fmt.Errorf("filter dependency mismatch"))
				}
			}
		}
	}
	return nil
}
