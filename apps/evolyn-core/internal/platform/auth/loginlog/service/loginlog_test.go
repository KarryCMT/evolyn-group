package service

import (
	"context"
	"testing"
	"time"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure/ipregion"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/auth/loginlog/model"

	"github.com/stretchr/testify/assert"
)

// fakeRepo 捕获写入参数；真库读写路径由集成测试覆盖
type fakeRepo struct {
	created []*model.LoginLog

	listAccountID uint
	listStart     time.Time
	listEndExcl   time.Time
	listOffset    int
	listLimit     int
}

func (f *fakeRepo) Create(_ context.Context, log *model.LoginLog) error {
	f.created = append(f.created, log)
	return nil
}

func (f *fakeRepo) ListByAccount(_ context.Context, accountID uint, start, endExcl time.Time, offset, limit int) ([]model.LoginLog, int64, error) {
	f.listAccountID, f.listStart, f.listEndExcl = accountID, start, endExcl
	f.listOffset, f.listLimit = offset, limit
	return nil, 0, nil
}

func (f *fakeRepo) Migrate() error { return nil }

// newTestService 用内嵌真实 IP 库构造（离线、无外部依赖），归属地断言
// 只做「非未知」级别，不绑定具体城市（xdb 数据随版本更新漂移）
func newTestService(t *testing.T) (*loginLogService, *fakeRepo) {
	t.Helper()
	resolver, err := ipregion.New()
	assert.NoError(t, err)
	repo := &fakeRepo{}
	return NewService(repo, resolver), repo
}

func TestRecord(t *testing.T) {
	svc, repo := newTestService(t)

	ctx := contextx.NewRequestMetaContext(context.Background(), contextx.RequestMeta{
		IP:        "210.21.226.222",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126.0.0.0",
		RequestID: "req-001",
	})
	svc.Record(ctx, Entry{AccountID: 7, TenantID: 3, MemberID: 9, Method: model.MethodPassword})

	assert.Len(t, repo.created, 1)
	log := repo.created[0]
	assert.Equal(t, uint(7), log.AccountID)
	assert.Equal(t, uint(3), log.TenantID)
	assert.Equal(t, uint(9), log.MemberID)
	assert.Equal(t, model.MethodPassword, log.Method)
	assert.Equal(t, model.ClientWeb, log.Client)
	assert.Equal(t, "210.21.226.222", log.IP)
	assert.Equal(t, "req-001", log.RequestID)
	assert.NotEqual(t, ipregion.UnknownLocation, log.Location, "公网 IP 应解析出归属地")
}

// 脱离 HTTP 链路的调用方（无请求元数据）不 panic：client 归 unknown、归属地回退未知
func TestRecordWithoutMeta(t *testing.T) {
	svc, repo := newTestService(t)

	svc.Record(context.Background(), Entry{AccountID: 1, Method: model.MethodOAuth + "github"})

	assert.Len(t, repo.created, 1)
	log := repo.created[0]
	assert.Equal(t, model.ClientUnknown, log.Client)
	assert.Equal(t, ipregion.UnknownLocation, log.Location)
	assert.Equal(t, "oauth_github", log.Method)
}

func TestListByAccount(t *testing.T) {
	svc, repo := newTestService(t)
	// 期望值与实现同一时区来源（Asia/Shanghai，缺 tzdata 环境回退 +08:00），
	// 避免自造 FixedZone 与实际 Location 对象不等
	cst := kernel.CSTLocation()

	// 分页规范化：page/pageSize 非法取默认，超限封顶
	_, err := svc.ListByAccount(context.Background(), 7, PageQuery{Page: 0, PageSize: 0})
	assert.NoError(t, err)
	assert.Equal(t, uint(7), repo.listAccountID)
	assert.Equal(t, 0, repo.listOffset, "第 1 页偏移为 0")
	assert.Equal(t, defaultPageSize, repo.listLimit)
	assert.True(t, repo.listStart.IsZero() && repo.listEndExcl.IsZero(), "空日期不过滤")

	// 日期闭区间换算：endDate 2026-08-22 → 上界 2026-08-23 零点（开区间）
	_, err = svc.ListByAccount(context.Background(), 7, PageQuery{
		Page: 3, PageSize: 10, StartDate: "2026-08-01", EndDate: "2026-08-22",
	})
	assert.NoError(t, err)
	assert.Equal(t, 20, repo.listOffset, "第 3 页每页 10 条偏移 20")
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, cst), repo.listStart)
	assert.Equal(t, time.Date(2026, 8, 23, 0, 0, 0, 0, cst), repo.listEndExcl)

	// pageSize 超限封顶
	_, err = svc.ListByAccount(context.Background(), 7, PageQuery{PageSize: 500})
	assert.NoError(t, err)
	assert.Equal(t, maxPageSize, repo.listLimit)

	// 非法日期格式报错
	_, err = svc.ListByAccount(context.Background(), 7, PageQuery{StartDate: "2026/08/01"})
	assert.ErrorContains(t, err, "startDate 格式非法")
	_, err = svc.ListByAccount(context.Background(), 7, PageQuery{EndDate: "abc"})
	assert.ErrorContains(t, err, "endDate 格式非法")
}

func TestParseClient(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want string
	}{
		{"电脑浏览器", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126.0.0.0", model.ClientWeb},
		{"iPhone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Safari/604.1", model.ClientWap},
		{"Android", "Mozilla/5.0 (Linux; Android 14) Chrome/126.0.0.0 Mobile Safari/537.36", model.ClientWap},
		{"iPad", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) Safari/604.1", model.ClientWap},
		{"空 UA", "", model.ClientUnknown},
		{"空白 UA", "  ", model.ClientUnknown},
		// 非浏览器客户端（curl 等）无移动特征词，粗分类归 web
		{"命令行", "curl/8.4.0", model.ClientWeb},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, ParseClient(c.ua))
		})
	}
}
