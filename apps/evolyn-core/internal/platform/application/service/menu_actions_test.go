package service

import (
	"context"
	"errors"
	"testing"

	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- ADR-011：对成员隐藏 / 按钮能力投影 / 节点管理 / 个人收藏 服务单测 ----

// menuAdminPerms 菜单管理员权限集（applications:* + forms:* + form-actions:*）
func menuAdminPerms() map[string]bool {
	return map[string]bool{
		"applications:get": true, "applications:list": true,
		"applications:create": true, "applications:patch": true,
		"applications:delete": true,
		"forms:create":        true, "forms:get": true, "forms:list": true,
		"forms:update": true, "forms:patch": true, "forms:delete": true,
		"form-actions:switch-type": true, "form-actions:copy-in-app": true,
		"form-actions:copy-cross-app": true, "form-actions:hide": true,
	}
}

// readOnlyPerms authenticated 基线（仅应用可读 + 提交）
func readOnlyPerms() map[string]bool {
	return map[string]bool{"applications:get": true, "form-records:create": true}
}

// recordingMenuRepo 在 fakeMenuRepo 上记录 UpdateEntryFields 写入，供移动/
// 改名/隐藏断言
type recordingMenuRepo struct {
	fakeMenuRepo
	updatedFields map[string]map[string]interface{} // entryCode → fields（按快照内编码）
}

func (f *recordingMenuRepo) UpdateEntryFields(ctx context.Context, applicationID, entryID uint, fields map[string]interface{}) error {
	if f.updatedFields == nil {
		f.updatedFields = map[string]map[string]interface{}{}
	}
	for i := range f.snapshots {
		for _, entry := range f.snapshots[i].Entries {
			if entry.ID == entryID {
				f.updatedFields[entry.Code] = fields
			}
		}
	}
	return nil
}

func hiddenMenuSnapshot() *repository.MenuSnapshot {
	snap := emptySnapshot("app_a")
	group := menuEntryFixture(1, "menu_group", nil, model.MenuEntryTypeGroup, 1024)
	form := menuEntryFixture(2, "menu_form", ptrUint(1), model.MenuEntryTypeForm, 1024)
	form.Hidden = true // 对成员隐藏：分组唯一后代被隐藏后分组应随之裁剪
	dash := menuEntryFixture(3, "menu_dash", nil, model.MenuEntryTypeDashboard, 2048)
	snap.Entries = []model.MenuEntry{group, form, dash}
	return snap
}

func TestMenuHiddenVisibility(t *testing.T) {
	// 只读成员：隐藏节点按不存在裁剪，仅含隐藏后代的分组随之裁剪
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()}}
	svc := newMenuTestService(repo, readOnlyPerms())
	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.NotContains(t, menu.EntryMap, "menu_form")
	assert.NotContains(t, menu.EntryMap, "menu_group") // 无可见后代
	assert.Contains(t, menu.EntryMap, "menu_dash")

	// 菜单管理成员：隐藏节点保持可见（否则无法恢复显示）
	svcAdmin := newMenuTestService(repo, menuAdminPerms())
	menuAdmin, err := svcAdmin.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Contains(t, menuAdmin.EntryMap, "menu_form")
	assert.Contains(t, menuAdmin.EntryMap, "menu_group")

	// 收藏状态出网：当前成员已收藏节点 Favorited=true
	repoFav := &fakeMenuRepo{
		snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()},
		favorites: map[uint]bool{3: true},
	}
	svcFav := newMenuTestService(repoFav, readOnlyPerms())
	menuFav, err := svcFav.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.True(t, menuFav.EntryMap["menu_dash"].Favorited)
	assert.False(t, menuFav.EntryMap["menu_form"].Capabilities.Actions.Hide) // 只读成员无动作
}

func TestMenuUpdateEntryRename(t *testing.T) {
	snap := hiddenMenuSnapshot()
	repo := &recordingMenuRepo{fakeMenuRepo: fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}}
	svc := newMenuTestService(repo, menuAdminPerms())

	// 分组改名成功：修订号推进 + name 写入
	out, err := svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_group",
		&model.UpdateMenuEntryRequest{Name: strPtr("新分组"), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Equal(t, "menu_group", out.EntryID)
	assert.Equal(t, int64(2), out.MenuRevision)
	assert.Equal(t, "新分组", repo.updatedFields["menu_group"]["name"])

	// 资产节点改名拒绝：名称以资产域为事实源（须经 PATCH /forms/:code）
	resetRevision(snap, 1)
	_, err = svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuEntryRequest{Name: strPtr("旁路改名"), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrMenuEntryRenameForbidden))
}

func TestMenuUpdateEntryHidden(t *testing.T) {
	snap := hiddenMenuSnapshot()
	repo := &recordingMenuRepo{fakeMenuRepo: fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}}
	svc := newMenuTestService(repo, menuAdminPerms())

	// 恢复显示（hidden=false）成功
	_, err := svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuEntryRequest{Hidden: boolPtr(false), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Equal(t, false, repo.updatedFields["menu_form"]["hidden"])

	// 分组节点不支持对成员隐藏
	resetRevision(snap, 1)
	_, err = svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_group",
		&model.UpdateMenuEntryRequest{Hidden: boolPtr(true), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrMenuHiddenInvalid))

	// 缺 form-actions:hide 动作键：即使持 applications:patch 也拒绝
	perms := menuAdminPerms()
	delete(perms, "form-actions:hide")
	svcNoHide := newMenuTestService(repo, perms)
	resetRevision(snap, 1)
	_, err = svcNoHide.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_dash",
		&model.UpdateMenuEntryRequest{Hidden: boolPtr(true), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrForbidden))
}

func TestMenuUpdateEntryMove(t *testing.T) {
	snap := hiddenMenuSnapshot()
	// 追加第二层分组，构造「分组移动到自身后代」用例
	nested := menuEntryFixture(4, "menu_nested", ptrUint(1), model.MenuEntryTypeGroup, 3072)
	snap.Entries = append(snap.Entries, nested)
	repo := &recordingMenuRepo{fakeMenuRepo: fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}}
	svc := newMenuTestService(repo, menuAdminPerms())

	// 表单节点移动到根级（空串父编码）
	_, err := svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuEntryRequest{ParentEntryCode: strPtr(""), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Contains(t, repo.updatedFields["menu_form"], "parent_entry_id")
	assert.Nil(t, repo.updatedFields["menu_form"]["parent_entry_id"])

	// 分组移动到自身后代：APP_MENU_MOVE_INVALID
	resetRevision(snap, 1)
	_, err = svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_group",
		&model.UpdateMenuEntryRequest{ParentEntryCode: strPtr("menu_nested"), BaseMenuRevision: 1})
	assert.True(t, errors.Is(err, apperrors.ErrMenuMoveInvalid))

	// 分组移动到根级：合法（两层结构的正常形态）
	resetRevision(snap, 1)
	_, err = svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_nested",
		&model.UpdateMenuEntryRequest{ParentEntryCode: strPtr(""), BaseMenuRevision: 1})
	assert.NoError(t, err)
	assert.Contains(t, repo.updatedFields["menu_nested"], "parent_entry_id")
}

func TestMenuUpdateEntryRevisionConflict(t *testing.T) {
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()}}
	svc := newMenuTestService(repo, menuAdminPerms())
	// baseMenuRevision 与服务端不一致：APP_MENU_VERSION_CONFLICT
	_, err := svc.UpdateEntry(alphaCtx(), alphaMember(), "app_a", "menu_form",
		&model.UpdateMenuEntryRequest{Hidden: boolPtr(true), BaseMenuRevision: 99})
	assert.True(t, errors.Is(err, apperrors.ErrMenuVersionConflict))
}

func TestMenuFavorites(t *testing.T) {
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": hiddenMenuSnapshot()}}
	svc := newMenuTestService(repo, readOnlyPerms())

	// 收藏成功（普通成员即可，个人状态动作）
	out, err := svc.AddFavorite(alphaCtx(), alphaMember(), "app_a", "menu_dash")
	assert.NoError(t, err)
	assert.Equal(t, "menu_dash", out.EntryID)
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

func strPtr(v string) *string { return &v }
func boolPtr(v bool) *bool    { return &v }

// resetRevision 重置快照修订号：fakeMenuRepo.BumpMenuRevisionFrom 会推进
// 共享快照，同用例多次调用 UpdateEntry 时需回传一致的 baseMenuRevision
func resetRevision(snap *repository.MenuSnapshot, revision int64) {
	snap.MenuRevision = revision
}

var _ = gorm.ErrRecordNotFound // 保留与既有桩一致的导入引用
