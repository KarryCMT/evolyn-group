package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/infrastructure"
	apperrors "evolyn/internal/platform/app"
	"evolyn/internal/platform/app/model"
	"evolyn/internal/platform/app/repository"

	"github.com/stretchr/testify/assert"
)

// ---- SEC-MENU-* 真实 PostgreSQL 集成测试矩阵（M2-菜单-1 只读骨架）----
//
// 验证链路覆盖：迁移链 000016（menu_revision 列 + 菜单节点表）→
// MenuRepository.GetSnapshot 单条 SQL 快照（显式 tenant_id 条件）→
// MenuService 树校验/裁剪/capabilities。未配置 TEST_PG_DSN 时自动
// Skip（离线套件保持全绿），与 sec_app_integration_test.go 同约定。

// menuEnv 复用 appEnv 的双租户环境，另挂菜单仓储与服务
type menuEnv struct {
	*appEnv
	menuRepo repository.MenuRepository
	menuSvc  AppMenuService
}

func newMenuEnv(t *testing.T) *menuEnv {
	t.Helper()
	base := newAppEnv(t)
	menuRepo := repository.NewMenuRepository(base.db)
	return &menuEnv{
		appEnv:   base,
		menuRepo: menuRepo,
		menuSvc:  NewMenuService(infrastructure.NewTxManager(base.db), menuRepo, nil, NewRBACAccessEvaluator(base.iamRepo.User(), base.iamRepo.Group())),
	}
}

// insertMenuNode 直写菜单节点（绕过 Service：写路径随 M2-菜单-3 落地，
// 测试读取/校验/裁剪链路）
func (e *menuEnv) insertMenuNode(t *testing.T, ctx context.Context, node *model.MenuNode) {
	t.Helper()
	assert.NoError(t, e.db.WithContext(ctx).Create(node).Error)
}

// SEC-MENU-001：空应用菜单为空树（200 语义），menuRevision 出网
func TestSECMENU001EmptyMenu(t *testing.T) {
	env := newMenuEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("菜单应用"))
	assert.NoError(t, err)

	menu, err := env.menuSvc.GetMenu(appCtx(env.alpha.ID), env.alphaMember, created.Code)
	assert.NoError(t, err)
	assert.Equal(t, created.Code, menu.AppCode)
	assert.Equal(t, int64(1), menu.MenuRevision)
	assert.Empty(t, menu.RootMenuIDs)
	assert.Empty(t, menu.NodeMap)
	assert.False(t, menu.Features.Workflow)
}

// SEC-MENU-007：分组创建推进修订号，且菜单管理成员可在读取快照中看到空分组。
func TestSECMENU007CreateGroup(t *testing.T) {
	env := newMenuEnv(t)
	ctx := appCtx(env.alpha.ID)
	createdApp, err := env.appSvc.CreateBlank(ctx, env.alphaMember, blankReq("分组创建应用"))
	assert.NoError(t, err)

	group, err := env.menuSvc.CreateGroup(ctx, env.alphaMember, createdApp.Code, &model.CreateMenuGroupRequest{
		Name:             "订单管理",
		BaseMenuRevision: 1,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), group.MenuRevision)

	menu, err := env.menuSvc.GetMenu(ctx, env.alphaMember, createdApp.Code)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), menu.MenuRevision)
	assert.Equal(t, []string{group.MenuID}, menu.RootMenuIDs)
	assert.Equal(t, "订单管理", menu.NodeMap[group.MenuID].Name)

	_, err = env.menuSvc.CreateGroup(ctx, env.alphaMember, createdApp.Code, &model.CreateMenuGroupRequest{
		Name:             "陈旧写入",
		BaseMenuRevision: 1,
	})
	assert.True(t, errors.Is(err, apperrors.ErrMenuVersionConflict))
}

// SEC-MENU-002：租户 A 上下文读取租户 B 的应用菜单 → NotFound；
// 租户 B 自身可读。快照 SQL 显式携带 tenant_id（Raw 查询不经租户
// Callback，此用例是该条件的直接回归）
func TestSECMENU002CrossTenantIsolation(t *testing.T) {
	env := newMenuEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.beta.ID), env.betaMember, blankReq("beta 菜单"))
	assert.NoError(t, err)

	_, err = env.menuSvc.GetMenu(appCtx(env.alpha.ID), env.alphaMember, created.Code)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))

	menu, err := env.menuSvc.GetMenu(appCtx(env.beta.ID), env.betaMember, created.Code)
	assert.NoError(t, err)
	assert.Equal(t, created.Code, menu.AppCode)
}

// SEC-MENU-003：软删应用菜单不可读（统一 NotFound）
func TestSECMENU003SoftDeletedApp(t *testing.T) {
	env := newMenuEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("待删菜单"))
	assert.NoError(t, err)
	assert.NoError(t, env.appSvc.Delete(appCtx(env.alpha.ID), env.alphaMember, created.ID))

	_, err = env.menuSvc.GetMenu(appCtx(env.alpha.ID), env.alphaMember, created.Code)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
}

// SEC-MENU-004：authenticated 基线成员（仅 apps:view 聚合展开的
// get/list）可读菜单，且 capabilities 不含管理能力
func TestSECMENU004PlainMemberReadOnly(t *testing.T) {
	env := newMenuEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("基线菜单"))
	assert.NoError(t, err)
	plain := env.createPlainMember(t, env.alpha, "menu-plain-a")

	menu, err := env.menuSvc.GetMenu(appCtx(env.alpha.ID), plain, created.Code)
	assert.NoError(t, err)
	assert.Empty(t, menu.RootMenuIDs)
	assert.False(t, menu.Features.Workflow)
}

// SEC-MENU-005：真库构造分组/资产树（资产域未落地，节点按菜单行出网、
// target 不投影）→ 分组保留、资产节点可见；空分组按 ADR-011 可见性规则
// 裁剪：菜单管理成员可见（可继续添加资产），普通成员被裁剪；结构损坏
// （跨应用父节点）返回 APP_MENU_INVALID
func TestSECMENU005PruningAndIntegrity(t *testing.T) {
	env := newMenuEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("裁剪应用"))
	assert.NoError(t, err)
	ctx := appCtx(env.alpha.ID)
	plain := env.createPlainMember(t, env.alpha, "menu-plain-prune")

	t.Run("资产节点出网且分组保留，空分组按成员可见性裁剪", func(t *testing.T) {
		emptyRoot := menuNodeFixture(0, "menu_empty", nil, model.MenuTypeGroup, 0)
		emptyRoot.AppID = created.ID
		root := menuNodeFixture(0, "menu_root", nil, model.MenuTypeGroup, 1024)
		root.AppID = created.ID
		// 两步写入让 root 先落库拿到自增 ID 供 form 引用
		env.insertMenuNode(t, ctx, &emptyRoot)
		env.insertMenuNode(t, ctx, &root)
		form := menuNodeFixture(0, "menu_form", ptrUint(root.ID), model.MenuTypeForm, 1024)
		form.AppID = created.ID
		env.insertMenuNode(t, ctx, &form)

		// 菜单管理成员（ADR-011）：空分组必须可见，否则无法向其中添加资产
		menu, err := env.menuSvc.GetMenu(ctx, env.alphaMember, created.Code)
		assert.NoError(t, err)
		assert.Equal(t, []string{"menu_empty", "menu_root"}, menu.RootMenuIDs)
		assert.Contains(t, menu.NodeMap, "menu_root")
		assert.Contains(t, menu.NodeMap, "menu_empty")
		assert.Contains(t, menu.NodeMap, "menu_form")
		// 资产域未落地：target 不投影（M2-资产-1 接入后映射资产公开编码）
		assert.Nil(t, menu.NodeMap["menu_form"].Target)

		// 普通成员：无可见后代的空分组被裁剪，分组与资产正常出网
		plainMenu, err := env.menuSvc.GetMenu(ctx, plain, created.Code)
		assert.NoError(t, err)
		assert.Equal(t, []string{"menu_root"}, plainMenu.RootMenuIDs)
		assert.Contains(t, plainMenu.NodeMap, "menu_root")
		assert.Contains(t, plainMenu.NodeMap, "menu_form")
		assert.NotContains(t, plainMenu.NodeMap, "menu_empty") // 空分组被裁剪
	})

	t.Run("跨应用父节点返回 APP_MENU_INVALID", func(t *testing.T) {
		// FK 只保证 parent ID 存在，不保证同应用（方案 §5.1：单列外键
		// 表达不了同应用约束）：把另一应用的节点作父，快照集合内父缺失，
		// 服务层结构校验须按完整性故障拦截
		other, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("另一应用"))
		assert.NoError(t, err)
		otherRoot := menuNodeFixture(0, "menu_other_root", nil, model.MenuTypeGroup, 1024)
		otherRoot.AppID = other.ID
		env.insertMenuNode(t, ctx, &otherRoot)

		cross := menuNodeFixture(0, "menu_cross", ptrUint(otherRoot.ID), model.MenuTypeGroup, 1024)
		cross.AppID = created.ID
		env.insertMenuNode(t, ctx, &cross)

		_, err = env.menuSvc.GetMenu(ctx, env.alphaMember, created.Code)
		assert.True(t, errors.Is(err, apperrors.ErrMenuInvalid))
	})
}

// SEC-MENU-006：软删节点（deleted_at 置位）从快照排除
func TestSECMENU006SoftDeletedEntryExcluded(t *testing.T) {
	env := newMenuEnv(t)

	created, err := env.appSvc.CreateBlank(appCtx(env.alpha.ID), env.alphaMember, blankReq("软删节点"))
	assert.NoError(t, err)
	ctx := appCtx(env.alpha.ID)

	root := menuNodeFixture(0, "menu_root2", nil, model.MenuTypeGroup, 1024)
	root.AppID = created.ID
	env.insertMenuNode(t, ctx, &root)
	form := menuNodeFixture(0, "menu_form2", ptrUint(root.ID), model.MenuTypeForm, 1024)
	form.AppID = created.ID
	env.insertMenuNode(t, ctx, &form)

	// 软删两个节点（先叶子后根，避免悬空父引用）：快照应不含任何节点
	assert.NoError(t, env.db.WithContext(ctx).
		Where("app_id = ? AND code = ?", created.ID, "menu_form2").
		Delete(&model.MenuNode{}).Error)
	assert.NoError(t, env.db.WithContext(ctx).
		Where("app_id = ? AND code = ?", created.ID, "menu_root2").
		Delete(&model.MenuNode{}).Error)

	snap, err := env.menuRepo.GetSnapshot(ctx, env.alpha.ID, created.Code)
	assert.NoError(t, err)
	assert.Empty(t, snap.Nodes)
}
