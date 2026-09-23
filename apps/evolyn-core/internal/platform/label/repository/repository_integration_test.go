package repository

import (
	"context"
	"fmt"
	"testing"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	formmodel "evolyn/internal/platform/form/model"
	labelmodel "evolyn/internal/platform/label/model"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestTemplateRepositoryTenantAndBindingInvariants 用真实 PostgreSQL 验证持久化
// 层最后一道边界：租户隔离、每表单唯一模板、软删除后可重新绑定以及草稿 CAS。
func TestTemplateRepositoryTenantAndBindingInvariants(t *testing.T) {
	db := testsupport.NewPostgres(t)
	repo := NewTemplateRepository(db)

	alphaCtx := contextx.NewTenantContext(context.Background(), 101)
	betaCtx := contextx.NewTenantContext(context.Background(), 202)
	alphaForm := createForm(t, db, alphaCtx, "form_alpha")
	betaForm := createForm(t, db, betaCtx, "form_beta")

	alpha := newTemplate("label_shared", alphaForm)
	_, err := repo.Create(alphaCtx, alpha)
	require.NoError(t, err)
	require.Equal(t, uint(101), alpha.TenantID)

	// 同租户同表单不能存在两个有效模板，即使公开编码不同也由条件唯一索引拒绝。
	_, err = repo.Create(alphaCtx, newTemplate("label_duplicate", alphaForm))
	require.Error(t, err)

	// 租户 callback 使相同公开编码在另一个租户内独立存在，并隐藏跨租户探测。
	beta := newTemplate("label_shared", betaForm)
	_, err = repo.Create(betaCtx, beta)
	require.NoError(t, err)
	_, err = repo.GetByCode(betaCtx, alpha.Code)
	require.NoError(t, err, "beta 应读取自己的同编码模板")

	alphaOnly := "label_alpha_only"
	extraForm := createForm(t, db, alphaCtx, "form_alpha_extra")
	_, err = repo.Create(alphaCtx, newTemplate(alphaOnly, extraForm))
	require.NoError(t, err)
	_, err = repo.GetByCode(betaCtx, alphaOnly)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	rows, hasMore, err := repo.List(betaCtx, ListParams{Limit: 20})
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Len(t, rows, 1)
	require.Equal(t, beta.ID, rows[0].ID)

	// CAS 仅接受当前修订；成功后旧修订立即失效。
	updatedSchema := labelmodel.SchemaContent(`{"version":"1.0","page":{"width":60,"height":40,"unit":"mm","dpi":300},"elements":[]}`)
	updated, err := repo.SaveDraft(alphaCtx, alpha.ID, 1, updatedSchema, 60, 40, "mm", 300)
	require.NoError(t, err)
	require.True(t, updated)
	updated, err = repo.SaveDraft(alphaCtx, alpha.ID, 1, updatedSchema, 60, 40, "mm", 300)
	require.NoError(t, err)
	require.False(t, updated, "过期修订不能覆盖已保存草稿")

	require.NoError(t, repo.SoftDelete(alphaCtx, alpha))
	rebound := newTemplate(alpha.Code, alphaForm)
	_, err = repo.Create(alphaCtx, rebound)
	require.NoError(t, err, "软删除后条件唯一索引应允许原表单重新绑定")
	require.NotEqual(t, alpha.ID, rebound.ID)
}

func createForm(t *testing.T, db *gorm.DB, ctx context.Context, code string) *formmodel.Form {
	t.Helper()
	form := &formmodel.Form{
		AppID:           1,
		Code:            code,
		Name:            fmt.Sprintf("集成测试表单 %s", code),
		FormType:        formmodel.FormTypeStandard,
		DraftContent:    formmodel.JSONContent(`{"content":{"items":[]}}`),
		DraftRevision:   1,
		ProtocolVersion: formmodel.CurrentProtocolVersion,
		CreatorMemberID: 1,
	}
	require.NoError(t, db.WithContext(ctx).Create(form).Error)
	return form
}

func newTemplate(code string, form *formmodel.Form) *labelmodel.Template {
	return &labelmodel.Template{
		Code:            code,
		Name:            "集成测试标签",
		AppID:           form.AppID,
		FormID:          form.ID,
		FormCode:        form.Code,
		Status:          labelmodel.StatusDraft,
		DraftSchema:     labelmodel.SchemaContent(`{"version":"1.0","page":{"width":50,"height":30,"unit":"mm","dpi":300},"elements":[]}`),
		DraftRevision:   1,
		Width:           50,
		Height:          30,
		Unit:            "mm",
		DPI:             300,
		CreatorMemberID: 1,
		TenantBaseModel: kernel.TenantBaseModel{},
	}
}
