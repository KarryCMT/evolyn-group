package service

import (
	"context"
	"errors"
	"testing"

	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeMaintenanceRepo 菜单维护用桩：在 fakeMenuRepo 基础上记录节点写入、
// 支持注入按编码可查的节点集合与同父最大排序值
type fakeMaintenanceRepo struct {
	fakeMenuRepo
	byCode     map[string]*model.MenuEntry
	maxSort    map[uint]int64 // key 为父节点 ID；根级用 0 表达
	created    []*model.MenuEntry
	revBumped  int
	createErr  error
	findErr    error
	maxSortErr error
}

func (f *fakeMaintenanceRepo) CreateFormEntry(ctx context.Context, entry *model.MenuEntry) (*model.MenuEntry, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = append(f.created, entry)
	return entry, nil
}

func (f *fakeMaintenanceRepo) FindByCode(ctx context.Context, applicationID uint, code string) (*model.MenuEntry, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	entry, ok := f.byCode[code]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return entry, nil
}

func (f *fakeMaintenanceRepo) MaxSortOrder(ctx context.Context, applicationID uint, parentEntryID *uint) (int64, error) {
	if f.maxSortErr != nil {
		return 0, f.maxSortErr
	}
	if parentEntryID == nil {
		return f.maxSort[0], nil
	}
	return f.maxSort[*parentEntryID], nil
}

func (f *fakeMaintenanceRepo) BumpMenuRevision(ctx context.Context, applicationID uint) error {
	f.revBumped++
	return nil
}

func TestAttachFormEntryRootAndGroup(t *testing.T) {
	group := menuEntryFixture(10, "menu_group", nil, model.MenuEntryTypeGroup, 1024)

	cases := []struct {
		name       string
		parentCode string
		parentID   *uint
	}{
		{"根级挂载（parentEntryCode 为空）", "", nil},
		{"分组下挂载", "menu_group", ptrUint(10)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeMaintenanceRepo{byCode: map[string]*model.MenuEntry{"menu_group": &group}}
			svc := NewMenuMaintenanceService(repo)

			err := svc.AttachFormEntry(alphaCtx(), 1, 77, "分组表单", tc.parentCode)

			assert.NoError(t, err)
			require.Len(t, repo.created, 1)
			entry := repo.created[0]
			assert.Equal(t, model.MenuEntryTypeForm, entry.EntryType)
			assert.Equal(t, uint(77), *entry.TargetID)
			if tc.parentID == nil {
				assert.Nil(t, entry.ParentEntryID)
			} else {
				assert.Equal(t, *tc.parentID, *entry.ParentEntryID)
			}
			assert.Equal(t, int64(1024), entry.SortOrder)
			assert.Equal(t, 1, repo.revBumped)
		})
	}
}

func TestAttachFormEntryParentInvalid(t *testing.T) {
	formEntry := menuEntryFixture(11, "menu_form", nil, model.MenuEntryTypeForm, 1024)

	cases := []struct {
		name     string
		parent   string
		byCode   map[string]*model.MenuEntry
		wantCode string
	}{
		{
			name:     "分组不存在（含跨应用/已软删编码）",
			parent:   "menu_missing",
			byCode:   map[string]*model.MenuEntry{},
			wantCode: apperrors.ErrMenuParentInvalid.Code,
		},
		{
			name:     "父节点不是分组（指向表单节点）",
			parent:   "menu_form",
			byCode:   map[string]*model.MenuEntry{"menu_form": &formEntry},
			wantCode: apperrors.ErrMenuParentInvalid.Code,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeMaintenanceRepo{byCode: tc.byCode}
			svc := NewMenuMaintenanceService(repo)

			err := svc.AttachFormEntry(alphaCtx(), 1, 77, "分组表单", tc.parent)

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
