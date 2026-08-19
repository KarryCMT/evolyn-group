package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"gorm.io/gorm"
)

// userService 成员服务（租户内语义）：登录身份相关见 AccountService（ADR-006）。
// 依赖说明：RBAC/部门仓储用于关系绑定的同租户校验（FIX-006）；
// 配额服务来自租户域（FIX-011，依赖方向 iam→tenant/service 单向无环）
type userService struct {
	userRepository       repository.UserRepository
	accountRepository    repository.AccountRepository
	rbacRepository       repository.RBACRepository
	departmentRepository repository.DepartmentRepository
	quota                tenantservice.QuotaService
	audit                auditservice.Recorder
}

func NewUserService(
	userRepository repository.UserRepository,
	accountRepository repository.AccountRepository,
	rbacRepository repository.RBACRepository,
	departmentRepository repository.DepartmentRepository,
	quota tenantservice.QuotaService,
	audit auditservice.Recorder,
) UserService {
	return &userService{
		userRepository:       userRepository,
		accountRepository:    accountRepository,
		rbacRepository:       rbacRepository,
		departmentRepository: departmentRepository,
		quota:                quota,
		audit:                audit,
	}
}

func (u *userService) List(ctx context.Context) (model.Users, error) {
	return u.userRepository.List(ctx)
}

func (u *userService) Get(ctx context.Context, id string) (*model.User, error) {
	return u.getUserByID(ctx, id)
}

// Update 成员资料更新（租户内昵称）；账号资料走 accounts 域
func (u *userService) Update(ctx context.Context, id string, member *model.User) (*model.User, error) {
	old, err := u.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if member.ID != 0 && old.ID != member.ID {
		return nil, fmt.Errorf("update member %s not match", id)
	}
	member.ID = old.ID

	updated, err := u.userRepository.Update(ctx, member)
	if err == nil && u.audit != nil {
		u.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "member",
			ResourceID: strconv.FormatUint(uint64(old.ID), 10),
			Before:     map[string]string{"nickname": old.Nickname},
			After:      map[string]string{"nickname": member.Nickname},
		})
	}
	return updated, err
}

func (u *userService) Delete(ctx context.Context, id string) error {
	member, err := u.getUserByID(ctx, id)
	if err != nil {
		return err
	}

	if err := u.userRepository.Delete(ctx, member); err != nil {
		return err
	}

	if u.audit != nil {
		u.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "delete", ResourceType: "member",
			ResourceID: id,
			After:      map[string]any{"accountId": member.AccountId, "nickname": member.Nickname},
		})
	}
	return nil
}

func (u *userService) GetGroups(ctx context.Context, id string) ([]model.Group, error) {
	member, err := u.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return u.userRepository.GetGroups(ctx, member)
}

// AddRole 成员绑定角色（FIX-006）：加载两端实体并校验同租户后才允许写入，
// 不允许裸 ID 盲写关系表
func (u *userService) AddRole(ctx context.Context, id, rid string) error {
	member, role, err := u.loadMemberAndRole(ctx, id, rid)
	if err != nil {
		return err
	}
	if err := ensureSameTenant(member.TenantID, role.TenantID, "member", member.ID, "role", role.ID); err != nil {
		return err
	}

	if err := u.userRepository.AddRole(ctx, role, member); err != nil {
		return err
	}

	if u.audit != nil {
		u.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "bind", ResourceType: "member_role",
			ResourceID: fmt.Sprintf("%d:%d", member.ID, role.ID),
			After:      map[string]string{"member": member.Nickname, "role": role.Name},
		})
	}
	return nil
}

func (u *userService) DelRole(ctx context.Context, id, rid string) error {
	member, role, err := u.loadMemberAndRole(ctx, id, rid)
	if err != nil {
		return err
	}
	if err := ensureSameTenant(member.TenantID, role.TenantID, "member", member.ID, "role", role.ID); err != nil {
		return err
	}

	if err := u.userRepository.DelRole(ctx, role, member); err != nil {
		return err
	}

	if u.audit != nil {
		u.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "unbind", ResourceType: "member_role",
			ResourceID: fmt.Sprintf("%d:%d", member.ID, role.ID),
			Before:     map[string]string{"member": member.Nickname, "role": role.Name},
		})
	}
	return nil
}

// AddMember 拉已有账号进当前租户（FIX-010）：
// 校验账号 → 重复成员 → 配额 → 创建成员 → 绑定部门/角色 → 审计
func (u *userService) AddMember(ctx context.Context, req *AddMemberRequest) (*model.User, error) {
	if req == nil || (req.AccountID == 0 && req.AccountName == "") {
		return nil, fmt.Errorf("accountId or accountName is required")
	}

	// 目标租户取自请求上下文（租户域路由链路由 TenantMiddleware 注入）
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}

	// 1. 账号必须存在（登录身份是平台级，无租户过滤）
	account, err := u.resolveAccount(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. 同租户重复成员拦截（FIX-004 服务层前置，数据库部分唯一索引兜底）
	if existing, err := u.userRepository.GetByAccountAndTenant(ctx, account.ID, tenantID); err == nil && existing != nil {
		return nil, fmt.Errorf("%w: account %s already a member of tenant %d", ErrDuplicateMember, account.Name, tenantID)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 3. 配额校验（FIX-011）
	if u.quota != nil {
		if err := u.quota.Check(ctx, tenantID, tenantmodel.QuotaMembers); err != nil {
			return nil, err
		}
	}

	// 4. 创建成员（显式归属目标租户）
	nickname := req.Nickname
	if nickname == "" {
		nickname = account.Nickname
		if nickname == "" {
			nickname = account.Name
		}
	}
	member := &model.User{AccountId: account.ID, Nickname: nickname}
	member.TenantID = tenantID
	member, err = u.userRepository.Create(ctx, member)
	if err != nil {
		return nil, err
	}

	// 5. 部门绑定（同租户校验，FIX-006）
	if len(req.DepartmentIDs) > 0 {
		if err := u.bindMemberDepartments(ctx, member, req.DepartmentIDs); err != nil {
			return nil, err
		}
	}

	// 6. 角色绑定（同租户校验，FIX-006）
	if len(req.RoleIDs) > 0 {
		if err := u.bindMemberRoles(ctx, member, req.RoleIDs); err != nil {
			return nil, err
		}
	}

	// 7. 审计（FIX-013）
	if u.audit != nil {
		u.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "create", ResourceType: "member",
			ResourceID: strconv.FormatUint(uint64(member.ID), 10),
			After:      map[string]any{"accountId": account.ID, "accountName": account.Name, "nickname": nickname},
		})
	}

	return member, nil
}

// bindMemberDepartments 逐个加载部门实体校验同租户后整体替换归属
func (u *userService) bindMemberDepartments(ctx context.Context, member *model.User, departmentIDs []uint) error {
	for _, did := range departmentIDs {
		dept, err := u.departmentRepository.GetByID(ctx, did)
		if err != nil {
			return fmt.Errorf("department %d not found", did)
		}
		if err := ensureSameTenant(member.TenantID, dept.TenantID, "member", member.ID, "department", dept.ID); err != nil {
			return err
		}
	}
	return u.departmentRepository.SetMemberDepartments(ctx, member, departmentIDs)
}

// bindMemberRoles 逐个加载角色实体校验同租户后绑定
func (u *userService) bindMemberRoles(ctx context.Context, member *model.User, roleIDs []uint) error {
	for _, rid := range roleIDs {
		role, err := u.rbacRepository.GetRoleByID(ctx, int(rid))
		if err != nil {
			return fmt.Errorf("role %d not found", rid)
		}
		if err := ensureSameTenant(member.TenantID, role.TenantID, "member", member.ID, "role", role.ID); err != nil {
			return err
		}
		if err := u.userRepository.AddRole(ctx, role, member); err != nil {
			return err
		}
	}
	return nil
}

func (u *userService) resolveAccount(ctx context.Context, req *AddMemberRequest) (*model.Account, error) {
	if req.AccountID > 0 {
		return u.accountRepository.GetByID(ctx, req.AccountID)
	}
	return u.accountRepository.GetByName(ctx, req.AccountName)
}

// loadMemberAndRole 按业务 ID 加载成员与角色实体（ctx 携带租户时
// GORM Callback 已施加租户过滤，跨租户 ID 直接 NotFound）
func (u *userService) loadMemberAndRole(ctx context.Context, id, rid string) (*model.User, *model.Role, error) {
	member, err := u.getUserByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return nil, nil, err
	}
	role, err := u.rbacRepository.GetRoleByID(ctx, roleId)
	if err != nil {
		return nil, nil, err
	}
	return member, role, nil
}

func (u *userService) getUserByID(ctx context.Context, id string) (*model.User, error) {
	uid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return u.userRepository.GetUserByID(ctx, uint(uid))
}

// ensureSameTenant 关系两端租户一致性校验（FIX-006 一期最低要求）：
// 双重防御——GORM Callback 已按租户过滤读取，此处显式比对兜底
func ensureSameTenant(a, b uint, aKind string, aID uint, bKind string, bID uint) error {
	if a != b {
		return fmt.Errorf("%w: %s %d (tenant %d) cannot bind %s %d (tenant %d)",
			ErrCrossTenantBinding, aKind, aID, a, bKind, bID, b)
	}
	return nil
}
