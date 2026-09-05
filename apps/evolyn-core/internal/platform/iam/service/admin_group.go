package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"

	"gorm.io/gorm"
)

// AdminGroupApplicationCatalog 管理组应用清单窄端口：校验 application 组提交的
// 应用 ID 是否存在于本租户。iam 不能反向依赖应用域（应用域已依赖 iam 鉴权），
// 由装配层以适配器桥接（server.go），测试以桩替身
type AdminGroupApplicationCatalog interface {
	// Exists 应用 ID 是否属于当前租户（不存在/已删即 false）
	Exists(ctx context.Context, id uint) (bool, error)
}

// AdminGroupTenantReader 管理组的租户归属窄读取端口：仅用于识别租户创建人。
// 通过接口避免 iam 服务直接依赖租户仓储实现，单测可提供最小替身。
type AdminGroupTenantReader interface {
	GetByID(ctx context.Context, id uint) (*tenantmodel.Tenant, error)
}

// adminGroupService 管理组服务（权限中心-管理员模块）：管理组 CRUD 与分区块
// 即时更新。内置系统管理员组的成员读写代理到 tenant-admin 角色绑定（单一
// 事实源，杜绝双写漂移）。租户创建人是内置系统管理员组的固定成员；管理组自身
// 的资源权限（admin-groups）只授予租户管理员，通讯录管理组成员无法经本服务自我扩权
type adminGroupService struct {
	tx           TxManager
	groups       repository.AdminGroupRepository
	users        repository.UserRepository
	departments  repository.DepartmentRepository
	rbac         repository.RBACRepository
	applications AdminGroupApplicationCatalog
	tenants      AdminGroupTenantReader
	audit        auditservice.Recorder
}

func NewAdminGroupService(
	tx TxManager,
	groups repository.AdminGroupRepository,
	users repository.UserRepository,
	departments repository.DepartmentRepository,
	rbac repository.RBACRepository,
	applications AdminGroupApplicationCatalog,
	tenants AdminGroupTenantReader,
	audit auditservice.Recorder,
) AdminGroupService {
	return &adminGroupService{
		tx:           tx,
		groups:       groups,
		users:        users,
		departments:  departments,
		rbac:         rbac,
		applications: applications,
		tenants:      tenants,
		audit:        audit,
	}
}

// List 按 scope 列出管理组概要：内置组排最前（读取侧幂等兜底补种），
// MemberCount 内置组为 tenant-admin 绑定数（包含企业创建人）、
// 自定义组为成员表计数
func (s *adminGroupService) List(ctx context.Context, scope string) ([]model.AdminGroupSummary, error) {
	if scope != "" && scope != model.AdminGroupScopeSystem && scope != model.AdminGroupScopeApplication {
		return nil, ErrAdminGroupConfigInvalid
	}
	if err := s.ensureBuiltin(ctx); err != nil {
		return nil, err
	}

	all, err := s.groups.ListByTenant(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := s.groups.MemberCounts(ctx)
	if err != nil {
		return nil, err
	}
	// 内置组成员数走角色绑定推导，企业创建人在租户开通时已绑定 tenant-admin，
	// 因而会作为内置系统管理员组的固定成员一并展示和计数。
	// 解析失败（异常库态）时降级为 0 而非报错，列表是读路径，不应因内置角色
	// 缺失整体失败。
	builtinCount := 0
	if roleID, err := s.groups.ResolveBuiltinRoleID(ctx); err == nil {
		if count, err := s.groups.CountBuiltinMembers(ctx, roleID); err == nil {
			builtinCount = int(count)
		}
	}

	summaries := make([]model.AdminGroupSummary, 0, len(all))
	for _, group := range all {
		if scope != "" && group.Scope != scope {
			continue
		}
		count := counts[group.ID]
		if group.BuiltIn {
			count = builtinCount
		}
		summaries = append(summaries, model.AdminGroupSummary{
			ID:          group.ID,
			Name:        group.Name,
			Scope:       group.Scope,
			BuiltIn:     group.BuiltIn,
			MemberCount: count,
		})
	}
	return summaries, nil
}

// Get 管理组详情：成员展示视图 + 范围配置展开（字段名对齐前端 AdministratorGroup）
func (s *adminGroupService) Get(ctx context.Context, id uint) (*model.AdminGroupDetailView, error) {
	group, err := s.groups.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminGroupNotFound
		}
		return nil, err
	}

	members, err := s.loadMembers(ctx, group)
	if err != nil {
		return nil, err
	}
	return buildAdminGroupDetailView(group, members), nil
}

// Create 创建自定义管理组：重名预检 + 默认范围配置（对齐前端新建组的初始态）。
// 内置组只经 seed 产生，请求侧不允许创建 built_in 组
func (s *adminGroupService) Create(ctx context.Context, req *AdminGroupCreateRequest) (*model.AdminGroupDetailView, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 30 {
		return nil, ErrAdminGroupNameInvalid
	}
	if req.Scope != model.AdminGroupScopeSystem && req.Scope != model.AdminGroupScopeApplication {
		return nil, ErrAdminGroupConfigInvalid
	}

	if _, err := s.groups.GetByName(ctx, name); err == nil {
		return nil, ErrAdminGroupDuplicateName
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	group := &model.AdminGroup{
		Name:        name,
		Scope:       req.Scope,
		BuiltIn:     false,
		ScopeConfig: defaultScopeConfig(req.Scope),
	}
	created, err := s.groups.Create(ctx, group)
	if err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "create", ResourceType: "admin_group",
			ResourceID: name,
			After:      map[string]any{"id": created.ID, "scope": created.Scope},
		})
	}
	return s.Get(ctx, created.ID)
}

// Update 分区块即时保存（对齐前端「每次勾选即保存」交互）：请求至多携带一个
// 区块，区块整体替换保证幂等。内置组仅允许 Members 区块（代理角色绑定），
// 名称/配置/删除一律拒绝
func (s *adminGroupService) Update(ctx context.Context, id uint, req *AdminGroupPatchRequest) (*model.AdminGroupDetailView, error) {
	group, err := s.groups.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminGroupNotFound
		}
		return nil, err
	}

	block := req.singleBlock()
	if block == "" {
		return nil, ErrAdminGroupConfigInvalid
	}

	switch block {
	case "members":
		return s.updateMembers(ctx, group, *req.Members)
	case "name":
		return s.rename(ctx, group, *req.Name)
	}

	if group.BuiltIn {
		return nil, ErrAdminGroupBuiltinImmutable
	}
	return s.updateScopeBlock(ctx, group, block, req)
}

// Delete 删除自定义管理组（内置组拒绝）：同一事务清理成员绑定后软删主表
func (s *adminGroupService) Delete(ctx context.Context, id uint) error {
	group, err := s.groups.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminGroupNotFound
		}
		return err
	}
	if group.BuiltIn {
		return ErrAdminGroupBuiltinImmutable
	}

	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.groups.DeleteMembersOfGroup(tctx, group.ID); err != nil {
			return err
		}
		return s.groups.Delete(tctx, group.ID)
	}); err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "delete", ResourceType: "admin_group",
			ResourceID: group.Name,
			Before:     map[string]any{"id": group.ID, "scope": group.Scope},
		})
	}
	return nil
}

// SeedBuiltin 租户开通事务内预置内置系统管理员组（幂等；成员不落表，
// 由开通流程绑定的 tenant-admin 角色实时推导）
func (s *adminGroupService) SeedBuiltin(ctx context.Context, tenantID uint) error {
	if _, err := s.groups.GetByName(ctx, model.AdminGroupBuiltinName); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	group := &model.AdminGroup{
		Name:        model.AdminGroupBuiltinName,
		Scope:       model.AdminGroupScopeSystem,
		BuiltIn:     true,
		ScopeConfig: model.AdminGroupScopeConfig{},
	}
	// seed 路径 ctx 携带目标租户上下文，TenantID 显式赋值兜底（同 fieldSeeder 口径）
	group.TenantID = tenantID
	_, err := s.groups.Create(ctx, group)
	return err
}

// ScopesOfMember 当前成员管理组身份聚合（/auth/admin-scopes）：SystemAdmin
// 即 tenant-admin 角色持有（内置组身份），Groups 仅含自定义管理组
func (s *adminGroupService) ScopesOfMember(ctx context.Context, memberID uint) (*model.MemberAdminScopes, error) {
	scopes := &model.MemberAdminScopes{Groups: make([]model.AdminGroupSummary, 0)}

	if member, err := s.users.GetUserByID(ctx, memberID); err == nil && member != nil {
		for _, role := range member.Roles {
			if role.Name == model.TenantAdminRoleName {
				scopes.SystemAdmin = true
				break
			}
		}
	}

	ids, err := s.groups.ListGroupIDsOfMember(ctx, memberID)
	if err != nil {
		return nil, err
	}
	groups, err := s.groups.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		scopes.Groups = append(scopes.Groups, model.AdminGroupSummary{
			ID: group.ID, Name: group.Name, Scope: group.Scope, BuiltIn: group.BuiltIn,
		})
	}
	return scopes, nil
}

// ---- 内部实现 ----

// ensureBuiltin 读取侧幂等兜底：内置系统管理员组缺失时补种（迁移回填与
// 开通 seed 之外的第三道保险，例如存量租户存在同名自定义组的边缘场景）
func (s *adminGroupService) ensureBuiltin(ctx context.Context) error {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil // 无租户上下文（启动期路径）不补种
	}
	if _, err := s.groups.GetByName(ctx, model.AdminGroupBuiltinName); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return s.SeedBuiltin(ctx, tenantID)
}

// loadMembers 组成员展示视图：内置组经 tenant-admin 角色绑定推导，包含租户
// 创建人这一固定成员；自定义组按 ID 逐个加载（管理组成员量为人工配置规模，
// N+1 可接受；GetMemberDetail 预加载账号与部门）。
func (s *adminGroupService) loadMembers(ctx context.Context, group *model.AdminGroup) ([]model.AdminGroupMemberView, error) {
	if group.BuiltIn {
		roleID, err := s.groups.ResolveBuiltinRoleID(ctx)
		if err != nil {
			// 内置角色缺失属异常库态：返回空成员而非整体报错，管理面仍可进入
			return []model.AdminGroupMemberView{}, nil //nolint:nilerr // 有意降级：异常库态保持管理面可用（见上注释）
		}
		users, err := s.groups.ListBuiltinMembers(ctx, roleID)
		if err != nil {
			return nil, err
		}
		return memberViews(users), nil
	}

	ids, err := s.groups.ListMemberIDs(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	views := make([]model.AdminGroupMemberView, 0, len(ids))
	for _, id := range ids {
		member, err := s.users.GetMemberDetail(ctx, id)
		if err != nil {
			// 悬挂成员 ID（离职清理前被删）静默跳过，读取侧不放大异常
			continue
		}
		views = append(views, memberView(member))
	}
	return views, nil
}

// updateMembers 成员区块更新：内置组代理到 tenant-admin 角色绑定（差量增删，
// 整体替换语义），自定义组整体重建成员表。两者都在同一事务内完成
func (s *adminGroupService) updateMembers(ctx context.Context, group *model.AdminGroup, memberIDs []uint) (*model.AdminGroupDetailView, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, errors.New("tenant context required")
	}
	ownerAccountID, err := s.ownerAccountID(ctx)
	if err != nil {
		return nil, err
	}

	// 目标成员逐个按 ID 加载：跨租户/不存在 ID 由 Callback 过滤为 NotFound，
	// 禁止裸 ID 盲写关系表（FIX-008 口径）；离职成员不允许担任管理员
	validated := make([]uint, 0, len(memberIDs))
	seen := make(map[uint]struct{}, len(memberIDs))
	creatorIncluded := ownerAccountID == 0
	for _, id := range memberIDs {
		if id == 0 {
			return nil, ErrAdminGroupMemberInvalid
		}
		if _, dup := seen[id]; dup {
			continue
		}
		member, err := s.users.GetMemberDetail(ctx, id)
		if err != nil {
			return nil, ErrAdminGroupMemberInvalid
		}
		if member.Status == model.MemberStatusResigned {
			return nil, ErrAdminGroupMemberInvalid
		}
		if ownerAccountID != 0 && member.AccountId == ownerAccountID {
			creatorIncluded = true
			// 创建人仅属于内置系统管理员组；自定义管理组不得重复授予其范围权限。
			if !group.BuiltIn {
				return nil, ErrAdminGroupTenantCreatorNotAllowed
			}
		}
		seen[id] = struct{}{}
		validated = append(validated, id)
	}
	// 企业创建人由开通流程初始化为 tenant-admin，因此必须始终保留在内置
	// 系统管理员组中。此处同时阻止其他管理员将创建人移出该组。
	if group.BuiltIn && !creatorIncluded {
		return nil, ErrAdminGroupTenantCreatorRequired
	}
	if err := s.ensureActorRemainsInGroup(ctx, group, validated); err != nil {
		return nil, err
	}

	if group.BuiltIn {
		return s.updateBuiltinMembers(ctx, tenantID, group, validated)
	}

	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		return s.groups.ReplaceMembers(tctx, group.ID, tenantID, validated)
	}); err != nil {
		return nil, err
	}

	s.recordMembersAudit(ctx, group, memberIDs)
	return s.Get(ctx, group.ID)
}

// ensureActorRemainsInGroup 防止成员在管理组编辑页误把自己移除。仅当操作者
// 当前已属于该管理组、且目标成员集合不再包含自己时拒绝；未认证的测试、后台
// 维护等上下文不触发此保护。内置组包含创建人这一固定成员，其移除还会由
// 创建人保留守卫拒绝。
func (s *adminGroupService) ensureActorRemainsInGroup(ctx context.Context, group *model.AdminGroup, targetMemberIDs []uint) error {
	actor, ok := contextx.ActorFromContext(ctx)
	if !ok || actor.MemberID == 0 || containsAdminGroupMember(targetMemberIDs, actor.MemberID) {
		return nil
	}

	currentMembers, err := s.loadMembers(ctx, group)
	if err != nil {
		return err
	}
	for _, member := range currentMembers {
		if member.ID == actor.MemberID {
			return ErrAdminGroupSelfRemovalNotAllowed
		}
	}
	return nil
}

func containsAdminGroupMember(ids []uint, target uint) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// updateBuiltinMembers 内置组成员更新：以 tenant-admin 角色绑定为事实源做
// 差量增删。守卫：替换后至少保留一名系统管理员（清空即拒绝，含把唯一管理员
// 换成空列表或全部移除自己的场景）
func (s *adminGroupService) updateBuiltinMembers(ctx context.Context, tenantID uint, group *model.AdminGroup, memberIDs []uint) (*model.AdminGroupDetailView, error) {
	if len(memberIDs) == 0 {
		return nil, ErrAdminGroupLastAdmin
	}
	roleID, err := s.groups.ResolveBuiltinRoleID(ctx)
	if err != nil {
		return nil, ErrAdminGroupConfigInvalid
	}
	current, err := s.groups.ListBuiltinMembers(ctx, roleID)
	if err != nil {
		return nil, err
	}
	currentIDs := make(map[uint]struct{}, len(current))
	for _, member := range current {
		currentIDs[member.ID] = struct{}{}
	}
	nextIDs := make(map[uint]struct{}, len(memberIDs))
	for _, id := range memberIDs {
		nextIDs[id] = struct{}{}
	}
	// 清空守卫：目标集合非空即天然满足，此处再显式断言一次防御未来重构
	if len(nextIDs) == 0 {
		return nil, ErrAdminGroupLastAdmin
	}

	role := &model.Role{ID: roleID}
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		for id := range nextIDs {
			if _, stays := currentIDs[id]; stays {
				continue
			}
			if err := s.users.AddRole(tctx, role, &model.User{ID: id}); err != nil {
				return err
			}
		}
		for id := range currentIDs {
			if _, keeps := nextIDs[id]; keeps {
				continue
			}
			if err := s.users.DelRole(tctx, role, &model.User{ID: id}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	s.recordMembersAudit(ctx, group, memberIDs)
	return s.Get(ctx, group.ID)
}

func (s *adminGroupService) rename(ctx context.Context, group *model.AdminGroup, name string) (*model.AdminGroupDetailView, error) {
	if group.BuiltIn {
		return nil, ErrAdminGroupBuiltinImmutable
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 30 {
		return nil, ErrAdminGroupNameInvalid
	}
	if existing, err := s.groups.GetByName(ctx, name); err == nil && existing.ID != group.ID {
		return nil, ErrAdminGroupDuplicateName
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	before := group.Name
	if err := s.groups.Rename(ctx, group.ID, name); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "admin_group",
			ResourceID: name,
			Before:     map[string]any{"name": before},
			After:      map[string]any{"name": name},
		})
	}
	return s.Get(ctx, group.ID)
}

// updateScopeBlock 范围区块更新：校验 scope 适用性与 ID 有效性后整体替换
// 对应区块，单事务写回整份 scope_config
func (s *adminGroupService) updateScopeBlock(ctx context.Context, group *model.AdminGroup, block string, req *AdminGroupPatchRequest) (*model.AdminGroupDetailView, error) {
	config := group.ScopeConfig

	switch block {
	case "departmentScope":
		scope := *req.DepartmentScope
		if err := s.validateMode(scope.Mode); err != nil {
			return nil, err
		}
		// mode=all 语义由模式表达，清单归空避免悬挂引用累积
		if scope.Mode == model.AdminScopeAll {
			scope.DepartmentIDs = nil
		} else if err := s.validateDepartmentIDs(ctx, scope.DepartmentIDs); err != nil {
			return nil, err
		}
		// application 组的分发范围无开关语义（主行直接选全部/部分），恒开启
		if group.Scope == model.AdminGroupScopeApplication {
			scope.Enabled = true
		}
		config.Department = &scope
	case "roleScope":
		scope := *req.RoleScope
		if err := s.validateMode(scope.Mode); err != nil {
			return nil, err
		}
		// 可管理必然隐含可见（先经可见才能操作），服务端兜底联动
		if scope.Manage && !scope.Visible {
			return nil, ErrAdminGroupConfigInvalid
		}
		if scope.Mode == model.AdminScopeAll {
			scope.RoleIDs = nil
		} else if err := s.validateRoleIDs(ctx, scope.RoleIDs); err != nil {
			return nil, err
		}
		config.Role = &scope
	case "externalOrg":
		config.ExternalOrg = req.ExternalOrg
	case "applicationScope":
		if group.Scope != model.AdminGroupScopeApplication {
			return nil, ErrAdminGroupScopeMismatch
		}
		scope := *req.ApplicationScope
		if !scope.AllApplications {
			if err := s.validateApplicationIDs(ctx, scope.ApplicationIDs); err != nil {
				return nil, err
			}
		} else {
			scope.ApplicationIDs = nil
		}
		config.Application = &scope
	case "addressBook":
		if group.Scope != model.AdminGroupScopeApplication {
			return nil, ErrAdminGroupScopeMismatch
		}
		config.AddressBook = req.AddressBook
	default:
		return nil, ErrAdminGroupConfigInvalid
	}

	before := group.ScopeConfig
	if err := s.groups.UpdateConfig(ctx, group.ID, config); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "admin_group",
			ResourceID: group.Name,
			Before:     map[string]any{"block": block, "config": before},
			After:      map[string]any{"block": block, "config": config},
		})
	}
	return s.Get(ctx, group.ID)
}

func (s *adminGroupService) validateMode(mode string) error {
	if mode != model.AdminScopeAll && mode != model.AdminScopePartial {
		return ErrAdminGroupConfigInvalid
	}
	return nil
}

// ownerAccountID 返回当前租户创建人的账号 ID。未注入租户读取端口或历史无主
// 租户均返回 0，使服务在测试及存量数据修复期间保持可用。
func (s *adminGroupService) ownerAccountID(ctx context.Context) (uint, error) {
	if s.tenants == nil {
		return 0, nil
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return 0, errors.New("tenant context required")
	}
	tenant, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	if tenant.OwnerAccountId == nil {
		return 0, nil
	}
	return *tenant.OwnerAccountId, nil
}

// validateDepartmentIDs 部门 ID 全部属于当前租户（partial 清单悬挂引用在写入侧拦截）
func (s *adminGroupService) validateDepartmentIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	existing, err := s.departments.List(ctx)
	if err != nil {
		return err
	}
	valid := make(map[uint]struct{}, len(existing))
	for _, dept := range existing {
		valid[dept.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := valid[id]; !ok {
			return ErrAdminGroupConfigInvalid
		}
	}
	return nil
}

// validateRoleIDs 角色 ID 全部属于当前租户
func (s *adminGroupService) validateRoleIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	existing, err := s.rbac.List(ctx)
	if err != nil {
		return err
	}
	valid := make(map[uint]struct{}, len(existing))
	for _, role := range existing {
		valid[role.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := valid[id]; !ok {
			return ErrAdminGroupConfigInvalid
		}
	}
	return nil
}

// validateApplicationIDs 应用 ID 全部属于当前租户（经装配层窄端口，
// 不引入 iam→application 反向依赖）
func (s *adminGroupService) validateApplicationIDs(ctx context.Context, ids []uint) error {
	if s.applications == nil {
		return nil // 端口未注入（测试/降级）：跳过校验，由读取侧悬挂丢弃兜底
	}
	for _, id := range ids {
		ok, err := s.applications.Exists(ctx, id)
		if err != nil {
			return err
		}
		if !ok {
			return ErrAdminGroupConfigInvalid
		}
	}
	return nil
}

func (s *adminGroupService) recordMembersAudit(ctx context.Context, group *model.AdminGroup, memberIDs []uint) {
	if s.audit == nil {
		return
	}
	s.audit.Record(ctx, auditservice.Entry{
		Module: "iam", Action: "update", ResourceType: "admin_group",
		ResourceID: group.Name,
		After:      map[string]any{"block": "members", "memberIds": memberIDs},
	})
}

// defaultScopeConfig 新建组的默认范围配置：对齐前端新建管理组的初始态
// （全关 + partial 空清单；application 组的分发范围恒开启）
func defaultScopeConfig(scope string) model.AdminGroupScopeConfig {
	config := model.AdminGroupScopeConfig{
		Department: &model.AdminDepartmentScope{
			Enabled: scope == model.AdminGroupScopeApplication,
			Mode:    model.AdminScopePartial,
		},
		Role: &model.AdminRoleScope{
			Mode: model.AdminScopePartial,
		},
		ExternalOrg: &model.AdminExternalOrgScope{},
	}
	if scope == model.AdminGroupScopeApplication {
		config.Application = &model.AdminApplicationScope{}
		config.AddressBook = &model.AdminAddressBookScope{}
	}
	return config
}

// memberViews 将 tenant-admin 角色绑定投影为内置管理组成员，包含企业创建人。
func memberViews(users []model.User) []model.AdminGroupMemberView {
	views := make([]model.AdminGroupMemberView, 0, len(users))
	for i := range users {
		views = append(views, memberView(&users[i]))
	}
	return views
}

// memberView 展示名回落链：租户内昵称 → 账号昵称 → 账号登录名；
// 部门取首个（多部门仅标签展示）
func memberView(member *model.User) model.AdminGroupMemberView {
	view := model.AdminGroupMemberView{ID: member.ID, Name: member.Nickname}
	if view.Name == "" && member.Account != nil {
		view.Name = member.Account.Nickname
		if view.Name == "" {
			view.Name = member.Account.Name
		}
	}
	if len(member.Departments) > 0 {
		view.Department = member.Departments[0].Name
	}
	return view
}

// buildAdminGroupDetailView 聚合详情读模型：内置组恒全量权限（不读配置），
// 自定义组按 scope_config 展开；application 专属区块仅 application 组出网
func buildAdminGroupDetailView(group *model.AdminGroup, members []model.AdminGroupMemberView) *model.AdminGroupDetailView {
	view := &model.AdminGroupDetailView{
		ID:      group.ID,
		Name:    group.Name,
		Scope:   group.Scope,
		BuiltIn: group.BuiltIn,
		Members: members,
	}
	if group.BuiltIn {
		// 内置系统管理员组：全产品/模块的全量管理与数据权限（前端头部文案口径）
		view.DepartmentEnabled = true
		view.DepartmentMode = model.AdminScopeAll
		view.RoleVisible = true
		view.RoleManage = true
		view.RoleMode = model.AdminScopeAll
		view.ExternalEnabled = true
		return view
	}

	config := group.ScopeConfig
	if config.Department != nil {
		view.DepartmentEnabled = config.Department.Enabled
		view.DepartmentMode = config.Department.Mode
		view.DepartmentIDs = nonEmptyIDs(config.Department.DepartmentIDs)
		if view.DepartmentMode == "" {
			view.DepartmentMode = model.AdminScopePartial
		}
	}
	if config.Role != nil {
		view.RoleVisible = config.Role.Visible
		view.RoleManage = config.Role.Manage
		view.RoleMode = config.Role.Mode
		view.RoleIDs = nonEmptyIDs(config.Role.RoleIDs)
		if view.RoleMode == "" {
			view.RoleMode = model.AdminScopePartial
		}
	}
	if config.ExternalOrg != nil {
		view.ExternalEnabled = config.ExternalOrg.Enabled
	}
	if group.Scope == model.AdminGroupScopeApplication {
		if config.Application != nil {
			view.AllApplications = config.Application.AllApplications
			view.ApplicationManage = config.Application.Manage
			view.ApplicationIDs = nonEmptyIDs(config.Application.ApplicationIDs)
		}
		view.AddressBook = config.AddressBook
	}
	return view
}

// nonEmptyIDs nil 清单统一出网为空数组（前端 v-for 安全），并按需去重保序
func nonEmptyIDs(ids []uint) []uint {
	out := make([]uint, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
