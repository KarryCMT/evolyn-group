// ---- SEC-FPERM-* 真实 PostgreSQL 集成测试矩阵（表单权限组 P1） ----
//
// 验证链路覆盖：迁移链 000058（tn_asset_permission_groups + subjects，含外键
// 与 CHECK）→ 权限组 CRUD（乐观锁/上限/主体存在性/软删级联硬删）→
// FormPermissionEvaluator（S4/S5/S2/S7 + 管理员旁路）→ 执行点（菜单裁剪
// view∨add、运行时 permissions 投影、提交权限管线、switch-type/发布阻塞）。
// 未配置 TEST_PG_DSN 时自动 Skip（离线套件保持全绿），与 SEC-APP-*/SEC-MENU-* 同约定。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	applicationmodel "evolyn/internal/platform/application/model"
	applicationrepository "evolyn/internal/platform/application/repository"
	applicationservice "evolyn/internal/platform/application/service"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// fpermEnv 双租户表单权限测试环境：alpha/beta 各含 tenant-admin owner 成员
// （租户开通基线已含 forms:*/form-permissions:*/form-data:*，表单权限 P1 起）
type fpermEnv struct {
	db          *gorm.DB
	iamRepo     *iamrepository.Repositories
	formRepo    repository.FormRepository
	versionRepo repository.FormVersionRepository
	recordRepo  repository.FormRecordRepository
	permRepo    repository.PermissionGroupRepository
	auditSvc    auditservice.Recorder
	access      AccessEvaluator
	appSvc      applicationservice.ApplicationService
	menuSvc     applicationservice.ApplicationMenuService
	formSvc     FormService
	permSvc     PermissionGroupService
	evaluator   FormPermissionEvaluator

	alpha, beta *tenantmodel.Tenant
	alphaOwner  *iammodel.User // tenant-admin（基线管理员）
	betaOwner   *iammodel.User
	plainMember *iammodel.User // alpha 普通成员（仅 authenticated 基线）
}

func newFpermEnv(t *testing.T) *fpermEnv {
	t.Helper()

	db := testsupport.NewPostgres(t)
	rdb := testsupport.DisabledRedis()
	iamRepo := iamrepository.NewRepositories(db, rdb)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	auditRepo := auditrepository.NewRepository(db)
	auditSvc := auditservice.NewService(auditRepo)
	appRepo := applicationrepository.NewRepository(db)
	menuRepo := applicationrepository.NewMenuRepository(db)
	formRepo := repository.NewRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	recordRepo := repository.NewRecordRepository(db)
	permRepo := repository.NewPermissionGroupRepository(db)
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), appRepo)
	// 表单配额计数器注入（与 server 装配同口径，缺省即「配额表面未配置」拒绝）
	if injector, ok := quotaSvc.(tenantservice.QuotaFormCounterInjector); ok {
		injector.UseFormCounter(formRepo)
	}
	txManager := infrastructure.NewTxManager(db)
	tenantSvc := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, 0)

	access := applicationservice.NewRBACAccessEvaluator(iamRepo.User(), iamRepo.Group())
	appSvc := applicationservice.NewApplicationService(txManager, appRepo, quotaSvc, auditSvc, access)
	menuSvc := applicationservice.NewMenuService(txManager, menuRepo, auditSvc, access)
	// 表单生命周期菜单节点维护端口（创建表单时挂 form 资产节点）
	menuMaintenance := applicationservice.NewMenuMaintenanceService(menuRepo)

	formSvc := NewFormService(txManager, formRepo, versionRepo, recordRepo, quotaSvc, auditSvc,
		access, fpermAppDirectory{apps: appRepo}, menuMaintenance)

	env := &fpermEnv{
		db: db, iamRepo: iamRepo, formRepo: formRepo, versionRepo: versionRepo,
		recordRepo: recordRepo, permRepo: permRepo, auditSvc: auditSvc, access: access,
		appSvc: appSvc, menuSvc: menuSvc, formSvc: formSvc,
	}

	env.alpha = env.openTenant(t, tenantSvc, "fperm-alpha", "fperm-owner-a")
	env.beta = env.openTenant(t, tenantSvc, "fperm-beta", "fperm-owner-b")
	env.alphaOwner = env.ownerMember(t, iamRepo, env.alpha, "fperm-owner-a")
	env.betaOwner = env.ownerMember(t, iamRepo, env.beta, "fperm-owner-b")
	env.plainMember = env.createPlainMember(t, env.alpha, "fperm-plain-a")

	// 判定器 + 配置面服务（主体解析/展示名适配与 server 装配同构）
	env.evaluator = NewFormPermissionEvaluator(
		permRepo, formRepo, versionRepo, access,
		fpermSubjectSource{iam: iamRepo},
	)
	if injector, ok := formSvc.(PermissionEvaluatorInjector); ok {
		injector.UsePermissionEvaluator(env.evaluator)
	}
	if injector, ok := formSvc.(PermissionGroupSourceInjector); ok {
		injector.UsePermissionGroupSource(NewPermissionGroupReadSource(permRepo))
	}
	env.permSvc = NewPermissionGroupService(txManager, permRepo, formRepo, versionRepo, auditSvc, access,
		fpermSubjectDirectory{iam: iamRepo})
	// 菜单读侧接入表单目录与权限裁剪端口（装配同构）
	if injector, ok := menuSvc.(applicationservice.MenuFormDirectoryInjector); ok {
		injector.UseFormDirectory(fpermFormDirectory{forms: formRepo})
	}
	if injector, ok := menuSvc.(applicationservice.FormPermissionDirectoryInjector); ok {
		injector.UseFormPermissionDirectory(fpermMenuDirectory{evaluator: env.evaluator})
	}
	return env
}

func (e *fpermEnv) openTenant(t *testing.T, svc tenantservice.TenantService, code, ownerName string) *tenantmodel.Tenant {
	t.Helper()
	tenant, err := svc.Open(context.Background(), &tenantservice.OpenTenantRequest{
		Code: code, Name: code, Plan: tenantmodel.PlanFree,
		OwnerName: ownerName, OwnerPassword: "secret123",
	})
	if err != nil {
		t.Fatalf("open tenant %s: %v", code, err)
	}
	return tenant
}

func (e *fpermEnv) ownerMember(t *testing.T, iamRepo *iamrepository.Repositories, tenant *tenantmodel.Tenant, ownerName string) *iammodel.User {
	t.Helper()
	account, err := iamRepo.Account().GetByName(context.Background(), ownerName)
	if err != nil {
		t.Fatalf("load owner account %s: %v", ownerName, err)
	}
	member, err := iamRepo.User().GetByAccountAndTenant(context.Background(), account.ID, tenant.ID)
	if err != nil {
		t.Fatalf("load owner member: %v", err)
	}
	return member
}

func (e *fpermEnv) createPlainMember(t *testing.T, tenant *tenantmodel.Tenant, name string) *iammodel.User {
	t.Helper()
	account := &iammodel.Account{Name: name, Password: "secret123"}
	account, err := e.iamRepo.Account().Create(context.Background(), account)
	if err != nil {
		t.Fatalf("create plain account: %v", err)
	}
	member := &iammodel.User{AccountId: account.ID, Nickname: "普通成员"}
	member.TenantID = tenant.ID
	if _, err = e.iamRepo.User().Create(contextx.NewTenantContext(context.Background(), tenant.ID), member); err != nil {
		t.Fatalf("create plain member: %v", err)
	}
	return member
}

func (e *fpermEnv) createAppWithForm(t *testing.T, ctx context.Context, member *iammodel.User, name string) (*applicationmodel.ApplicationDetail, *model.FormDetail) {
	t.Helper()
	app, err := e.appSvc.CreateBlank(ctx, member, &applicationmodel.CreateBlankRequest{Name: name})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	form, err := e.formSvc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: app.ID, Name: name + "表单", FormType: model.FormTypeStandard,
	})
	if err != nil {
		t.Fatalf("create form: %v", err)
	}
	return app, form
}

// saveAndPublish 保存草稿并发布（走真实 Service 校验链路）
func (e *fpermEnv) saveAndPublish(t *testing.T, ctx context.Context, member *iammodel.User, form *model.FormDetail, doc string) *model.PublishResult {
	t.Helper()
	saved, err := e.formSvc.SaveDraft(ctx, member, form.Code, &model.SaveDraftRequest{
		DraftRevision: form.DraftRevision, ProtocolVersion: model.CurrentProtocolVersion,
		Content: model.JSONContent(doc),
	})
	if err != nil {
		t.Fatalf("save draft: %v", err)
	}
	published, err := e.formSvc.Publish(ctx, member, form.Code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	return published
}

func fpermCtx(tenantID uint) context.Context {
	return contextx.NewTenantContext(context.Background(), tenantID)
}

// ---- 发布可用文档助手（协议校验要求完整属性） ----

// fpermField 发布文档字段项：name → type/label/allowBlank
type fpermField struct {
	name       string
	label      string
	widgetType string
	allowBlank bool
}

func fpermText(name, label string, allowBlank bool) fpermField {
	return fpermField{name: name, label: label, widgetType: "text", allowBlank: allowBlank}
}

func fpermNumber(name, label string) fpermField {
	return fpermField{name: name, label: label, widgetType: "number", allowBlank: true}
}

// fpermRadio 单选字段（options 固定 sales/ops/finance，供数据条件 in/eq 判定）
func fpermRadio(name, label string) fpermField {
	return fpermField{name: name, label: label, widgetType: "radiogroup", allowBlank: true}
}

// fpermDoc 组装发布可用文档（field_layout 依 items 顺序列出全部字段键）
func fpermDoc(items ...fpermField) string {
	body := ""
	layout := ""
	for i, item := range items {
		if i > 0 {
			body += ","
			layout += ","
		}
		widget := fmt.Sprintf(`{"type":"%s","widgetName":"%s","enable":true,"visible":true,"allowBlank":%t`,
			item.widgetType, item.name, item.allowBlank)
		if item.widgetType == "radiogroup" {
			widget += `,"options":[{"value":"sales","label":"销售"},{"value":"ops","label":"运营"},{"value":"finance","label":"财务"}]`
		}
		widget += `}`
		body += fmt.Sprintf(`{"widget":%s,"label":"%s","description":"","labelHidden":false,"lineWidth":6}`, widget, item.label)
		layout += fmt.Sprintf("%q", item.name)
	}
	return fmt.Sprintf(`{"content":{"type":"form","layout":"grid-2","items":[%s],"layout_fields":[],"field_layout":[%s],"fieldShowRules":[],"submitRule":2,"widget_submit_rules":{}}}`, body, layout)
}

// ---- 适配器（与 server 装配同构的最小测试适配） ----

type fpermAppDirectory struct {
	apps applicationrepository.ApplicationRepository
}

func (d fpermAppDirectory) ApplicationByID(ctx context.Context, id uint) (ApplicationView, bool, error) {
	app, err := d.apps.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ApplicationView{}, true, nil
		}
		return ApplicationView{}, false, err
	}
	return ApplicationView{ID: app.ID, Status: app.Status}, false, nil
}

func (d fpermAppDirectory) ApplicationByCode(ctx context.Context, code string) (ApplicationView, bool, error) {
	app, err := d.apps.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ApplicationView{}, true, nil
		}
		return ApplicationView{}, false, err
	}
	return ApplicationView{ID: app.ID, Status: app.Status}, false, nil
}

type fpermSubjectSource struct{ iam *iamrepository.Repositories }

func (s fpermSubjectSource) MemberSubject(ctx context.Context, memberID uint) ([]uint, []uint, error) {
	user, err := s.iam.User().GetUserByID(ctx, memberID)
	if err != nil {
		return nil, nil, err
	}
	roleIDs := make([]uint, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
	}
	for _, group := range user.Groups {
		for _, role := range group.Roles {
			roleIDs = append(roleIDs, role.ID)
		}
	}
	detail, err := s.iam.User().GetMemberDetail(ctx, memberID)
	if err != nil {
		return nil, nil, err
	}
	direct := make([]uint, 0, len(detail.Departments))
	for _, dept := range detail.Departments {
		direct = append(direct, dept.ID)
	}
	if len(direct) == 0 {
		return direct, roleIDs, nil
	}
	all, err := s.iam.Department().List(ctx)
	if err != nil {
		return nil, nil, err
	}
	parentByID := make(map[uint]*uint, len(all))
	for i := range all {
		parentByID[all[i].ID] = all[i].ParentId
	}
	expanded := map[uint]bool{}
	queue := append([]uint{}, direct...)
	for _, id := range direct {
		expanded[id] = true
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		parent := parentByID[current]
		if parent == nil || expanded[*parent] {
			continue
		}
		expanded[*parent] = true
		queue = append(queue, *parent)
	}
	departmentIDs := make([]uint, 0, len(expanded))
	for id := range expanded {
		departmentIDs = append(departmentIDs, id)
	}
	return departmentIDs, roleIDs, nil
}

type fpermSubjectDirectory struct{ iam *iamrepository.Repositories }

func (d fpermSubjectDirectory) SubjectExists(ctx context.Context, subjectType string, subjectID uint) (bool, error) {
	switch subjectType {
	case model.PermissionSubjectMember:
		_, err := d.iam.User().GetUserByID(ctx, subjectID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return err == nil, err
	case model.PermissionSubjectDepartment:
		_, err := d.iam.Department().GetByID(ctx, subjectID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return err == nil, err
	case model.PermissionSubjectRole:
		_, err := d.iam.RBAC().GetRoleByID(ctx, int(subjectID))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return err == nil, err
	}
	return false, nil
}

func (d fpermSubjectDirectory) SubjectNames(ctx context.Context, subjects []model.AssetPermissionGroupSubject) (map[string]map[uint]string, error) {
	names := map[string]map[uint]string{model.PermissionSubjectMember: {}, model.PermissionSubjectDepartment: {}, model.PermissionSubjectRole: {}}
	members, err := d.iam.User().List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range members {
		names[model.PermissionSubjectMember][members[i].ID] = members[i].Nickname
	}
	departments, err := d.iam.Department().List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range departments {
		names[model.PermissionSubjectDepartment][departments[i].ID] = departments[i].Name
	}
	roles, err := d.iam.RBAC().List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range roles {
		names[model.PermissionSubjectRole][roles[i].ID] = roles[i].Name
	}
	return names, nil
}

type fpermFormDirectory struct{ forms repository.FormRepository }

func (d fpermFormDirectory) ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]applicationservice.FormTargetProjection, error) {
	existing, err := d.forms.ExistingFormTargets(ctx, ids)
	if err != nil {
		return nil, err
	}
	targets := make(map[uint]applicationservice.FormTargetProjection, len(existing))
	for id, target := range existing {
		targets[id] = applicationservice.FormTargetProjection{Code: target.Code, FormType: string(target.FormType)}
	}
	return targets, nil
}

type fpermMenuDirectory struct{ evaluator FormPermissionEvaluator }

func (d fpermMenuDirectory) VisibleFormIDs(ctx context.Context, memberID uint, formIDs []uint) (map[uint]bool, error) {
	resolved, err := d.evaluator.EvaluateForForms(ctx, &iammodel.User{ID: memberID}, formIDs)
	if err != nil {
		return nil, err
	}
	visible := make(map[uint]bool, len(resolved))
	for formID, permission := range resolved {
		if permission.EntranceAllowed() {
			visible[formID] = true
		}
	}
	return visible, nil
}

// ---- 用例 ----

// SEC-FPERM-001：权限组 CRUD 全链路（真库）：创建（主体同租户校验 + 唯一
// 编码）→ 清单 → 整组乐观锁更新（冲突 409）→ 删除（subjects 同事务硬删）
func TestSECFPERM001GroupCRUDAndCascade(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	_, form := env.createAppWithForm(t, ctx, env.alphaOwner, "CRUD 应用")
	env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(fpermText("name", "姓名", false), fpermNumber("amount", "金额")))

	created, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name:       "管理全部数据",
		Operations: []string{"view", "add", "edit", "delete"},
		FieldPermissions: []model.PermissionFieldRule{
			{Field: "name", Visible: true, Editable: true},
			{Field: "amount", Visible: false, Editable: false},
		},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.alphaOwner.ID}},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, created.Code)
	assert.Equal(t, int64(1), created.Revision)
	assert.Equal(t, env.alphaOwner.Nickname, created.Subjects[0].Name, "主体展示名经窄端口解析")

	// 主体存在性校验：不存在的成员 ID 拒绝
	_, err = env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "坏主体", Operations: []string{"view"},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: 999999}},
	})
	assert.ErrorIs(t, err, apperrors.ErrPermissionSubjectInvalid)

	// 乐观锁：陈旧 baseRevision → 409
	_, err = env.permSvc.UpdateGroup(ctx, env.alphaOwner, form.Code, created.Code, &model.UpdatePermissionGroupRequest{
		CreatePermissionGroupRequest: model.CreatePermissionGroupRequest{
			Name: "陈旧写入", Operations: []string{"view"},
		},
		BaseRevision: 99,
	})
	assert.ErrorIs(t, err, apperrors.ErrPermissionRevisionConflict)

	updated, err := env.permSvc.UpdateGroup(ctx, env.alphaOwner, form.Code, created.Code, &model.UpdatePermissionGroupRequest{
		CreatePermissionGroupRequest: model.CreatePermissionGroupRequest{
			Name: "管理全部数据 v2", Operations: []string{"view"},
			FieldPermissions: []model.PermissionFieldRule{{Field: "amount", Visible: true, Editable: false}},
		},
		BaseRevision: 1,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), updated.Revision)
	assert.Equal(t, "管理全部数据 v2", updated.Name)

	// 删除：subjects 同事务硬删（软删不触发外键，Service 显式清理）
	assert.NoError(t, env.permSvc.DeleteGroup(ctx, env.alphaOwner, form.Code, created.Code))
	var subjectCount int64
	assert.NoError(t, env.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM tn_asset_permission_group_subjects s
		 JOIN tn_asset_permission_groups g ON g.id = s.group_id WHERE g.code = ?`, created.Code,
	).Scan(&subjectCount).Error)
	assert.Equal(t, int64(0), subjectCount, "删除权限组须同事务硬删主体行")
	rows, err := env.permSvc.ListGroups(ctx, env.alphaOwner, form.Code)
	assert.NoError(t, err)
	assert.Empty(t, rows, "软删组不再出现在清单")
}

// SEC-FPERM-002：跨租户隔离：beta 上下文读取/删除 alpha 的权限组 → NotFound；
// 同租户跨表单组编码归属不符 → NotFound
func TestSECFPERM002CrossTenantAndCrossForm(t *testing.T) {
	env := newFpermEnv(t)
	alphaCtx := fpermCtx(env.alpha.ID)
	betaCtx := fpermCtx(env.beta.ID)
	_, form := env.createAppWithForm(t, alphaCtx, env.alphaOwner, "隔离应用")
	env.saveAndPublish(t, alphaCtx, env.alphaOwner, form, fpermDoc(fpermText("name", "姓名", false)))
	created, err := env.permSvc.CreateGroup(alphaCtx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "alpha 组", Operations: []string{"view"},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.alphaOwner.ID}},
	})
	assert.NoError(t, err)

	// 跨租户：统一 NotFound（租户 Callback 过滤；删除先撞表单门）
	_, err = env.permSvc.ListGroups(betaCtx, env.betaOwner, form.Code)
	assert.ErrorIs(t, err, apperrors.ErrFormNotFound)
	err = env.permSvc.DeleteGroup(betaCtx, env.betaOwner, form.Code, created.Code)
	assert.ErrorIs(t, err, apperrors.ErrFormNotFound)

	// 跨应用同租户：组编码不属于该表单 → NotFound（不泄露归属）
	_, otherForm := env.createAppWithForm(t, alphaCtx, env.alphaOwner, "另一应用")
	err = env.permSvc.DeleteGroup(alphaCtx, env.alphaOwner, otherForm.Code, created.Code)
	assert.ErrorIs(t, err, apperrors.ErrPermissionGroupNotFound)
}

// SEC-FPERM-003：S5 收口与菜单裁剪（view∨add）：有组未命中的成员菜单隐藏
// 表单节点、运行时 403；命中成员可见；管理员（form-data:*）旁路可见
func TestSECFPERM003MenuTrimmingAndEntrance(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	app, form := env.createAppWithForm(t, ctx, env.alphaOwner, "收口应用")
	env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(fpermText("name", "姓名", false), fpermNumber("amount", "金额")))
	created, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "仅查看", Operations: []string{"view"},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.alphaOwner.ID}},
	})
	assert.NoError(t, err)

	// 普通成员未命中：菜单隐藏、运行时 403
	menu, err := env.menuSvc.GetMenu(fpermCtx(env.alpha.ID), env.plainMember, app.Code)
	assert.NoError(t, err)
	assert.False(t, menuHasFormTarget(t, menu, form.Code), "未命中成员的表单节点隐藏")
	_, err = env.formSvc.GetRuntime(fpermCtx(env.alpha.ID), env.plainMember, app.Code, form.Code)
	assert.ErrorIs(t, err, apperrors.ErrPermissionDenied)

	// 普通成员命中（加入主体）后可见
	_, err = env.permSvc.UpdateGroup(ctx, env.alphaOwner, form.Code, created.Code, &model.UpdatePermissionGroupRequest{
		CreatePermissionGroupRequest: model.CreatePermissionGroupRequest{
			Name: created.Name, Operations: []string{"view"},
			SubjectIds: []model.PermissionSubjectInput{
				{Type: model.PermissionSubjectMember, ID: env.alphaOwner.ID},
				{Type: model.PermissionSubjectMember, ID: env.plainMember.ID},
			},
		},
		BaseRevision: created.Revision,
	})
	assert.NoError(t, err)
	menu, err = env.menuSvc.GetMenu(fpermCtx(env.alpha.ID), env.plainMember, app.Code)
	assert.NoError(t, err)
	assert.True(t, menuHasFormTarget(t, menu, form.Code), "命中成员的表单节点可见")
	runtime, err := env.formSvc.GetRuntime(fpermCtx(env.alpha.ID), env.plainMember, app.Code, form.Code)
	assert.NoError(t, err)
	assert.NotNil(t, runtime.Permissions)
	assert.Empty(t, runtime.Permissions.Operations, "view-only 组不含新建记录语义操作")
	assert.False(t, runtime.Permissions.ViewFields["name"].Visible, "S7：组未配置字段矩阵 → deny-by-default")
	assert.False(t, runtime.Permissions.AddFields["name"].Editable)

	// 管理员旁路：alphaOwner 非 S5 命中问题——组存在且 owner 是主体，直接可读
	//（S3 数据面旁路用禁用组场景验证：owner 持 form-data:*，禁用全部组后仍可见）
	_, err = env.permSvc.UpdateGroup(ctx, env.alphaOwner, form.Code, created.Code, &model.UpdatePermissionGroupRequest{
		CreatePermissionGroupRequest: model.CreatePermissionGroupRequest{
			Name: created.Name, Enabled: submitBool(false), Operations: []string{"view"},
			SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.alphaOwner.ID}},
		},
		BaseRevision: 2,
	})
	assert.NoError(t, err)
	runtime, err = env.formSvc.GetRuntime(fpermCtx(env.alpha.ID), env.alphaOwner, app.Code, form.Code)
	assert.NoError(t, err, "禁用组维持收口，但管理员经 form-data:admin 旁路")
	assert.True(t, runtime.Permissions.ViewFields["amount"].Visible)
}

// menuHasFormTarget 判断菜单快照中是否存在指向指定表单编码的资产节点
func menuHasFormTarget(t *testing.T, menu *applicationmodel.MenuSnapshot, formCode string) bool {
	t.Helper()
	for _, detail := range menu.EntryMap {
		if detail.Target != nil && detail.Target.Code == formCode {
			return true
		}
	}
	return false
}

// SEC-FPERM-004：提交执行点：无 add 组 → FORM_PERMISSION_DENIED；命中后
// 权限隐藏字段空 data 通过且旧语义（必填/类型）保留；不可编辑字段携带数据
// 整体拒绝
func TestSECFPERM004SubmitPermissionPipeline(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	app, form := env.createAppWithForm(t, ctx, env.alphaOwner, "提交应用")
	published := env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(
		fpermText("name", "姓名", false),
		fpermNumber("amount", "金额"),
		fpermText("secret", "密级", true),
	))

	envelope := func(entries map[string]string) map[string]model.SubmitFieldValue {
		values := map[string]model.SubmitFieldValue{}
		for _, field := range []string{"name", "amount", "secret"} {
			values[field] = model.SubmitFieldValue{Visible: submitBool(true)}
		}
		for name, data := range entries {
			values[name] = model.SubmitFieldValue{Data: model.JSONContent(data), Visible: submitBool(true)}
		}
		return values
	}
	submit := func(member *iammodel.User, dataOp string, entries map[string]string) error {
		_, err := env.formSvc.SubmitRecord(fpermCtx(env.alpha.ID), member, &model.SubmitRecordRequest{
			AppCode: app.Code, FormCode: form.Code, PublishedVersion: published.PublishedVersion,
			SchemaRevision: published.SchemaRevision, HasResult: submitBool(true),
			DataOpID: dataOp, Values: envelope(entries),
		})
		return err
	}

	// S5：表单存在启用组但普通成员未命中 → 提交拒绝
	_, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "仅管理员", Operations: []string{"add"},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.alphaOwner.ID}},
	})
	assert.NoError(t, err)
	err = submit(env.plainMember, "6e243bbb-7d57-4e59-952b-d530c53c6561", map[string]string{"name": `"李四"`})
	assert.ErrorIs(t, err, apperrors.ErrPermissionDenied)

	// 命中 add 组：不可编辑字段（未配置 → S7 deny-by-default）携带数据整体拒绝
	_, err = env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "录入组", Operations: []string{"add"},
		FieldPermissions: []model.PermissionFieldRule{
			{Field: "name", Visible: true, Editable: true},
			{Field: "amount", Visible: true, Editable: true},
		},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.plainMember.ID}},
	})
	assert.NoError(t, err)
	err = submit(env.plainMember, "6e243bbb-7d57-4e59-952b-d530c53c6562", map[string]string{
		"name": `"李四"`, "secret": `"越权写入"`,
	})
	assert.ErrorIs(t, err, apperrors.ErrRecordInvalid)

	// 权限隐藏字段空 data 通过：记录落库，secret 为 null
	result, err := env.formSvc.SubmitRecord(fpermCtx(env.alpha.ID), env.plainMember, &model.SubmitRecordRequest{
		AppCode: app.Code, FormCode: form.Code, PublishedVersion: published.PublishedVersion,
		SchemaRevision: published.SchemaRevision, HasResult: submitBool(true),
		DataOpID: "6e243bbb-7d57-4e59-952b-d530c53c6563",
		Values:   envelope(map[string]string{"name": `"李四"`, "amount": "88"}),
	})
	assert.NoError(t, err)
	assert.NotZero(t, result.RecordID)
	record, err := env.recordRepo.GetByID(fpermCtx(env.alpha.ID), result.RecordID)
	assert.NoError(t, err)
	var stored map[string]any
	assert.NoError(t, json.Unmarshal([]byte(record.Values), &stored))
	assert.Nil(t, stored["secret"], "不可编辑字段以 null 落库")
}

// SEC-FPERM-REC-001：记录列表必须在数据库分页之前合并 view 数据范围，且
// 不能借跨租户 formCode 或伪造字段名绕过受控查询编译器。
func TestSECFPERMRecordListScopeTenantAndUnknownField(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	app, form := env.createAppWithForm(t, ctx, env.alphaOwner, "记录查询应用")
	published := env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(
		fpermText("name", "姓名", true), fpermRadio("department", "部门"),
	))
	submit := func(operationID, name, department string) {
		_, err := env.formSvc.SubmitRecord(ctx, env.alphaOwner, &model.SubmitRecordRequest{
			AppCode: app.Code, FormCode: form.Code, PublishedVersion: published.PublishedVersion,
			SchemaRevision: published.SchemaRevision, HasResult: submitBool(true), DataOpID: operationID,
			Values: map[string]model.SubmitFieldValue{
				"name":       {Data: model.JSONContent(fmt.Sprintf("%q", name)), Visible: submitBool(true)},
				"department": {Data: model.JSONContent(fmt.Sprintf("%q", department)), Visible: submitBool(true)},
			},
		})
		assert.NoError(t, err)
	}
	submit("87d8ce09-1b7d-4fb9-90b9-5326c04670b1", "销售记录", "sales")
	submit("87d8ce09-1b7d-4fb9-90b9-5326c04670b2", "运营记录", "ops")
	_, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "仅销售数据", Operations: []string{model.PermissionOpView},
		FieldPermissions: []model.PermissionFieldRule{
			{Field: "name", Visible: true}, {Field: "department", Visible: true},
		},
		DataScope: &model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{{
			Field: "department", Operator: "eq", Value: []any{"sales"},
		}}},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.plainMember.ID}},
	})
	assert.NoError(t, err)

	page, err := env.formSvc.ListRecords(ctx, env.plainMember, form.Code, model.RecordQueryDocument{
		Version: 1, Paging: model.RecordQueryPaging{Page: 1, PageSize: 1},
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), page.Total, "total must use the permission predicate before paging")
	assert.Len(t, page.Items, 1)
	assert.Equal(t, "销售记录", page.Items[0].Values["name"])
	noViewMember := env.createPlainMember(t, env.alpha, "fperm-no-record-view")
	page, err = env.formSvc.ListRecords(ctx, noViewMember, form.Code, model.RecordQueryDocument{Version: 1})
	assert.NoError(t, err)
	assert.Empty(t, page.Items, "unmatched view member must not receive rows")
	assert.Zero(t, page.Total)

	_, err = env.formSvc.ListRecords(fpermCtx(env.beta.ID), env.betaOwner, form.Code, model.RecordQueryDocument{Version: 1})
	assert.ErrorIs(t, err, apperrors.ErrFormNotFound, "tenant callback must hide alpha form and records")
	_, err = env.formSvc.ListRecords(ctx, env.plainMember, form.Code, model.RecordQueryDocument{
		Version: 1, Filter: &model.RecordQueryExpression{Type: "condition", Field: "values ->> 'secret'", Operator: "eq", Value: "x"},
	})
	assert.ErrorIs(t, err, apperrors.ErrRecordQueryInvalid)
}

// SEC-FPERM-005：仅 add 成员入口与填写模式（S8）：菜单可见、运行时
// operations=[add,import]、viewFields 全拒绝、addFields 全量放行
func TestSECFPERM005AddOnlyEntrance(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	app, form := env.createAppWithForm(t, ctx, env.alphaOwner, "仅录入应用")
	env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(fpermText("name", "姓名", false), fpermNumber("amount", "金额")))
	_, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "仅录入", Operations: []string{"add"},
		FieldPermissions: []model.PermissionFieldRule{
			{Field: "name", Visible: true, Editable: true},
			{Field: "amount", Visible: true, Editable: true},
		},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.plainMember.ID}},
	})
	assert.NoError(t, err)

	runtime, err := env.formSvc.GetRuntime(fpermCtx(env.alpha.ID), env.plainMember, app.Code, form.Code)
	assert.NoError(t, err)
	assert.NotNil(t, runtime.Permissions)
	assert.Equal(t, []string{"add"}, runtime.Permissions.Operations, "组仅授 add：投影不含 import")
	assert.False(t, runtime.Permissions.ViewFields["name"].Visible, "仅 add 成员无查看入口：viewFields 全拒绝")
	assert.True(t, runtime.Permissions.AddFields["name"].Editable, "填写模式按 addFields 放行")
}

// SEC-FPERM-006：switch-type 阻塞与放行两向（§3.3）：含 workflow_* 操作的组
// 阻塞 workflow→standard；清理后放行；standard→workflow 不阻塞
func TestSECFPERM006SwitchTypeBlocked(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	_, form := env.createAppWithForm(t, ctx, env.alphaOwner, "切换应用")
	env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(fpermText("name", "姓名", false)))
	// 切为流程表单
	_, err := env.formSvc.SwitchType(ctx, env.alphaOwner, form.Code, &model.SwitchFormTypeRequest{FormType: model.FormTypeWorkflow})
	assert.NoError(t, err)
	created, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "流程操作组", Operations: []string{"view", model.PermissionOpWorkflowTerminate},
	})
	assert.NoError(t, err)

	_, err = env.formSvc.SwitchType(ctx, env.alphaOwner, form.Code, &model.SwitchFormTypeRequest{FormType: model.FormTypeStandard})
	assert.ErrorIs(t, err, apperrors.ErrPermissionBlockedTypeSwitch)

	// 管理员清理组后放行
	assert.NoError(t, env.permSvc.DeleteGroup(ctx, env.alphaOwner, form.Code, created.Code))
	updated, err := env.formSvc.SwitchType(ctx, env.alphaOwner, form.Code, &model.SwitchFormTypeRequest{FormType: model.FormTypeStandard})
	assert.NoError(t, err)
	assert.Equal(t, model.FormTypeStandard, updated.FormType)

	// standard → workflow 不阻塞（合法集为超集）
	_, err = env.formSvc.SwitchType(ctx, env.alphaOwner, form.Code, &model.SwitchFormTypeRequest{FormType: model.FormTypeWorkflow})
	assert.NoError(t, err)
}

// SEC-FPERM-007：发布阻塞（§5.2 字段生命周期）：删除被启用组 data_scope
// 引用的字段 → FORM_PERMISSION_BLOCKED_PUBLISH 且 data.fields 列出冲突字段；
// 调整权限组后放行；禁用组引用不阻塞
func TestSECFPERM007PublishBlocked(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	_, form := env.createAppWithForm(t, ctx, env.alphaOwner, "发布阻塞应用")
	env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(fpermText("name", "姓名", false), fpermNumber("amount", "金额")))
	_, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "金额范围组", Operations: []string{"view"},
		DataScope: &model.PermissionDataScopeSpec{
			Match: model.PermissionScopeMatchAll,
			Conditions: []model.PermissionDataCondition{
				{Field: "amount", Operator: "gte", Value: []any{float64(100)}},
			},
		},
	})
	assert.NoError(t, err)

	// 新草稿删除 amount → 发布阻塞
	detail, err := env.formSvc.Get(ctx, env.alphaOwner, form.Code)
	assert.NoError(t, err)
	blockedDoc := fpermDoc(fpermText("name", "姓名", false))
	_, err = env.formSvc.SaveDraft(ctx, env.alphaOwner, form.Code, &model.SaveDraftRequest{
		DraftRevision: detail.DraftRevision, ProtocolVersion: model.CurrentProtocolVersion,
		Content: model.JSONContent(blockedDoc),
	})
	assert.NoError(t, err)
	_, err = env.formSvc.Publish(ctx, env.alphaOwner, form.Code, &model.PublishRequest{DraftRevision: detail.DraftRevision + 1})
	assert.ErrorIs(t, err, apperrors.ErrPermissionBlockedPublish)

	// 禁用组引用不阻塞（EnabledDataScopeFields 仅启用组）
	groups, err := env.permSvc.ListGroups(ctx, env.alphaOwner, form.Code)
	assert.NoError(t, err)
	_, err = env.permSvc.UpdateGroup(ctx, env.alphaOwner, form.Code, groups[0].Code, &model.UpdatePermissionGroupRequest{
		CreatePermissionGroupRequest: model.CreatePermissionGroupRequest{
			Name: groups[0].Name, Enabled: submitBool(false), Operations: []string{"view"},
		},
		BaseRevision: groups[0].Revision,
	})
	assert.NoError(t, err)
	result, err := env.formSvc.Publish(ctx, env.alphaOwner, form.Code, &model.PublishRequest{DraftRevision: detail.DraftRevision + 1})
	assert.NoError(t, err)
	assert.Equal(t, 2, result.PublishedVersion)
}

// SEC-FPERM-008：S2 组绑定串联越权回归（真库判定器路径）：A 组编辑 X 范围 +
// B 组查看 Y 范围 → AllowsOperation(edit, Y) 为假（内存匹配镜像 §5.2）
func TestSECFPERM008GroupBindingNoChain(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	_, form := env.createAppWithForm(t, ctx, env.alphaOwner, "绑定应用")
	env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(fpermRadio("dept", "部门"), fpermNumber("amount", "金额")))
	// A 组：edit + dept in [sales]；B 组：view + dept in [ops]
	_, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "A 编辑销售", Operations: []string{"edit"},
		DataScope: &model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{
			{Field: "dept", Operator: "in", Value: []any{"sales"}},
		}},
	})
	assert.NoError(t, err)
	_, err = env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "B 查看运营", Operations: []string{"view"},
		DataScope: &model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{
			{Field: "dept", Operator: "in", Value: []any{"ops"}},
		}},
	})
	assert.NoError(t, err)

	formRow, err := env.formRepo.GetByCode(ctx, form.Code)
	assert.NoError(t, err)
	resolved, err := env.evaluator.Evaluate(ctx, env.plainMember, formRow.ID)
	assert.NoError(t, err)
	// plainMember 未命中任何主体 → S5 全拒绝（入口关闭）
	assert.False(t, resolved.EntranceAllowed())

	_, err = env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "A2 编辑销售", Operations: []string{"edit"},
		DataScope: &model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{
			{Field: "dept", Operator: "in", Value: []any{"sales"}},
		}},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.plainMember.ID}},
	})
	assert.NoError(t, err)
	_, err = env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "B2 查看运营", Operations: []string{"view"},
		DataScope: &model.PermissionDataScopeSpec{Conditions: []model.PermissionDataCondition{
			{Field: "dept", Operator: "in", Value: []any{"ops"}},
		}},
		SubjectIds: []model.PermissionSubjectInput{{Type: model.PermissionSubjectMember, ID: env.plainMember.ID}},
	})
	assert.NoError(t, err)
	resolved, err = env.evaluator.Evaluate(ctx, env.plainMember, formRow.ID)
	assert.NoError(t, err)
	yRecord := map[string]any{"dept": "ops", "amount": float64(100)}
	assert.True(t, resolved.AllowsOperation("view", yRecord))
	assert.False(t, resolved.AllowsOperation("edit", yRecord), "串联越权回归：B 组范围不得借 A 组 edit 放行")
	assert.True(t, resolved.AllowsOperation("edit", map[string]any{"dept": "sales"}))
}

// SEC-FPERM-009：审计事件落库（事务提交后 best-effort 写入）
func TestSECFPERM009AuditEvents(t *testing.T) {
	env := newFpermEnv(t)
	ctx := fpermCtx(env.alpha.ID)
	_, form := env.createAppWithForm(t, ctx, env.alphaOwner, "审计应用")
	env.saveAndPublish(t, ctx, env.alphaOwner, form, fpermDoc(fpermText("name", "姓名", false)))
	created, err := env.permSvc.CreateGroup(ctx, env.alphaOwner, form.Code, &model.CreatePermissionGroupRequest{
		Name: "审计组", Operations: []string{"view"},
	})
	assert.NoError(t, err)
	assert.NoError(t, env.permSvc.DeleteGroup(ctx, env.alphaOwner, form.Code, created.Code))

	var count int64
	assert.NoError(t, env.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM tn_audit_logs WHERE module = 'form' AND resource_type = 'form_permission_group' AND action = 'create'`,
	).Scan(&count).Error)
	assert.GreaterOrEqual(t, count, int64(1), "权限组创建审计事件落库")
}
