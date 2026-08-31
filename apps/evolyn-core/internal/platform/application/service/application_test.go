package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/contextx"
	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/repository"
	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- M2-A Service 单元测试（真实 PostgreSQL 链路见 sec_app_integration_test.go）----

// passThroughTx 不携带事务语义、直接执行 fn
type passThroughTx struct{}

func (passThroughTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeAppRepo 应用仓储桩：记录写入与字段更新，可注入读取结果
type fakeAppRepo struct {
	apps       map[uint]*model.Application
	installs   map[uint]*model.Installation
	updated    map[uint]map[string]interface{}
	deleted    map[uint]bool
	nextID     uint
	getByIDErr error
}

func newFakeAppRepo() *fakeAppRepo {
	return &fakeAppRepo{
		apps:     map[uint]*model.Application{},
		installs: map[uint]*model.Installation{},
		updated:  map[uint]map[string]interface{}{},
		deleted:  map[uint]bool{},
		nextID:   100,
	}
}

func (f *fakeAppRepo) Create(ctx context.Context, app *model.Application) (*model.Application, error) {
	f.nextID++
	app.ID = f.nextID
	clone := *app
	f.apps[app.ID] = &clone
	return &clone, nil
}

func (f *fakeAppRepo) CreateInstallation(ctx context.Context, inst *model.Installation) error {
	clone := *inst
	f.installs[inst.ApplicationID] = &clone
	return nil
}

func (f *fakeAppRepo) GetByID(ctx context.Context, id uint) (*model.Application, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	if app, ok := f.apps[id]; ok && !f.deleted[id] {
		clone := *app
		return &clone, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// GetByCode 按 code 遍历匹配（fake 无索引，语义与真实部分唯一索引一致）
func (f *fakeAppRepo) GetByCode(ctx context.Context, code string) (*model.Application, error) {
	for _, app := range f.apps {
		if app.Code == code && !f.deleted[app.ID] {
			clone := *app
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeAppRepo) List(ctx context.Context, params repository.ListParams) ([]model.Application, bool, error) {
	return nil, false, nil
}

func (f *fakeAppRepo) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	app, ok := f.apps[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	for k, v := range fields {
		f.setField(app, k, v)
	}
	f.updated[id] = fields
	return nil
}

func (f *fakeAppRepo) setField(app *model.Application, key string, value interface{}) {
	switch key {
	case "name":
		app.Name = value.(string)
	case "icon":
		app.Icon = value.(model.ApplicationIcon)
	case "color":
		app.Color = value.(string)
	case "status":
		app.Status = value.(string)
	case "sort_order":
		app.SortOrder = value.(int64)
	}
}

func (f *fakeAppRepo) SoftDelete(ctx context.Context, app *model.Application) error {
	f.deleted[app.ID] = true
	return nil
}

func (f *fakeAppRepo) CountBillableByTenant(ctx context.Context, tenantID uint) (int64, error) {
	var count int64
	for _, app := range f.apps {
		if app.TenantID == tenantID && !f.deleted[app.ID] {
			count++
		}
	}
	return count, nil
}

func (f *fakeAppRepo) Migrate() error { return nil }

// fakeQuota 可配置失败/透传的配额桩
type fakeQuota struct {
	err error
}

func (q fakeQuota) Check(ctx context.Context, tenantID uint, key string) error { return q.err }
func (q fakeQuota) Usage(ctx context.Context, tenantID uint, key string) (int64, error) {
	return 0, nil
}
func (q fakeQuota) CheckAndReserve(ctx context.Context, tenantID uint, key string, fn func(ctx context.Context) error) error {
	if q.err != nil {
		return q.err
	}
	return fn(ctx)
}

// fakeAudit 审计桩：记录事件顺序（验证「事务提交后才审计」）
type fakeAudit struct {
	entries []auditservice.Entry
}

func (a *fakeAudit) Record(ctx context.Context, e auditservice.Entry) {
	a.entries = append(a.entries, e)
}

// fakeAccess 访问判定桩：直接返回可配置权限集（真实 RBAC 同源合并链路
// 见 access.go 与集成测试）
type fakeAccess struct {
	perms map[string]bool
}

func (f fakeAccess) Permissions(ctx context.Context, member *iammodel.User) map[string]bool {
	return f.perms
}

// fullPerms 全量权限集（模拟 tenant-admin 的 applications:* 规则展开）
func fullPerms() map[string]bool {
	return map[string]bool{
		"applications:get": true, "applications:list": true,
		"applications:create": true, "applications:patch": true,
		"applications:delete": true,
	}
}

// fakeGroupLoader 系统组加载桩：返回可配置的 authenticated 组（含角色）
type fakeGroupLoader struct {
	group *iammodel.Group
	err   error
}

func (f fakeGroupLoader) GetGroupByName(ctx context.Context, name string) (*iammodel.Group, error) {
	return f.group, f.err
}

// fakeMemberLoader 成员重载桩：按 ID 返回数据库侧的真实成员快照
// （evaluator 不信任调用方传入对象，角色一律以重载结果为准）
type fakeMemberLoader struct {
	members map[uint]*iammodel.User
	err     error
}

func (f fakeMemberLoader) GetUserByID(ctx context.Context, id uint) (*iammodel.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	if m, ok := f.members[id]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// TestRBACAccessEvaluatorGuard evaluator 权限来源守卫：按 ID 重载成员后
// 计算权限——伪造的 TenantID/Roles 不生效；未认证/无租户上下文/成员
// 不存在一律空集；本租户成员合并 authenticated 基线规则
func TestRBACAccessEvaluatorGuard(t *testing.T) {
	// authenticated 基线组：applications:view（展开为 get+list）
	authGroup := &iammodel.Group{
		Name: iammodel.AuthenticatedGroup,
		Roles: []iammodel.Role{{Rules: iammodel.Rules{
			{Resource: "applications", Operation: iammodel.ViewOperation},
		}}},
	}
	// 数据库侧真实成员：8 = 本租户普通成员（无显式角色）；9 = 他租成员
	store := map[uint]*iammodel.User{
		8: {Nickname: "member-a"},
		9: {Nickname: "member-b"},
	}
	store[8].ID, store[8].TenantID = 8, 1
	store[9].ID, store[9].TenantID = 9, 2

	evaluator := NewRBACAccessEvaluator(
		fakeMemberLoader{members: store},
		fakeGroupLoader{group: authGroup},
	)

	t.Run("伪造通配角色的快照被忽略（以重载角色为准）", func(t *testing.T) {
		// 调用方构造：真实成员 ID + 伪造 applications:* 角色
		forged := alphaMember(iammodel.Rule{Resource: "*", Operation: iammodel.AllOperation})
		perms := evaluator.Permissions(alphaCtx(), forged)
		assert.True(t, perms["applications:get"], "基线 view 经系统组合并")
		assert.True(t, perms["applications:list"])
		assert.False(t, perms["applications:patch"], "伪造的通配角色不得生效")
		assert.False(t, perms["applications:delete"])
	})

	t.Run("重载结果租户不符兜底拒绝", func(t *testing.T) {
		// ID 9 重载后属于租户 2（真实仓储下该重载已被租户过滤拦截）
		assert.Empty(t, evaluator.Permissions(alphaCtx(), &iammodel.User{ID: 9}))
	})

	t.Run("成员不存在（含跨租户 ID 被过滤）权限集为空", func(t *testing.T) {
		ghost := alphaMember(iammodel.Rule{Resource: "*", Operation: iammodel.AllOperation})
		ghost.ID = 404
		assert.Empty(t, evaluator.Permissions(alphaCtx(), ghost))
	})

	t.Run("未认证成员权限集为空", func(t *testing.T) {
		assert.Empty(t, evaluator.Permissions(alphaCtx(), nil))
		zero := alphaMember()
		zero.ID = 0
		assert.Empty(t, evaluator.Permissions(alphaCtx(), zero))
	})

	t.Run("无租户上下文权限集为空", func(t *testing.T) {
		assert.Empty(t, evaluator.Permissions(context.Background(), alphaMember()))
	})

	t.Run("成员重载失败权限集为空", func(t *testing.T) {
		broken := NewRBACAccessEvaluator(
			fakeMemberLoader{err: assert.AnError},
			fakeGroupLoader{group: authGroup},
		)
		assert.Empty(t, broken.Permissions(alphaCtx(), alphaMember()))
	})

	t.Run("系统组读取失败降级为重载后的显式角色集", func(t *testing.T) {
		admin := &iammodel.User{Nickname: "admin"}
		admin.ID, admin.TenantID = 8, 1
		admin.Roles = []iammodel.Role{{Rules: iammodel.Rules{
			{Resource: "applications", Operation: iammodel.AllOperation},
		}}}
		degraded := NewRBACAccessEvaluator(
			fakeMemberLoader{members: map[uint]*iammodel.User{8: admin}},
			fakeGroupLoader{err: assert.AnError},
		)
		perms := degraded.Permissions(alphaCtx(), &iammodel.User{ID: 8})
		assert.True(t, perms["applications:patch"], "显式角色不受系统组降级影响")
		assert.True(t, perms["applications:get"])
	})
}

func newTestService(repo repository.ApplicationRepository, quota fakeQuota, audit auditservice.Recorder, access ApplicationAccessEvaluator) ApplicationService {
	return NewApplicationService(passThroughTx{}, repo, quota, audit, access)
}

// alphaMember 构造租户 1 的成员（可带角色规则）
func alphaMember(rules ...iammodel.Rule) *iammodel.User {
	member := &iammodel.User{Nickname: "member-a"}
	member.ID = 8
	member.TenantID = 1
	if len(rules) > 0 {
		member.Roles = []iammodel.Role{{ID: 1, Name: "r", Rules: rules}}
	}
	return member
}

func alphaCtx() context.Context {
	return contextx.NewTenantContext(context.Background(), 1)
}

func TestCreateBlankValidation(t *testing.T) {
	svc := newTestService(newFakeAppRepo(), fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
	member := alphaMember()
	ctx := alphaCtx()

	cases := []struct {
		name string
		req  model.CreateBlankRequest
		want error
	}{
		{"空名称", model.CreateBlankRequest{Name: "  "}, apperrors.ErrNameInvalid},
		{"名称超长", model.CreateBlankRequest{Name: string(make([]byte, 129))}, apperrors.ErrNameInvalid},
		{"非法图标", model.CreateBlankRequest{Name: "应用", Icon: &model.ApplicationIcon{Type: "remix", Name: "RiBookmarkFill"}}, apperrors.ErrIconInvalid},
		{"非法颜色", model.CreateBlankRequest{Name: "应用", Color: "#409eff"}, apperrors.ErrColorInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateBlank(ctx, member, &tc.req)
			assert.True(t, errors.Is(err, tc.want), "got: %v", err)
		})
	}
}

func TestCreateBlankMemberGuard(t *testing.T) {
	repo := newFakeAppRepo()
	svc := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})

	// 跨租户成员：ctx 是租户 1，成员属于租户 2 → 拒绝且不落任何行
	beta := alphaMember()
	beta.TenantID = 2
	_, err := svc.CreateBlank(alphaCtx(), beta, &model.CreateBlankRequest{Name: "应用"})
	assert.True(t, errors.Is(err, apperrors.ErrMemberInvalid))
	assert.Empty(t, repo.apps)

	// 无成员（未认证路径防御）
	_, err = svc.CreateBlank(alphaCtx(), nil, &model.CreateBlankRequest{Name: "应用"})
	assert.True(t, errors.Is(err, apperrors.ErrMemberInvalid))
}

func TestCreateBlankSuccess(t *testing.T) {
	repo := newFakeAppRepo()
	audit := &fakeAudit{}
	svc := newTestService(repo, fakeQuota{}, audit, fakeAccess{perms: fullPerms()})

	detail, err := svc.CreateBlank(alphaCtx(), alphaMember(), &model.CreateBlankRequest{Name: " 客户管理 ", Color: ""})
	assert.NoError(t, err)

	// 应用记录：blank/self 语义 + 同步 ready；名称去空格、外观取默认
	assert.NotEmpty(t, detail.ID)
	assert.Equal(t, "客户管理", detail.Name)
	assert.Equal(t, model.ApplicationIcon{Type: "remix", Name: "bookmark", Background: "#f7be54,#eda426"}, detail.Icon)
	assert.Equal(t, "primary", detail.Color)
	assert.Equal(t, model.SourceTypeBlank, detail.Source.Type)
	assert.Equal(t, model.InstallChannelSelf, detail.Source.Channel)
	assert.Equal(t, model.ApplicationStatusActive, detail.Status)
	assert.Equal(t, model.ProvisionStatusReady, detail.ProvisionStatus)
	assert.Equal(t, model.ApplicationHomeModeBuilder, detail.HomeMode)

	// 安装记录成对写入
	app := repo.apps[detail.ID]
	assert.NotNil(t, app)
	inst := repo.installs[detail.ID]
	assert.NotNil(t, inst)
	assert.Equal(t, model.SourceTypeBlank, inst.SourceType)
	assert.Equal(t, model.InstallChannelSelf, inst.Channel)

	// 审计一次 create 事件
	assert.Len(t, audit.entries, 1)
	assert.Equal(t, "create", audit.entries[0].Action)
	assert.Equal(t, "application", audit.entries[0].ResourceType)

	// owner 视角 capabilities 全开（M2-A）
	assert.True(t, detail.Capabilities.View)
	assert.True(t, detail.Capabilities.Edit)
	assert.True(t, detail.Capabilities.Delete)
}

func TestCreateBlankQuotaExceededRollback(t *testing.T) {
	repo := newFakeAppRepo()
	audit := &fakeAudit{}
	svc := newTestService(repo, fakeQuota{err: tenantservice.ErrQuotaExceeded}, audit, fakeAccess{perms: fullPerms()})

	_, err := svc.CreateBlank(alphaCtx(), alphaMember(), &model.CreateBlankRequest{Name: "应用"})
	assert.True(t, errors.Is(err, tenantservice.ErrQuotaExceeded))
	// 配额拒绝：应用/安装记录均未写入，也不记审计
	assert.Empty(t, repo.apps)
	assert.Empty(t, repo.installs)
	assert.Empty(t, audit.entries)
}

func TestUpdateWhitelistAndTransitions(t *testing.T) {
	repo := newFakeAppRepo()
	svc := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
	created, err := svc.CreateBlank(alphaCtx(), alphaMember(), &model.CreateBlankRequest{Name: "应用"})
	assert.NoError(t, err)

	// 白名单字段更新
	newName := "客户管理"
	updated, err := svc.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{Name: &newName})
	assert.NoError(t, err)
	assert.Equal(t, "客户管理", updated.Name)

	// 归档：status active→archived
	archived := model.ApplicationStatusArchived
	updated, err = svc.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{Status: &archived})
	assert.NoError(t, err)
	assert.Equal(t, model.ApplicationStatusArchived, updated.Status)
	// 归档态不可编辑（capabilities 派生）
	assert.False(t, updated.Capabilities.Edit)

	// 恢复：archived→active
	active := model.ApplicationStatusActive
	updated, err = svc.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{Status: &active})
	assert.NoError(t, err)
	assert.Equal(t, model.ApplicationStatusActive, updated.Status)

	// 非法状态值拒绝
	bad := "deleted"
	_, err = svc.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{Status: &bad})
	assert.True(t, errors.Is(err, apperrors.ErrStatusInvalid))

	// 空请求幂等返回
	same, err := svc.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{})
	assert.NoError(t, err)
	assert.Equal(t, updated.Name, same.Name)
}

func TestProvisioningGuard(t *testing.T) {
	repo := newFakeAppRepo()
	svc := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
	created, err := svc.CreateBlank(alphaCtx(), alphaMember(), &model.CreateBlankRequest{Name: "应用"})
	assert.NoError(t, err)

	// 直接改桩数据模拟异步实例化进行中（M2-C 场景）
	repo.apps[created.ID].ProvisionStatus = model.ProvisionStatusPending

	_, err = svc.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{})
	assert.True(t, errors.Is(err, apperrors.ErrProvisioning))

	err = svc.Delete(alphaCtx(), alphaMember(), created.ID)
	assert.True(t, errors.Is(err, apperrors.ErrProvisioning))
}

func TestGetNotFoundWrapped(t *testing.T) {
	svc := newTestService(newFakeAppRepo(), fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
	_, err := svc.Get(alphaCtx(), alphaMember(), 404)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
}

// TestGetInfraErrorNotNotFound 基础设施错误（连接中断/超时等）不得误映射
// 为 404：原样上抛，由 Controller 统一出口脱敏为 500——否则「更新成功后
// 重载失败」会被客户端误读为「应用不存在」
func TestGetInfraErrorNotNotFound(t *testing.T) {
	repo := newFakeAppRepo()
	repo.getByIDErr = errors.New("connection refused")
	svc := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})

	_, err := svc.Get(alphaCtx(), alphaMember(), 1)
	assert.False(t, errors.Is(err, apperrors.ErrNotFound), "基础设施错误不得映射为 APP_NOT_FOUND")
	assert.EqualError(t, err, "connection refused")
}

// TestGetByCode 按 code 查详情：与按 ID 查询同口径（定位、权限复核、
// 出网 capabilities）
func TestGetByCode(t *testing.T) {
	repo := newFakeAppRepo()
	seed := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
	created, err := seed.CreateBlank(alphaCtx(), alphaMember(), blankReq("应用"))
	assert.NoError(t, err)

	t.Run("code 命中返回与 ID 查询一致的详情", func(t *testing.T) {
		svc := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
		detail, err := svc.GetByCode(alphaCtx(), alphaMember(), created.Code)
		assert.NoError(t, err)
		assert.Equal(t, created.ID, detail.ID)
		assert.Equal(t, created.Code, detail.Code)
		assert.True(t, detail.Capabilities.View)
	})

	t.Run("code 不存在 NotFound", func(t *testing.T) {
		svc := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
		_, err := svc.GetByCode(alphaCtx(), alphaMember(), "app_notexist")
		assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	})

	t.Run("无 get 权限拒绝", func(t *testing.T) {
		svc := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: map[string]bool{}})
		_, err := svc.GetByCode(alphaCtx(), alphaMember(), created.Code)
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))
	})
}

func TestCapabilitiesByPermission(t *testing.T) {
	// capabilities 严格等于「有效权限集 ∩ 应用状态」：owner 不放大能力
	//（与路由 RBAC 拒绝结果保持一致），权限集由 AccessEvaluator 提供
	viewOnly := map[string]bool{"applications:get": true, "applications:list": true}

	cases := []struct {
		name       string
		perms      map[string]bool
		appStatus  string
		wantView   bool
		wantEdit   bool
		wantDelete bool
	}{
		{"全量权限 + active", fullPerms(), model.ApplicationStatusActive, true, true, true},
		{"只读权限（authenticated 基线）", viewOnly, model.ApplicationStatusActive, true, false, false},
		{"无权限", map[string]bool{}, model.ApplicationStatusActive, false, false, false},
		{"全量权限 + 归档态（不可编辑）", fullPerms(), model.ApplicationStatusArchived, true, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := &model.Application{
				ID: 1, Status: tc.appStatus, ProvisionStatus: model.ProvisionStatusReady,
				HomeMode: model.ApplicationHomeModeApplication,
			}
			svc := newTestService(newFakeAppRepo(), fakeQuota{}, nil, fakeAccess{perms: tc.perms}).(*applicationService)
			detail := svc.detailFor(tc.perms, app)
			assert.Equal(t, tc.wantView, detail.Capabilities.View)
			assert.Equal(t, tc.wantEdit, detail.Capabilities.Edit)
			assert.Equal(t, tc.wantDelete, detail.Capabilities.Delete)
			assert.Equal(t, model.ApplicationHomeModeApplication, detail.HomeMode)
		})
	}
}

// TestWritePathAccessGuards 写路径 Service 复核：权限缺失直接 403，
// 不依赖 HTTP 中间件（内部调用路径同样受控，§9.1）
// viewOnly 只读权限集（authenticated 基线形态）
var viewOnly = fakeAccess{perms: map[string]bool{"applications:get": true, "applications:list": true}}

func TestWritePathAccessGuards(t *testing.T) {

	t.Run("Get 无 get 权限拒绝", func(t *testing.T) {
		repo := newFakeAppRepo()
		seed := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
		seeded, err := seed.CreateBlank(alphaCtx(), alphaMember(), blankReq("应用"))
		assert.NoError(t, err)

		noPerm := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: map[string]bool{}})
		_, err = noPerm.Get(alphaCtx(), alphaMember(), seeded.ID)
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))
	})

	t.Run("CreateBlank 无 create 权限拒绝且不落库", func(t *testing.T) {
		repo := newFakeAppRepo()
		readonly := newTestService(repo, fakeQuota{}, nil, viewOnly)
		_, err := readonly.CreateBlank(alphaCtx(), alphaMember(), blankReq("应用"))
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))
		assert.Empty(t, repo.apps)
		assert.Empty(t, repo.installs)
	})

	t.Run("List 无 list 权限拒绝", func(t *testing.T) {
		repo := newFakeAppRepo()
		seed := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
		_, err := seed.CreateBlank(alphaCtx(), alphaMember(), blankReq("应用"))
		assert.NoError(t, err)

		noList := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: map[string]bool{"applications:get": true}})
		_, err = noList.List(alphaCtx(), alphaMember(), model.ListApplicationsQuery{})
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))
	})

	t.Run("仅 update 动词不放行编辑（路由只认 patch）", func(t *testing.T) {
		repo := newFakeAppRepo()
		seed := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
		created, err := seed.CreateBlank(alphaCtx(), alphaMember(), blankReq("应用"))
		assert.NoError(t, err)

		updateOnly := newTestService(repo, fakeQuota{}, nil,
			fakeAccess{perms: map[string]bool{"applications:update": true, "applications:get": true}})
		_, err = updateOnly.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{})
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))

		// capabilities.edit 同口径：仅 update 动词不产生编辑能力
		svc := updateOnly.(*applicationService)
		detail := svc.detailFor(map[string]bool{"applications:update": true}, repo.apps[created.ID])
		assert.False(t, detail.Capabilities.Edit)
	})

	t.Run("Update/Delete 无写权限拒绝且不落库", func(t *testing.T) {
		repo := newFakeAppRepo()
		seed := newTestService(repo, fakeQuota{}, nil, fakeAccess{perms: fullPerms()})
		created, err := seed.CreateBlank(alphaCtx(), alphaMember(), blankReq("应用"))
		assert.NoError(t, err)

		readonly := newTestService(repo, fakeQuota{}, nil, viewOnly)
		_, err = readonly.Update(alphaCtx(), alphaMember(), created.ID, &model.UpdateApplicationRequest{})
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))

		err = readonly.Delete(alphaCtx(), alphaMember(), created.ID)
		assert.True(t, errors.Is(err, apperrors.ErrForbidden))
		assert.False(t, repo.deleted[created.ID])
	})
}

func TestCursorCodec(t *testing.T) {
	cursor := encodeListCursor(100, 101)
	sort, id, has, err := decodeListCursor(cursor)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, int64(100), sort)
	assert.Equal(t, uint(101), id)

	// 空游标 = 首页
	_, _, has, err = decodeListCursor("")
	assert.NoError(t, err)
	assert.False(t, has)

	// 非法游标（非 base64 / 缺 ID）一律拒绝，且不得携带部分解析结果
	sort, id, has, err = decodeListCursor("!!not-base64!!")
	assert.Error(t, err)
	assert.Zero(t, sort)
	assert.Zero(t, id)
	assert.False(t, has)
	sort, id, has, err = decodeListCursor(encodeListCursor(0, 0))
	assert.Error(t, err)
	assert.Zero(t, sort)
	assert.Zero(t, id)
	assert.False(t, has)
}
