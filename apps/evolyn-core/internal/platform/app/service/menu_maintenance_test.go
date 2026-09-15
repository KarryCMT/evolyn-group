package service

import (
	"context"
	"errors"
	"testing"

	apperrors "evolyn/internal/platform/app"
	"evolyn/internal/platform/app/model"
	"evolyn/internal/platform/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeMaintenanceRepo 菜单维护用桩：在 fakeMenuRepo 基础上记录节点写入、
// 支持注入按编码可查的节点集合与同父最大排序值
type fakeMaintenanceRepo struct {
	fakeMenuRepo
	byCode     map[string]*model.MenuNode
	maxSort    map[uint]int64 // key 为父节点 ID；根级用 0 表达
	created    []*model.MenuNode
	revBumped  int
	createErr  error
	findErr    error
	maxSortErr error
}

func (f *fakeMaintenanceRepo) CreateFormNode(ctx context.Context, node *model.MenuNode) (*model.MenuNode, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = append(f.created, node)
	return node, nil
}

func (f *fakeMaintenanceRepo) FindByCode(ctx context.Context, appID uint, code string) (*model.MenuNode, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	node, ok := f.byCode[code]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return node, nil
}

func (f *fakeMaintenanceRepo) MaxSortOrder(ctx context.Context, appID uint, parentMenuID *uint) (int64, error) {
	if f.maxSortErr != nil {
		return 0, f.maxSortErr
	}
	if parentMenuID == nil {
		return f.maxSort[0], nil
	}
	return f.maxSort[*parentMenuID], nil
}

func (f *fakeMaintenanceRepo) BumpMenuRevision(ctx context.Context, appID uint) error {
	f.revBumped++
	return nil
}

func TestAttachFormNodeRootAndGroup(t *testing.T) {
	group := menuNodeFixture(10, "menu_group", nil, model.MenuTypeGroup, 1024)

	cases := []struct {
		name       string
		parentCode string
		parentID   *uint
	}{
		{"根级挂载（parentMenuCode 为空）", "", nil},
		{"分组下挂载", "menu_group", ptrUint(10)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeMaintenanceRepo{byCode: map[string]*model.MenuNode{"menu_group": &group}}
			svc := NewMenuMaintenanceService(repo)

			err := svc.AttachFormNode(alphaCtx(), 1, 77, "分组表单", tc.parentCode)

			assert.NoError(t, err)
			require.Len(t, repo.created, 1)
			node := repo.created[0]
			assert.Equal(t, model.MenuTypeForm, node.MenuType)
			assert.Equal(t, uint(77), *node.TargetID)
			if tc.parentID == nil {
				assert.Nil(t, node.ParentMenuID)
			} else {
				assert.Equal(t, *tc.parentID, *node.ParentMenuID)
			}
			assert.Equal(t, int64(1024), node.SortOrder)
			assert.Equal(t, 1, repo.revBumped)
		})
	}
}

func TestAttachFormNodeParentInvalid(t *testing.T) {
	formNode := menuNodeFixture(11, "menu_form", nil, model.MenuTypeForm, 1024)

	cases := []struct {
		name     string
		parent   string
		byCode   map[string]*model.MenuNode
		wantCode string
	}{
		{
			name:     "分组不存在（含跨应用/已软删编码）",
			parent:   "menu_missing",
			byCode:   map[string]*model.MenuNode{},
			wantCode: apperrors.ErrMenuParentInvalid.Code,
		},
		{
			name:     "父节点不是分组（指向表单节点）",
			parent:   "menu_form",
			byCode:   map[string]*model.MenuNode{"menu_form": &formNode},
			wantCode: apperrors.ErrMenuParentInvalid.Code,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeMaintenanceRepo{byCode: tc.byCode}
			svc := NewMenuMaintenanceService(repo)

			err := svc.AttachFormNode(alphaCtx(), 1, 77, "分组表单", tc.parent)

			var biz *httpx.BizError
			if !errors.As(err, &biz) {
				t.Fatalf("期望 BizError，实际 %v", err)
			}
			assert.Equal(t, tc.wantCode, biz.Code)
			// 非法父分组在写节点前拦截：无节点写入、修订号不递增
			assert.Empty(t, repo.created)
			assert.Zero(t, repo.revBumped)
		})
	}
}
