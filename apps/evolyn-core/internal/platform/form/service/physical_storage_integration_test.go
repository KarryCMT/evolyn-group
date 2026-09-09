// 物理表存储真库集成测试（方案 §16 验收）：发布 → DDL Worker → 物理表/RLS
// 真实落库 → 提交（values=NULL + 物理行）→ 列表 → 审批写回的完整链路，
// 以及跨租户 RLS 隔离、Query DSL 注入面、子表单原子性与并发发布收口。
// 服务装配用真仓储 + 真 TxManager + dynamicddl 执行器；权限/配额/应用端口
// 用内存桩放行（聚焦存储链路本身）。
package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"evolyn/internal/infrastructure"
	"evolyn/internal/infrastructure/dynamicddl"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/form/worker"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/testsupport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// physEnv 真库物理链路环境。
type physEnv struct {
	db             *gorm.DB
	tx             *infrastructure.TxManager
	mu             sync.Mutex
	probeRoleReady bool
	formSvc        FormService
	recordRepo     repository.FormRecordRepository
	storageRepo    repository.FormStorageRepository
	schemaRepo     repository.StorageSchemaVersionRepository
	childRepo      repository.StorageChildRepository
	ddlWorker      *worker.DDLJobWorker
}

// newPhysEnv 组装真库 + 真物理链路。
func newPhysEnv(t *testing.T) *physEnv {
	t.Helper()
	db := testsupport.NewPostgres(t)
	formRepo := repository.NewRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	recordRepo := repository.NewRecordRepository(db)
	storageRepo := repository.NewStorageRepository(db)
	schemaRepo := repository.NewStorageSchemaVersionRepository(db)
	jobRepo := repository.NewDDLJobRepository(db)
	childRepo := repository.NewStorageChildRepository(db)
	physicalRepo := repository.NewPhysicalValueRepository(db)
	txManager := infrastructure.NewTxManager(db)

	svc := NewFormService(
		txManager, formRepo, versionRepo, recordRepo,
		&fakeQuota{limit: -1}, &fakeRecorder{}, fakeAccess{perms: adminPerms},
		fakeApps{apps: map[uint]string{7: "active"}, codeToID: map[string]uint{"app_phys": 7}},
		&fakeMenuPort{},
	)
	if injector, ok := svc.(PhysicalStorageInjector); ok {
		injector.UsePhysicalStorage(storageRepo, schemaRepo, jobRepo, physicalRepo)
	}
	if injector, ok := svc.(StorageChildInjector); ok {
		injector.UseStorageChildren(childRepo)
	}
	ddlWorker := worker.NewDDLJobWorker(
		txManager, jobRepo, schemaRepo, storageRepo, versionRepo, formRepo,
		dynamicddl.NewExecutor(db), nil,
	)
	return &physEnv{
		db: db, tx: txManager, formSvc: svc, recordRepo: recordRepo,
		storageRepo: storageRepo, schemaRepo: schemaRepo, childRepo: childRepo,
		ddlWorker: ddlWorker,
	}
}

// physDoc v8 发布草稿（text + number + date 三标量字段）。
func physDoc(items string, names string) model.JSONContent {
	return model.JSONContent(fmt.Sprintf(
		`{"content":{"type":"form","layout":"grid-2","items":%s,"layout_fields":[],"field_layout":[%s],"fieldShowRules":[],"submitRule":2,"widget_submit_rules":{},"validators":[],"preSubmitConfirm":{"enable":false,"title":"请确认提交","content":"确认提交当前内容？"}}}`,
		items, names))
}

var physScalarItems = `[
	{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
	{"widget":{"type":"number","widgetName":"_widget_n","fieldId":"aaaaaaaa02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6},
	{"widget":{"type":"datetime","widgetName":"_widget_d","fieldId":"aaaaaaaa03","enable":true,"visible":true,"allowBlank":true,"format":"date"},"label":"日期","description":"","labelHidden":false,"lineWidth":6}]`

// publishPhysical 发布并执行 DDL 到就绪（返回表单与物理表名）。
func (env *physEnv) publishPhysical(t *testing.T, items, names string, member *iammodel.User) (string, string) {
	t.Helper()
	ctx := tenantCtx(member.TenantID)
	created, err := env.formSvc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: 7, Name: "物理链路", FormType: model.FormTypeStandard,
	})
	require.NoError(t, err)
	saved, err := env.formSvc.SaveDraft(ctx, member, created.Code, &model.SaveDraftRequest{
		DraftRevision: 1, ProtocolVersion: model.CurrentProtocolVersion, Content: physDoc(items, names),
	})
	require.NoError(t, err)
	published, err := env.formSvc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	require.NoError(t, err)
	require.True(t, published.Async, "structural change must publish async")

	// 执行 DDL Job（真库：claim+执行+回写同事务）
	processed, err := env.ddlWorker.ProcessOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed, "ddl job must be claimed and executed")

	binding, err := env.storageRepo.GetByFormID(ctx, env.formIDOf(t, created.Code))
	require.NoError(t, err)
	require.Equal(t, model.StorageStateReady, binding.State)
	return created.Code, binding.PhysicalTable
}

// ---- 用例 ----

// SEC-PHYS-001 全链路：物理表 + RLS 落库 → 提交 values=NULL、物理行有值 →
// 列表出网 → 审批写回替换物理行（方案 §16.3/§16.4）。
func TestPhysINTFullChain(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	code, tableName := env.publishPhysical(t, physScalarItems, `"_widget_a","_widget_n","_widget_d"`, member)
	ctx := tenantCtx(1)

	// 物理表存在、列齐备（系统预置 + f_* 用户列）、RLS 启用并带租户策略
	assert.True(t, env.tableExists(t, tableName))
	for _, column := range []string{"tenant_id", "record_id", "workflow_instance_no", "workflow_status", "workflow_updated_at", `"f_aaaaaaaa01"`, `"f_aaaaaaaa02"`, `"f_aaaaaaaa03"`} {
		assert.True(t, env.columnExists(t, tableName, column), "column %s missing", column)
	}
	assert.Equal(t, 1, env.countInt(t,
		fmt.Sprintf("SELECT count(*) FROM pg_class WHERE relname = '%s' AND relrowsecurity AND relforcerowsecurity", tableName)),
		"RLS must be enabled and forced")
	assert.Equal(t, 1, env.countInt(t,
		fmt.Sprintf("SELECT count(*) FROM pg_policy WHERE polname = '%s'", "p_"+tableName+"_tenant")),
		"tenant policy must exist")

	// 提交：信封 values=NULL、物理行各类型值正确
	result, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
		HasResult: submitBool(true), DataOpID: "11111111-1111-4111-8111-111111111111",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"张三"`), Visible: submitBool(true)},
			"_widget_n": {Data: model.JSONContent(`88`), Visible: submitBool(true)},
			"_widget_d": {Data: model.JSONContent(`"2026-09-04"`), Visible: submitBool(true)},
		},
	})
	require.NoError(t, err)

	// values 为 NULL（不双写，方案 §5.1）
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		"SELECT count(*) FROM tn_form_records WHERE id = %d AND values IS NULL", result.RecordID)))
	// 物理行：文本/数字（EXTRACT 日期形状校验）/引用形状逐列断言
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE record_id = %d AND "f_aaaaaaaa01" = '张三' AND "f_aaaaaaaa02" = 88 AND "f_aaaaaaaa03" = '2026-09-04'::date AND workflow_status = 'NONE'`,
		tableName, result.RecordID)))

	// 列表：物理值出网（decode 回协议形态）
	page, err := env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{
		Version: 1,
		Filter:  &model.RecordQueryExpression{Type: "condition", Field: "_widget_n", Operator: "gt", Value: float64(80)},
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "张三", page.Items[0].Values["_widget_a"])
	assert.Equal(t, float64(88), page.Items[0].Values["_widget_n"])
	assert.Equal(t, "2026-09-04", page.Items[0].Values["_widget_d"])
	assert.Equal(t, "NONE", page.Items[0].WorkflowStatus)

	// 审批写回：物理行整体替换 + 信封 updated_at 刷新
	store := env.formSvc.(interface {
		UpdateRecordValues(ctx context.Context, recordID uint, patch map[string]any) error
	})
	require.NoError(t, store.UpdateRecordValues(ctx, result.RecordID, map[string]any{"_widget_a": "李四"}))
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE record_id = %d AND "f_aaaaaaaa01" = '李四' AND "f_aaaaaaaa02" = 88`,
		tableName, result.RecordID)))
}

// SEC-PHYS-002 跨租户 RLS 隔离（方案 §16.6）：租户 2 事务与无租户事务均
// 读不到租户 1 的物理行（fail-closed）；record 复合外键防跨租户绑定。
func TestPhysINTRowLevelSecurityIsolation(t *testing.T) {
	env := newPhysEnv(t)
	owner := memberOfTenant(1)
	code, tableName := env.publishPhysical(t, physScalarItems, `"_widget_a","_widget_n","_widget_d"`, owner)
	ctx := tenantCtx(1)
	result, err := env.formSvc.SubmitRecord(ctx, owner, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
		HasResult: submitBool(true), DataOpID: "22222222-2222-4222-8222-222222222222",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"机密"`), Visible: submitBool(true)},
			"_widget_n": {Visible: submitBool(true)},
			"_widget_d": {Visible: submitBool(true)},
		},
	})
	require.NoError(t, err)

	// 租户 1 自己可见
	assert.Equal(t, 1, env.countInTenant(t, tableName, 1, result.RecordID))
	// 租户 2 事务：RLS 拒绝（0 行）
	assert.Equal(t, 0, env.countInTenant(t, tableName, 2, result.RecordID))
	// 无租户上下文事务：fail-closed（0 行）
	assert.Equal(t, 0, env.probeCountAsNoTenant(t, tableName, result.RecordID),
		"session without tenant must see nothing (fail-closed)")

	// 复合外键：向物理表插跨租户 record_id 组合必须被外键拒绝
	err = env.db.WithContext(tenantCtx(2)).Transaction(func(tx *gorm.DB) error {
		return tx.Exec(fmt.Sprintf(
			`INSERT INTO %q (tenant_id, record_id) VALUES (2, %d)`, tableName, result.RecordID)).Error
	})
	require.Error(t, err, "composite FK must reject cross-tenant binding")
}

// SEC-PHYS-003 注入面（方案 §16.7）：恶意字段名/排序/值只进绑定参数或被
// 白名单拒绝，不产生 SQL 错误。
func TestPhysINTQueryInjectionSurface(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	code, _ := env.publishPhysical(t, physScalarItems, `"_widget_a","_widget_n","_widget_d"`, member)
	ctx := tenantCtx(1)
	revision := env.firstSchemaRevision(t, ctx, code)
	_, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: revision,
		HasResult: submitBool(true), DataOpID: "33333333-3333-4333-8333-333333333333",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"Robert'); DROP TABLE students;--"`), Visible: submitBool(true)},
			"_widget_n": {Visible: submitBool(true)},
			"_widget_d": {Visible: submitBool(true)},
		},
	})
	require.NoError(t, err, "quoted value must be parameter-bound")

	malicious := []model.RecordQueryExpression{
		{Type: "condition", Field: `_widget_a" OR 1=1 --`, Operator: "eq", Value: "x"},
		{Type: "condition", Field: "values", Operator: "eq", Value: "x"},
		{Type: "condition", Field: "sys.submittedAt", Operator: "eq", Value: "2026' ; --"},
	}
	for _, filter := range malicious {
		_, err := env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{Version: 1, Filter: &filter})
		assert.ErrorIs(t, err, apperrors.ErrRecordQueryInvalid, "field %q must be rejected", filter.Field)
	}
	_, err = env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{
		Version: 1, Sorts: []model.RecordQuerySort{{Field: "id; DROP TABLE tn_form_records", Direction: "asc"}},
	})
	assert.ErrorIs(t, err, apperrors.ErrRecordQueryInvalid)
	// 注入尝试后表仍存在且数据完整
	assert.Equal(t, 1, env.countInt(t, "SELECT count(*) FROM tn_form_records WHERE form_id > 0"))
}

// SEC-PHYS-004 子表单（方案 §16.10）：子表建表 → 提交子行 → 集合替换 →
// 无孤儿子行。
func TestPhysINTSubformCollectionReplace(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	items := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"subform","widgetName":"_widget_s","fieldId":"aaaaaaaa04","enable":true,"visible":true,"allowBlank":true,"items":[
			{"widget":{"type":"text","widgetName":"_widget_r1","fieldId":"bbbbbbbb01","enable":true,"visible":true,"allowBlank":true},"label":"明细","description":"","labelHidden":false,"lineWidth":6},
			{"widget":{"type":"number","widgetName":"_widget_r2","fieldId":"bbbbbbbb02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6}],
			"subformCreate":true,"subformInsert":true,"subformEdit":true,"subformDelete":true,"quickFill":true,
			"pcStickyColumn":{"enable":true,"limit":1},"mobileStickyColumn":{"enable":false,"limit":1},
			"mobileViewStyle":"vertical","mobileSummaryFieldCount":3},"label":"明细表","description":"","labelHidden":false,"lineWidth":12}]`
	code, tableName := env.publishPhysical(t, items, `"_widget_a","_widget_s"`, member)
	ctx := tenantCtx(1)

	childTable := env.childTableOf(t, ctx, tableName)
	assert.True(t, env.tableExists(t, childTable), "child table must be created: %s", childTable)

	submitValues := func(dataOpID, rows string) *model.SubmitRecordResult {
		result, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
			AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
			HasResult: submitBool(true), DataOpID: dataOpID,
			Values: map[string]model.SubmitFieldValue{
				"_widget_a": {Data: model.JSONContent(`"甲"`), Visible: submitBool(true)},
				"_widget_s": {Data: model.JSONContent(rows), Visible: submitBool(true)},
			},
		})
		require.NoError(t, err)
		return result
	}
	first := submitValues("44444444-4444-4444-8444-444444444444", `[{"_widget_r1":"行一","_widget_r2":1},{"_widget_r1":"行二","_widget_r2":2}]`)
	assert.Equal(t, 2, env.countInt(t, fmt.Sprintf("SELECT count(*) FROM %q WHERE parent_record_id = %d", childTable, first.RecordID)))
	// 子行顺序持久化（sort_order）
	assert.Equal(t, "行一", env.textOf(t, fmt.Sprintf(
		"SELECT \"f_bbbbbbbb01\" FROM %q WHERE parent_record_id = %d ORDER BY sort_order ASC LIMIT 1", childTable, first.RecordID)))

	// 集合替换：两行换一行，无孤儿残留
	store := env.formSvc.(interface {
		UpdateRecordValues(ctx context.Context, recordID uint, patch map[string]any) error
	})
	require.NoError(t, store.UpdateRecordValues(ctx, first.RecordID, map[string]any{
		"_widget_s": []any{map[string]any{"_widget_r1": "新行", "_widget_r2": 9}},
	}))
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		"SELECT count(*) FROM %q WHERE parent_record_id = %d", childTable, first.RecordID)),
		"collection replace must not leave orphan rows")
	assert.Equal(t, "新行", env.textOf(t, fmt.Sprintf(
		"SELECT \"f_bbbbbbbb01\" FROM %q WHERE parent_record_id = %d", childTable, first.RecordID)))
}

// SEC-PHYS-005 并发发布收口（方案 §16.1）：存在未完成 Job 时再次发布拒绝。
func TestPhysINTConcurrentPublishBusy(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	ctx := tenantCtx(1)
	created, err := env.formSvc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: 7, Name: "并发发布", FormType: model.FormTypeStandard,
	})
	require.NoError(t, err)
	saved, err := env.formSvc.SaveDraft(ctx, member, created.Code, &model.SaveDraftRequest{
		DraftRevision: 1, ProtocolVersion: model.CurrentProtocolVersion, Content: physDoc(physScalarItems, `"_widget_a","_widget_n","_widget_d"`),
	})
	require.NoError(t, err)
	first, err := env.formSvc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	require.NoError(t, err)
	require.True(t, first.Async)

	// Job 未执行：同表单再次发布 → FORM_STORAGE_BUSY
	_, err = env.formSvc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: 2})
	require.ErrorIs(t, err, apperrors.ErrStorageBusy)
}

// ---- 助手 ----

func (env *physEnv) formIDOf(t *testing.T, code string) uint {
	t.Helper()
	formSvc := env.formSvc.(*formService)
	form, err := formSvc.loadByCode(tenantCtx(1), code)
	require.NoError(t, err)
	return form.ID
}

func (env *physEnv) firstSchemaRevision(t *testing.T, ctx context.Context, code string) string {
	t.Helper()
	return env.schemaRevisionOf(t, ctx, code, 1)
}

// schemaRevisionOf 指定版本号的修订口令（发布期间旧/新版本按各自口令提交）。
func (env *physEnv) schemaRevisionOf(t *testing.T, ctx context.Context, code string, versionNo int) string {
	t.Helper()
	formSvc := env.formSvc.(*formService)
	form, err := formSvc.loadByCode(ctx, code)
	require.NoError(t, err)
	version, err := formSvc.versions.GetByFormAndVersionNo(ctx, form.ID, versionNo)
	require.NoError(t, err)
	return fmt.Sprintf("%d", version.SchemaRevision)
}

func (env *physEnv) childTableOf(t *testing.T, ctx context.Context, parentTable string) string {
	t.Helper()
	children, err := env.childRepo.ListByStorage(ctx, env.storageIDOfTable(t, ctx, parentTable))
	require.NoError(t, err)
	require.NotEmpty(t, children)
	return children[0].PhysicalTable
}

func (env *physEnv) storageIDOfTable(t *testing.T, ctx context.Context, parentTable string) uint {
	t.Helper()
	var id uint
	require.NoError(t, env.db.WithContext(ctx).Raw(
		"SELECT id FROM tn_form_storages WHERE table_name = ?", parentTable).Scan(&id).Error)
	return id
}

func (env *physEnv) tableExists(t *testing.T, name string) bool {
	t.Helper()
	return env.countInt(t, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '"+name+"'") == 1
}

func (env *physEnv) columnExists(t *testing.T, table, column string) bool {
	t.Helper()
	trimmed := strings.Trim(column, `"`)
	return env.countInt(t, fmt.Sprintf(
		"SELECT count(*) FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s'", table, trimmed)) == 1
}

func (env *physEnv) countInt(t *testing.T, sql string) int {
	t.Helper()
	var n int
	require.NoError(t, env.db.Raw(sql).Scan(&n).Error)
	return n
}

func (env *physEnv) textOf(t *testing.T, sql string) string {
	t.Helper()
	var text string
	require.NoError(t, env.db.Raw(sql).Scan(&text).Error)
	return text
}

// countInTenant 以指定租户上下文开事务统计可见行（RLS 语义验证）。
// 关键：PostgreSQL 超级用户绕过一切 RLS（即使 FORCE），测试库连接是
// postgres 超级用户——必须 SET LOCAL ROLE 降为普通探测角色才能验证策略；
// 该角色只授予动态表 SELECT（与生产运行时角色同位：非 owner、无 DDL）。
func (env *physEnv) countInTenant(t *testing.T, table string, tenantID, recordID uint) int {
	t.Helper()
	return env.probeCount(t, table, recordID,
		fmt.Sprintf("SELECT set_config('app.current_tenant', '%d', true)", tenantID))
}

// probeCountAsNoTenant 以普通角色但未设置租户会话变量统计可见行（fail-closed 验证）。
func (env *physEnv) probeCountAsNoTenant(t *testing.T, table string, recordID uint) int {
	t.Helper()
	return env.probeCount(t, table, recordID, "")
}

func (env *physEnv) probeCount(t *testing.T, table string, recordID uint, setupSQL string) int {
	t.Helper()
	env.ensureRLSProbeRole(t)
	// 动态表对探测角色授权 SELECT（幂等；owner 权限执行）
	require.NoError(t, env.db.Exec(fmt.Sprintf("GRANT SELECT ON %q TO evolyn_rls_probe", table)).Error)
	var n int
	require.NoError(t, env.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE evolyn_rls_probe").Error; err != nil {
			return err
		}
		if setupSQL != "" {
			if err := tx.Exec(setupSQL).Error; err != nil {
				return err
			}
		}
		return tx.Raw(fmt.Sprintf("SELECT count(*) FROM %q WHERE record_id = ?", table), recordID).Scan(&n).Error
	}))
	return n
}

// ensureRLSProbeRole 建普通探测角色并授予动态表 SELECT（幂等）。
func (env *physEnv) ensureRLSProbeRole(t *testing.T) {
	t.Helper()
	env.mu.Lock()
	defer env.mu.Unlock()
	if env.probeRoleReady {
		return
	}
	env.countInt(t, "SELECT 1") // 连接健康检查
	require.NoError(t, env.db.Exec(
		"DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'evolyn_rls_probe') THEN CREATE ROLE evolyn_rls_probe NOLOGIN; END IF; END $$").Error)
	require.NoError(t, env.db.Exec("GRANT USAGE ON SCHEMA public TO evolyn_rls_probe").Error)
	// 动态表在测试内才创建，逐表授权放在建表后由 probeCount 首次调用补齐
	env.probeRoleReady = true
}

// SEC-PHYS-006 流程投影一致性（方案 §10.2/§16.5）：投影端口同事务刷新信封
// 与物理表的单号/状态/时间列，状态枚举外值拒绝。
func TestPhysINTWorkflowProjectionConsistency(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	code, tableName := env.publishPhysical(t, physScalarItems, `"_widget_a","_widget_n","_widget_d"`, member)
	ctx := tenantCtx(1)
	result, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
		HasResult: submitBool(true), DataOpID: "55555555-5555-4555-8555-555555555555",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"审批中"`), Visible: submitBool(true)},
			"_widget_n": {Visible: submitBool(true)},
			"_widget_d": {Visible: submitBool(true)},
		},
	})
	require.NoError(t, err)

	updater := env.formSvc.(interface {
		UpdateWorkflowProjection(ctx context.Context, recordID uint, projection WorkflowProjection) error
	})
	// 非法状态拒绝（枚举外值不出网也不落库）
	require.Error(t, updater.UpdateWorkflowProjection(ctx, result.RecordID, WorkflowProjection{Status: "PAUSED"}))

	// RUNNING：信封与物理表同事务同值
	require.NoError(t, updater.UpdateWorkflowProjection(ctx, result.RecordID, WorkflowProjection{
		InstanceNo: "WF-20260909-000001", Status: "RUNNING", UpdatedAt: time.Now(),
	}))
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM tn_form_records WHERE id = %d AND workflow_instance_no = 'WF-20260909-000001' AND workflow_status = 'RUNNING' AND workflow_updated_at IS NOT NULL`,
		result.RecordID)))
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE record_id = %d AND workflow_instance_no = 'WF-20260909-000001' AND workflow_status = 'RUNNING' AND workflow_updated_at IS NOT NULL`,
		tableName, result.RecordID)))

	// 终态推进：COMPLETED 双侧同步刷新
	require.NoError(t, updater.UpdateWorkflowProjection(ctx, result.RecordID, WorkflowProjection{
		InstanceNo: "WF-20260909-000001", Status: "COMPLETED",
	}))
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM tn_form_records WHERE id = %d AND workflow_status = 'COMPLETED'`, result.RecordID)))
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE record_id = %d AND workflow_status = 'COMPLETED'`, tableName, result.RecordID)))

	// 校准端点（§10.2）：注入 fake 校准器验证管理入口编排与审计口径
	svc := env.formSvc.(*formService)
	svc.UseProjectionRecalibrator(fakeRecalibrator{instances: 3})
	detail, err := env.formSvc.RecalibrateWorkflowProjection(ctx, member, code)
	require.NoError(t, err)
	assert.EqualValues(t, 3, detail.Instances)
}

// fakeRecalibrator 校准端口桩。
type fakeRecalibrator struct{ instances int }

func (f fakeRecalibrator) RecalibrateByForm(ctx context.Context, formID uint) (int, error) {
	return f.instances, nil
}

// SEC-PHYS-007 Job 查询/重试服务面（方案 §14）：发布后可查任务状态；非
// FAILED 的重试拒绝（不接受新模型语义）。
func TestPhysINTStorageJobLifecycle(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	ctx := tenantCtx(1)
	created, err := env.formSvc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: 7, Name: "任务查询", FormType: model.FormTypeStandard,
	})
	require.NoError(t, err)
	saved, err := env.formSvc.SaveDraft(ctx, member, created.Code, &model.SaveDraftRequest{
		DraftRevision: 1, ProtocolVersion: model.CurrentProtocolVersion,
		Content: physDoc(physScalarItems, `"_widget_a","_widget_n","_widget_d"`),
	})
	require.NoError(t, err)
	published, err := env.formSvc.Publish(ctx, member, created.Code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	require.NoError(t, err)
	require.True(t, published.Async)

	// 查询：PENDING 期携带受控信息与双口令
	detail, err := env.formSvc.GetStorageJob(ctx, member, created.Code, *published.JobID)
	require.NoError(t, err)
	assert.Equal(t, "PENDING", detail.Status)
	assert.Equal(t, published.PublishedVersion, detail.PublishedVersion)
	assert.Equal(t, published.SchemaRevision, detail.SchemaRevision)

	// 非 FAILED 重试拒绝
	_, err = env.formSvc.RetryStorageJob(ctx, member, created.Code, *published.JobID)
	assert.ErrorIs(t, err, apperrors.ErrStorageDDLFailed)

	// 归属复核：另一表单编码查询同 Job → NOT_FOUND
	other, err := env.formSvc.Create(ctx, member, &model.CreateFormRequest{
		ApplicationID: 7, Name: "另一表单", FormType: model.FormTypeStandard,
	})
	require.NoError(t, err)
	_, err = env.formSvc.GetStorageJob(ctx, member, other.Code, *published.JobID)
	assert.ErrorIs(t, err, apperrors.ErrStorageJobNotFound)
}

// SEC-PHYS-008 PUBLISHING 窗口放行（202 发布语义）：结构变更 DDL 在途时，
// 发布指针仍指向旧版本——旧版本提交/列表按已应用模型继续服务（物理结构是
// 历次已应用字段的并集，旧快照字段集 ⊆ 已应用模型）；DDL 完成后新版本可
// 提交。流程状态筛选在物理模式走物理表同名列（d. 前缀命中预置索引）。
func TestPhysINTPublishingWindowServesOldSnapshot(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	code, _ := env.publishPhysical(t, physScalarItems, `"_widget_a","_widget_n","_widget_d"`, member)
	ctx := tenantCtx(1)

	// v2：新增字段 _widget_e（fieldId aaaaaaaa05）→ 结构变更 → 202 + PUBLISHING
	v2Items := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"number","widgetName":"_widget_n","fieldId":"aaaaaaaa02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"datetime","widgetName":"_widget_d","fieldId":"aaaaaaaa03","enable":true,"visible":true,"allowBlank":true,"format":"date"},"label":"日期","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"text","widgetName":"_widget_e","fieldId":"aaaaaaaa05","enable":true,"visible":true,"allowBlank":true},"label":"备注","description":"","labelHidden":false,"lineWidth":6}]`
	saved, err := env.formSvc.SaveDraft(ctx, member, code, &model.SaveDraftRequest{
		DraftRevision: 2, ProtocolVersion: model.CurrentProtocolVersion,
		Content: physDoc(v2Items, `"_widget_a","_widget_n","_widget_d","_widget_e"`),
	})
	require.NoError(t, err)
	published, err := env.formSvc.Publish(ctx, member, code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	require.NoError(t, err)
	require.True(t, published.Async, "add-column change must publish async")

	// 存储态=PUBLISHING（DDL Job 未执行）
	binding, err := env.storageRepo.GetByFormID(ctx, env.formIDOf(t, code))
	require.NoError(t, err)
	require.Equal(t, model.StorageStatePublishing, binding.State)

	// PUBLISHING 窗口内：按旧版本（v1）提交成功（P1-1 放行语义）
	_, err = env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
		HasResult: submitBool(true), DataOpID: "77777777-7777-4777-8777-777777777777",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"窗口期"`), Visible: submitBool(true)},
			"_widget_n": {Data: model.JSONContent(`7`), Visible: submitBool(true)},
			"_widget_d": {Visible: submitBool(true)},
		},
	})
	require.NoError(t, err, "old-snapshot submission must be served during PUBLISHING window")

	// 列表同口径放行：旧版本映射 + 已应用模型列投影；流程状态筛选走物理表
	// 同名列（sys.workflowStatus → d.workflow_status，P1-4）
	statusFilter := model.RecordQueryExpression{Type: "condition", Field: "sys.workflowStatus", Operator: "eq", Value: "NONE"}
	page, err := env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{Version: 1, Filter: &statusFilter})
	require.NoError(t, err, "list must be served during PUBLISHING window")
	require.Len(t, page.Items, 1)
	assert.Equal(t, "窗口期", page.Items[0].Values["_widget_a"])

	// 执行 DDL → READY；新版本（v2，含新字段）可提交
	processed, err := env.ddlWorker.ProcessOnce(ctx)
	require.NoError(t, err)
	require.True(t, processed)
	_, err = env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 2, SchemaRevision: env.schemaRevisionOf(t, ctx, code, 2),
		HasResult: submitBool(true), DataOpID: "88888888-8888-4888-8888-888888888888",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"新版本"`), Visible: submitBool(true)},
			"_widget_n": {Visible: submitBool(true)},
			"_widget_d": {Visible: submitBool(true)},
			"_widget_e": {Data: model.JSONContent(`"新列"`), Visible: submitBool(true)},
		},
	})
	require.NoError(t, err)

	// 列表出网两张记录（发布指针已推进 v2）
	page, err = env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{Version: 1})
	require.NoError(t, err)
	assert.Len(t, page.Items, 2)
}

// SEC-PHYS-009 弃用列基线（P1-2）：字段删除后物理列保留（仅元数据弃用），
// 旧版本快照的提交继续落列、记录读取/审批写回保留弃用字段基线、最新版本
// 列表出网不暴露弃用字段（最新快照白名单）。
func TestPhysINTDeprecatedColumnBaseline(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	twoFieldItems := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"number","widgetName":"_widget_n","fieldId":"aaaaaaaa02","enable":true,"visible":true,"allowBlank":true},"label":"数量","description":"","labelHidden":false,"lineWidth":6}]`
	code, tableName := env.publishPhysical(t, twoFieldItems, `"_widget_a","_widget_n"`, member)
	ctx := tenantCtx(1)

	// v1 记录：a + n 均有值
	first, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
		HasResult: submitBool(true), DataOpID: "99999999-9999-4999-8999-999999999999",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"甲"`), Visible: submitBool(true)},
			"_widget_n": {Data: model.JSONContent(`3`), Visible: submitBool(true)},
		},
	})
	require.NoError(t, err)

	// v2：删除 _widget_n（纯弃用，无物理变更 → 同步发布，存储保持 READY）
	oneFieldItems := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6}]`
	saved, err := env.formSvc.SaveDraft(ctx, member, code, &model.SaveDraftRequest{
		DraftRevision: 2, ProtocolVersion: model.CurrentProtocolVersion,
		Content: physDoc(oneFieldItems, `"_widget_a"`),
	})
	require.NoError(t, err)
	published, err := env.formSvc.Publish(ctx, member, code, &model.PublishRequest{DraftRevision: saved.DraftRevision})
	require.NoError(t, err)
	assert.False(t, published.Async, "pure deprecation has no physical change")
	binding, err := env.storageRepo.GetByFormID(ctx, env.formIDOf(t, code))
	require.NoError(t, err)
	assert.Equal(t, model.StorageStateReady, binding.State)

	// 记录读取（表达式取数/审批合并基线）：弃用字段值仍完整返回
	store := env.formSvc.(WorkflowRecordStore)
	_, _, values, err := store.RecordData(ctx, first.RecordID)
	require.NoError(t, err)
	assert.Equal(t, "甲", values["_widget_a"])
	assert.Equal(t, float64(3), values["_widget_n"])

	// 旧版本（v1）快照新提交：仍含已删字段，物理列继续写入（弃用列含写入）
	oldSnap, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
		HasResult: submitBool(true), DataOpID: "aaaaaaa1-1111-4111-8111-111111111111",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"乙"`), Visible: submitBool(true)},
			"_widget_n": {Data: model.JSONContent(`5`), Visible: submitBool(true)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE record_id = %d AND "f_aaaaaaaa02" = 5`, tableName, oldSnap.RecordID)),
		"deprecated column must keep accepting old-snapshot submissions")

	// 审批写回（受信 patch）：未触及的弃用字段基线保留
	require.NoError(t, store.UpdateRecordValues(ctx, first.RecordID, map[string]any{"_widget_a": "丙"}))
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE record_id = %d AND "f_aaaaaaaa01" = '丙' AND "f_aaaaaaaa02" = 3`,
		tableName, first.RecordID)), "untouched deprecated column must be preserved on write-back")

	// 最新版本（v2）列表：弃用字段不出网（最新快照白名单）
	page, err := env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{Version: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	for _, item := range page.Items {
		assert.NotContains(t, item.Values, "_widget_n", "deprecated field must not leak to latest-snapshot list")
		assert.Contains(t, item.Values, "_widget_a")
	}
}

// SEC-PHYS-011 month/time 原形 TEXT 直存（支持矩阵扩展）：年月/时间字段
// 发布建 TEXT 列 → 提交原形入库 → 出网无损往返 → 字典序==时间序（between
// 范围筛选正确命中）；format 缺省的 datetime 兜底 TIMESTAMP（与提交校验
// value.go 的缺省口径一致）。
func TestPhysINTMonthTimeFields(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	items := `[
		{"widget":{"type":"text","widgetName":"_widget_a","fieldId":"aaaaaaaa01","enable":true,"visible":true,"allowBlank":true},"label":"姓名","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"datetime","widgetName":"_widget_m","fieldId":"aaaaaaaa06","enable":true,"visible":true,"allowBlank":true,"format":"month"},"label":"月份","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"datetime","widgetName":"_widget_t","fieldId":"aaaaaaaa07","enable":true,"visible":true,"allowBlank":true,"format":"time"},"label":"时间","description":"","labelHidden":false,"lineWidth":6},
		{"widget":{"type":"datetime","widgetName":"_widget_x","fieldId":"aaaaaaaa08","enable":true,"visible":true,"allowBlank":true},"label":"裸日期","description":"","labelHidden":false,"lineWidth":6}]`
	code, tableName := env.publishPhysical(t, items, `"_widget_a","_widget_m","_widget_t","_widget_x"`, member)
	ctx := tenantCtx(1)

	result, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: code, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, code),
		HasResult: submitBool(true), DataOpID: "eeeeeee1-5555-4555-8555-555555555555",
		Values: map[string]model.SubmitFieldValue{
			"_widget_a": {Data: model.JSONContent(`"甲"`), Visible: submitBool(true)},
			"_widget_m": {Data: model.JSONContent(`"2026-09"`), Visible: submitBool(true)},
			"_widget_t": {Data: model.JSONContent(`"09:30"`), Visible: submitBool(true)},
			"_widget_x": {Data: model.JSONContent(`"2026-09-04 12:30:00"`), Visible: submitBool(true)},
		},
	})
	require.NoError(t, err, "month/time and format-less datetime must publish and submit through physical storage")

	// 物理行原形 TEXT 直存；裸 datetime 兜底 TIMESTAMP 列
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE record_id = %d AND "f_aaaaaaaa06" = '2026-09' AND "f_aaaaaaaa07" = '09:30' AND "f_aaaaaaaa08" = '2026-09-04 12:30:00'::timestamp`,
		tableName, result.RecordID)))

	// 出网无损往返（列表 + 记录读取两路）
	page, err := env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{Version: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "2026-09", page.Items[0].Values["_widget_m"])
	assert.Equal(t, "09:30", page.Items[0].Values["_widget_t"])
	assert.Equal(t, "2026-09-04 12:30:00", page.Items[0].Values["_widget_x"])

	store := env.formSvc.(WorkflowRecordStore)
	_, _, values, err := store.RecordData(ctx, result.RecordID)
	require.NoError(t, err)
	assert.Equal(t, "2026-09", values["_widget_m"])
	assert.Equal(t, "09:30", values["_widget_t"])

	// 字典序==时间序：between 命中当年区间、跨年区间外不命中
	inRange := model.RecordQueryExpression{Type: "condition", Field: "_widget_m", Operator: "between", Value: []any{"2026-01", "2026-12"}}
	page, err = env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{Version: 1, Filter: &inRange})
	require.NoError(t, err)
	assert.Len(t, page.Items, 1, "lexicographic between must hit for same-year range")
	outRange := model.RecordQueryExpression{Type: "condition", Field: "_widget_m", Operator: "between", Value: []any{"2027-01", "2027-12"}}
	page, err = env.formSvc.ListRecords(ctx, member, code, model.RecordQueryDocument{Version: 1, Filter: &outRange})
	require.NoError(t, err)
	assert.Empty(t, page.Items, "out-of-range between must not hit")
}

// physSubformItems 子表单发布草稿（children 为子字段 JSON；空串=零子字段）。
func physSubformItems(children string) string {
	return fmt.Sprintf(`[
		{"widget":{"type":"subform","widgetName":"_widget_s","fieldId":"aaaaaaaa04","enable":true,"visible":true,"allowBlank":true,"items":[%s],
			"subformCreate":true,"subformInsert":true,"subformEdit":true,"subformDelete":true,"quickFill":true,
			"pcStickyColumn":{"enable":true,"limit":1},"mobileStickyColumn":{"enable":false,"limit":1},
			"mobileViewStyle":"vertical","mobileSummaryFieldCount":3},"label":"明细表","description":"","labelHidden":false,"lineWidth":12}]`, children)
}

var physSubformChildText = `{"widget":{"type":"text","widgetName":"_widget_r1","fieldId":"bbbbbbbb01","enable":true,"visible":true,"allowBlank":true},"label":"明细","description":"","labelHidden":false,"lineWidth":6}`

// SEC-PHYS-010 零列物理表全链路（P1-3）：无字段表单（父表零用户列）、纯
// 子表单表单（父表零用户列 + 子表有列）、子表单暂无子字段（子表零用户列
// 但有行）三种合法形态的提交/读取/列表/写回均不产生非法 SQL。
func TestPhysINTZeroColumnForms(t *testing.T) {
	env := newPhysEnv(t)
	member := memberOfTenant(1)
	ctx := tenantCtx(1)

	// 形态一：无字段表单——发布（建零用户列父表）→ 空值提交 → 读取/列表/空写回
	zeroCode, zeroTable := env.publishPhysical(t, "[]", "", member)
	assert.True(t, env.tableExists(t, zeroTable))
	assert.True(t, env.columnExists(t, zeroTable, "record_id"), "preset columns must exist on zero-column parent")
	zero, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: zeroCode, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, zeroCode),
		HasResult: submitBool(true), DataOpID: "bbbbbbb1-2222-4222-8222-222222222222",
		Values: map[string]model.SubmitFieldValue{},
	})
	require.NoError(t, err, "zero-column form submission must not produce invalid SQL")
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		"SELECT count(*) FROM %q WHERE record_id = %d AND workflow_status = 'NONE'", zeroTable, zero.RecordID)))

	store := env.formSvc.(WorkflowRecordStore)
	_, _, zeroValues, err := store.RecordData(ctx, zero.RecordID)
	require.NoError(t, err)
	assert.Empty(t, zeroValues)

	zeroPage, err := env.formSvc.ListRecords(ctx, member, zeroCode, model.RecordQueryDocument{Version: 1})
	require.NoError(t, err, "zero-column form list must not produce invalid SQL")
	require.Len(t, zeroPage.Items, 1)
	assert.Empty(t, zeroPage.Items[0].Values)

	require.NoError(t, store.UpdateRecordValues(ctx, zero.RecordID, map[string]any{}),
		"empty patch on zero-column form must skip UPDATE instead of invalid SQL")

	// 形态二：纯子表单表单——父表零用户列，子表携带明细行
	subCode, subTable := env.publishPhysical(t, physSubformItems(physSubformChildText), `"_widget_s"`, member)
	subChild := env.childTableOf(t, ctx, subTable)
	subResult, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: subCode, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, subCode),
		HasResult: submitBool(true), DataOpID: "ccccccc1-3333-4333-8333-333333333333",
		Values: map[string]model.SubmitFieldValue{
			"_widget_s": {Data: model.JSONContent(`[{"_widget_r1":"行一"}]`), Visible: submitBool(true)},
		},
	})
	require.NoError(t, err, "subform-only form submission must not produce invalid SQL")
	assert.Equal(t, 1, env.countInt(t, fmt.Sprintf(
		`SELECT count(*) FROM %q WHERE parent_record_id = %d AND "f_bbbbbbbb01" = '行一'`, subChild, subResult.RecordID)))
	_, _, subValues, err := store.RecordData(ctx, subResult.RecordID)
	require.NoError(t, err)
	require.Len(t, subValues["_widget_s"].([]any), 1)
	assert.Equal(t, "行一", subValues["_widget_s"].([]any)[0].(map[string]any)["_widget_r1"])
	_, err = env.formSvc.ListRecords(ctx, member, subCode, model.RecordQueryDocument{Version: 1})
	require.NoError(t, err, "subform-only form list must not produce invalid SQL")

	// 形态三：子表单暂无子字段——零列子表按行数保留空对象行序
	emptyChildCode, emptyChildTable := env.publishPhysical(t, physSubformItems(""), `"_widget_s"`, member)
	emptyChild := env.childTableOf(t, ctx, emptyChildTable)
	emptyResult, err := env.formSvc.SubmitRecord(ctx, member, &model.SubmitRecordRequest{
		AppCode: "app_phys", FormCode: emptyChildCode, PublishedVersion: 1, SchemaRevision: env.firstSchemaRevision(t, ctx, emptyChildCode),
		HasResult: submitBool(true), DataOpID: "ddddddd1-4444-4444-8444-444444444444",
		Values: map[string]model.SubmitFieldValue{
			"_widget_s": {Data: model.JSONContent(`[{},{}]`), Visible: submitBool(true)},
		},
	})
	require.NoError(t, err, "zero-column child submission must not produce invalid SQL")
	assert.Equal(t, 2, env.countInt(t, fmt.Sprintf(
		"SELECT count(*) FROM %q WHERE parent_record_id = %d", emptyChild, emptyResult.RecordID)),
		"row count must survive without value columns")
	_, _, emptyValues, err := store.RecordData(ctx, emptyResult.RecordID)
	require.NoError(t, err)
	assert.Len(t, emptyValues["_widget_s"].([]any), 2, "zero-column child rows must read back by count")
}
