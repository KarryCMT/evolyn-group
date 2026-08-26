package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	editionmodel "evolyn/internal/platform/edition/model"
	iammodel "evolyn/internal/platform/iam/model"
	apperrors "evolyn/internal/platform/tenantproduct"
	tpmodel "evolyn/internal/platform/tenantproduct/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 产品中心服务单元测试（文档 11 用例的可离线子集；跨租户/并发真库
// 语义由迁移与 SQL 约束保证，可按 SEC-* 模式补充集成用例）----

// passThroughTx 不携带事务语义、直接执行 fn
type passThroughTx struct{}

func (passThroughTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeRepo 仓储替身：内存态模拟本域 SQL 语义（租户过滤、乐观更新、
// 范围全量替换、active 成员/部门口径、部门-成员归属）
type fakeRepo struct {
	catalogs    []*tpmodel.ProductCatalog
	configs     []*tpmodel.TenantProductConfig
	scopeDepts  map[uint][]uint // configID → departmentIDs
	scopeMember map[uint][]uint // configID → memberIDs
	departments map[uint][]iammodel.Department
	members     map[uint][]iammodel.User
	// memberDepts: tenantID → memberID → departmentIDs（模拟 department_users）
	memberDepts map[uint]map[uint][]uint
	updateErr   error // 注入更新故障（验证审计只在提交成功后记录）
	nextID      uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		scopeDepts:  map[uint][]uint{},
		scopeMember: map[uint][]uint{},
		departments: map[uint][]iammodel.Department{},
		members:     map[uint][]iammodel.User{},
		memberDepts: map[uint]map[uint][]uint{},
		nextID:      100,
	}
}

func (f *fakeRepo) id() uint { f.nextID++; return f.nextID }

func (f *fakeRepo) addCatalog(code, status string) *tpmodel.ProductCatalog {
	c := &tpmodel.ProductCatalog{ID: f.id(), Code: code, Name: code, Status: status, EntryPath: "/workspace"}
	f.catalogs = append(f.catalogs, c)
	return c
}

func (f *fakeRepo) addConfig(tenantID, productID uint, enabled bool, mode string) *tpmodel.TenantProductConfig {
	cfg := &tpmodel.TenantProductConfig{ID: f.id(), ProductID: productID, Enabled: enabled, ScopeMode: mode, Revision: 1}
	// TenantID 经 TenantBaseModel 内嵌提升，字面量不可直接赋值
	cfg.TenantID = tenantID
	f.configs = append(f.configs, cfg)
	return cfg
}

func (f *fakeRepo) addDepartment(tenantID uint, id uint, parent *uint, status string) iammodel.Department {
	dept := iammodel.Department{ID: id, ParentId: parent, Name: "dept", Status: status}
	dept.TenantID = tenantID
	f.departments[tenantID] = append(f.departments[tenantID], dept)
	return dept
}

func (f *fakeRepo) addMember(tenantID uint, id uint, status string) iammodel.User {
	member := iammodel.User{ID: id, Nickname: "member", Status: status}
	member.TenantID = tenantID
	f.members[tenantID] = append(f.members[tenantID], member)
	return member
}

func (f *fakeRepo) bindDepartments(tenantID, memberID uint, deptIDs ...uint) {
	if f.memberDepts[tenantID] == nil {
		f.memberDepts[tenantID] = map[uint][]uint{}
	}
	f.memberDepts[tenantID][memberID] = append(f.memberDepts[tenantID][memberID], deptIDs...)
}

func (f *fakeRepo) ListCatalog(ctx context.Context) ([]tpmodel.ProductCatalog, error) {
	result := make([]tpmodel.ProductCatalog, 0, len(f.catalogs))
	for _, c := range f.catalogs {
		result = append(result, *c)
	}
	return result, nil
}

func (f *fakeRepo) GetCatalogByCode(ctx context.Context, code string) (*tpmodel.ProductCatalog, error) {
	for _, c := range f.catalogs {
		if c.Code == code {
			clone := *c
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) ListConfigsByTenant(ctx context.Context, tenantID uint) ([]tpmodel.TenantProductConfig, error) {
	result := make([]tpmodel.TenantProductConfig, 0)
	for _, c := range f.configs {
		if c.TenantID == tenantID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (f *fakeRepo) getConfig(tenantID, productID uint) *tpmodel.TenantProductConfig {
	for _, c := range f.configs {
		if c.TenantID == tenantID && c.ProductID == productID {
			return c
		}
	}
	return nil
}

func (f *fakeRepo) GetConfig(ctx context.Context, tenantID, productID uint) (*tpmodel.TenantProductConfig, error) {
	if c := f.getConfig(tenantID, productID); c != nil {
		clone := *c
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) LockConfig(ctx context.Context, tenantID, productID uint) (*tpmodel.TenantProductConfig, error) {
	return f.GetConfig(ctx, tenantID, productID)
}

func (f *fakeRepo) CreateConfig(ctx context.Context, config *tpmodel.TenantProductConfig) error {
	config.ID = f.id()
	f.configs = append(f.configs, config)
	return nil
}

func (f *fakeRepo) UpdateEnabledWithRevision(ctx context.Context, id uint, fromRevision int64, enabled bool) (bool, error) {
	for _, c := range f.configs {
		if c.ID == id {
			if c.Revision != fromRevision {
				return false, nil
			}
			if f.updateErr != nil {
				return false, f.updateErr
			}
			c.Enabled = enabled
			c.Revision++
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) UpdateScopeWithRevision(ctx context.Context, id uint, fromRevision int64, scopeMode string) (bool, error) {
	for _, c := range f.configs {
		if c.ID == id {
			if c.Revision != fromRevision {
				return false, nil
			}
			if f.updateErr != nil {
				return false, f.updateErr
			}
			c.ScopeMode = scopeMode
			c.Revision++
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) ListScopeDepartments(ctx context.Context, configID uint) ([]uint, error) {
	return append([]uint{}, f.scopeDepts[configID]...), nil
}

func (f *fakeRepo) ListScopeMembers(ctx context.Context, configID uint) ([]uint, error) {
	return append([]uint{}, f.scopeMember[configID]...), nil
}

func (f *fakeRepo) ReplaceScope(ctx context.Context, config *tpmodel.TenantProductConfig, departmentIDs, memberIDs []uint) error {
	f.scopeDepts[config.ID] = append([]uint{}, departmentIDs...)
	f.scopeMember[config.ID] = append([]uint{}, memberIDs...)
	return nil
}

func (f *fakeRepo) ListTenantDepartments(ctx context.Context, tenantID uint) ([]iammodel.Department, error) {
	return append([]iammodel.Department{}, f.departments[tenantID]...), nil
}

func (f *fakeRepo) memberOf(tenantID, memberID uint) *iammodel.User {
	for i := range f.members[tenantID] {
		if f.members[tenantID][i].ID == memberID {
			return &f.members[tenantID][i]
		}
	}
	return nil
}

func (f *fakeRepo) GetMember(ctx context.Context, tenantID, memberID uint) (*iammodel.User, error) {
	if m := f.memberOf(tenantID, memberID); m != nil {
		clone := *m
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) ListMembersByIDs(ctx context.Context, tenantID uint, ids []uint) ([]iammodel.User, error) {
	result := make([]iammodel.User, 0)
	for _, id := range ids {
		if m := f.memberOf(tenantID, id); m != nil {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (f *fakeRepo) CountActiveMembers(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	for i := range f.members[tenantID] {
		if f.members[tenantID][i].Status == iammodel.MemberStatusActive {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepo) CountActiveMembersInScope(ctx context.Context, tenantID uint, memberIDs, deptIDs []uint) (int64, error) {
	deptSet := map[uint]struct{}{}
	for _, id := range deptIDs {
		deptSet[id] = struct{}{}
	}
	memberSet := map[uint]struct{}{}
	for _, id := range memberIDs {
		memberSet[id] = struct{}{}
	}
	var count int64
	for i := range f.members[tenantID] {
		m := &f.members[tenantID][i]
		if m.Status != iammodel.MemberStatusActive {
			continue
		}
		if _, ok := memberSet[m.ID]; ok {
			count++
			continue
		}
		for _, deptID := range f.memberDepts[tenantID][m.ID] {
			if _, ok := deptSet[deptID]; ok {
				count++
				break
			}
		}
	}
	return count, nil
}

func (f *fakeRepo) MemberInDepartments(ctx context.Context, tenantID, memberID uint, deptIDs []uint) (bool, error) {
	deptSet := map[uint]struct{}{}
	for _, id := range deptIDs {
		deptSet[id] = struct{}{}
	}
	for _, deptID := range f.memberDepts[tenantID][memberID] {
		if _, ok := deptSet[deptID]; ok {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) Migrate() error { return nil }

// fakeEditions 版本信息窄端口替身：固定返回试用版投影
type fakeEditions struct{}

func (fakeEditions) GetCurrent(ctx context.Context, tenantID uint) (*editionmodel.CurrentEdition, error) {
	return &editionmodel.CurrentEdition{
		Subscription: editionmodel.SubscriptionView{
			PlanCode: "trial", PlanName: "试用版", Status: "active",
		},
	}, nil
}

// fakeAudit 审计记录替身（仅捕获动作标识，断言提交成功后 best-effort 落审计）
type fakeAudit struct{ entries []map[string]any }

func (f *fakeAudit) Record(ctx context.Context, e auditservice.Entry) {
	f.entries = append(f.entries, map[string]any{
		"module": e.Module, "action": e.Action, "resource": e.ResourceType,
	})
}

func newService(repo *fakeRepo) (TenantProductService, *fakeAudit) {
	audit := &fakeAudit{}
	return NewTenantProductService(passThroughTx{}, repo, fakeEditions{}, audit), audit
}

// TestListDefaultAfterSeed 用例 1/9：种子初始化后默认启用、范围 all，
// 版本名称来自 edition 域
func TestListDefaultAfterSeed(t *testing.T) {
	repo := newFakeRepo()
	active := repo.addCatalog("lingyanyun", tpmodel.CatalogStatusActive)
	repo.addCatalog("paused", tpmodel.CatalogStatusInactive)
	repo.addMember(1, 10, iammodel.MemberStatusActive)
	repo.addMember(1, 11, iammodel.MemberStatusDisabled)
	repo.addMember(1, 12, iammodel.MemberStatusResigned)

	svc, _ := newService(repo)
	require.NoError(t, svc.SeedDefaults(context.Background(), 1))

	view, err := svc.List(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, view.Items, 2)

	first := view.Items[0]
	assert.Equal(t, "lingyanyun", first.Code)
	assert.True(t, first.Enabled)
	assert.Equal(t, int64(1), first.Revision)
	assert.Equal(t, "trial", first.Edition.PlanCode)
	assert.Equal(t, "试用版", first.Edition.PlanName)
	assert.Equal(t, tpmodel.ScopeModeAll, first.AccessScope.Mode)
	// 有效成员只统计 active：3 个成员中 1 个 active
	assert.Equal(t, int64(1), first.AccessScope.EligibleMemberCount)

	// 种子幂等：重复调用不产生重复配置；inactive 目录不建配置
	require.NoError(t, svc.SeedDefaults(context.Background(), 1))
	configs, _ := repo.ListConfigsByTenant(context.Background(), 1)
	require.Len(t, configs, 1)
	assert.Equal(t, active.ID, configs[0].ProductID)
}

// TestListUninitializedConfig 目录先于回填到达：保守合成停用卡片
func TestListUninitializedConfig(t *testing.T) {
	repo := newFakeRepo()
	repo.addCatalog("lingyanyun", tpmodel.CatalogStatusActive)
	repo.addMember(1, 10, iammodel.MemberStatusActive)

	svc, _ := newService(repo)
	view, err := svc.List(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, view.Items, 1)
	assert.False(t, view.Items[0].Enabled)
	assert.Equal(t, int64(0), view.Items[0].Revision)
	assert.Equal(t, int64(1), view.Items[0].AccessScope.EligibleMemberCount)
}

// TestSetEnabledOptimisticLock 用例 5：旧 revision 更新得到 409 冲突，
// 只有一个成功
func TestSetEnabledOptimisticLock(t *testing.T) {
	repo := newFakeRepo()
	catalog := repo.addCatalog("lingyanyun", tpmodel.CatalogStatusActive)
	repo.addConfig(1, catalog.ID, true, tpmodel.ScopeModeAll)

	svc, _ := newService(repo)

	// 第一次以 revision=1 停用：成功且版本递增
	card, err := svc.SetEnabled(context.Background(), 1, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: false, Revision: 1})
	require.NoError(t, err)
	assert.False(t, card.Enabled)
	assert.Equal(t, int64(2), card.Revision)

	// 第二个管理员仍持旧 revision=1：拒绝
	_, err = svc.SetEnabled(context.Background(), 1, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: true, Revision: 1})
	assert.ErrorIs(t, err, apperrors.ErrRevisionConflict)

	// 其他租户配置不受影响
	other := repo.addConfig(2, catalog.ID, true, tpmodel.ScopeModeAll)
	assert.True(t, other.Enabled)
}

// TestSetEnabledUninitializedAndAudit 未初始化返回 404；提交成功后记录审计
func TestSetEnabledUninitializedAndAudit(t *testing.T) {
	repo := newFakeRepo()
	repo.addCatalog("lingyanyun", tpmodel.CatalogStatusActive)

	svc, audit := newService(repo)
	_, err := svc.SetEnabled(context.Background(), 1, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: false, Revision: 1})
	assert.ErrorIs(t, err, apperrors.ErrProductNotFound)
	assert.Empty(t, audit.entries)

	repo.addConfig(1, repo.catalogs[0].ID, true, tpmodel.ScopeModeAll)
	_, err = svc.SetEnabled(context.Background(), 1, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: false, Revision: 1})
	require.NoError(t, err)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "update_enabled", audit.entries[0]["action"])

	// 事务失败（更新故障）不产生成功审计（用例 10）
	repo.updateErr = errors.New("db down")
	_, err = svc.SetEnabled(context.Background(), 1, "lingyanyun", &tpmodel.UpdateEnabledRequest{Enabled: true, Revision: 2})
	assert.Error(t, err)
	assert.Len(t, audit.entries, 1)
}

// TestUpdateScopeValidation 用例 3：非法模式/all 携带 ID/partial 空范围/
// 无效成员/无效部门均返回稳定业务错误码
func TestUpdateScopeValidation(t *testing.T) {
	repo := newFakeRepo()
	catalog := repo.addCatalog("lingyanyun", tpmodel.CatalogStatusActive)
	repo.addConfig(1, catalog.ID, true, tpmodel.ScopeModeAll)

	// 部门树：1（active 根）→ 2（active 子）；3（disabled）；租户 2 的部门 9
	repo.addDepartment(1, 1, nil, iammodel.DeptActive)
	repo.addDepartment(1, 2, ptrUint(1), iammodel.DeptActive)
	repo.addDepartment(1, 3, nil, iammodel.DeptDisabled)
	repo.addDepartment(2, 9, nil, iammodel.DeptActive)
	// 成员：10 active（租户 1）、11 resigned（租户 1）、20 active（租户 2）
	repo.addMember(1, 10, iammodel.MemberStatusActive)
	repo.addMember(1, 11, iammodel.MemberStatusResigned)
	repo.addMember(2, 20, iammodel.MemberStatusActive)

	svc, _ := newService(repo)
	ctx := context.Background()

	_, err := svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{Mode: "someone"})
	assert.ErrorIs(t, err, apperrors.ErrScopeInvalid)

	_, err = svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{Mode: tpmodel.ScopeModeAll, MemberIds: []uint{10}})
	assert.ErrorIs(t, err, apperrors.ErrScopeInvalid)

	_, err = svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{Mode: tpmodel.ScopeModePartial})
	assert.ErrorIs(t, err, apperrors.ErrScopeEmpty)

	// 跨租户部门（用例 2）：租户 1 提交租户 2 的部门 9
	_, err = svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{Mode: tpmodel.ScopeModePartial, DepartmentIds: []uint{9}, Revision: 1})
	assert.ErrorIs(t, err, apperrors.ErrDepartmentInvalid)

	// 停用部门不可用于范围
	_, err = svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{Mode: tpmodel.ScopeModePartial, DepartmentIds: []uint{3}, Revision: 1})
	assert.ErrorIs(t, err, apperrors.ErrDepartmentInvalid)

	// 跨租户成员：租户 1 提交租户 2 的成员 20
	_, err = svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{Mode: tpmodel.ScopeModePartial, MemberIds: []uint{20}, Revision: 1})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)

	// 离职成员无效
	_, err = svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{Mode: tpmodel.ScopeModePartial, MemberIds: []uint{11}, Revision: 1})
	assert.ErrorIs(t, err, apperrors.ErrMemberInvalid)

	// 全部失败路径均未落库
	depts, _ := repo.ListScopeDepartments(ctx, repo.configs[0].ID)
	assert.Empty(t, depts)
}

// TestUpdateScopeReplacesAndDedupes 用例 4：重复 ID 去重；范围更新后旧关联
// 不再生效（视图与访问判定均以最新关联为准）
func TestUpdateScopeReplacesAndDedupes(t *testing.T) {
	repo := newFakeRepo()
	catalog := repo.addCatalog("lingyanyun", tpmodel.CatalogStatusActive)
	repo.addConfig(1, catalog.ID, true, tpmodel.ScopeModeAll)
	repo.addDepartment(1, 1, nil, iammodel.DeptActive)
	repo.addMember(1, 10, iammodel.MemberStatusActive)
	repo.addMember(1, 11, iammodel.MemberStatusActive)

	svc, audit := newService(repo)
	ctx := context.Background()

	card, err := svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{
		Mode: tpmodel.ScopeModePartial, DepartmentIds: []uint{1, 1}, MemberIds: []uint{10, 10, 11, 11}, Revision: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, tpmodel.ScopeModePartial, card.AccessScope.Mode)
	assert.Equal(t, []uint{1}, card.AccessScope.DepartmentIds)
	assert.Equal(t, []uint{10, 11}, card.AccessScope.MemberIds)
	// 成员 10/11 未绑定部门，部门 1 也无成员：有效成员 = 直接成员 2 人
	assert.Equal(t, int64(2), card.AccessScope.EligibleMemberCount)

	// 全量替换为仅成员 11：旧关联（部门 1、成员 10）不再生效
	card, err = svc.UpdateAccessScope(ctx, 1, "lingyanyun", &tpmodel.UpdateAccessScopeRequest{
		Mode: tpmodel.ScopeModePartial, MemberIds: []uint{11}, Revision: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, []uint{11}, card.AccessScope.MemberIds)
	assert.Empty(t, card.AccessScope.DepartmentIds)

	// 两次成功的范围替换各记录一条审计
	require.Len(t, audit.entries, 2)
	for _, entry := range audit.entries {
		assert.Equal(t, "update_scope", entry["action"])
	}
}

// TestCanAccess 访问判定语义（用例 6/7/8）：判定链与范围命中
func TestCanAccess(t *testing.T) {
	repo := newFakeRepo()
	catalog := repo.addCatalog("lingyanyun", tpmodel.CatalogStatusActive)
	repo.addCatalog("paused", tpmodel.CatalogStatusInactive)
	repo.addCatalog("missing-config", tpmodel.CatalogStatusActive)

	// 部门树：1 → 2 → 3（active 链），1 → 4（disabled 分支），独立部门 5
	repo.addDepartment(1, 1, nil, iammodel.DeptActive)
	repo.addDepartment(1, 2, ptrUint(1), iammodel.DeptActive)
	repo.addDepartment(1, 3, ptrUint(2), iammodel.DeptActive)
	repo.addDepartment(1, 4, ptrUint(1), iammodel.DeptDisabled)
	repo.addDepartment(1, 5, nil, iammodel.DeptActive)

	// 成员：10 直属部门 1；11 直属部门 3（孙子部门）；12 直属部门 4（停用分支）；
	// 13 无部门；14 resigned；20 属于租户 2
	repo.addMember(1, 10, iammodel.MemberStatusActive)
	repo.bindDepartments(1, 10, 1)
	repo.addMember(1, 11, iammodel.MemberStatusActive)
	repo.bindDepartments(1, 11, 3)
	repo.addMember(1, 12, iammodel.MemberStatusActive)
	repo.bindDepartments(1, 12, 4)
	repo.addMember(1, 13, iammodel.MemberStatusActive)
	repo.addMember(1, 14, iammodel.MemberStatusResigned)
	repo.addMember(2, 20, iammodel.MemberStatusActive)

	// 租户 1 配置：partial，选中部门 1 + 直接成员 13
	config := repo.addConfig(1, catalog.ID, true, tpmodel.ScopeModePartial)
	repo.scopeDepts[config.ID] = []uint{1}
	repo.scopeMember[config.ID] = []uint{13}
	repo.addConfig(2, catalog.ID, true, tpmodel.ScopeModeAll)

	evaluator := NewTenantProductAccessEvaluator(repo)
	ctx := contextx.NewTenantContext(context.Background(), 1)

	// all 范围允许所有有效成员（租户 2 的成员 20 在租户 2 上下文中放行）
	ok, err := evaluator.CanAccess(contextx.NewTenantContext(context.Background(), 2), "lingyanyun", 20)
	require.NoError(t, err)
	assert.True(t, ok)

	// partial：直接成员、选中部门成员、子部门（孙）成员命中
	for _, memberID := range []uint{10, 11, 13} {
		ok, err := evaluator.CanAccess(ctx, "lingyanyun", memberID)
		require.NoError(t, err)
		assert.True(t, ok, "member %d should access", memberID)
	}

	// 部门停用分支（4）虽是选中部门 1 的子级，但不授予访问
	ok, err = evaluator.CanAccess(ctx, "lingyanyun", 12)
	require.NoError(t, err)
	assert.False(t, ok)

	// 不在范围内的无部门成员拒绝
	ok, err = evaluator.CanAccess(ctx, "lingyanyun", 99)
	require.NoError(t, err)
	assert.False(t, ok)

	// 离职成员拒绝（用例 8）
	ok, err = evaluator.CanAccess(ctx, "lingyanyun", 14)
	require.NoError(t, err)
	assert.False(t, ok)

	// 跨租户成员：ctx 为租户 1 时成员 20 表现为不存在
	ok, err = evaluator.CanAccess(ctx, "lingyanyun", 20)
	require.NoError(t, err)
	assert.False(t, ok)

	// 平台停用目录拒绝
	ok, err = evaluator.CanAccess(ctx, "paused", 10)
	require.NoError(t, err)
	assert.False(t, ok)

	// 未初始化配置拒绝
	ok, err = evaluator.CanAccess(ctx, "missing-config", 10)
	require.NoError(t, err)
	assert.False(t, ok)

	// 租户停用后命中范围成员也被拒（用例 6）
	config.Enabled = false
	ok, err = evaluator.CanAccess(ctx, "lingyanyun", 10)
	require.NoError(t, err)
	assert.False(t, ok)
	config.Enabled = true

	// 无租户上下文/未知产品拒绝
	ok, err = evaluator.CanAccess(context.Background(), "lingyanyun", 10)
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = evaluator.CanAccess(ctx, "unknown", 10)
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestExpandActiveDescendants 子树展开纯函数：停用节点及其子树忽略
func TestExpandActiveDescendants(t *testing.T) {
	repo := newFakeRepo()
	repo.addDepartment(1, 1, nil, iammodel.DeptActive)
	repo.addDepartment(1, 2, ptrUint(1), iammodel.DeptActive)
	repo.addDepartment(1, 3, ptrUint(2), iammodel.DeptActive)
	repo.addDepartment(1, 4, ptrUint(1), iammodel.DeptDisabled)
	repo.addDepartment(1, 5, ptrUint(4), iammodel.DeptActive) // 停用节点下的 active 子级

	expanded := expandActiveDescendants([]uint{1}, repo.departments[1])
	assert.ElementsMatch(t, []uint{1, 2, 3}, expanded)

	// 选中停用部门：整棵忽略
	assert.Empty(t, expandActiveDescendants([]uint{4}, repo.departments[1]))

	// 选中不存在部门：忽略
	assert.Empty(t, expandActiveDescendants([]uint{99}, repo.departments[1]))
}

func ptrUint(v uint) *uint { return &v }
