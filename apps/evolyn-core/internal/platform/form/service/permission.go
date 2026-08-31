// 权限组配置面服务（表单权限 P1，设计 §4/§6/§7）：权限组 CRUD 与权限配置
// 字段清单。四要素（主体范围×操作集×字段矩阵×数据范围）整体提交、整体生效
// （S1），写入前经配置期校验器终审；组删除同事务软删组行 + 硬删 subjects
// （授权关系无保留价值，外键 CASCADE 仅兜底物理清理路径）。
//
// 鉴权口径：配置面资源 form-permissions 的 list/create/update/delete 在
// Service 层按权限集复核（路由挂 /forms 首段，中间件 URL 门解析为 forms:*，
// 两道门独立生效——持 forms:* 而无 form-permissions:* 的成员过不了本层）。
// 配置面权限不触发数据面旁路（S3：能配置权限组 ≠ 拥有全部数据权限）。
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

// 权限组业务上限
const (
	maxPermissionGroupNameRunes   = 64
	maxPermissionGroupDescRunes   = 200
	maxPermissionGroupsPerForm    = 50  // 单表单权限组数量上限
	maxPermissionSubjectsPerGroup = 200 // 单权限组主体数量上限
)

// PermissionSubjectDirectory 权限组主体校验/展示名解析窄端口（装配层由 iam
// 仓储适配；写入时校验同租户存在性，读取时解析展示名，域内不直连 iam 表）。
type PermissionSubjectDirectory interface {
	// SubjectExists 校验主体在当前租户存在（跨租户/不存在/已删除 → false）
	SubjectExists(ctx context.Context, subjectType string, subjectID uint) (bool, error)
	// SubjectNames 批量解析主体展示名（键为主体类型；解析不到为空串）
	SubjectNames(ctx context.Context, subjects []model.AssetPermissionGroupSubject) (map[string]map[uint]string, error)
}

// TxManager 事务边界抽象（与 formService 同形，实现在 infrastructure）
type permissionTxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// PermissionGroupService 权限组配置面服务接口。
type PermissionGroupService interface {
	// ListGroups 表单下权限组清单（含禁用组，S5 收口事实源对管理员完整可见）
	ListGroups(ctx context.Context, member *iammodel.User, formCode string) ([]model.PermissionGroupView, error)
	// CreateGroup 创建权限组（revision=1；数量上限 50/表）
	CreateGroup(ctx context.Context, member *iammodel.User, formCode string, req *model.CreatePermissionGroupRequest) (*model.PermissionGroupView, error)
	// UpdateGroup 整组全量更新（baseRevision 整组乐观锁；冲突 409）
	UpdateGroup(ctx context.Context, member *iammodel.User, formCode, groupCode string, req *model.UpdatePermissionGroupRequest) (*model.PermissionGroupView, error)
	// DeleteGroup 删除权限组（软删组行 + 同事务硬删 subjects）
	DeleteGroup(ctx context.Context, member *iammodel.User, formCode, groupCode string) error
	// ListPermissionFields 权限配置字段清单（最新发布版本 schema，未发布回落草稿）
	ListPermissionFields(ctx context.Context, member *iammodel.User, formCode string) ([]model.PermissionFieldView, error)
}

type permissionGroupService struct {
	tx        permissionTxManager
	groups    repository.PermissionGroupRepository
	forms     repository.FormRepository
	versions  repository.FormVersionRepository
	audit     auditservice.Recorder
	access    AccessEvaluator
	directory PermissionSubjectDirectory
}

// NewPermissionGroupService 构造权限组配置面服务（directory 为主体窄端口，
// audit 可为 nil：跳过审计，便于单测桩）。
func NewPermissionGroupService(
	tx permissionTxManager,
	groups repository.PermissionGroupRepository,
	forms repository.FormRepository,
	versions repository.FormVersionRepository,
	audit auditservice.Recorder,
	access AccessEvaluator,
	directory PermissionSubjectDirectory,
) PermissionGroupService {
	return &permissionGroupService{tx: tx, groups: groups, forms: forms, versions: versions, audit: audit, access: access, directory: directory}
}

// ---- 权限组清单 ----

func (s *permissionGroupService) ListGroups(ctx context.Context, member *iammodel.User, formCode string) ([]model.PermissionGroupView, error) {
	if err := s.checkPermission(ctx, member, "form-permissions:list"); err != nil {
		return nil, err
	}
	form, err := s.loadForm(ctx, formCode)
	if err != nil {
		return nil, err
	}
	groups, err := s.groups.ListByAsset(ctx, model.PermissionAssetTypeForm, form.ID)
	if err != nil {
		return nil, err
	}
	return s.buildViews(ctx, groups), nil
}

// ---- 创建 ----

func (s *permissionGroupService) CreateGroup(
	ctx context.Context, member *iammodel.User, formCode string, req *model.CreatePermissionGroupRequest,
) (*model.PermissionGroupView, error) {
	if err := s.checkPermission(ctx, member, "form-permissions:create"); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, httpx.Wrap(apperrors.ErrPermissionNameInvalid, fmt.Errorf("create permission group request is nil"))
	}
	form, err := s.loadForm(ctx, formCode)
	if err != nil {
		return nil, err
	}
	normalized, err := s.normalizeRequest(ctx, form, req)
	if err != nil {
		return nil, err
	}
	count, err := s.groups.CountByAsset(ctx, model.PermissionAssetTypeForm, form.ID)
	if err != nil {
		return nil, err
	}
	if count >= maxPermissionGroupsPerForm {
		return nil, httpx.Wrap(apperrors.ErrPermissionLimitExceeded,
			fmt.Errorf("form %s already has %d permission groups", formCode, count))
	}
	code, err := newPermissionGroupCode()
	if err != nil {
		return nil, err
	}
	group := &model.AssetPermissionGroup{
		ApplicationID:    form.ApplicationID,
		AssetType:        model.PermissionAssetTypeForm,
		AssetID:          form.ID,
		Code:             code,
		Name:             normalized.name,
		Description:      normalized.description,
		Enabled:          normalized.enabled,
		Operations:       model.PermissionOperations(normalized.operations),
		FieldPermissions: model.PermissionFieldRules(normalized.fieldRules),
		DataScope:        model.PermissionDataScopeValue(normalized.dataScope),
		Revision:         1,
	}
	group.TenantID = form.TenantID

	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if _, err := s.groups.Create(tctx, group); err != nil {
			return err
		}
		return s.groups.ReplaceSubjects(tctx, group.ID, normalized.subjects)
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "create", ResourceType: "form_permission_group",
			ResourceID: group.Code,
			After: map[string]any{
				"formCode": form.Code, "name": group.Name, "enabled": group.Enabled,
				"operations": group.Operations.String(),
			},
		})
	}
	return s.buildView(ctx, group, normalized.subjects), nil
}

// ---- 更新 ----

func (s *permissionGroupService) UpdateGroup(
	ctx context.Context, member *iammodel.User, formCode, groupCode string, req *model.UpdatePermissionGroupRequest,
) (*model.PermissionGroupView, error) {
	if err := s.checkPermission(ctx, member, "form-permissions:update"); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, httpx.Wrap(apperrors.ErrPermissionNameInvalid, fmt.Errorf("update permission group request is nil"))
	}
	form, err := s.loadForm(ctx, formCode)
	if err != nil {
		return nil, err
	}
	group, err := s.loadGroup(ctx, form, groupCode)
	if err != nil {
		return nil, err
	}
	normalized, err := s.normalizeRequest(ctx, form, &req.CreatePermissionGroupRequest)
	if err != nil {
		return nil, err
	}

	updated := false
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		ok, uerr := s.groups.UpdateWithRevision(tctx, group.ID, req.BaseRevision, map[string]interface{}{
			"name":              normalized.name,
			"description":       normalized.description,
			"enabled":           normalized.enabled,
			"operations":        model.PermissionOperations(normalized.operations),
			"field_permissions": model.PermissionFieldRules(normalized.fieldRules),
			"data_scope":        model.PermissionDataScopeValue(normalized.dataScope),
		})
		if uerr != nil {
			return uerr
		}
		if !ok {
			return httpx.Wrap(apperrors.ErrPermissionRevisionConflict,
				fmt.Errorf("permission group %s revision %d stale", groupCode, req.BaseRevision))
		}
		updated = true
		return s.groups.ReplaceSubjects(tctx, group.ID, normalized.subjects)
	}); err != nil {
		return nil, err
	}

	if updated && s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "update", ResourceType: "form_permission_group",
			ResourceID: group.Code,
			Before:     map[string]any{"name": group.Name, "enabled": group.Enabled, "revision": group.Revision},
			After: map[string]any{
				"formCode": form.Code, "name": normalized.name, "enabled": normalized.enabled,
				"operations": strings.Join(normalized.operations, ","), "revision": req.BaseRevision + 1,
			},
		})
	}
	group.Name = normalized.name
	group.Description = normalized.description
	group.Enabled = normalized.enabled
	group.Operations = model.PermissionOperations(normalized.operations)
	group.FieldPermissions = model.PermissionFieldRules(normalized.fieldRules)
	group.DataScope = model.PermissionDataScopeValue(normalized.dataScope)
	group.Revision = req.BaseRevision + 1
	return s.buildView(ctx, group, normalized.subjects), nil
}

// ---- 删除 ----

func (s *permissionGroupService) DeleteGroup(ctx context.Context, member *iammodel.User, formCode, groupCode string) error {
	if err := s.checkPermission(ctx, member, "form-permissions:delete"); err != nil {
		return err
	}
	form, err := s.loadForm(ctx, formCode)
	if err != nil {
		return err
	}
	group, err := s.loadGroup(ctx, form, groupCode)
	if err != nil {
		return err
	}
	// 软删组行 + 同事务硬删 subjects（外键 CASCADE 只对物理 DELETE 生效，
	// 软删路径必须显式清理，保证不产生孤儿授权行）
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.groups.SoftDelete(tctx, group); err != nil {
			return err
		}
		return s.groups.DeleteSubjectsByGroupIDs(tctx, []uint{group.ID})
	}); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "delete", ResourceType: "form_permission_group",
			ResourceID: group.Code,
			After:      map[string]any{"formCode": form.Code, "name": group.Name},
		})
	}
	return nil
}

// ---- 权限配置字段清单 ----

func (s *permissionGroupService) ListPermissionFields(
	ctx context.Context, member *iammodel.User, formCode string,
) ([]model.PermissionFieldView, error) {
	if err := s.checkPermission(ctx, member, "form-permissions:list"); err != nil {
		return nil, err
	}
	form, err := s.loadForm(ctx, formCode)
	if err != nil {
		return nil, err
	}
	fieldList, err := s.currentFieldList(ctx, form)
	if err != nil {
		return nil, err
	}
	views := make([]model.PermissionFieldView, 0, len(fieldList))
	for _, field := range fieldList {
		views = append(views, model.PermissionFieldView{
			Field:    field.Key,
			Label:    field.Label,
			Type:     field.WidgetType,
			Required: field.Required,
		})
	}
	return views, nil
}

// ---- 内部助手 ----

// checkPermission 配置面权限复核（form-permissions:* 稳定键）
func (s *permissionGroupService) checkPermission(ctx context.Context, member *iammodel.User, key string) error {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if s.access == nil || !s.access.Permissions(ctx, member)[key] {
		return httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member %d lacks %s", member.ID, key))
	}
	return nil
}

// loadForm 按稳定公开编码加载表单（跨租户表现为 NotFound）
func (s *permissionGroupService) loadForm(ctx context.Context, formCode string) (*model.Form, error) {
	form, err := s.forms.GetByCode(ctx, formCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Wrap(apperrors.ErrFormNotFound, err)
		}
		return nil, err
	}
	return form, nil
}

// loadGroup 按公开编码加载权限组并复核资产归属（跨表单/跨应用的组编码
// 统一 NotFound，不泄露组归属）
func (s *permissionGroupService) loadGroup(ctx context.Context, form *model.Form, groupCode string) (*model.AssetPermissionGroup, error) {
	group, err := s.groups.GetByCode(ctx, groupCode)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Wrap(apperrors.ErrPermissionGroupNotFound, err)
		}
		return nil, err
	}
	if group.AssetID != form.ID || group.ApplicationID != form.ApplicationID ||
		group.AssetType != model.PermissionAssetTypeForm {
		return nil, httpx.Wrap(apperrors.ErrPermissionGroupNotFound,
			fmt.Errorf("permission group %s does not belong to form %s", groupCode, form.Code))
	}
	return group, nil
}

// normalizedPermissionGroup 校验后的组四要素（写入形态）
type normalizedPermissionGroup struct {
	name        string
	description string
	enabled     bool
	operations  []string
	fieldRules  []model.PermissionFieldRule
	dataScope   model.PermissionDataScopeSpec
	subjects    []model.AssetPermissionGroupSubject
}

// normalizeRequest 四要素整体校验（S1）：名称 → 操作集 → 字段矩阵 → 数据
// 范围 → 主体。字段矩阵与数据范围以当前字段清单为事实源（最新发布版本，
// 未发布回落草稿）。
func (s *permissionGroupService) normalizeRequest(
	ctx context.Context, form *model.Form, req *model.CreatePermissionGroupRequest,
) (*normalizedPermissionGroup, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > maxPermissionGroupNameRunes {
		return nil, httpx.Wrap(apperrors.ErrPermissionNameInvalid, fmt.Errorf("invalid permission group name length"))
	}
	description := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(description) > maxPermissionGroupDescRunes {
		return nil, httpx.Wrap(apperrors.ErrPermissionNameInvalid, fmt.Errorf("permission group description too long"))
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	operations, err := ValidatePermissionOperations(form.FormType, req.Operations)
	if err != nil {
		return nil, err
	}

	fieldList, err := s.currentFieldList(ctx, form)
	if err != nil {
		return nil, err
	}
	fieldRules := make([]model.PermissionFieldRule, 0, len(req.FieldPermissions))
	fieldRules = append(fieldRules, req.FieldPermissions...)
	if err := ValidatePermissionFieldRules(fieldList, fieldRules, operations); err != nil {
		return nil, err
	}

	dataScope := model.PermissionDataScopeSpec{}
	if req.DataScope != nil {
		dataScope = *req.DataScope
	}
	dataScope.Normalize()
	if err := ValidatePermissionDataScope(fieldList, &dataScope); err != nil {
		return nil, err
	}

	subjects, err := s.normalizeSubjects(ctx, form.TenantID, req.SubjectIds)
	if err != nil {
		return nil, err
	}

	return &normalizedPermissionGroup{
		name:        name,
		description: description,
		enabled:     enabled,
		operations:  operations,
		fieldRules:  fieldRules,
		dataScope:   dataScope,
		subjects:    subjects,
	}, nil
}

// normalizeSubjects 主体清单校验：类型枚举、去重、上限 200、同租户存在性
// （禁止裸 ID 盲写授权关系）
func (s *permissionGroupService) normalizeSubjects(
	ctx context.Context, tenantID uint, inputs []model.PermissionSubjectInput,
) ([]model.AssetPermissionGroupSubject, error) {
	if len(inputs) > maxPermissionSubjectsPerGroup {
		return nil, httpx.Wrap(apperrors.ErrPermissionSubjectInvalid,
			fmt.Errorf("subject count %d exceeds limit %d", len(inputs), maxPermissionSubjectsPerGroup))
	}
	seen := make(map[model.AssetPermissionGroupSubject]bool, len(inputs))
	subjects := make([]model.AssetPermissionGroupSubject, 0, len(inputs))
	for _, input := range inputs {
		if !model.ValidPermissionSubjectType(input.Type) {
			return nil, httpx.Wrap(apperrors.ErrPermissionSubjectInvalid,
				fmt.Errorf("unknown subject type %q", input.Type))
		}
		if input.ID == 0 {
			return nil, httpx.Wrap(apperrors.ErrPermissionSubjectInvalid,
				fmt.Errorf("subject id is required"))
		}
		subject := model.AssetPermissionGroupSubject{
			GroupID:     0, // 由调用方在落库前回填
			SubjectType: input.Type,
			SubjectID:   input.ID,
		}
		subject.TenantID = tenantID
		if seen[subject] {
			continue
		}
		if s.directory != nil {
			exists, err := s.directory.SubjectExists(ctx, input.Type, input.ID)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, httpx.Wrap(apperrors.ErrPermissionSubjectInvalid,
					fmt.Errorf("subject %s/%d not found in tenant %d", input.Type, input.ID, tenantID))
			}
		}
		seen[subject] = true
		subjects = append(subjects, subject)
	}
	return subjects, nil
}

// currentFieldList 当前字段清单：最新发布版本 schema，未发布回落草稿
// （同一提取器，与判定器 S7 合并、数据范围类型分派共用一份事实源）
func (s *permissionGroupService) currentFieldList(ctx context.Context, form *model.Form) ([]permissionFieldMeta, error) {
	content := form.DraftContent
	if form.LatestVersionID != nil {
		version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				content = form.DraftContent
			} else {
				return nil, err
			}
		} else {
			content = version.Content
		}
	}
	return snapshotPermissionFieldList(content)
}

// buildViews 批量出网视图（组 → 主体 → 展示名解析）
func (s *permissionGroupService) buildViews(ctx context.Context, groups []model.AssetPermissionGroup) []model.PermissionGroupView {
	groupIDs := make([]uint, 0, len(groups))
	for i := range groups {
		groupIDs = append(groupIDs, groups[i].ID)
	}
	subjects, err := s.groups.ListSubjectsByGroupIDs(ctx, groupIDs)
	if err != nil {
		// 读取侧主体解析失败不吞组清单：降级为主体为空的视图
		subjects = []model.AssetPermissionGroupSubject{}
	}
	subjectsByGroup := make(map[uint][]model.AssetPermissionGroupSubject, len(groups))
	for _, subject := range subjects {
		subjectsByGroup[subject.GroupID] = append(subjectsByGroup[subject.GroupID], subject)
	}
	names := s.resolveSubjectNames(ctx, subjects)

	views := make([]model.PermissionGroupView, 0, len(groups))
	for i := range groups {
		groupSubjects := subjectsByGroup[groups[i].ID]
		views = append(views, permissionGroupViewOf(&groups[i], groupSubjects, names))
	}
	return views
}

// buildView 单组出网视图
func (s *permissionGroupService) buildView(
	ctx context.Context, group *model.AssetPermissionGroup, subjects []model.AssetPermissionGroupSubject,
) *model.PermissionGroupView {
	names := s.resolveSubjectNames(ctx, subjects)
	view := permissionGroupViewOf(group, subjects, names)
	return &view
}

func (s *permissionGroupService) resolveSubjectNames(
	ctx context.Context, subjects []model.AssetPermissionGroupSubject,
) map[string]map[uint]string {
	if s.directory == nil || len(subjects) == 0 {
		return map[string]map[uint]string{}
	}
	names, err := s.directory.SubjectNames(ctx, subjects)
	if err != nil || names == nil {
		return map[string]map[uint]string{}
	}
	return names
}

// permissionGroupViewOf 组行 + 主体 → 出网视图（dataScope 归一缺省后出网）
func permissionGroupViewOf(
	group *model.AssetPermissionGroup,
	subjects []model.AssetPermissionGroupSubject,
	names map[string]map[uint]string,
) model.PermissionGroupView {
	operations := make([]string, 0, len(group.Operations))
	operations = append(operations, group.Operations...)
	if operations == nil {
		operations = []string{}
	}
	fieldRules := make([]model.PermissionFieldRule, 0, len(group.FieldPermissions))
	fieldRules = append(fieldRules, group.FieldPermissions...)
	dataScope := model.PermissionDataScopeSpec(group.DataScope)
	dataScope.Normalize()

	subjectViews := make([]model.PermissionSubjectView, 0, len(subjects))
	for _, subject := range subjects {
		name := ""
		if byID, ok := names[subject.SubjectType]; ok {
			name = byID[subject.SubjectID]
		}
		subjectViews = append(subjectViews, model.PermissionSubjectView{
			Type: subject.SubjectType,
			ID:   subject.SubjectID,
			Name: name,
		})
	}
	return model.PermissionGroupView{
		Code:             group.Code,
		Name:             group.Name,
		Description:      group.Description,
		Enabled:          group.Enabled,
		Operations:       operations,
		FieldPermissions: fieldRules,
		DataScope:        dataScope,
		Revision:         group.Revision,
		Subjects:         subjectViews,
	}
}

// newPermissionGroupCode 生成权限组公开编码：fpg_ + 16 位随机 hex
// （对齐 form_/menu_ 惯例，内部自增 ID 不出网；唯一约束兜底）
func newPermissionGroupCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate permission group code: %w", err)
	}
	return "fpg_" + hex.EncodeToString(buf), nil
}
