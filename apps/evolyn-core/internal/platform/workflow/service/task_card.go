package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	enginemodel "evolyn/internal/engine/workflow/model"
	"evolyn/internal/engine/workflow/provider"
	"evolyn/internal/platform/workflow/model"
)

// taskCardContext 将同页实例与发布版本批量读取，成员与表单内容按页缓存。
// 不跨请求缓存业务值，审批完成后刷新始终读取最新记录。
type taskCardContext struct {
	instances map[uint]model.WfInstance
	versions  map[uint]model.WfDefinitionVersion
	names     map[uint]string
	starters  map[uint]string
	contents  map[uint]json.RawMessage
}

func (s *runtimeService) loadTaskCardContext(ctx context.Context, rows []model.WfTask) (*taskCardContext, error) {
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.InstanceID)
	}
	instances, err := s.reader.ListInstanceRowsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	definitionIDs, versionIDs := []uint{}, []uint{}
	result := &taskCardContext{instances: map[uint]model.WfInstance{}, starters: map[uint]string{}, contents: map[uint]json.RawMessage{}}
	for _, instance := range instances {
		result.instances[instance.ID] = instance
		definitionIDs = append(definitionIDs, instance.DefinitionID)
		versionIDs = append(versionIDs, instance.DefinitionVersionID)
	}
	result.names, result.versions, err = s.definitions.LoadCardDefinitions(ctx, definitionIDs, versionIDs)
	return result, err
}

// populateTaskCard 在成员任务查询授权后组装卡片；业务数据只经表单窄端口读取。
func (s *runtimeService) populateTaskCard(ctx context.Context, item *model.TaskSummary, cards *taskCardContext) error {
	instance, exists := cards.instances[item.InstanceID]
	if !exists {
		return fmt.Errorf("workflow instance %d missing", item.InstanceID)
	}
	code := fmt.Sprintf("流程 #%d", instance.DefinitionID)
	item.InstanceNo = instance.InstanceNo
	item.Title = code
	item.NodeName = item.NodeKey
	item.SummaryFields = []model.TaskSummaryField{}
	if name := cards.names[instance.DefinitionID]; name != "" {
		item.Title = name
	}
	if s.identity != nil {
		name, found := cards.starters[instance.StarterMemberID]
		if !found {
			name = s.identity.MemberDisplayName(ctx, instance.TenantID, instance.StarterMemberID)
			cards.starters[instance.StarterMemberID] = name
		}
		item.StarterName = name
	}
	row, found := cards.versions[instance.DefinitionVersionID]
	if !found {
		return fmt.Errorf("workflow version %d missing", instance.DefinitionVersionID)
	}
	var snapshot enginemodel.Document
	if err := json.Unmarshal(row.DSLSnapshot, &snapshot); err != nil {
		return err
	}
	node, ok := snapshot.NodeOf(item.NodeKey)
	if ok && node.Name != "" {
		item.NodeName = node.Name
	}
	if !ok || instance.FormVersionID == 0 || s.formDir == nil || s.formData == nil {
		return nil
	}
	content, found := cards.contents[instance.FormVersionID]
	if !found {
		var err error
		content, _, _, err = s.formDir.GetVersionContent(ctx, instance.FormVersionID)
		if err != nil {
			return err
		}
		cards.contents[instance.FormVersionID] = content
	}
	values, err := s.formData.GetData(ctx, provider.BusinessRef{
		TenantID: instance.TenantID, AppID: instance.AppID, FormID: instance.FormID,
		FormVersionID: instance.FormVersionID, BusinessID: instance.BusinessID,
	})
	if err != nil {
		return err
	}
	item.SummaryFields = taskCardFields(content, values, node.Config.FormPermissions)
	return nil
}

// taskCardFields 按冻结表单顺序提取明确授权字段；复杂对象、隐藏/未授权字段不出网。
func taskCardFields(content json.RawMessage, values map[string]any, permissions map[string]enginemodel.FieldPermission) []model.TaskSummaryField {
	var document struct {
		Content struct {
			Items []struct {
				Label  string `json:"label"`
				Widget struct {
					WidgetName string `json:"widgetName"`
				} `json:"widget"`
			} `json:"items"`
		} `json:"content"`
	}
	fields := []model.TaskSummaryField{}
	if json.Unmarshal(content, &document) != nil {
		return fields
	}
	for _, field := range document.Content.Items {
		key := field.Widget.WidgetName
		permission := permissions[key]
		if permission != enginemodel.FieldPermissionReadonly && permission != enginemodel.FieldPermissionEditable {
			continue
		}
		var value string
		switch v := values[key].(type) {
		case string:
			value = v
		case float64, float32, int, int64, json.Number:
			value = fmt.Sprint(v)
		case bool:
			if v {
				value = "是"
			} else {
				value = "否"
			}
		default:
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		// 卡片仅保留短摘要，长文本在审批详情内阅读。
		runes := []rune(value)
		if len(runes) > 120 {
			value = string(runes[:120]) + "…"
		}
		fields = append(fields, model.TaskSummaryField{Label: field.Label, Value: value})
		if len(fields) == 3 {
			break
		}
	}
	return fields
}
