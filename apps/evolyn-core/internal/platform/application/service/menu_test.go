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

// ---- M2-菜单-1 Service 单元测试（真实 PostgreSQL 链路见 sec_menu_integration_test.go）----

// fakeMenuRepo 菜单仓储桩：按编码返回可注入快照
type fakeMenuRepo struct {
	snapshots map[string]*repository.MenuSnapshot
}

func (f *fakeMenuRepo) GetSnapshot(ctx context.Context, tenantID uint, code string) (*repository.MenuSnapshot, error) {
	snap, ok := f.snapshots[code]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return snap, nil
}

func (f *fakeMenuRepo) Migrate() error { return nil }

// M2-资产-1 写路径桩（菜单维护用例见 menu_maintenance_test.go；只读用例不触达）
func (f *fakeMenuRepo) CreateFormEntry(ctx context.Context, entry *model.MenuEntry) (*model.MenuEntry, error) {
	return entry, nil
}
func (f *fakeMenuRepo) UpdateNameByFormTarget(ctx context.Context, applicationID, formID uint, name string) error {
	return nil
}
func (f *fakeMenuRepo) SoftDeleteByFormTarget(ctx context.Context, applicationID, formID uint) error {
	return nil
}
func (f *fakeMenuRepo) MaxSortOrder(ctx context.Context, applicationID uint, parentEntryID *uint) (int64, error) {
	return 0, nil
}
func (f *fakeMenuRepo) FindByCode(ctx context.Context, applicationID uint, code string) (*model.MenuEntry, error) {
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeMenuRepo) BumpMenuRevision(ctx context.Context, applicationID uint) error {
	return nil
}

// menuEntryFixture 构造菜单节点（测试内联便捷函数）
func menuEntryFixture(id uint, code string, parent *uint, entryType string, sortOrder int64) model.MenuEntry {
	entry := model.MenuEntry{
		ID: id, Code: code, ParentEntryID: parent, EntryType: entryType,
		Name: code, SortOrder: sortOrder, ApplicationID: 1,
	}
	if entryType != model.MenuEntryTypeGroup {
		// 非分组节点必须带资产引用（CHECK 约束）；group 的 TargetType 保持 NULL
		targetType := entryType
		entry.TargetType = &targetType
		targetID := uint(900 + id)
		entry.TargetID = &targetID
	}
	return entry
}

func ptrUint(v uint) *uint { return &v }

func newMenuTestService(repo repository.MenuRepository, perms map[string]bool) ApplicationMenuService {
	return NewMenuService(repo, fakeAccess{perms: perms})
}

// emptySnapshot 空菜单快照（应用就绪、无节点——M2-菜单-1 的常态）
func emptySnapshot(code string) *repository.MenuSnapshot {
	return &repository.MenuSnapshot{
		ApplicationID: 1, ApplicationCode: code,
		Status: model.ApplicationStatusActive, ProvisionStatus: model.ProvisionStatusReady,
		MenuRevision: 1,
	}
}

func TestMenuGetEmpty(t *testing.T) {
	// 空应用菜单：空树是合法结果（200 语义），rootEntryIds/entryMap 为空
	// 集合而非 null；menuRevision 原样出网供后续管理接口乐观并发
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": emptySnapshot("app_a")}}
	svc := newMenuTestService(repo, fullPerms())

	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Equal(t, "app_a", menu.ApplicationCode)
	assert.Equal(t, int64(1), menu.MenuRevision)
	assert.NotNil(t, menu.RootEntryIDs)
	assert.NotNil(t, menu.EntryMap)
	assert.Empty(t, menu.RootEntryIDs)
	assert.Empty(t, menu.EntryMap)
	assert.False(t, menu.Features.Workflow) // 流程引擎未接入恒 false
}

func TestMenuGetNotFound(t *testing.T) {
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{}}

	t.Run("应用不存在 NotFound", func(t *testing.T) {
		svc := newMenuTestService(repo, fullPerms())
		_, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_notexist")
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("无读取权限统一 NotFound（不泄露应用存在性，§6.1）", func(t *testing.T) {
		repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": emptySnapshot("app_a")}}
		svc := newMenuTestService(repo, map[string]bool{"applications:patch": true})
		_, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})
}

func TestMenuSnapshotVisibilityAndOrdering(t *testing.T) {
	// 资产域落地前资产节点默认出网（target 为 null），构造「分组 → 资产」
	// 树验证裁剪与排序（不可见资产的裁剪回归见 TestMenuAssetInvisiblePruning）
	emptyRoot := menuEntryFixture(1, "menu_empty", nil, model.MenuEntryTypeGroup, 0)
	root2 := menuEntryFixture(2, "menu_root2", nil, model.MenuEntryTypeGroup, 1024)
	root3 := menuEntryFixture(3, "menu_root3", nil, model.MenuEntryTypeGroup, 512)
	asset1 := menuEntryFixture(4, "menu_form1", ptrUint(2), model.MenuEntryTypeForm, 2048)
	child := menuEntryFixture(5, "menu_child", ptrUint(2), model.MenuEntryTypeGroup, 1024)
	asset2 := menuEntryFixture(6, "menu_dash2", ptrUint(5), model.MenuEntryTypeDashboard, 1024)
	asset3 := menuEntryFixture(7, "menu_form3", ptrUint(3), model.MenuEntryTypeForm, 1024)

	snap := emptySnapshot("app_a")
	snap.Entries = []model.MenuEntry{emptyRoot, root2, root3, asset1, child, asset2, asset3}
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
	svc := newMenuTestService(repo, fullPerms())

	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	// 根顺序：sortOrder ASC（512 先于 1024）；空分组被裁剪不出现
	assert.Equal(t, []string{"menu_root3", "menu_root2"}, menu.RootEntryIDs)
	// 空分组与不可见节点不进 entryMap；其余节点齐备且父子编码正确
	assert.NotContains(t, menu.EntryMap, "menu_empty")
	assert.Contains(t, menu.EntryMap, "menu_form1")
	assert.Contains(t, menu.EntryMap, "menu_child")
	assert.Contains(t, menu.EntryMap, "menu_dash2")
	assert.Contains(t, menu.EntryMap, "menu_form3")
	assert.Equal(t, "menu_root2", *menu.EntryMap["menu_form1"].ParentEntryID)
	assert.Equal(t, "menu_root3", *menu.EntryMap["menu_form3"].ParentEntryID)
	assert.Nil(t, menu.EntryMap["menu_root2"].ParentEntryID)
	// 资产节点出网但资产域未落地：target 投影为 null（公开编码待 M2-资产-1）
	assert.Nil(t, menu.EntryMap["menu_form1"].Target)
}

func TestMenuAssetInvisiblePruning(t *testing.T) {
	// 资产级授权落地后的裁剪回归：资产不可见（目录注入空存在集模拟）时
	// 排除其菜单节点，且没有可见后代的分组整体裁剪（方案 §6.3）
	root := menuEntryFixture(1, "menu_root", nil, model.MenuEntryTypeGroup, 1024)
	form := menuEntryFixture(2, "menu_form", ptrUint(1), model.MenuEntryTypeForm, 1024)
	snap := emptySnapshot("app_a")
	snap.Entries = []model.MenuEntry{root, form}

	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
	svc := newMenuTestService(repo, fullPerms()).(*menuService)
	svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]bool{}})

	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Empty(t, menu.RootEntryIDs)
	assert.Empty(t, menu.EntryMap)
}

// fakeFormDirectory 表单目录端口桩（M2-资产-1）：existing 即存在集
type fakeFormDirectory struct {
	existing map[uint]bool
}

func (f fakeFormDirectory) ExistingFormIDs(ctx context.Context, ids []uint) (map[uint]bool, error) {
	return f.existing, nil
}

func TestMenuIntegrityFailures(t *testing.T) {
	cases := []struct {
		name    string
		entries []model.MenuEntry
	}{
		{"孤儿节点（父引用不存在）", []model.MenuEntry{
			menuEntryFixture(1, "menu_orphan", ptrUint(99), model.MenuEntryTypeGroup, 1024),
		}},
		{"父节点非分组", []model.MenuEntry{
			menuEntryFixture(1, "menu_form", nil, model.MenuEntryTypeForm, 1024),
			menuEntryFixture(2, "menu_child", ptrUint(1), model.MenuEntryTypeGroup, 1024),
		}},
		{"父链循环", []model.MenuEntry{
			menuEntryFixture(1, "menu_a", ptrUint(2), model.MenuEntryTypeGroup, 1024),
			menuEntryFixture(2, "menu_b", ptrUint(1), model.MenuEntryTypeGroup, 1024),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := emptySnapshot("app_a")
			snap.Entries = tc.entries
			repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
			svc := newMenuTestService(repo, fullPerms())

			_, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
			// 服务端数据完整性故障返回 APP_MENU_INVALID（500），非 409
			assert.True(t, errors.Is(err, apperrors.ErrMenuInvalid))
		})
	}
}

func TestMenuCapabilities(t *testing.T) {
	// capabilities 口径对齐应用域 detailFor：manage/move=patch∩可编辑，
	// delete=delete∩非初始化中，favorite 恒 false（资产节点默认出网）
	form := menuEntryFixture(2, "menu_form", nil, model.MenuEntryTypeForm, 1024)
	cases := []struct {
		name   string
		perms  map[string]bool
		status string
		want   model.MenuEntryCapabilities
	}{
		{"全量权限 + active", fullPerms(), model.ApplicationStatusActive,
			model.MenuEntryCapabilities{View: true, Manage: true, Move: true, Delete: true}},
		{"只读权限（authenticated 基线）", map[string]bool{"applications:get": true}, model.ApplicationStatusActive,
			model.MenuEntryCapabilities{View: true}},
		// 归档态禁止菜单管理（方案 §8.2）：节点级 manage/move/delete 全关，
		// 与应用级 Delete（归档可删应用本身）口径不同
		{"全量权限 + 归档（不可管理）", fullPerms(), model.ApplicationStatusArchived,
			model.MenuEntryCapabilities{View: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := emptySnapshot("app_a")
			snap.Status = tc.status
			snap.Entries = []model.MenuEntry{form}
			repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
			svc := newMenuTestService(repo, tc.perms)

			menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
			assert.NoError(t, err)
			assert.Equal(t, tc.want, menu.EntryMap["menu_form"].Capabilities)
			assert.False(t, menu.EntryMap["menu_form"].Capabilities.Favorite)
		})
	}
}

// TestMenuCapabilities（见上）覆盖节点级 capabilities 派生；
// 租户隔离/软删应用不可见等依赖真实 SQL 租户过滤的链路由
// sec_menu_integration_test.go 覆盖（fake 桩无法表达 tenant_id 条件）
