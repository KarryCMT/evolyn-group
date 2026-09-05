package repository

import (
	"context"
	"encoding/json"
	"errors"

	enginemodel "evolyn/internal/engine/workflow/model"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/workflow/model"

	"gorm.io/gorm"
)

// ---- 引擎内核只读端口与详情查询辅助 ----

// EngineDefinitionReader 引擎内核 runtime.DefinitionReader 的 GORM 适配：
// Runtime 发起流程时按 code/版本号/版本 ID 读取发布快照。
type EngineDefinitionReader struct{ base *gorm.DB }

// NewEngineDefinitionReader 构造引擎定义只读端口。
func NewEngineDefinitionReader(base *gorm.DB) *EngineDefinitionReader {
	return &EngineDefinitionReader{base: base}
}

func (r *EngineDefinitionReader) FindDefinitionByCode(ctx context.Context, tenantID uint, code string) (*enginemodel.Definition, error) {
	var row model.WfDefinition
	if err := infrastructure.ResolveDB(ctx, r.base).Where("code = ?", code).First(&row).Error; err != nil {
		return nil, err
	}
	return definitionFromRow(ctx, r.base, &row)
}

func (r *EngineDefinitionReader) FindVersion(ctx context.Context, tenantID, definitionID uint, versionNo int) (*enginemodel.DefinitionVersion, error) {
	var row model.WfDefinitionVersion
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("definition_id = ? AND version_no = ?", definitionID, versionNo).First(&row).Error; err != nil {
		return nil, err
	}
	return versionFromRow(&row)
}

func (r *EngineDefinitionReader) FindVersionByID(ctx context.Context, tenantID, versionID uint) (*enginemodel.DefinitionVersion, error) {
	var row model.WfDefinitionVersion
	if err := infrastructure.ResolveDB(ctx, r.base).Where("id = ?", versionID).First(&row).Error; err != nil {
		return nil, err
	}
	return versionFromRow(&row)
}

// FindVersionNoByID 详情投影用：按版本行 ID 取发布号。
func (r *EngineDefinitionReader) FindVersionNoByID(ctx context.Context, versionID uint) (int, error) {
	var row model.WfDefinitionVersion
	if err := infrastructure.ResolveDB(ctx, r.base).Select("version_no").Where("id = ?", versionID).First(&row).Error; err != nil {
		return 0, err
	}
	return row.VersionNo, nil
}

// FindCodeByID 详情投影用：按定义行 ID 取公开编码。
func (r *EngineDefinitionReader) FindCodeByID(ctx context.Context, definitionID uint) (string, error) {
	var row model.WfDefinition
	if err := infrastructure.ResolveDB(ctx, r.base).Select("code").Where("id = ?", definitionID).First(&row).Error; err != nil {
		return "", err
	}
	return row.Code, nil
}

// FindDefinitionCodeByID 引擎内核 DefinitionReader 端口实现：实例级动作
// （重提交）构造表达式上下文时按定义行 ID 反查公开编码。
func (r *EngineDefinitionReader) FindDefinitionCodeByID(ctx context.Context, tenantID, definitionID uint) (string, error) {
	return r.FindCodeByID(ctx, definitionID)
}

func definitionFromRow(ctx context.Context, base *gorm.DB, row *model.WfDefinition) (*enginemodel.Definition, error) {
	draft, err := decodeDocument([]byte(row.DraftContent))
	if err != nil {
		return nil, err
	}
	def := &enginemodel.Definition{
		ID:               row.ID,
		TenantID:         row.TenantID,
		Code:             row.Code,
		Name:             row.Name,
		Description:      row.Description,
		Draft:            *draft,
		DraftRevision:    row.DraftRevision,
		LatestVersionID:  row.LatestVersionID,
		PublishedVersion: row.PublishedVersion,
	}
	switch {
	case row.DeletedAt.Valid:
		def.Status = enginemodel.DefinitionStatusDeleted
	case row.PublishedVersion > 0:
		def.Status = enginemodel.DefinitionStatusPublished
	default:
		def.Status = enginemodel.DefinitionStatusDraft
	}
	_ = ctx
	return def, nil
}

func versionFromRow(row *model.WfDefinitionVersion) (*enginemodel.DefinitionVersion, error) {
	snapshot, err := decodeDocument([]byte(row.DSLSnapshot))
	if err != nil {
		return nil, err
	}
	return &enginemodel.DefinitionVersion{
		ID:           row.ID,
		DefinitionID: row.DefinitionID,
		VersionNo:    row.VersionNo,
		Snapshot:     *snapshot,
	}, nil
}

func decodeDocument(raw []byte) (*enginemodel.Document, error) {
	doc := new(enginemodel.Document)
	if err := json.Unmarshal(raw, doc); err != nil {
		return nil, errors.New("DSL 快照不是合法 JSON 文档")
	}
	return doc, nil
}

// RuntimeReader 运行态详情查询（实例/节点/任务/参与人/操作流水行读取，
// 供服务层组装出网 DTO；行锁路径走引擎 SPI，不在此处）。
type RuntimeReader interface {
	FindInstanceRow(ctx context.Context, instanceID uint) (*model.WfInstance, error)
	ListNodeRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfNodeInstance, error)
	ListTaskRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfTask, error)
	ListActorRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfTaskActor, error)
	ListOperationRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfOperation, error)
	// FindTaskRow 按任务行 ID 读取（ctx 租户过滤兜底；任务详情上下文）
	FindTaskRow(ctx context.Context, taskID uint) (*model.WfTask, error)
	// ListActorRowsByTaskIDs 批量读取任务参与人快照（列表页组装）
	ListActorRowsByTaskIDs(ctx context.Context, taskIDs []uint) ([]model.WfTaskActor, error)
	// ListTaskRowsByMemberAndStatuses 我的待办/我的已办（第 20.4/28 章主查询：
	// 经 wf_task_actor 参与人快照定位本人任务，游标按 task id 倒序）
	ListTaskRowsByMemberAndStatuses(ctx context.Context, memberID uint, statuses []string, formCode string, limit int, afterID uint) ([]model.WfTask, bool, error)
	// ListCCRowsByMember 抄送我的（000051 追加写记录，游标按 id 倒序）
	ListCCRowsByMember(ctx context.Context, memberID uint, formCode string, limit int, afterID uint) ([]model.WfCCRecord, bool, error)
	// CountPendingTasksByMember 按表单分组计算当前成员待办，供左侧菜单徽标使用。
	CountPendingTasksByMember(ctx context.Context, memberID uint, statuses []string) ([]model.PendingTaskFormCount, error)
	// ListInstanceRowsByStarter 我发起的（游标按 id 倒序）
	ListInstanceRowsByStarter(ctx context.Context, memberID uint, limit int, afterID uint) ([]model.WfInstance, bool, error)
}

type runtimeReader struct{ base *gorm.DB }

// NewRuntimeReader 构造运行态详情查询。
func NewRuntimeReader(base *gorm.DB) RuntimeReader { return &runtimeReader{base: base} }

func (r *runtimeReader) FindInstanceRow(ctx context.Context, instanceID uint) (*model.WfInstance, error) {
	var row model.WfInstance
	if err := infrastructure.ResolveDB(ctx, r.base).Where("id = ?", instanceID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *runtimeReader) ListNodeRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfNodeInstance, error) {
	rows := make([]model.WfNodeInstance, 0)
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error
	return rows, err
}

func (r *runtimeReader) ListTaskRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfTask, error) {
	rows := make([]model.WfTask, 0)
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error
	return rows, err
}

func (r *runtimeReader) ListActorRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfTaskActor, error) {
	rows := make([]model.WfTaskActor, 0)
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("task_id IN (?)", infrastructure.ResolveDB(ctx, r.base).
			Model(&model.WfTask{}).Select("id").Where("instance_id = ?", instanceID)).
		Order("id").Find(&rows).Error
	return rows, err
}

func (r *runtimeReader) ListOperationRowsByInstance(ctx context.Context, instanceID uint) ([]model.WfOperation, error) {
	rows := make([]model.WfOperation, 0)
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error
	return rows, err
}

func (r *runtimeReader) FindTaskRow(ctx context.Context, taskID uint) (*model.WfTask, error) {
	var row model.WfTask
	if err := infrastructure.ResolveDB(ctx, r.base).Where("id = ?", taskID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *runtimeReader) ListActorRowsByTaskIDs(ctx context.Context, taskIDs []uint) ([]model.WfTaskActor, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}
	rows := make([]model.WfTaskActor, 0)
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("task_id IN ?", taskIDs).Order("id").Find(&rows).Error
	return rows, err
}

// pageTaskRowsByMember 待办/已办共用游标分页：参与人快照定位本人任务
// （快照在任务创建时一次性写入，运行中不随组织变化重算），limit+1 探测下一页。
func (r *runtimeReader) pageTaskRowsByMember(ctx context.Context, memberID uint, statuses []string, formCode string, limit int, afterID uint) ([]model.WfTask, bool, error) {
	query := infrastructure.ResolveDB(ctx, r.base).
		Model(&model.WfTask{}).
		Joins("JOIN wf_task_actor ON wf_task_actor.task_id = wf_task.id AND wf_task_actor.member_id = ?", memberID).
		Where("wf_task.status IN ?", statuses)
	if formCode != "" {
		query = query.Joins("JOIN wf_instance ON wf_instance.id = wf_task.instance_id").
			Joins("JOIN wf_definition ON wf_definition.id = wf_instance.definition_id").
			Where("wf_definition.form_code = ?", formCode)
	}
	if afterID > 0 {
		query = query.Where("wf_task.id < ?", afterID)
	}
	rows := make([]model.WfTask, 0)
	if err := query.
		Order("wf_task.id DESC").
		Limit(limit + 1).
		Select("wf_task.*").
		Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return rows, hasMore, nil
}

func (r *runtimeReader) ListTaskRowsByMemberAndStatuses(ctx context.Context, memberID uint, statuses []string, formCode string, limit int, afterID uint) ([]model.WfTask, bool, error) {
	return r.pageTaskRowsByMember(ctx, memberID, statuses, formCode, limit, afterID)
}

func (r *runtimeReader) ListCCRowsByMember(ctx context.Context, memberID uint, formCode string, limit int, afterID uint) ([]model.WfCCRecord, bool, error) {
	query := infrastructure.ResolveDB(ctx, r.base).
		Model(&model.WfCCRecord{}).
		Where("member_id = ?", memberID)
	if formCode != "" {
		query = query.Joins("JOIN wf_instance ON wf_instance.id = wf_cc_record.instance_id").
			Joins("JOIN wf_definition ON wf_definition.id = wf_instance.definition_id").
			Where("wf_definition.form_code = ?", formCode)
	}
	if afterID > 0 {
		query = query.Where("id < ?", afterID)
	}
	rows := make([]model.WfCCRecord, 0)
	if err := query.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return rows, hasMore, nil
}

func (r *runtimeReader) CountPendingTasksByMember(ctx context.Context, memberID uint, statuses []string) ([]model.PendingTaskFormCount, error) {
	rows := make([]model.PendingTaskFormCount, 0)
	err := infrastructure.ResolveDB(ctx, r.base).
		Model(&model.WfTask{}).
		Select("COALESCE(wf_definition.form_code, '') AS form_code, COUNT(DISTINCT wf_task.id) AS count").
		Joins("JOIN wf_task_actor ON wf_task_actor.task_id = wf_task.id AND wf_task_actor.member_id = ?", memberID).
		Joins("JOIN wf_instance ON wf_instance.id = wf_task.instance_id").
		Joins("LEFT JOIN wf_definition ON wf_definition.id = wf_instance.definition_id").
		Where("wf_task.status IN ?", statuses).
		Group("wf_definition.form_code").
		Find(&rows).Error
	return rows, err
}

func (r *runtimeReader) ListInstanceRowsByStarter(ctx context.Context, memberID uint, limit int, afterID uint) ([]model.WfInstance, bool, error) {
	query := infrastructure.ResolveDB(ctx, r.base).
		Model(&model.WfInstance{}).
		Where("starter_member_id = ?", memberID)
	if afterID > 0 {
		query = query.Where("id < ?", afterID)
	}
	rows := make([]model.WfInstance, 0)
	if err := query.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return rows, hasMore, nil
}
