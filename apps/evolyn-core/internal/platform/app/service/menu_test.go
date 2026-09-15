package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	apperrors "evolyn/internal/platform/app"
	"evolyn/internal/platform/app/model"
	"evolyn/internal/platform/app/repository"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- M2-菜单-1 Service 单元测试（真实 PostgreSQL 链路见 sec_menu_integration_test.go）----

// fakeMenuRepo 菜单仓储桩：按编码返回可注入快照
type fakeMenuRepo struct {
	snapshots map[string]*repository.MenuSnapshot
	created   []*model.MenuNode
	nextID    uint
	favorites map[uint]bool // member 无关的收藏集合（读侧投影用例注入）
}

func (f *fakeMenuRepo) GetSnapshot(ctx context.Context, tenantID uint, code string) (*repository.MenuSnapshot, error) {
	snap, ok := f.snapshots[code]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return snap, nil
}

func (f *fakeMenuRepo) Migrate() error { return nil }

func (f *fakeMenuRepo) CreateGroupEntry(ctx context.Context, node *model.MenuNode) (*model.MenuNode, error) {
	f.nextID++
	node.ID = f.nextID
	node.Code = fmt.Sprintf("menu_created_%d", f.nextID)
	f.created = append(f.created, node)
	return node, nil
}

// M2-资产-1 写路径桩（菜单维护用例见 menu_maintenance_test.go；只读用例不触达）
func (f *fakeMenuRepo) CreateFormEntry(ctx context.Context, node *model.MenuNode) (*model.MenuNode, error) {
	return node, nil
}
func (f *fakeMenuRepo) UpdateNameByFormTarget(ctx context.Context, appID, formID uint, name string) error {
	return nil
}
func (f *fakeMenuRepo) UpdateAppearanceByFormTarget(ctx context.Context, appID, formID uint, icon, color string) error {
	return nil
}
func (f *fakeMenuRepo) SoftDeleteByFormTarget(ctx context.Context, appID, formID uint) error {
	return nil
}
func (f *fakeMenuRepo) MaxSortOrder(ctx context.Context, appID uint, parentMenuID *uint) (int64, error) {
	return 0, nil
}
func (f *fakeMenuRepo) FindByCode(ctx context.Context, appID uint, code string) (*model.MenuNode, error) {
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeMenuRepo) BumpMenuRevision(ctx context.Context, appID uint) error {
	return nil
}
func (f *fakeMenuRepo) BumpMenuRevisionFrom(ctx context.Context, appID uint, baseRevision int64) (bool, error) {
	for _, snapshot := range f.snapshots {
		if snapshot.AppID != appID || snapshot.MenuRevision != baseRevision {
			continue
		}
		snapshot.MenuRevision++
		return true, nil
	}
	return false, nil
}

// ADR-011 新增接口的默认桩：只读用例不触达；节点管理/收藏用例在需要时覆写
func (f *fakeMenuRepo) UpdateEntryFields(ctx context.Context, appID, entryID uint, fields map[string]interface{}) error {
	return nil
}
func (f *fakeMenuRepo) CreateFavorite(ctx context.Context, fav *model.MenuFavorite) error {
	return nil
}
func (f *fakeMenuRepo) DeleteFavoriteByCode(ctx context.Context, tenantID, memberID uint, menuCode string) (bool, error) {
	return true, nil
}
func (f *fakeMenuRepo) FavoriteMenuIDs(ctx context.Context, tenantID, memberID, appID uint) (map[uint]bool, error) {
	if f.favorites == nil {
		return map[uint]bool{}, nil
	}
	return f.favorites, nil
}
func (f *fakeMenuRepo) DeleteFavoritesByFormTarget(ctx context.Context, appID, formID uint) error {
	return nil
}
func (f *fakeMenuRepo) ListFormMenuReferences(ctx context.Context, tenantID, formID uint) ([]repository.FormMenuReference, error) {
	return nil, nil
}

// menuNodeFixture 构造菜单节点（测试内联便捷函数）
func menuNodeFixture(id uint, code string, parent *uint, menuType string, sortOrder int64) model.MenuNode {
	node := model.MenuNode{
		ID: id, Code: code, ParentMenuID: parent, MenuType: menuType,
		Name: code, SortOrder: sortOrder, AppID: 1,
	}
	if menuType != model.MenuTypeGroup {
		// 非分组节点必须带资产引用（CHECK 约束）；group 的 TargetType 保持 NULL
		targetType := menuType
		node.TargetType = &targetType
		targetID := 900 + id
		node.TargetID = &targetID
	}
	return node
}

func ptrUint(v uint) *uint { return &v }

func newMenuTestService(repo repository.MenuRepository, perms map[string]bool) AppMenuService {
	return NewMenuService(passThroughTx{}, repo, nil, fakeAccess{perms: perms})
}

// emptySnapshot 空菜单快照（应用就绪、无节点——M2-菜单-1 的常态）
func emptySnapshot(code string) *repository.MenuSnapshot {
	return &repository.MenuSnapshot{
		AppID: 1, AppCode: code,
		Status: model.AppStatusActive, ProvisionStatus: model.ProvisionStatusReady,
		MenuRevision: 1,
	}
}

func TestMenuGetEmpty(t *testing.T) {
	// 空应用菜单：空树是合法结果（200 语义），rootMenuIds/nodeMap 为空
	// 集合而非 null；menuRevision 原样出网供后续管理接口乐观并发
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": emptySnapshot("app_a")}}
	svc := newMenuTestService(repo, fullPerms())

	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Equal(t, "app_a", menu.AppCode)
	assert.Equal(t, int64(1), menu.MenuRevision)
	assert.NotNil(t, menu.RootMenuIDs)
	assert.NotNil(t, menu.NodeMap)
	assert.Empty(t, menu.RootMenuIDs)
	assert.Empty(t, menu.NodeMap)
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
		svc := newMenuTestService(repo, map[string]bool{"apps:patch": true})
		_, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})
}

func TestMenuCreateGroup(t *testing.T) {
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": emptySnapshot("app_a")}}
	audit := &fakeAudit{}
	svc := NewMenuService(passThroughTx{}, repo, audit, fakeAccess{perms: fullPerms()})

	created, err := svc.CreateGroup(alphaCtx(), alphaMember(), "app_a", &model.CreateMenuGroupRequest{
		Name:             " 订单管理 ",
		BaseMenuRevision: 1,
	})
	assert.NoError(t, err)
	assert.Equal(t, "订单管理", created.Name)
	assert.Equal(t, int64(2), created.MenuRevision)
	assert.Nil(t, created.ParentMenuID)
	assert.Len(t, repo.created, 1)
	assert.Equal(t, model.MenuTypeGroup, repo.created[0].MenuType)
	assert.Nil(t, repo.created[0].TargetID)
	assert.Len(t, audit.entries, 1)
}

func TestMenuCreateGroupValidatesParentDepthAndRevision(t *testing.T) {
	root := menuNodeFixture(10, "menu_root", nil, model.MenuTypeGroup, 1024)
	child := menuNodeFixture(11, "menu_child", ptrUint(10), model.MenuTypeGroup, 1024)
	form := menuNodeFixture(12, "menu_form", nil, model.MenuTypeForm, 2048)

	tests := []struct {
		name    string
		parent  *string
		base    int64
		wantErr error
	}{
		{name: "允许根分组下创建二级分组", parent: ptrString("menu_root"), base: 1},
		{name: "拒绝三级分组", parent: ptrString("menu_child"), base: 1, wantErr: apperrors.ErrMenuDepthExceeded},
		{name: "拒绝资产作为父节点", parent: ptrString("menu_form"), base: 1, wantErr: apperrors.ErrMenuParentInvalid},
		{name: "拒绝陈旧修订号", base: 9, wantErr: apperrors.ErrMenuVersionConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := emptySnapshot("app_a")
			snap.Nodes = []model.MenuNode{root, child, form}
			repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
			svc := newMenuTestService(repo, fullPerms())
			created, err := svc.CreateGroup(alphaCtx(), alphaMember(), "app_a", &model.CreateMenuGroupRequest{
				Name: "分组", ParentMenuID: tc.parent, BaseMenuRevision: tc.base,
			})
			if tc.wantErr != nil {
				assert.True(t, errors.Is(err, tc.wantErr))
				assert.Nil(t, created)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.parent, created.ParentMenuID)
		})
	}
}

func ptrString(v string) *string { return &v }

func TestMenuSnapshotVisibilityAndOrdering(t *testing.T) {
	// 资产域落地前资产节点默认出网（target 为 null），构造「分组 → 资产」
	// 树验证裁剪与排序（不可见资产的裁剪回归见 TestMenuAssetInvisiblePruning）
	emptyRoot := menuNodeFixture(1, "menu_empty", nil, model.MenuTypeGroup, 0)
	root2 := menuNodeFixture(2, "menu_root2", nil, model.MenuTypeGroup, 1024)
	root3 := menuNodeFixture(3, "menu_root3", nil, model.MenuTypeGroup, 512)
	asset1 := menuNodeFixture(4, "menu_form1", ptrUint(2), model.MenuTypeForm, 2048)
	child := menuNodeFixture(5, "menu_child", ptrUint(2), model.MenuTypeGroup, 1024)
	asset2 := menuNodeFixture(6, "menu_dash2", ptrUint(5), model.MenuTypeDashboard, 1024)
	asset3 := menuNodeFixture(7, "menu_form3", ptrUint(3), model.MenuTypeForm, 1024)

	snap := emptySnapshot("app_a")
	snap.Nodes = []model.MenuNode{emptyRoot, root2, root3, asset1, child, asset2, asset3}
	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
	svc := newMenuTestService(repo, fullPerms())

	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	// 管理成员可见空分组，以便创建后继续向其中添加资产；根顺序仍按
	// sortOrder ASC（0、512、1024）稳定输出。
	assert.Equal(t, []string{"menu_empty", "menu_root3", "menu_root2"}, menu.RootMenuIDs)
	assert.Contains(t, menu.NodeMap, "menu_empty")
	assert.Contains(t, menu.NodeMap, "menu_form1")
	assert.Contains(t, menu.NodeMap, "menu_child")
	assert.Contains(t, menu.NodeMap, "menu_dash2")
	assert.Contains(t, menu.NodeMap, "menu_form3")
	assert.Equal(t, "menu_root2", *menu.NodeMap["menu_form1"].ParentMenuID)
	assert.Equal(t, "menu_root3", *menu.NodeMap["menu_form3"].ParentMenuID)
	assert.Nil(t, menu.NodeMap["menu_root2"].ParentMenuID)
	// 资产节点出网但资产域未落地：target 投影为 null（公开编码待 M2-资产-1）
	assert.Nil(t, menu.NodeMap["menu_form1"].Target)
}

func TestMenuAssetInvisiblePruning(t *testing.T) {
	// 资产级授权落地后的裁剪回归：资产不可见（目录注入空存在集模拟）时
	// 排除其菜单节点，且没有可见后代的分组整体裁剪（方案 §6.3）
	root := menuNodeFixture(1, "menu_root", nil, model.MenuTypeGroup, 1024)
	form := menuNodeFixture(2, "menu_form", ptrUint(1), model.MenuTypeForm, 1024)
	snap := emptySnapshot("app_a")
	snap.Nodes = []model.MenuNode{root, form}

	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
	// 只读成员看不到无可见资产后代的分组；管理成员则保留空分组。
	svc := newMenuTestService(repo, map[string]bool{"apps:get": true}).(*menuService)
	svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{}})

	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	assert.Empty(t, menu.RootMenuIDs)
	assert.Empty(t, menu.NodeMap)
}

func TestMenuFormTargetUsesPublicCodeAndFormType(t *testing.T) {
	form := menuNodeFixture(2, "menu_form", nil, model.MenuTypeForm, 1024)
	snap := emptySnapshot("app_a")
	snap.Nodes = []model.MenuNode{form}

	repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
	svc := newMenuTestService(repo, fullPerms()).(*menuService)
	svc.UseFormDirectory(fakeFormDirectory{existing: map[uint]FormTargetProjection{
		902: {Code: "form_0123456789abcdef", FormType: "workflow"},
	}})

	menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
	assert.NoError(t, err)
	target := menu.NodeMap["menu_form"].Target
	if assert.NotNil(t, target) {
		assert.Equal(t, model.MenuTypeForm, target.Type)
		assert.Equal(t, "form_0123456789abcdef", target.Code)
		assert.Equal(t, "workflow", target.FormType)
		payload, marshalErr := json.Marshal(target)
		assert.NoError(t, marshalErr)
		assert.JSONEq(t, `{"type":"form","code":"form_0123456789abcdef","formType":"workflow"}`, string(payload))
	}
}

// fakeFormDirectory 表单目录端口桩（M2-资产-1）：existing 为内部 ID → 菜单目标投影。
type fakeFormDirectory struct {
	existing map[uint]FormTargetProjection
}

func (f fakeFormDirectory) ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]FormTargetProjection, error) {
	return f.existing, nil
}

func TestMenuIntegrityFailures(t *testing.T) {
	cases := []struct {
		name  string
		nodes []model.MenuNode
	}{
		{"孤儿节点（父引用不存在）", []model.MenuNode{
			menuNodeFixture(1, "menu_orphan", ptrUint(99), model.MenuTypeGroup, 1024),
		}},
		{"父节点非分组", []model.MenuNode{
			menuNodeFixture(1, "menu_form", nil, model.MenuTypeForm, 1024),
			menuNodeFixture(2, "menu_child", ptrUint(1), model.MenuTypeGroup, 1024),
		}},
		{"父链循环", []model.MenuNode{
			menuNodeFixture(1, "menu_a", ptrUint(2), model.MenuTypeGroup, 1024),
			menuNodeFixture(2, "menu_b", ptrUint(1), model.MenuTypeGroup, 1024),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := emptySnapshot("app_a")
			snap.Nodes = tc.nodes
			repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
			svc := newMenuTestService(repo, fullPerms())

			_, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
			// 服务端数据完整性故障返回 APP_MENU_INVALID（500），非 409
			assert.True(t, errors.Is(err, apperrors.ErrMenuInvalid))
		})
	}
}

func TestMenuCapabilities(t *testing.T) {
	// capabilities 已收敛为 view/favorite/actions 三字段（ADR-011）：actions
	// 是按钮唯一事实源（动作注册表 × 权限集 × 可编辑状态派生）；favorite 为
	// 个人状态能力（凡可见即可收藏，与权限/应用状态无关）
	// 菜单管理员全量权限集（模拟 tenant-admin：apps:* + forms:* +
	// form-actions:*，ADR-011 动作键齐备）
	menuAdminPerms := func() map[string]bool {
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
	form := menuNodeFixture(2, "menu_form", nil, model.MenuTypeForm, 1024)
	cases := []struct {
		name   string
		perms  map[string]bool
		status string
		want   model.MenuNodeCapabilities
	}{
		{"全量权限 + active", menuAdminPerms(), model.AppStatusActive,
			model.MenuNodeCapabilities{View: true, Favorite: true,
				Actions: model.MenuNodeActions{Edit: true, Rename: true, SwitchType: true, ReferenceView: true,
					CopyInApp: true, CopyCrossApp: true, Move: true, Hide: true, Delete: true}}},
		{"只读权限（authenticated 基线）", map[string]bool{"apps:get": true}, model.AppStatusActive,
			model.MenuNodeCapabilities{View: true, Favorite: true}},
		// 归档态禁止一切按钮动作（可编辑是动作公共因子）；favorite 随可见
		// 保持可收藏
		{"全量权限 + 归档（不可管理）", menuAdminPerms(), model.AppStatusArchived,
			model.MenuNodeCapabilities{View: true, Favorite: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := emptySnapshot("app_a")
			snap.Status = tc.status
			snap.Nodes = []model.MenuNode{form}
			repo := &fakeMenuRepo{snapshots: map[string]*repository.MenuSnapshot{"app_a": snap}}
			svc := newMenuTestService(repo, tc.perms)

			menu, err := svc.GetMenu(alphaCtx(), alphaMember(), "app_a")
			assert.NoError(t, err)
			assert.Equal(t, tc.want, menu.NodeMap["menu_form"].Capabilities)
			assert.False(t, menu.NodeMap["menu_form"].Favorited) // 未注入收藏集合
		})
	}
}

// TestMenuCapabilities（见上）覆盖节点级 capabilities 派生；
// 租户隔离/软删应用不可见等依赖真实 SQL 租户过滤的链路由
// sec_menu_integration_test.go 覆盖（fake 桩无法表达 tenant_id 条件）
