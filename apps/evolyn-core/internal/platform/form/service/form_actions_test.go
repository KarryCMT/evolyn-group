package service

import (
	"context"
	"testing"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"

	"github.com/stretchr/testify/assert"
)

// ---- ADR-011：切换类型 / 复制（双动作码）/ 引用视图 服务单测 ----

// actionPerms 构造可配置权限集：forms 全量基础上叠加 form-actions 动作键
func actionPerms(actions ...string) map[string]bool {
	perms := map[string]bool{
		"forms:create": true, "forms:get": true, "forms:list": true,
		"forms:patch": true, "forms:update": true, "forms:delete": true,
	}
	for _, a := range actions {
		perms[a] = true
	}
	return perms
}

// newActionTestService 以指定权限集构造服务（fakeApps：7/9 active、8 archived）
func newActionTestService(perms map[string]bool, formRepo *fakeFormRepo) FormService {
	return NewFormService(
		passThroughTx{}, formRepo, newFakeVersionRepo(), &fakeRecordRepo{},
		&fakeQuota{limit: 100}, &fakeRecorder{}, fakeAccess{perms: perms},
		fakeApps{
			apps:     map[uint]string{7: "active", 8: "archived", 9: "active"},
			codeToID: map[string]uint{"app_x": 7},
		},
		&fakeMenuPort{},
	)
}

func seedWorkflowForm(repo *fakeFormRepo) {
	form := &model.Form{
		ApplicationID: 7, Code: "form_src", Name: "请假申请", FormType: model.FormTypeWorkflow,
		DraftContent: validDraft(), DraftRevision: 1, ProtocolVersion: 1, CreatorMemberID: 11,
	}
	form.TenantID = 1
	if _, err := repo.Create(tenantCtx(1), form); err != nil {
		panic(err)
	}
}

func TestSwitchFormType(t *testing.T) {
	// 流程表单切换为标准表单：类型变更成功（ADR-011：流程数据保留仅是
	// 运行时语义，服务层只负责类型事实源切换）
	repo := newFakeFormRepo()
	seedWorkflowForm(repo)
	svc := newActionTestService(actionPerms("form-actions:switch-type"), repo)

	detail, err := svc.SwitchType(tenantCtx(1), memberOfTenant(1), "form_src", &model.SwitchFormTypeRequest{FormType: model.FormTypeStandard})
	assert.NoError(t, err)
	assert.Equal(t, model.FormTypeStandard, detail.FormType)

	// 类型未变化：FORM_TYPE_UNCHANGED
	_, err = svc.SwitchType(tenantCtx(1), memberOfTenant(1), "form_src", &model.SwitchFormTypeRequest{FormType: model.FormTypeStandard})
	assert.ErrorIs(t, err, apperrors.ErrFormTypeUnchanged)

	// 缺动作授权键：即使 forms 全量也拒绝（动作键独立于 URL 门强裁决）
	svc2 := newActionTestService(actionPerms(), repo)
	_, err = svc2.SwitchType(tenantCtx(1), memberOfTenant(1), "form_src", &model.SwitchFormTypeRequest{FormType: model.FormTypeWorkflow})
	assert.ErrorIs(t, err, apperrors.ErrForbidden)

	// 非法类型枚举
	svc3 := newActionTestService(actionPerms("form-actions:switch-type"), repo)
	_, err = svc3.SwitchType(tenantCtx(1), memberOfTenant(1), "form_src", &model.SwitchFormTypeRequest{FormType: "dashboard"})
	assert.ErrorIs(t, err, apperrors.ErrFormTypeInvalid)
}

func TestCopyFormInApp(t *testing.T) {
	// 应用内复制（copy-in-app）：新编码、名称追加「（副本）」、草稿全文与
	// 类型复制、归属源应用
	repo := newFakeFormRepo()
	seedWorkflowForm(repo)
	svc := newActionTestService(actionPerms("form-actions:copy-in-app"), repo)

	detail, err := svc.Copy(tenantCtx(1), memberOfTenant(1), "form_src", &model.CopyFormRequest{})
	assert.NoError(t, err)
	assert.NotEqual(t, "form_src", detail.Code)
	assert.Equal(t, "请假申请（副本）", detail.Name)
	assert.Equal(t, model.FormTypeWorkflow, detail.FormType)
	assert.Equal(t, uint(7), detail.ApplicationID)
	// 草稿全文随复制携带
	assert.JSONEq(t, string(validDraft()), string(detail.Draft))

	// 缺 copy-in-app 动作键：拒绝
	svc2 := newActionTestService(actionPerms(), repo)
	_, err = svc2.Copy(tenantCtx(1), memberOfTenant(1), "form_src", &model.CopyFormRequest{})
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
}

func TestCopyFormCrossApp(t *testing.T) {
	repo := newFakeFormRepo()
	seedWorkflowForm(repo)
	cross := uint(9)

	// 跨应用成功：copy-cross-app 动作 + 目标应用可用
	svc := newActionTestService(actionPerms("form-actions:copy-cross-app"), repo)
	detail, err := svc.Copy(tenantCtx(1), memberOfTenant(1), "form_src", &model.CopyFormRequest{TargetApplicationID: &cross})
	assert.NoError(t, err)
	assert.Equal(t, uint(9), detail.ApplicationID)

	// 目标应用归档：FORM_APP_INVALID
	archived := uint(8)
	_, err = svc.Copy(tenantCtx(1), memberOfTenant(1), "form_src", &model.CopyFormRequest{TargetApplicationID: &archived})
	assert.ErrorIs(t, err, apperrors.ErrFormAppInvalid)

	// 只授 copy-in-app 未授 copy-cross-app：跨应用拒绝
	svc2 := newActionTestService(actionPerms("form-actions:copy-in-app"), repo)
	_, err = svc2.Copy(tenantCtx(1), memberOfTenant(1), "form_src", &model.CopyFormRequest{TargetApplicationID: &cross})
	assert.ErrorIs(t, err, apperrors.ErrForbidden)
}

func TestListReferences(t *testing.T) {
	repo := newFakeFormRepo()
	seedWorkflowForm(repo)

	// 端口未注入：空集（不报错）
	svc := newActionTestService(actionPerms(), repo)
	refs, err := svc.ListReferences(tenantCtx(1), memberOfTenant(1), "form_src")
	assert.NoError(t, err)
	assert.Empty(t, refs)

	// 端口注入：透传反查结果
	stub := &fakeReferenceSource{items: []FormReference{{
		ApplicationCode: "app_x", ApplicationName: "示例应用",
		EntryID: "menu_a", EntryName: "请假申请",
	}}}
	if impl, ok := svc.(FormReferenceSourceInjector); ok {
		impl.UseReferenceSource(stub)
	}
	refs, err = svc.ListReferences(tenantCtx(1), memberOfTenant(1), "form_src")
	assert.NoError(t, err)
	assert.Equal(t, stub.items, refs)
	assert.Equal(t, uint(101), stub.lastFormID) // fakeFormRepo.nextID 自 100 起
}

type fakeReferenceSource struct {
	items      []FormReference
	lastFormID uint
}

func (f *fakeReferenceSource) ListFormReferences(ctx context.Context, formID uint) ([]FormReference, error) {
	f.lastFormID = formID
	return f.items, nil
}
