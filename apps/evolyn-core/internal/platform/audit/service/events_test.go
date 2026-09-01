package service

import (
	"context"
	"testing"

	"evolyn/internal/platform/audit/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuditRepo 捕获写入行的内存仓储（Record 是 best-effort，只验证投影结果）
type fakeAuditRepo struct {
	created []model.AuditLog
}

func (f *fakeAuditRepo) Create(_ context.Context, log *model.AuditLog) error {
	f.created = append(f.created, *log)
	return nil
}

func (f *fakeAuditRepo) List(context.Context, uint, int, int) ([]model.AuditLog, error) {
	return nil, nil
}

func (f *fakeAuditRepo) Migrate() error { return nil }

// stubActorNamer 固定返回的显示名解析器
type stubActorNamer struct{ name string }

func (s stubActorNamer) MemberDisplayName(context.Context, uint) string { return s.name }

func TestEventRegistryDerivation(t *testing.T) {
	// 已登记资源：事件码机械拼接 + 分类/摘要模板生成
	repo := &fakeAuditRepo{}
	NewService(repo).Record(context.Background(), Entry{
		Module: "iam", Action: "update", ResourceType: "member", ResourceID: "7",
		Before: map[string]string{"nickname": "旧名"},
		After:  map[string]string{"nickname": "新名"},
	})

	require.Len(t, repo.created, 1)
	log := repo.created[0]
	assert.Equal(t, "iam.member.update", log.EventCode)
	assert.Equal(t, CategoryMemberManagement, log.CategoryCode)
	// 目标名从 After 优先提取（展示级键）
	assert.Equal(t, "新名", log.TargetNameSnapshot)
	assert.Equal(t, "更新成员「新名」", log.Summary)
}

func TestEventRegistryUnknownResourceDegrades(t *testing.T) {
	// 未登记资源：投影为空，读取侧降级「历史操作记录」
	repo := &fakeAuditRepo{}
	NewService(repo).Record(context.Background(), Entry{
		Module: "future", Action: "update", ResourceType: "widget",
	})

	require.Len(t, repo.created, 1)
	log := repo.created[0]
	assert.Empty(t, log.EventCode)
	assert.Empty(t, log.CategoryCode)
	assert.Empty(t, log.Summary)
}

func TestRecordExplicitProjectionOverride(t *testing.T) {
	// 显式事件码/分类/摘要/快照优先于注册表推导（新业务专用通道）
	repo := &fakeAuditRepo{}
	NewService(repo, stubActorNamer{"操作者甲"}).Record(context.Background(), Entry{
		Module: "enterpriselog", Action: "create", ResourceType: "export",
		MemberID:   5,
		EventCode:  "enterpriselog.export.create",
		TargetName: "登录日志",
		Summary:    "导出登录日志，共 3 条",
	})

	require.Len(t, repo.created, 1)
	log := repo.created[0]
	assert.Equal(t, "enterpriselog.export.create", log.EventCode)
	assert.Equal(t, CategoryLogExport, log.CategoryCode) // 按事件码回查注册表补全
	assert.Equal(t, "登录日志", log.TargetNameSnapshot)
	assert.Equal(t, "导出登录日志，共 3 条", log.Summary)
	// 操作人快照经 ActorNamer 按成员 ID 解析（Entry 未显式提供时）
	assert.Equal(t, "操作者甲", log.ActorNameSnapshot)
}

func TestSummaryTruncatedToColumnLimit(t *testing.T) {
	// 超长目标名生成的摘要按 varchar(1000) 语义截断（Record 投影出口统一处理）
	longName := make([]rune, 2000)
	for i := range longName {
		longName[i] = '名'
	}
	repo := &fakeAuditRepo{}
	NewService(repo).Record(context.Background(), Entry{
		Module: "iam", Action: "create", ResourceType: "department",
		After: map[string]string{"name": string(longName)},
	})

	require.Len(t, repo.created, 1)
	assert.Len(t, []rune(repo.created[0].Summary), summaryMaxRunes)
}

func TestSummaryNeverTouchesNonDisplayKeys(t *testing.T) {
	// 快照中的手机号等敏感键绝不进摘要/目标名（脱敏口径）
	repo := &fakeAuditRepo{}
	NewService(repo).Record(context.Background(), Entry{
		Module: "iam", Action: "update", ResourceType: "account",
		After: map[string]interface{}{"phone": "13800001111", "name": "张三"},
	})

	require.Len(t, repo.created, 1)
	log := repo.created[0]
	assert.Equal(t, "张三", log.TargetNameSnapshot)
	assert.NotContains(t, log.Summary, "13800001111")
}

func TestCatalogCategoriesCoversRegisteredEvents(t *testing.T) {
	categories := CatalogCategories()
	require.NotEmpty(t, categories)

	byCode := map[string]CategoryView{}
	for _, c := range categories {
		byCode[c.Code] = c
	}

	// 设计文档点名的核心分类齐备（企业日志目录不含产品分类）
	for _, code := range []string{
		CategoryMemberManagement, CategoryOrganization, CategoryRolePermission,
		CategoryTenantSettings, CategoryLogExport,
	} {
		assert.Contains(t, byCode, code)
	}
	assert.NotContains(t, byCode, CategoryProductApplication, "应用管理归属产品日志目录")

	member := byCode[CategoryMemberManagement]
	found := false
	for _, e := range member.Events {
		if e.Code == "iam.member.update" && e.Name == "更新成员" {
			found = true
		}
	}
	assert.True(t, found, "成员管理分类应包含 iam.member.update 事件")
}

func TestKnownCodes(t *testing.T) {
	assert.True(t, KnownEvent("iam.member.update"))
	assert.False(t, KnownEvent("iam.member.unknown_action_of_doom"))
	assert.False(t, KnownEvent("not-an-event"))
	assert.True(t, KnownCategory(CategoryTenantSettings))
	assert.False(t, KnownCategory("no_such_category"))
}
