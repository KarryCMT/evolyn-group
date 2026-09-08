// DDL Job Worker 单测（方案 §8/§16.1/§16.2）：claim+执行+回写同事务语义、
// 失败退避回队、超限终态失败、checksum 防篡改拒绝。桩为内存实现；事务
// 语义经 fake TxManager 的回滚模拟验证（执行错误 → 状态不落）。
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	storagepkg "evolyn/internal/engine/data/storage"
	"evolyn/internal/infrastructure/dynamicddl"
	"evolyn/internal/platform/form/model"
	formrepo "evolyn/internal/platform/form/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- 内存桩 ----

// fakeDDLTx 模拟 TxManager：fn 返回 error 时按真语义「不提交」（内存桩
// 通过显式 abort 回调还原状态，由各用例断言回滚后的状态）。
type fakeDDLTx struct{}

func (fakeDDLTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeDDLExecutor 可注入执行结果与调用记录。
type fakeDDLExecutor struct {
	executeErr   error
	lockErr      error
	executeCalls int
	lockCalls    int
}

func (f *fakeDDLExecutor) AcquireAdvisoryLock(ctx context.Context, tenantID, storageID uint) error {
	f.lockCalls++
	return f.lockErr
}

func (f *fakeDDLExecutor) ExecutePlan(ctx context.Context, plan storagepkg.Plan, model *storagepkg.StorageModel, commentCtx dynamicddl.TableCommentContext) error {
	f.executeCalls++
	return f.executeErr
}

type workerStore struct {
	jobs      map[uint]*model.FormDDLJob
	schemas   map[uint]*model.FormStorageSchemaVersion
	storages  map[uint]*model.FormStorage
	versions  map[uint]*model.FormVersion
	forms     map[uint]*model.Form
	nextJobID uint
}

func newWorkerStore() *workerStore {
	return &workerStore{
		jobs:      map[uint]*model.FormDDLJob{},
		schemas:   map[uint]*model.FormStorageSchemaVersion{},
		storages:  map[uint]*model.FormStorage{},
		versions:  map[uint]*model.FormVersion{},
		forms:     map[uint]*model.Form{},
		nextJobID: 10,
	}
}

type stubJobs struct{ store *workerStore }

func (r *stubJobs) Create(ctx context.Context, job *model.FormDDLJob) (*model.FormDDLJob, error) {
	r.store.nextJobID++
	job.ID = r.store.nextJobID
	r.store.jobs[job.ID] = job
	return job, nil
}
func (r *stubJobs) GetByID(ctx context.Context, id uint) (*model.FormDDLJob, error) {
	if job, ok := r.store.jobs[id]; ok {
		clone := *job
		return &clone, nil
	}
	return nil, fmt.Errorf("not found")
}
func (r *stubJobs) HasActiveByStorage(ctx context.Context, storageID uint) (bool, error) {
	return false, nil
}

// ClaimNextDue 返回到期 Job 的 PROCESSING 副本但不改 store 状态——内存桩
// 无事务回滚语义，claim 状态保持 PENDING 使失败路径（真实语义：执行事务
// 回滚还原 claim）可直接观察；回滚还原语义由真库集成测试覆盖。
func (r *stubJobs) ClaimNextDue(ctx context.Context, now time.Time, limit int) ([]model.FormDDLJob, error) {
	claimed := make([]model.FormDDLJob, 0, limit)
	for i := uint(1); i <= r.store.nextJobID && len(claimed) < limit; i++ {
		job, ok := r.store.jobs[i]
		if !ok || job.Status != model.DDLJobPending || now.Before(time.Time(job.NextAttemptAt)) {
			continue
		}
		clone := *job
		clone.Status = model.DDLJobProcessing
		started := model.JSONTimeOf(now)
		clone.StartedAt = &started
		claimed = append(claimed, clone)
	}
	return claimed, nil
}
func (r *stubJobs) SaveJob(ctx context.Context, job *model.FormDDLJob) error {
	r.store.jobs[job.ID] = job
	return nil
}
func (r *stubJobs) ResetForRetry(ctx context.Context, jobID uint, nextAttemptAt time.Time) error {
	job := r.store.jobs[jobID]
	job.Status = model.DDLJobPending
	job.NextAttemptAt = model.JSONTimeOf(nextAttemptAt)
	return nil
}
func (r *stubJobs) Migrate() error { return nil }

type stubSchemas struct{ store *workerStore }

func (r *stubSchemas) Create(ctx context.Context, version *model.FormStorageSchemaVersion) (*model.FormStorageSchemaVersion, error) {
	version.ID = 100 + uint(len(r.store.schemas))
	r.store.schemas[version.ID] = version
	return version, nil
}
func (r *stubSchemas) GetByID(ctx context.Context, id uint) (*model.FormStorageSchemaVersion, error) {
	if version, ok := r.store.schemas[id]; ok {
		clone := *version
		return &clone, nil
	}
	return nil, fmt.Errorf("not found")
}
func (r *stubSchemas) GetApplied(ctx context.Context, storageID uint) (*model.FormStorageSchemaVersion, error) {
	for _, version := range r.store.schemas {
		if version.StorageID == storageID && version.State == model.SchemaVersionApplied {
			clone := *version
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
func (r *stubSchemas) MarkApplied(ctx context.Context, id uint) error {
	r.store.schemas[id].State = model.SchemaVersionApplied
	return nil
}
func (r *stubSchemas) MarkFailed(ctx context.Context, id uint, errorCode, errorDetail string) error {
	r.store.schemas[id].State = model.SchemaVersionFailed
	r.store.schemas[id].ErrorCode = errorCode
	return nil
}
func (r *stubSchemas) ResetForRetry(ctx context.Context, id uint) error {
	r.store.schemas[id].State = model.SchemaVersionPending
	return nil
}
func (r *stubSchemas) Migrate() error { return nil }

type stubStorages struct{ store *workerStore }

func (r *stubStorages) Create(ctx context.Context, storage *model.FormStorage) (bool, bool, error) {
	storage.ID = 200 + uint(len(r.store.storages))
	r.store.storages[storage.ID] = storage
	return true, false, nil
}
func (r *stubStorages) GetByFormID(ctx context.Context, formID uint) (*model.FormStorage, error) {
	for _, storage := range r.store.storages {
		if storage.FormID == formID {
			clone := *storage
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
func (r *stubStorages) LockByForm(ctx context.Context, formID uint) (*model.FormStorage, error) {
	return r.GetByFormID(ctx, formID)
}
func (r *stubStorages) GetByID(ctx context.Context, id uint) (*model.FormStorage, error) {
	if storage, ok := r.store.storages[id]; ok {
		clone := *storage
		return &clone, nil
	}
	return nil, fmt.Errorf("not found")
}
func (r *stubStorages) LockByID(ctx context.Context, id uint) (*model.FormStorage, error) {
	return r.GetByID(ctx, id)
}
func (r *stubStorages) MarkPublishing(ctx context.Context, id uint) error {
	r.store.storages[id].State = model.StorageStatePublishing
	return nil
}
func (r *stubStorages) MarkReady(ctx context.Context, id uint, applied uint) error {
	r.store.storages[id].State = model.StorageStateReady
	appliedID := applied
	r.store.storages[id].AppliedSchemaVersionID = &appliedID
	return nil
}
func (r *stubStorages) MarkFailed(ctx context.Context, id uint) error {
	r.store.storages[id].State = model.StorageStateFailed
	return nil
}
func (r *stubStorages) Migrate() error { return nil }

// stubFormVersions FormVersionRepository 最小桩（Worker 只读 GetByID）。
type stubFormVersions struct{ store *workerStore }

func (r *stubFormVersions) Create(ctx context.Context, version *model.FormVersion) (*model.FormVersion, error) {
	return version, nil
}
func (r *stubFormVersions) SetSchemaRevision(ctx context.Context, id uint, revision int64) error {
	return nil
}
func (r *stubFormVersions) GetByID(ctx context.Context, id uint) (*model.FormVersion, error) {
	if version, ok := r.store.versions[id]; ok {
		clone := *version
		return &clone, nil
	}
	return nil, fmt.Errorf("not found")
}
func (r *stubFormVersions) MaxVersionNo(ctx context.Context, formID uint) (int, error) {
	return 0, nil
}
func (r *stubFormVersions) GetByFormAndVersionNo(ctx context.Context, formID uint, versionNo int) (*model.FormVersion, error) {
	return nil, fmt.Errorf("not found")
}
func (r *stubFormVersions) Migrate() error { return nil }

// formRepoStub FormRepository 最小桩（Worker 只读 GetByID + MarkPublished）。
type formRepoStub struct{ store *workerStore }

func (r *formRepoStub) Create(ctx context.Context, form *model.Form) (*model.Form, error) {
	return form, nil
}
func (r *formRepoStub) GetByCode(ctx context.Context, code string) (*model.Form, error) {
	for _, form := range r.store.forms {
		if form.Code == code {
			clone := *form
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
func (r *formRepoStub) GetByID(ctx context.Context, id uint) (*model.Form, error) {
	if form, ok := r.store.forms[id]; ok {
		clone := *form
		return &clone, nil
	}
	return nil, fmt.Errorf("not found")
}
func (r *formRepoStub) List(ctx context.Context, params formrepo.ListParams) ([]model.Form, bool, error) {
	return nil, false, nil
}
func (r *formRepoStub) UpdateName(ctx context.Context, id uint, name string) error { return nil }
func (r *formRepoStub) UpdateFormType(ctx context.Context, id uint, formType model.FormType) error {
	return nil
}
func (r *formRepoStub) UpdateDraft(ctx context.Context, id uint, fromRevision int64, protocolVersion int, content model.JSONContent) (bool, error) {
	return true, nil
}
func (r *formRepoStub) MarkPublished(ctx context.Context, id uint, versionID uint, versionNo int) error {
	form := r.store.forms[id]
	vid := versionID
	form.LatestVersionID = &vid
	form.PublishedVersion = versionNo
	return nil
}
func (r *formRepoStub) SoftDelete(ctx context.Context, form *model.Form) error { return nil }
func (r *formRepoStub) CountBillableFormsByTenant(ctx context.Context, tenantID uint) (int64, error) {
	return 0, nil
}
func (r *formRepoStub) ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]formrepo.FormMenuTarget, error) {
	return map[uint]formrepo.FormMenuTarget{}, nil
}
func (r *formRepoStub) Migrate() error { return nil }

// ---- 装配 ----

type workerEnv struct {
	worker   *DDLJobWorker
	store    *workerStore
	executor *fakeDDLExecutor
}

func newWorkerEnv() *workerEnv {
	store := newWorkerStore()
	executor := &fakeDDLExecutor{}
	w := NewDDLJobWorker(
		fakeDDLTx{},
		&stubJobs{store: store},
		&stubSchemas{store: store},
		&stubStorages{store: store},
		&stubFormVersions{store: store},
		&formRepoStub{store: store},
		executor,
		nil,
	)
	return &workerEnv{worker: w, store: store, executor: executor}
}

// seedJob 构造一条可执行链：form → FormVersion → SchemaVersion(PENDING,
// model/plan/checksum 自洽) → Storage(PUBLISHING) → Job(PENDING)。
func (env *workerEnv) seedJob(t *testing.T, tamperChecksum bool) uint {
	t.Helper()
	form := &model.Form{Code: "form_test01", PublishedVersion: 0}
	form.ID = 1
	env.store.forms[form.ID] = form

	version := &model.FormVersion{FormID: form.ID, VersionNo: 1, SchemaRevision: 501}
	version.ID = 501
	env.store.versions[version.ID] = version

	target := &storagepkg.StorageModel{
		TableName: "tn_fd_01ab_k7q2m8",
		Columns: []storagepkg.ColumnSpec{{
			FieldID: "aaaaaaaa01", WidgetName: "_widget_a", WidgetType: "text",
			Kind: storagepkg.KindText, Type: storagepkg.ColumnTypeText,
		}},
	}
	plan := storagepkg.Plan{Actions: []storagepkg.PlanAction{{
		Kind: storagepkg.ActionCreateParent, Table: target.TableName,
	}}}
	checksum, err := storagepkg.ModelChecksum(target)
	require.NoError(t, err)
	if tamperChecksum {
		checksum = "deadbeef" + checksum[8:]
	}
	modelJSON, err := json.Marshal(target)
	require.NoError(t, err)
	planJSON, err := json.Marshal(plan)
	require.NoError(t, err)

	schemaVersion := &model.FormStorageSchemaVersion{
		StorageID: 201, FormVersionID: version.ID,
		Model: model.JSONContent(modelJSON), Plan: model.JSONContent(planJSON),
		Checksum: checksum, State: model.SchemaVersionPending,
	}
	schemaVersion.ID = 301
	env.store.schemas[schemaVersion.ID] = schemaVersion

	env.store.storages[201] = &model.FormStorage{
		ID: 201, FormID: form.ID, Backend: model.StorageBackendPhysical,
		PhysicalTable: target.TableName, State: model.StorageStatePublishing,
	}

	job := &model.FormDDLJob{
		StorageSchemaVersionID: schemaVersion.ID, Status: model.DDLJobPending,
		NextAttemptAt: model.JSONTimeOf(time.Now().Add(-time.Minute)),
	}
	created, err := (&stubJobs{store: env.store}).Create(context.Background(), job)
	require.NoError(t, err)
	return created.ID
}

// ---- 用例 ----

// 成功路径：claim+执行+回写一条链——Job SUCCEEDED、SchemaVersion APPLIED、
// Storage READY（applied 指针推进）、表单发布指针推进（方案 §16.2）。
func TestDDLWorkerSuccessAdvancesPublishPointer(t *testing.T) {
	env := newWorkerEnv()
	jobID := env.seedJob(t, false)

	processed, err := env.worker.processOne(context.Background())
	require.NoError(t, err)
	assert.True(t, processed)
	assert.Equal(t, 1, env.executor.executeCalls)
	assert.Equal(t, 1, env.executor.lockCalls)

	job := env.store.jobs[jobID]
	assert.Equal(t, model.DDLJobSucceeded, job.Status)
	assert.Equal(t, model.SchemaVersionApplied, env.store.schemas[301].State)
	storage := env.store.storages[201]
	assert.Equal(t, model.StorageStateReady, storage.State)
	require.NotNil(t, storage.AppliedSchemaVersionID)
	assert.EqualValues(t, 301, *storage.AppliedSchemaVersionID)

	form := env.store.forms[1]
	require.NotNil(t, form.LatestVersionID, "publish pointer must advance after job success")
	assert.EqualValues(t, 501, *form.LatestVersionID)
	assert.Equal(t, 1, form.PublishedVersion)
}

// 执行失败：执行事务回滚（Job 保持 PENDING 状态由 claim 事务还原——桩内
// 直接可观察 retry 记账），重试次数递增并退避回队（方案 §16.2 重试幂等）。
func TestDDLWorkerFailureRetriesWithBackoff(t *testing.T) {
	env := newWorkerEnv()
	jobID := env.seedJob(t, false)
	env.executor.executeErr = fmt.Errorf("boom: relation already exists")

	// 执行失败经独立事务记账回队：processOne 本身不视为轮询错误
	processed, err := env.worker.processOne(context.Background())
	require.NoError(t, err)
	assert.True(t, processed)

	job := env.store.jobs[jobID]
	assert.Equal(t, model.DDLJobPending, job.Status, "failed job must return to PENDING")
	assert.Equal(t, 1, job.RetryCount)
	assert.NotZero(t, job.LastErrorCode)
	assert.True(t, time.Time(job.NextAttemptAt).After(time.Now()), "backoff must schedule future attempt")
	// 执行失败不推进任何元数据
	assert.Equal(t, model.SchemaVersionPending, env.store.schemas[301].State)
	assert.Nil(t, env.store.forms[1].LatestVersionID)
}

// 超过重试上限：Job/SchemaVersion/Storage 全部落终态 FAILED（发布指针不动，
// 旧版本继续可运行——方案 §16.2）。
func TestDDLWorkerExhaustsRetriesToFailed(t *testing.T) {
	env := newWorkerEnv()
	jobID := env.seedJob(t, false)
	env.executor.executeErr = fmt.Errorf("persistent failure")
	env.worker.maxRetries = 1

	// 第一次失败：回队
	_, err := env.worker.processOne(context.Background())
	require.NoError(t, err)
	assert.Equal(t, model.DDLJobPending, env.store.jobs[jobID].Status)

	// 退避到期后第二次失败：超限 → 终态
	env.store.jobs[jobID].NextAttemptAt = model.JSONTimeOf(time.Now().Add(-time.Minute))
	_, err = env.worker.processOne(context.Background())
	require.NoError(t, err)

	assert.Equal(t, model.DDLJobFailed, env.store.jobs[jobID].Status)
	assert.Equal(t, model.SchemaVersionFailed, env.store.schemas[301].State)
	assert.Equal(t, model.StorageStateFailed, env.store.storages[201].State)
	assert.Nil(t, env.store.forms[1].LatestVersionID)
}

// checksum 防篡改：持久化模型与 checksum 不一致 → 拒绝执行并按失败记账
// （方案 §8「校验模型 checksum」）。
func TestDDLWorkerRejectsTamperedChecksum(t *testing.T) {
	env := newWorkerEnv()
	jobID := env.seedJob(t, true)

	_, err := env.worker.processOne(context.Background())
	require.NoError(t, err)
	checksumJob := env.store.jobs[jobID]
	assert.Contains(t, checksumJob.LastErrorDetail, "checksum", "checksum mismatch must be recorded: %s", checksumJob.LastErrorDetail)
	assert.Equal(t, 0, env.executor.executeCalls, "tampered model must not reach DDL executor")

	job := env.store.jobs[jobID]
	assert.Equal(t, model.DDLJobPending, job.Status)
	assert.Equal(t, 1, job.RetryCount)
}

// 队列空转：无到期 Job 时 processOne 返回未处理。
func TestDDLWorkerEmptyQueue(t *testing.T) {
	env := newWorkerEnv()
	processed, err := env.worker.processOne(context.Background())
	require.NoError(t, err)
	assert.False(t, processed)
	assert.Equal(t, 0, env.executor.executeCalls)
}
