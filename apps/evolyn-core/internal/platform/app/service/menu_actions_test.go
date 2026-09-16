package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	apperrors "evolyn/internal/platform/app"
	"evolyn/internal/platform/app/model"
	"evolyn/internal/platform/app/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- ADR-011：对成员隐藏 / 按钮能力投影 / 节点管理 / 个人收藏 服务单测 ----

// menuAdminPerms 菜单管理员权限集（apps:* + forms:* + form-actions:*）
func menuAdminPerms() map[string]bool {
	return map[string]bool{
		"apps:get": true, "apps:list": true,
		"apps:create": true, "apps:patch": true,
		"apps:delete":  true,
		"forms:create": true, "forms:get": true, "forms:list": true,
		"forms:update": true, "forms:patch": true, "forms:delete": true,
		"form-actions:switch-type": true, "form-actions:copy-in-app": true,
		"form-actions:copy-cross-app": true, "form-actions:hide": true,
	}
}

// readOnlyPerms authenticated 基线（仅应用可读 + 提交）
func readOnlyPerms() map[string]bool {
	return map[string]bool{"apps:get": true, "form-records:create": true}
}

// recordingMenuRepo 在 fakeMenuRepo 上记录 UpdateNodeFields 写入，供移动/
// 改名/隐藏断言
type recordingMenuRepo struct {
	fakeMenuRepo
	updatedFields map[string]map[string]interface{} // menuCode → fields（按快照内编码）
}

func (f *recordingMenuRepo) UpdateNodeFields(ctx context.Context, appID, nodeID uint, fields map[string]interface{}) error {
	if f.updatedFields == nil {
		f.updatedFields = map[string]map[string]interface{}{}
	}
	for i := range f.snapshots {
		for _, node := range f.snapshots[i].Nodes {
			if node.ID == nodeID {
				f.updatedFields[node.Code] = fields
			}
		}
	}
	return nil
}

func hiddenMenuSnapshot() *repository.MenuSnapshot {
	snap := emptySnapshot("app_a")
	group := menuNodeFixture(1, "menu_group", nil, model.MenuTypeGroup, 1024)
	form := menuNodeFixture(2, "menu_form", ptrUint(1), model.MenuTypeForm, 1024)
	form.Hidden = true // 对成员隐藏：分组唯一后代被隐藏后分组应随之裁剪
	dash := menuNodeFixture(3, "menu_dash", nil, model.MenuTypeDashboard, 2048)
	snap.Nodes = []model.MenuNode{group, form, dash}
	return snap
}

func TestMenuHiddenVisibility(t *testing.T) {
	// 只读成员：隐藏节点按不存在裁剪，仅含隐藏后代的分组随之裁剪
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()}}
	svc := newMenuTestService(repo, readOnlyPerms())
	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.NotContains(t, menu.NodeMap, "menu_form")
	assert.NotContains(t, menu.NodeMap, "menu_group") // 无可见后代
	assert.Contains(t, menu.NodeMap, "menu_dash")

	// 菜单管理成员：隐藏节点保持可见（否则无法恢复显示）
	svcAdmin := newMenuTestService(repo, menuAdminPerms())
	menuAdmin, err := svcAdmin.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Contains(t, menuAdmin.NodeMap, "menu_form")
	assert.Contains(t, menuAdmin.NodeMap, "menu_group")

	// 收藏状态出网：当前成员已收藏节点 Favorited=true
	repoFav := &fakeMenuRepo{
		snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()},
		favorites: map[uint]bool{3: true},
	}
	svcFav := newMenuTestService(repoFav, readOnlyPerms())
	menuFav, err := svcFav.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.True(t, menuFav.NodeMap["menu_dash"].Favorited)
	assert.False(t, menuFav.NodeMap["menu_form"].Capabilities.Actions.Hide) // 只读成员无动作
}

func TestMenuUpdateNodeRename(t *testing.T) {
	snap := hiddenMenuSnapshot()
	repo := &recordingMenuRepo{fakeMenuRepo: fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}}
	svc := newMenuTestService(repo, menuAdminPerms())

	// 分组改名成功：修订号推进 + name 写入
	out, err := svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_group",
		&model.UpdateMenuNodeRequest{Name: strPtr("新分组"), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Equal(t, "menu_group", out.MenuID)
	assert.Equal(t, int64(2), out.MenuRevision)
	assert.Equal(t, "新分组", repo.updatedFields["menu_group"]["name"])

	// 资产节点改名拒绝：名称以资产域为事实源（须经 PATCH /forms/:code）
	resetRevision(snap, 1)
	_, err = svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuNodeRequest{Name: strPtr("旁路改名"), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrMenuNodeRenameForbidden))
}

func TestMenuUpdateNodeHidden(t *testing.T) {
	snap := hiddenMenuSnapshot()
	repo := &recordingMenuRepo{fakeMenuRepo: fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}}
	svc := newMenuTestService(repo, menuAdminPerms())

	// 恢复显示（hidden=false）成功
	_, err := svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuNodeRequest{Hidden: boolPtr(false), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Equal(t, false, repo.updatedFields["menu_form"]["hidden"])

	// 分组节点不支持对成员隐藏
	resetRevision(snap, 1)
	_, err = svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_group",
		&model.UpdateMenuNodeRequest{Hidden: boolPtr(true), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrMenuHiddenInvalid))

	// 缺 form-actions:hide 动作键：即使持 apps:patch 也拒绝
	perms := menuAdminPerms()
	delete(perms, "form-actions:hide")
	svcNoHide := newMenuTestService(repo, perms)
	resetRevision(snap, 1)
	_, err = svcNoHide.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_dash",
		&model.UpdateMenuNodeRequest{Hidden: boolPtr(true), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrForbidden))
}

func TestMenuUpdateNodeMove(t *testing.T) {
	snap := hiddenMenuSnapshot()
	// 追加第二层分组，构造「分组移动到自身后代」用例
	nested := menuNodeFixture(4, "menu_nested", ptrUint(1), model.MenuTypeGroup, 3072)
	snap.Nodes = append(snap.Nodes, nested)
	repo := &recordingMenuRepo{fakeMenuRepo: fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}}
	svc := newMenuTestService(repo, menuAdminPerms())

	// 表单节点移动到根级（空串父编码）
	_, err := svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuNodeRequest{ParentMenuCode: strPtr(""), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Contains(t, repo.updatedFields["menu_form"], "parent_menu_id")
	assert.Nil(t, repo.updatedFields["menu_form"]["parent_menu_id"])

	// 分组移动到自身后代：APP_MENU_MOVE_INVALID
	resetRevision(snap, 1)
	_, err = svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_group",
		&model.UpdateMenuNodeRequest{ParentMenuCode: strPtr("menu_nested"), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrMenuMoveInvalid))

	// 分组移动到根级：合法（两层结构的正常形态）
	resetRevision(snap, 1)
	_, err = svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_nested",
		&model.UpdateMenuNodeRequest{ParentMenuCode: strPtr(""), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Contains(t, repo.updatedFields["menu_nested"], "parent_menu_id")
}

func TestMenuUpdateNodeRevisionConflict(t *testing.T) {
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()}}
	svc := newMenuTestService(repo, menuAdminPerms())
	// baseMenuRevision 与服务端不一致：APP_MENU_VERSION_CONFLICT
	_, err := svc.UpdateNode(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuNodeRequest{Hidden: boolPtr(true), BaseMenuRevision: 99})
	assert.True(t, errors.Is(err, apperrors.ErrMenuVersionConflict))
}

func TestMenuFavorites(t *testing.T) {
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()}}
	svc := newMenuTestService(repo, readOnlyPerms())

	// 收藏成功（普通成员即可，个人状态动作；dashboard 可见资产节点）
	out, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_dash")
	assert.NoError(t, err)
	assert.Equal(t, "menu_dash", out.MenuID)
	assert.True(t, out.Favorited)

	// 节点不存在：APP_MENU_FAVORITE_INVALID
	_, err = svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_missing")
	assert.True(t, errors.Is(err, apperrors.ErrMenuFavoriteInvalid))

	// 应用不存在：APP_NOT_FOUND
	_, err = svc.AddFavorite(alphaCtx(), alphaMember(), "app_missing", "menu_dash")
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))

	// 取消收藏幂等
	out, err = svc.RemoveFavorite(alphaCtx(), alphaMember(), "menu_dash")
	assert.NoError(t, err)
	assert.False(t, out.Favorited)
}

// TestMenuFavoriteCanFavoritePolicy P1 统一收藏策略：分组节点、隐藏节点
// （非菜单管理成员）、无 apps:get、归档应用、表单软删与表单入口权限失效
// 均拒绝收藏，且错误不泄露隐藏节点细节（统一 APP_MENU_FAVORITE_INVALID）
func TestMenuFavoriteCanFavoritePolicy(t *testing.T) {
	newRepo := func() *fakeMenuRepo {
		return &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()}}
	}

	t.Run("分组节点不可收藏", func(t *testing.T) {
		svc := newMenuTestService(newRepo(), readOnlyPerms())
		_, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_group")
		assert.True(t, errors.Is(err, apperrors.ErrMenuFavoriteInvalid))
	})

	t.Run("隐藏节点对普通成员不可收藏", func(t *testing.T) {
		svc := newMenuTestService(newRepo(), readOnlyPerms())
		_, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_form")
		assert.True(t, errors.Is(err, apperrors.ErrMenuFavoriteInvalid))
	})

	t.Run("隐藏节点对菜单管理成员可收藏（恢复显示口径同 GetMenu）", func(t *testing.T) {
		svc := newMenuTestService(newRepo(), menuAdminPerms())
		out, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_form")
		assert.NoError(t, err)
		assert.True(t, out.Favorited)
	})

	t.Run("无 apps:get 统一 APP_NOT_FOUND（不泄露应用存在性）", func(t *testing.T) {
		svc := newMenuTestService(newRepo(), map[string]bool{"form-records:create": true})
		_, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_dash")
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("归档应用不可收藏：APP_STATUS_INVALID", func(t *testing.T) {
		repo := newRepo()
		repo.snapshots["app_a"].Status = model.AppStatusArchived
		svc := newMenuTestService(repo, readOnlyPerms())
		_, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_dash")
		assert.True(t, errors.Is(err, apperrors.ErrStatusInvalid))
	})

	t.Run("表单软删（目录存在集未命中）不可收藏", func(t *testing.T) {
		repo := newRepo()
		svc := newMenuTestService(repo, readOnlyPerms()).(*menuService)
		svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{}})
		_, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_form")
		assert.True(t, errors.Is(err, apperrors.ErrMenuFavoriteInvalid))
	})

	t.Run("表单入口权限失效（view ∨ add 未命中）不可收藏", func(t *testing.T) {
		repo := newRepo()
		svc := newMenuTestService(repo, menuAdminPerms()).(*menuService)
		svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{
			902: {Code: "form_0123456789abcdef", FormType: "standard"},
		}})
		svc.UseFormPermissionDirectory(fakeFormPermissionDirectory{visible: map[uint]bool{}})
		_, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_form")
		assert.True(t, errors.Is(err, apperrors.ErrMenuFavoriteInvalid))
	})
}

// favoriteRowFixture 构造收藏列表行（含应用/节点投影）
func favoriteRowFixture(favoriteID uint, appCode, menuType string, hidden bool, targetID *uint, at time.Time) repository.FavoriteRow {
	row := repository.FavoriteRow{
		FavoriteID:         favoriteID,
		CreatedAt:          at,
		AppCode:            appCode,
		AppName:            "应用" + appCode,
		AppStatus:          model.AppStatusActive,
		AppProvisionStatus: model.ProvisionStatusReady,
		MenuID:             100 + favoriteID,
		MenuCode:           fmt.Sprintf("menu_row_%d", favoriteID),
		MenuType:           menuType,
		MenuName:           fmt.Sprintf("节点%d", favoriteID),
		Hidden:             hidden,
		TargetID:           targetID,
	}
	if menuType != model.MenuTypeGroup {
		tt := menuType
		row.TargetType = &tt
	}
	return row
}

// listFavoritePerms 收藏列表所需最小权限集（authenticated 基线 + list）
func listFavoritePerms(extra ...string) map[string]bool {
	perms := map[string]bool{"apps:get": true, "menu-favorites:list": true}
	for _, key := range extra {
		perms[key] = true
	}
	return perms
}

// TestMenuListFavorites P2 我的收藏列表：读侧复用菜单同口径可见性裁剪
// （失效记录只过滤不删除）+ 游标分页
func TestMenuListFavorites(t *testing.T) {
	base := time.Date(2026, 9, 15, 10, 0, 0, 0, time.Local)
	formTarget := ptrUint(9001)

	buildRows := func() []repository.FavoriteRow {
		return []repository.FavoriteRow{
			// 5. 归档应用：保留行但读侧过滤
			func() repository.FavoriteRow {
				row := favoriteRowFixture(5, "app_archived", model.MenuTypeDashboard, false, nil, base.Add(4*time.Minute))
				row.AppStatus = model.AppStatusArchived
				return row
			}(),
			// 4. 可见表单节点：目录命中 + 权限命中 → 展示（含 target 投影）
			favoriteRowFixture(4, "app_a", model.MenuTypeForm, false, formTarget, base.Add(3*time.Minute)),
			// 3. 隐藏节点：普通成员过滤，菜单管理成员展示
			favoriteRowFixture(3, "app_a", model.MenuTypeDashboard, true, nil, base.Add(2*time.Minute)),
			// 2. 表单入口权限失效：过滤
			favoriteRowFixture(2, "app_b", model.MenuTypeForm, false, ptrUint(9002), base.Add(1*time.Minute)),
			// 1. 正常仪表盘节点
			favoriteRowFixture(1, "app_c", model.MenuTypeDashboard, false, nil, base),
		}
	}

	t.Run("默认按时间倒序 + 可见性过滤", func(t *testing.T) {
		repo := &fakeMenuRepo{favoriteRows: buildRows()}
		svc := newMenuTestService(repo, listFavoritePerms()).(*menuService)
		svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{
			9001: {Code: "form_visible00001", FormType: "workflow"},
			9002: {Code: "form_denied000001", FormType: "standard"},
		}})
		svc.UseFormPermissionDirectory(fakeFormPermissionDirectory{visible: map[uint]bool{9001: true}})

		page, err := svc.ListFavorites(alphaCtx(), alphaMember(), model.ListMenuFavoritesQuery{})
		assert.NoError(t, err)
		// 归档应用/权限失效被过滤；隐藏节点对普通成员过滤
		ids := make([]string, 0, len(page.Items))
		for _, item := range page.Items {
			ids = append(ids, item.Node.MenuID)
		}
		assert.Equal(t, []string{"menu_row_4", "menu_row_1"}, ids)
		// 应用与节点投影 + target 出网
		assert.Equal(t, "app_a", page.Items[0].App.Code)
		assert.Equal(t, "应用app_a", page.Items[0].App.Name)
		assert.Equal(t, model.MenuTypeForm, page.Items[0].Node.Type)
		assert.Equal(t, "form_visible00001", page.Items[0].Node.Target.Code)
		assert.Equal(t, "workflow", page.Items[0].Node.Target.FormType)
		assert.Empty(t, page.NextCursor) // 行集耗尽（行数 < 扫描批量）
	})

	t.Run("菜单管理成员可见自己收藏的隐藏节点", func(t *testing.T) {
		repo := &fakeMenuRepo{favoriteRows: buildRows()}
		svc := newMenuTestService(repo, listFavoritePerms("apps:patch")).(*menuService)
		svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{
			9001: {Code: "form_visible00001", FormType: "workflow"},
			9002: {Code: "form_denied000001", FormType: "standard"},
		}})
		svc.UseFormPermissionDirectory(fakeFormPermissionDirectory{visible: map[uint]bool{9001: true}})

		page, err := svc.ListFavorites(alphaCtx(), alphaMember(), model.ListMenuFavoritesQuery{})
		assert.NoError(t, err)
		ids := make([]string, 0, len(page.Items))
		for _, item := range page.Items {
			ids = append(ids, item.Node.MenuID)
		}
		assert.Equal(t, []string{"menu_row_4", "menu_row_3", "menu_row_1"}, ids)
	})

	t.Run("游标分页：limit 截断 + 续拉收尾", func(t *testing.T) {
		repo := &fakeMenuRepo{favoriteRows: buildRows()}
		svc := newMenuTestService(repo, listFavoritePerms()).(*menuService)
		svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{
			9001: {Code: "form_visible00001", FormType: "workflow"},
			9002: {Code: "form_denied000001", FormType: "standard"},
		}})
		svc.UseFormPermissionDirectory(fakeFormPermissionDirectory{visible: map[uint]bool{9001: true}})

		// 首页：limit=1 只取最新一条，下发游标（锚定最后一条已返回条目）
		first, err := svc.ListFavorites(alphaCtx(), alphaMember(), model.ListMenuFavoritesQuery{Limit: 1})
		assert.NoError(t, err)
		assert.Len(t, first.Items, 1)
		assert.Equal(t, "menu_row_4", first.Items[0].Node.MenuID)
		assert.NotEmpty(t, first.NextCursor)

		// 续拉：游标之后剩余可见条目，无更多数据
		second, err := svc.ListFavorites(alphaCtx(), alphaMember(), model.ListMenuFavoritesQuery{Limit: 1, Cursor: first.NextCursor})
		assert.NoError(t, err)
		assert.Len(t, second.Items, 1)
		assert.Equal(t, "menu_row_1", second.Items[0].Node.MenuID)
		assert.Empty(t, second.NextCursor)
	})

	t.Run("无 menu-favorites:list 返回 FORBIDDEN", func(t *testing.T) {
		repo := &fakeMenuRepo{favoriteRows: buildRows()}
		svc := newMenuTestService(repo, map[string]bool{"apps:get": true})
		_, err := svc.ListFavorites(alphaCtx(), alphaMember(), model.ListMenuFavoritesQuery{})
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))
	})

	t.Run("非法游标返回 APP_CURSOR_INVALID", func(t *testing.T) {
		repo := &fakeMenuRepo{favoriteRows: buildRows()}
		svc := newMenuTestService(repo, listFavoritePerms())
		_, err := svc.ListFavorites(alphaCtx(), alphaMember(), model.ListMenuFavoritesQuery{Cursor: "not-a-cursor"})
		assert.True(t, errors.Is(err, apperrors.ErrCursorInvalid))
	})
}

func strPtr(v string) *string { return &v }
func boolPtr(v bool) *bool    { return &v }

// resetRevision 重置快照修订号：fakeMenuRepo.BumpMenuRevisionFrom 会推进
// 共享快照，同用例多次调用 UpdateNode 时需回传一致的 baseMenuRevision
func resetRevision(snap *repository.MenuSnapshot, revision int64) {
	snap.MenuRevision = revision
}

var _ = gorm.ErrRecordNotFound // 保留与既有桩一致的导入引用
