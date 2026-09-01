package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProductCatalogMutualExclusion 产品/企业两目录互斥：同一分类码不得
// 同时出现在两侧目录（查询范围按 category_code 切分的事实基础）
func TestProductCatalogMutualExclusion(t *testing.T) {
	for _, enterprise := range categoryCatalog {
		assert.False(t, IsProductCategory(enterprise.Code),
			"企业日志分类 %s 不应归属产品日志目录", enterprise.Code)
	}
	for _, product := range productCategoryCatalog {
		assert.Equal(t, "", enterpriseCategoryNameOf(product.Code),
			"产品日志分类 %s 不应出现在企业日志目录", product.Code)
	}
}

// enterpriseCategoryNameOf 企业日志目录内查分类展示名；不在目录返回空串
func enterpriseCategoryNameOf(code string) string {
	for _, c := range categoryCatalog {
		if c.Code == code {
			return c.Name
		}
	}
	return ""
}

func TestCatalogProductCategoriesCoversCoreEvents(t *testing.T) {
	categories := CatalogProductCategories()
	assert.Len(t, categories, 6)
	assert.Equal(t, "application", categories[0].Code)
	assert.Equal(t, "应用管理", categories[0].Name)

	// 设计文档 §3.3 点名的核心事件齐备（3 段式事件码与注册表机械拼接一致）
	expect := map[string]bool{
		"application.application.create":            false,
		"application.application_menu_entry.create": false,
		"form.form.create":                          false,
		"form.form.delete":                          false,
		"workflow.workflow.publish":                 false,
		"form.form_record.submit":                   false,
		"form.form_permission_group.create":         false,
	}
	for _, category := range categories {
		for _, event := range category.Events {
			if _, ok := expect[event.Code]; ok {
				expect[event.Code] = true
			}
		}
	}
	for code, found := range expect {
		assert.True(t, found, "产品日志目录应包含事件 %s", code)
	}
}

func TestKnownProductCodes(t *testing.T) {
	assert.True(t, KnownProductCategory(CategoryProductForm))
	assert.True(t, KnownProductCategory("application")) // 应用管理归产品日志
	assert.False(t, KnownProductCategory(CategoryMemberManagement))
	assert.False(t, KnownProductCategory("no_such"))

	assert.True(t, KnownProductEvent("form.form.create"))
	assert.True(t, KnownProductEvent("form.form_record.submit"))
	// 未登记动作（草稿保存刻意不进筛选项）与企业日志事件不可用作产品筛选
	assert.False(t, KnownProductEvent("form.form.update-draft"))
	assert.False(t, KnownProductEvent("iam.member.update"))
	assert.False(t, KnownProductEvent("form.form.no_such"))
}

func TestProductEventNamesUseResourceVerbs(t *testing.T) {
	// 资源级动词覆盖生效：应用/表单用「创建」而非全局「添加」
	assert.Equal(t, "创建应用", EventName("application.application.create"))
	assert.Equal(t, "删除表单", EventName("form.form.delete"))
	assert.Equal(t, "提交表单数据", EventName("form.form_record.submit"))
	assert.Equal(t, "发布流程", EventName("workflow.workflow.publish"))
	// 草稿保存共用「更新」口径展示（不进筛选项，读取侧仍可解析名称）
	assert.Equal(t, "更新表单", EventName("form.form.update-draft"))
	assert.Equal(t, "删除表单「采购申请」", BuildSummary("form", "form", "delete", "采购申请"))
	assert.Equal(t, "创建表单「采购申请」", BuildSummary("form", "form", "create", "采购申请"))
}

func TestRecordCarriesApplicationSnapshot(t *testing.T) {
	// 应用维度快照（000064）：Entry 填写应用三元组时写时固化；ApplicationID
	// 为 0 视为非应用内操作，落 NULL/空串
	repo := &fakeAuditRepo{}
	svc := NewService(repo)
	svc.Record(context.Background(), Entry{
		Module: "form", Action: "delete", ResourceType: "form", ResourceID: "f_1",
		TargetName:      "采购申请",
		ApplicationID:   21,
		ApplicationCode: "app_21",
		ApplicationName: "测试应用",
	})
	require.Len(t, repo.created, 1)
	log := repo.created[0]
	require.NotNil(t, log.ApplicationID)
	assert.Equal(t, uint(21), *log.ApplicationID)
	assert.Equal(t, "app_21", log.ApplicationCode)
	assert.Equal(t, "测试应用", log.ApplicationNameSnapshot)
	// 产品分类与资源动词覆盖同步生效
	assert.Equal(t, "form.form.delete", log.EventCode)
	assert.Equal(t, CategoryProductForm, log.CategoryCode)
	assert.Equal(t, "删除表单「采购申请」", log.Summary)

	repo2 := &fakeAuditRepo{}
	NewService(repo2).Record(context.Background(), Entry{
		Module: "iam", Action: "update", ResourceType: "member", ResourceID: "7",
	})
	require.Len(t, repo2.created, 1)
	assert.Nil(t, repo2.created[0].ApplicationID)
	assert.Empty(t, repo2.created[0].ApplicationCode)
	assert.Empty(t, repo2.created[0].ApplicationNameSnapshot)
}
