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
