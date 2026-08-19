package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	"evolyn/internal/utils/request"

	"gorm.io/gorm"
)

type rbacService struct {
	rbacRepository repository.RBACRepository
	audit          auditservice.Recorder
}

func NewRBACService(rbacRepository repository.RBACRepository, audit auditservice.Recorder) RBACService {
	return &rbacService{
		rbacRepository: rbacRepository,
		audit:          audit,
	}
}

func (rbac *rbacService) List(ctx context.Context) ([]model.Role, error) {
	return rbac.rbacRepository.List(ctx)
}

// Create 创建角色（FIX-002）：名称租户内唯一，服务层预检 + 部分唯一索引兜底
func (rbac *rbacService) Create(ctx context.Context, role *model.Role) (*model.Role, error) {
	if err := rbac.ensureNameAvailable(ctx, role.Name, 0); err != nil {
		return nil, err
	}

	created, err := rbac.rbacRepository.Create(ctx, role)
	if err == nil && rbac.audit != nil {
		rbac.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "create", ResourceType: "role",
			ResourceID: strconv.FormatUint(uint64(created.ID), 10),
			After:      map[string]any{"name": created.Name, "rules": created.Rules},
		})
	}
	return created, err
}

func (rbac *rbacService) Get(ctx context.Context, id string) (*model.Role, error) {
	rid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return rbac.rbacRepository.GetRoleByID(ctx, rid)
}

func (rbac *rbacService) Update(ctx context.Context, id string, role *model.Role) (*model.Role, error) {
	rid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	// 先加载再更新（FIX-022）：伪造他租角色 ID 时 GORM 租户过滤使 Update
	// 影响 0 行却返回成功，形成「假成功 + 假审计」；加载即被过滤拒绝
	if _, err := rbac.rbacRepository.GetRoleByID(ctx, rid); err != nil {
		return nil, err
	}
	// 改名时校验租户内唯一（排除自身）
	if err := rbac.ensureNameAvailable(ctx, role.Name, uint(rid)); err != nil {
		return nil, err
	}

	role.ID = uint(rid)
	updated, err := rbac.rbacRepository.Update(ctx, role)
	if err == nil && rbac.audit != nil {
		rbac.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "role",
			ResourceID: id,
			After:      map[string]any{"name": role.Name, "rules": role.Rules},
		})
	}
	return updated, err
}

// Delete 角色删除为软删除（FIX-001：Role 模型已带 DeletedAt，
// 普通 Delete 即软删，确需物理删除由仓储显式 Unscoped）
func (rbac *rbacService) Delete(ctx context.Context, id string) error {
	rid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	// 先加载再删除（FIX-022）：跨租户 ID 必须显式拒绝而非静默 0 行成功
	if _, err := rbac.rbacRepository.GetRoleByID(ctx, rid); err != nil {
		return err
	}

	if err := rbac.rbacRepository.Delete(ctx, uint(rid)); err != nil {
		return err
	}

	if rbac.audit != nil {
		rbac.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "delete", ResourceType: "role", ResourceID: id,
		})
	}
	return nil
}

func (rbac *rbacService) ListResources(ctx context.Context) ([]model.Resource, error) {
	return rbac.rbacRepository.ListResources(ctx)
}

func (rbac *rbacService) ListOperations() ([]model.Operation, error) {
	return []model.Operation{
		model.AllOperation,
		model.EditOperation,
		model.ViewOperation,
		request.CreateOperation,
		request.PatchOperation,
		request.UpdateOperation,
		request.GetOperation,
		request.ListOperation,
		request.DeleteOperation,
		"log",
		"exec",
		"proxy",
	}, nil
}

// ensureNameAvailable 租户内重名校验（FIX-002）：excludeSelf 用于更新场景排除自身。
// 软删除角色不占名（GetRoleByName 自动过滤已删行），删后可重建同名
func (rbac *rbacService) ensureNameAvailable(ctx context.Context, name string, excludeSelf uint) error {
	if name == "" {
		return fmt.Errorf("role name is required")
	}
	existing, err := rbac.rbacRepository.GetRoleByName(ctx, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != excludeSelf {
		return fmt.Errorf("%w: role name %s already exists in tenant", ErrDuplicateName, name)
	}
	return nil
}
