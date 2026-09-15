package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"evolyn/internal/platform/httpx"
	apperrors "evolyn/internal/platform/workbench"
	"evolyn/internal/platform/workbench/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- 企业自定义工作台服务单测（000078 租户语义）：存取/乐观锁/校验终审/
// 开通种子幂等（仓储替身模拟本域 SQL 语义；真库约束行为由迁移与集成测试保证）----

// passThroughTx 不携带事务语义、直接执行 fn
type passThroughTx struct{}

func (passThroughTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeRepo 仓储替身：内存态模拟按租户定位、乐观更新与并发写入唯一冲突
type fakeRepo struct {
	records       []*model.TenantWorkbench
	createViolate bool // 注入：Create 恒返回唯一约束冲突（模拟并发种子/首存）
	nextID        uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{nextID: 100}
}

func (f *fakeRepo) FindByTenant(ctx context.Context, tenantID uint) (*model.TenantWorkbench, error) {
	for _, record := range f.records {
		if record.TenantID == tenantID {
			return record, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeRepo) Create(ctx context.Context, record *model.TenantWorkbench) error {
	if f.createViolate {
		return &fakePGError{sqlState: "23505"}
	}
	f.nextID++
	record.ID = f.nextID
	stored := *record
	f.records = append(f.records, &stored)
	return nil
}

func (f *fakeRepo) UpdateContentWithRevision(
	ctx context.Context, id uint, fromRevision int64, content model.Content,
) (bool, error) {
	for _, record := range f.records {
		if record.ID != id || record.Revision != fromRevision {
			continue
		}
		record.Content = content
		record.Revision++
		return true, nil
	}
	return false, nil
}

// Migrate 满足仓储接口（替身无开发迁移路径）
func (f *fakeRepo) Migrate() error { return nil }

// fakePGError 模拟 pgconn 错误的 GetSQLState 接口（唯一约束冲突注入）
type fakePGError struct{ sqlState string }

func (e *fakePGError) GetSQLState() string { return e.sqlState }
func (e *fakePGError) Error() string       { return fmt.Sprintf("pg error %s", e.sqlState) }

func newWorkbenchService(repo *fakeRepo) WorkbenchService {
	return NewWorkbenchService(passThroughTx{}, repo)
}

const validDocument = `{"version":1,"widgets":[{"id":"greeting-0-2","type":"greeting","title":"问候语","x":0,"y":2,"w":3,"h":1,"minW":3,"maxH":1}]}`

func TestWorkbenchGetMissingRowReturnsNil(t *testing.T) {
	svc := newWorkbenchService(newFakeRepo())

	view, err := svc.Get(context.Background(), 1)
	require.NoError(t, err)
	assert.Nil(t, view, "行缺失（种子兜底外的异常路径）应返回 nil，前端回退默认布局")
}

func TestWorkbenchGetReturnsStoredDocument(t *testing.T) {
	repo := newFakeRepo()
	svc := newWorkbenchService(repo)

	_, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: 0, Document: []byte(validDocument),
	})
	require.NoError(t, err)

	view, err := svc.Get(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, view)
	assert.EqualValues(t, 1, view.Revision)
	assert.JSONEq(t, validDocument, string(view.Document))

	// 租户隔离：其他租户读不到
	other, err := svc.Get(context.Background(), 2)
	require.NoError(t, err)
	assert.Nil(t, other)
}

func TestWorkbenchSaveFirstInsertsRevisionOne(t *testing.T) {
	repo := newFakeRepo()
	svc := newWorkbenchService(repo)

	view, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: 0, Document: []byte(validDocument),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, view.Revision)
	assert.JSONEq(t, validDocument, string(view.Document))

	stored, err := repo.FindByTenant(context.Background(), 1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, stored.Revision)
	assert.Equal(t, uint(1), stored.TenantID)
}

func TestWorkbenchSaveFirstWithNonZeroRevisionConflicts(t *testing.T) {
	svc := newWorkbenchService(newFakeRepo())

	_, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: 3, Document: []byte(validDocument),
	})
	require.ErrorIs(t, err, apperrors.ErrRevisionConflict)
}

func TestWorkbenchSaveUpdatesWithMatchingRevision(t *testing.T) {
	repo := newFakeRepo()
	svc := newWorkbenchService(repo)

	first, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: 0, Document: []byte(validDocument),
	})
	require.NoError(t, err)

	updated := `{"version":1,"widgets":[{"id":"todo-0-0","type":"todo","title":"流程中心","x":0,"y":0,"w":3,"h":4}]}`
	second, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: first.Revision, Document: []byte(updated),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, second.Revision)
	assert.JSONEq(t, updated, string(second.Document))

	view, err := svc.Get(context.Background(), 1)
	require.NoError(t, err)
	assert.EqualValues(t, 2, view.Revision)
	assert.JSONEq(t, updated, string(view.Document))
}

func TestWorkbenchSaveStaleRevisionConflicts(t *testing.T) {
	repo := newFakeRepo()
	svc := newWorkbenchService(repo)

	first, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: 0, Document: []byte(validDocument),
	})
	require.NoError(t, err)

	stale := first.Revision - 1
	_, err = svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: stale, Document: []byte(validDocument),
	})
	require.ErrorIs(t, err, apperrors.ErrRevisionConflict)

	// 冲突按稳定码出网（前端按 errCode 分支）
	var biz *httpx.BizError
	require.True(t, errors.As(err, &biz))
	assert.Equal(t, "WORKBENCH_REVISION_CONFLICT", biz.Code)
}

func TestWorkbenchSaveConcurrentInsertConflicts(t *testing.T) {
	// 并发首存：唯一约束冲突映射为乐观锁冲突，不让 23505 裸错误出网
	repo := newFakeRepo()
	repo.createViolate = true
	svc := newWorkbenchService(repo)

	_, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: 0, Document: []byte(validDocument),
	})
	require.ErrorIs(t, err, apperrors.ErrRevisionConflict)
}

func TestWorkbenchSaveRejectsInvalidDocument(t *testing.T) {
	repo := newFakeRepo()
	svc := newWorkbenchService(repo)

	cases := []struct {
		name string
		raw  string
	}{
		{"未知卡片类型", `{"version":1,"widgets":[{"id":"a","type":"nope","title":"t","x":0,"y":0,"w":1,"h":1}]}`},
		{"坐标越界", `{"version":1,"widgets":[{"id":"a","type":"todo","title":"t","x":-1,"y":0,"w":1,"h":1}]}`},
		{"结构损坏", `{"version":1,"widgets":"bad"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
				Revision: 0, Document: []byte(tc.raw),
			})
			require.ErrorIs(t, err, apperrors.ErrDocumentInvalid)

			// 拒绝的文档不落库
			stored, findErr := repo.FindByTenant(context.Background(), 1)
			require.ErrorIs(t, findErr, gorm.ErrRecordNotFound)
			assert.Nil(t, stored)
		})
	}
}

func TestWorkbenchSaveNormalizesStoredDocument(t *testing.T) {
	repo := newFakeRepo()
	svc := newWorkbenchService(repo)

	// 携带未知字段与多余根键：保存后库存为规范化文档
	raw := `{"version":1,"extra":1,"widgets":[{"id":"a","type":"todo","title":"t","x":0,"y":0,"w":1,"h":1,"junk":true}]}`
	_, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
		Revision: 0, Document: []byte(raw),
	})
	require.NoError(t, err)

	stored, err := repo.FindByTenant(context.Background(), 1)
	require.NoError(t, err)
	assert.NotContains(t, string(stored.Content), "extra")
	assert.NotContains(t, string(stored.Content), "junk")
}

func TestWorkbenchSeedDefaults(t *testing.T) {
	t.Run("写入默认文档", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newWorkbenchService(repo)

		require.NoError(t, svc.SeedDefaults(context.Background(), 1))

		stored, err := repo.FindByTenant(context.Background(), 1)
		require.NoError(t, err)
		assert.EqualValues(t, 1, stored.Revision)
		assert.JSONEq(t, model.DefaultWorkbenchDocument, string(stored.Content))
	})

	t.Run("已有行幂等", func(t *testing.T) {
		repo := newFakeRepo()
		svc := newWorkbenchService(repo)
		require.NoError(t, svc.SeedDefaults(context.Background(), 1))

		// 管理员改过布局后再次种子：保持已保存内容，不覆盖
		_, err := svc.Save(context.Background(), 1, &model.SaveWorkbenchRequest{
			Revision: 1, Document: []byte(validDocument),
		})
		require.NoError(t, err)
		require.NoError(t, svc.SeedDefaults(context.Background(), 1))

		stored, err := repo.FindByTenant(context.Background(), 1)
		require.NoError(t, err)
		assert.JSONEq(t, validDocument, string(stored.Content))
	})

	t.Run("并发种子竞态幂等", func(t *testing.T) {
		repo := newFakeRepo()
		repo.createViolate = true // 模拟另一请求已插入同租户行
		svc := newWorkbenchService(repo)

		require.NoError(t, svc.SeedDefaults(context.Background(), 1))
	})
}

func TestDefaultWorkbenchDocumentIsValid(t *testing.T) {
	// 默认文档必须过服务端校验器（与前端 defaultWorkbench.ts 镜像的守护）
	normalized, issue := ValidateWorkbenchDocument([]byte(model.DefaultWorkbenchDocument))
	require.Nil(t, issue)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(normalized, &doc))
	widgets := doc["widgets"].([]any)
	assert.Len(t, widgets, 8, "默认布局 8 张卡片（与前端 defaultWorkbench.ts 一致）")
}
