package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"strconv"
	"testing"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/ginctx"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	"evolyn/internal/platform/middleware"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/testsupport"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- FIX-022 跨租户攻击测试矩阵（真实 PostgreSQL 集成）----
//
// 验证链路覆盖：Tenant Context → Service → Repository → GORM Callback /
// Tenant Scope → PostgreSQL。每个用例除断言拒绝结果外，均直连数据库校验
// 未发生实际写入或关系污染（绕过服务层的原始行计数）。

// secEnv 双租户攻击测试环境：alpha/beta 各含 owner 成员、tenant-admin 角色、
// root 系统分组与一个部门；victim 为未入租的平台账号
type secEnv struct {
	db         *gorm.DB
	iamRepo    *iamrepository.Repositories
	tenantRepo tenantrepository.TenantRepository
	tenantSvc  tenantservice.TenantService
	userSvc    UserService
	groupSvc   GroupService
	roleSvc    RBACService
	deptSvc    DepartmentService

	alpha, beta             *tenantmodel.Tenant
	alphaMember, betaMember *iammodel.User
	roleAlpha, roleBeta     *iammodel.Role
	groupAlpha, groupBeta   *iammodel.Group
	deptAlpha, deptBeta     *iammodel.Department
	victim                  *iammodel.Account
}

func newSecEnv(t *testing.T) *secEnv {
	t.Helper()

	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iamRepo := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	auditSvc := auditservice.NewService(auditrepository.NewRepository(db))
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), nil)
	txManager := infrastructure.NewTxManager(db)

	env := &secEnv{
		db:         db,
		iamRepo:    iamRepo,
		tenantRepo: tenantRepo,
		tenantSvc:  tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, 0),
		userSvc:    NewUserService(txManager, iamRepo.User(), iamRepo.Account(), iamRepo.RBAC(), iamRepo.Department(), quotaSvc, auditSvc),
		groupSvc:   NewGroupService(iamRepo.Group(), iamRepo.User(), iamRepo.RBAC(), auditSvc),
		roleSvc:    NewRBACService(iamRepo.RBAC(), auditSvc),
		deptSvc:    NewDepartmentService(iamRepo.Department(), iamRepo.User(), auditSvc),
	}

	bg := context.Background()
	env.alpha = env.openTenant(t, "sec-alpha", "owner-alpha")
	env.beta = env.openTenant(t, "sec-beta", "owner-beta")

	env.alphaMember = env.ownerMember(t, env.alpha, "owner-alpha")
	env.betaMember = env.ownerMember(t, env.beta, "owner-beta")

	env.roleAlpha = env.roleByName(t, env.alpha, tenantservice.TenantAdminRole)
	env.roleBeta = env.roleByName(t, env.beta, tenantservice.TenantAdminRole)
	env.groupAlpha = env.groupByName(t, env.alpha, iammodel.RootGroup)
	env.groupBeta = env.groupByName(t, env.beta, iammodel.RootGroup)

	var err error
	env.deptAlpha, err = env.deptSvc.Create(itCtx(env.alpha.ID), &iammodel.Department{Name: "dept-alpha"})
	assert.NoError(t, err)
	env.deptBeta, err = env.deptSvc.Create(itCtx(env.beta.ID), &iammodel.Department{Name: "dept-beta"})
	assert.NoError(t, err)

	env.victim, err = env.iamRepo.Account().Create(bg, &iammodel.Account{Name: "victim", Nickname: "victim"})
	assert.NoError(t, err)

	return env
}

func (e *secEnv) openTenant(t *testing.T, code, ownerName string) *tenantmodel.Tenant {
	t.Helper()
	tenant, err := e.tenantSvc.Open(context.Background(), &tenantservice.OpenTenantRequest{
		Code: code, Name: code, Plan: tenantmodel.PlanFree,
		OwnerName: ownerName, OwnerPassword: "secret123",
	})
	if err != nil {
		t.Fatalf("open tenant %s: %v", code, err)
	}
	return tenant
}

func (e *secEnv) ownerMember(t *testing.T, tenant *tenantmodel.Tenant, ownerName string) *iammodel.User {
	t.Helper()
	account, err := e.iamRepo.Account().GetByName(context.Background(), ownerName)
	if err != nil {
		t.Fatalf("load owner account %s: %v", ownerName, err)
	}
	member, err := e.iamRepo.User().GetByAccountAndTenant(context.Background(), account.ID, tenant.ID)
	if err != nil {
		t.Fatalf("load owner member of tenant %d: %v", tenant.ID, err)
	}
	return member
}

func (e *secEnv) roleByName(t *testing.T, tenant *tenantmodel.Tenant, name string) *iammodel.Role {
	t.Helper()
	role, err := e.iamRepo.RBAC().GetRoleByName(itCtx(tenant.ID), name)
	if err != nil {
		t.Fatalf("load role %s of tenant %d: %v", name, tenant.ID, err)
	}
	return role
}

func (e *secEnv) groupByName(t *testing.T, tenant *tenantmodel.Tenant, name string) *iammodel.Group {
	t.Helper()
	group, err := e.iamRepo.Group().GetGroupByName(itCtx(tenant.ID), name)
	if err != nil {
		t.Fatalf("load group %s of tenant %d: %v", name, tenant.ID, err)
	}
	return group
}

// itCtx 租户上下文（与单测桩 tenantCtx 命名区分）
func itCtx(tenantID uint) context.Context {
	return contextx.NewTenantContext(context.Background(), tenantID)
}

// rawCount 绕过服务层的原始行计数（Background ctx 无租户过滤）
func (e *secEnv) rawCount(t *testing.T, sql string, args ...any) int64 {
	t.Helper()
	var count int64
	assert.NoError(t, e.db.Raw(sql, args...).Scan(&count).Error)
	return count
}

// ---- SEC-TENANT-001~011 ----

// SEC-TENANT-001：伪造他租成员 ID 读取（Tenant Scope 过滤 → NotFound）
func TestSECTenant001MemberCrossTenantRead(t *testing.T) {
	env := newSecEnv(t)

	_, err := env.userSvc.Get(itCtx(env.alpha.ID), strconv.FormatUint(uint64(env.betaMember.ID), 10))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	// 数据未受影响：beta 成员仍在库
	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM users WHERE id = ?", env.betaMember.ID))
}

// SEC-TENANT-002：伪造他租成员 ID 更新（加载即被过滤拒绝）
func TestSECTenant002MemberCrossTenantUpdate(t *testing.T) {
	env := newSecEnv(t)

	_, err := env.userSvc.Update(itCtx(env.alpha.ID), strconv.FormatUint(uint64(env.betaMember.ID), 10),
		&iammodel.User{Nickname: "hacked"})
	assert.Error(t, err)

	assert.EqualValues(t, 1, env.rawCount(t,
		"SELECT COUNT(*) FROM users WHERE id = ? AND nickname = ?", env.betaMember.ID, env.betaMember.Nickname),
		"beta 成员昵称不得被篡改")
}

// SEC-TENANT-003：伪造他租成员 ID 删除
func TestSECTenant003MemberCrossTenantDelete(t *testing.T) {
	env := newSecEnv(t)

	err := env.userSvc.Delete(itCtx(env.alpha.ID), strconv.FormatUint(uint64(env.betaMember.ID), 10))
	assert.Error(t, err)

	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM users WHERE id = ?", env.betaMember.ID),
		"beta 成员不得被删除")
}

// SEC-TENANT-004：成员绑定他租角色（伪造 roleID）
func TestSECTenant004MemberCrossTenantRoleBinding(t *testing.T) {
	env := newSecEnv(t)

	err := env.userSvc.AddRole(itCtx(env.alpha.ID),
		strconv.FormatUint(uint64(env.alphaMember.ID), 10),
		strconv.FormatUint(uint64(env.roleBeta.ID), 10))
	assert.Error(t, err)

	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role_id = ?", env.alphaMember.ID, env.roleBeta.ID),
		"user_roles 不得产生跨租户污染行")
}

// SEC-TENANT-005：把他租成员加入本租分组（group AddMember）
func TestSECTenant005GroupCrossTenantAddMember(t *testing.T) {
	env := newSecEnv(t)

	// 租户 alpha 上下文内把 alpha 成员塞进 beta 的分组：分组加载即被过滤
	err := env.groupSvc.AddUser(itCtx(env.alpha.ID), &iammodel.User{ID: env.alphaMember.ID},
		strconv.FormatUint(uint64(env.groupBeta.ID), 10))
	assert.Error(t, err)

	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM user_groups WHERE user_id = ? AND group_id = ?", env.alphaMember.ID, env.groupBeta.ID))
	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM user_groups WHERE group_id = ?", env.groupBeta.ID),
		"beta 分组成员关系不得被污染")
}

// SEC-TENANT-006：分组绑定他租角色（group AddRole）
func TestSECTenant006GroupCrossTenantAddRole(t *testing.T) {
	env := newSecEnv(t)

	err := env.groupSvc.AddRole(itCtx(env.alpha.ID),
		strconv.FormatUint(uint64(env.groupAlpha.ID), 10),
		strconv.FormatUint(uint64(env.roleBeta.ID), 10))
	assert.Error(t, err)

	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM group_roles WHERE group_id = ? AND role_id = ?", env.groupAlpha.ID, env.roleBeta.ID))
}

// SEC-TENANT-007：伪造他租角色/分组 ID 更新/删除——必须显式拒绝，
// 不允许「租户过滤致 0 行影响却返回成功」的假成功（FIX-022 整改项）
func TestSECTenant007RoleCrossTenantUpdateDelete(t *testing.T) {
	env := newSecEnv(t)

	rid := strconv.FormatUint(uint64(env.roleBeta.ID), 10)
	_, err := env.roleSvc.Update(itCtx(env.alpha.ID), rid, &iammodel.Role{Name: "hacked"})
	assert.Error(t, err)
	err = env.roleSvc.Delete(itCtx(env.alpha.ID), rid)
	assert.Error(t, err)

	assert.EqualValues(t, 1, env.rawCount(t,
		"SELECT COUNT(*) FROM roles WHERE id = ? AND name = ?", env.roleBeta.ID, env.roleBeta.Name),
		"beta 角色不得被篡改或删除")

	// 分组同维度：伪造他租分组 ID 的更新/删除同样必须拒绝
	gid := strconv.FormatUint(uint64(env.groupBeta.ID), 10)
	_, err = env.groupSvc.Update(itCtx(env.alpha.ID), gid, &iammodel.Group{Describe: "hacked"})
	assert.Error(t, err)
	err = env.groupSvc.Delete(itCtx(env.alpha.ID), gid)
	assert.Error(t, err)
	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM groups WHERE id = ?", env.groupBeta.ID),
		"beta 分组不得被删除")
}

// SEC-TENANT-008：拉人入租时伪造他租部门（FIX-021：整个 AddMember 回滚）。
// 真库上 GORM 租户过滤先生效（部门不可见 → NotFound），桩层是
// ensureSameTenant 拒绝——两条路径都必须使整体事务回滚
func TestSECTenant008AddMemberCrossTenantDepartment(t *testing.T) {
	env := newSecEnv(t)

	_, err := env.userSvc.AddMember(itCtx(env.alpha.ID), &AddMemberRequest{
		AccountID: env.victim.ID, DepartmentIDs: []uint{env.deptBeta.ID},
	})
	assert.Error(t, err)

	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM users WHERE account_id = ? AND tenant_id = ?", env.victim.ID, env.alpha.ID),
		"部门绑定失败后成员必须整体回滚（真实事务）")
	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM department_users WHERE department_id = ?", env.deptBeta.ID),
		"department_users 不得产生污染行")
}

// SEC-TENANT-009：拉人入租时伪造他租角色（FIX-021：整个 AddMember 回滚）
func TestSECTenant009AddMemberCrossTenantRole(t *testing.T) {
	env := newSecEnv(t)

	_, err := env.userSvc.AddMember(itCtx(env.alpha.ID), &AddMemberRequest{
		AccountID: env.victim.ID, RoleIDs: []uint{env.roleBeta.ID},
	})
	assert.Error(t, err)

	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM users WHERE account_id = ? AND tenant_id = ?", env.victim.ID, env.alpha.ID),
		"角色绑定失败后成员必须整体回滚（真实事务）")
	// 仅统计 alpha 侧成员的绑定行：beta owner 与其 tenant-admin 的合法
	// 绑定（role_id 相同）不得计入污染
	assert.EqualValues(t, 0, env.rawCount(t, `
		SELECT COUNT(*) FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		WHERE u.tenant_id = ? AND ur.role_id = ?`, env.alpha.ID, env.roleBeta.ID),
		"user_roles 不得产生跨租户污染行")
}

// SEC-TENANT-010：列表查询严格按租户隔离（无跨租户数据泄漏）
func TestSECTenant010TenantScopedLists(t *testing.T) {
	env := newSecEnv(t)

	members, err := env.userSvc.List(itCtx(env.alpha.ID))
	assert.NoError(t, err)
	for _, m := range members {
		assert.Equal(t, env.alpha.ID, m.TenantID, "alpha 列表不得混入他租成员")
	}

	roles, err := env.roleSvc.List(itCtx(env.alpha.ID))
	assert.NoError(t, err)
	for _, r := range roles {
		assert.Equal(t, env.alpha.ID, r.TenantID)
	}

	groups, err := env.groupSvc.List(itCtx(env.alpha.ID))
	assert.NoError(t, err)
	for _, g := range groups {
		assert.Equal(t, env.alpha.ID, g.TenantID)
	}

	depts, err := env.deptSvc.List(itCtx(env.alpha.ID))
	assert.NoError(t, err)
	for _, d := range depts {
		assert.Equal(t, env.alpha.ID, d.TenantID)
	}
}

// SEC-TENANT-011：租户 frozen 状态下请求级拦截（真实仓储链路）
func TestSECTenant011TenantStatusFrozenBlocks(t *testing.T) {
	env := newSecEnv(t)

	// 平台域语义：Background ctx 直接流转状态
	err := env.tenantSvc.SetStatus(context.Background(),
		strconv.FormatUint(uint64(env.beta.ID), 10), tenantmodel.TenantFrozen)
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/api/v1/users", nil)
	ginctx.SetTenant(c, env.beta.ID)

	middleware.TenantStatusMiddleware(env.tenantRepo)(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, 403, rec.Code)
	assert.Contains(t, rec.Body.String(), "TENANT_FROZEN")
}

// ---- 真实 PostgreSQL 事务验证（FIX-020/021 集成层补充）----

// TX-TENANT-INT：Open 全流程真实事务——成功路径基线完整 + owner 绑定本租户
// 角色（回归：旧实现按名回查会绑到其他租户的 tenant-admin）
func TestIntegrationTXTenantOpenAtomic(t *testing.T) {
	env := newSecEnv(t)

	tenant := env.openTenant(t, "sec-gamma", "owner-gamma")

	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM tenants WHERE code = ?", "sec-gamma"))
	assert.EqualValues(t, 3, env.rawCount(t, "SELECT COUNT(*) FROM roles WHERE tenant_id = ?", tenant.ID))
	assert.EqualValues(t, 3, env.rawCount(t, "SELECT COUNT(*) FROM groups WHERE tenant_id = ?", tenant.ID))
	assert.EqualValues(t, 3, env.rawCount(t, `
		SELECT COUNT(*) FROM group_roles gr
		JOIN groups g ON g.id = gr.group_id
		WHERE g.tenant_id = ?`, tenant.ID))

	// owner 成员恰好绑定一个角色，且该角色属于本租户（防跨租户绑定回归）
	assert.EqualValues(t, 1, env.rawCount(t, `
		SELECT COUNT(*) FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		JOIN roles r ON r.id = ur.role_id
		WHERE u.tenant_id = ? AND r.tenant_id = ? AND r.name = ?`,
		tenant.ID, tenant.ID, tenantservice.TenantAdminRole),
		"owner 必须绑定本租户的 tenant-admin")

	// 失败路径：owner 账号重名（accounts.name 唯一约束）在事务中途失败，
	// 不留任何脏数据（租户编码是新的，通过前置查重后落库失败）
	_, err := env.tenantSvc.Open(context.Background(), &tenantservice.OpenTenantRequest{
		Code: "sec-gamma-2", Name: "sec-gamma-2", Plan: tenantmodel.PlanFree,
		OwnerName: "owner-gamma", OwnerPassword: "secret123",
	})
	assert.Error(t, err, "重名账号必须使开通失败")
	assert.EqualValues(t, 0, env.rawCount(t, "SELECT COUNT(*) FROM tenants WHERE code = ?", "sec-gamma-2"),
		"失败开通不得留下租户行")
	assert.EqualValues(t, 1, env.rawCount(t, "SELECT COUNT(*) FROM accounts WHERE name = ?", "owner-gamma"),
		"不得产生重复账号")
}

// TX-MEMBER-INT：AddMember 事务中途失败（不存在的角色）→ 成员与部门绑定全部回滚
func TestIntegrationTXMemberAddMemberRollback(t *testing.T) {
	env := newSecEnv(t)

	victim2, err := env.iamRepo.Account().Create(context.Background(), &iammodel.Account{Name: "victim2"})
	assert.NoError(t, err)

	// 部门绑定成功后角色加载失败（id 不存在）：真实数据库回滚
	_, err = env.userSvc.AddMember(itCtx(env.alpha.ID), &AddMemberRequest{
		AccountID: victim2.ID, DepartmentIDs: []uint{env.deptAlpha.ID}, RoleIDs: []uint{99999999},
	})
	assert.Error(t, err)

	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM users WHERE account_id = ? AND tenant_id = ?", victim2.ID, env.alpha.ID),
		"成员必须整体回滚")
	assert.EqualValues(t, 0, env.rawCount(t,
		"SELECT COUNT(*) FROM department_users WHERE user_id NOT IN (SELECT id FROM users)"),
		"不得残留指向已回滚成员的部门关系")
}
