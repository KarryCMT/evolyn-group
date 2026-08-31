// 菜单权限裁剪单测（表单权限 P1，设计 §8.2 菜单快照裁剪执行点）：
// FormPermissionDirectory 注入后，成员侧 form 节点按「入口判定 view ∨ add」
// 二次裁剪；未命中（S5 含禁用组收口）的表单隐藏，空分组随既有裁剪规则收敛；
// 端口未注入时保持既有可见性行为。
package service

import (
	"context"
	"testing"

	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/repository"

	"github.com/stretchr/testify/assert"
)

// fakeFormPermissionDirectory 表单权限裁剪端口桩：直出「成员可入口表单集」
type fakeFormPermissionDirectory struct {
	visible map[uint]bool
}

func (f fakeFormPermissionDirectory) VisibleFormIDs(ctx context.Context, memberID uint, formIDs []uint) (map[uint]bool, error) {
	return f.visible, nil
}

// 仅 add 成员（仅录入场景）：表单对成员可见（入口判定 = view ∨ add）；
// 未命中成员：form 节点隐藏，空分组随之裁剪；菜单管理成员的空分组保留行为
// 不变（分组可见性由既有规则承载）
func TestMenuFormPermissionTrimming(t *testing.T) {
	root := menuEntryFixture(1, "menu_root", nil, model.MenuEntryTypeGroup, 1024)
	form := menuEntryFixture(2, "menu_form", ptrUint(1), model.MenuEntryTypeForm, 1024)
	form.TargetID = ptrUint(10) // 目标表单内部 ID
	snap := emptySnapshot("app_a")
	snap.Entries = []model.MenuEntry{root, form}
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}

	svc := newMenuTestService(repo, map[string]bool{"applications:get": true}).(*menuService)
	svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{
		10: {Code: "form_0123456789abcdef", FormType: "standard"},
	}})

	// 端口未注入：保持旧行为（仅存在性裁剪）
	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Contains(t, menu.EntryMap, "menu_form")

	// 端口注入，成员仅 add：入口判定放行
	svc.UseFormPermissionDirectory(fakeFormPermissionDirectory{visible: map[uint]bool{10: true}})
	menu, err = svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Contains(t, menu.EntryMap, "menu_form")
	assert.NotNil(t, menu.EntryMap["menu_form"].Target)

	// 端口注入，成员未命中任何启用组（S5 收口）：form 节点隐藏，空分组裁剪
	svc.UseFormPermissionDirectory(fakeFormPermissionDirectory{visible: map[uint]bool{}})
	menu, err = svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.NotContains(t, menu.EntryMap, "menu_form")
	assert.Empty(t, menu.RootEntryIDs, "无可见后代的分组随裁剪规则收敛")
}
