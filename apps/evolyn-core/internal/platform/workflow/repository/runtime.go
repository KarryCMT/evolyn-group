package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	enginemodel "evolyn/internal/engine/workflow/model"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/workflow/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- 引擎内核仓储 SPI 的 GORM 适配（ADR-012：platform/workflow →
// engine/workflow 单向依赖）。行锁统一经 clause.Locking 落 SELECT ... FOR
// UPDATE；租户过滤依赖 ctx（GORM 租户 Callback），跨租户 ID 即 NotFound。
// 本文件负责平台持久化模型 ↔ 引擎内核模型的双向转换。

// runtimeRepository 运行态仓储集合（同一 base 连接，各自实现引擎 SPI）。
type runtimeRepository struct {
	base *gorm.DB
}

// NewRuntimeRepositories 构造运行态仓储集合。
func NewRuntimeRepositories(base *gorm.DB) (*runtimeInstances, *runtimeExecutions, *runtimeNodes, *runtimeTasks, *runtimeOperations, *runtimeCCRecords, *runtimeJobs) {
	r := &runtimeRepository{base: base}
	return &runtimeInstances{r}, &runtimeExecutions{r}, &runtimeNodes{r}, &runtimeTasks{r}, &runtimeOperations{r}, &runtimeCCRecords{r}, &runtimeJobs{r}
}

// ---- 实例 ----

type runtimeInstances struct{ *runtimeRepository }

func (r *runtimeInstances) CreateInstance(ctx context.Context, instance *enginemodel.Instance) error {
	row := instanceToRow(instance)
	number, err := r.allocateInstanceNumber(ctx, instance.TenantID)
	if err != nil {
		return err
	}
	row.InstanceNo = number
	if err := infrastructure.ResolveDB(ctx, r.base).Create(row).Error; err != nil {
		return err
	}
	instance.ID = row.ID
	return nil
}

func (r *runtimeInstances) FindInstanceByID(ctx context.Context, tenantID, instanceID uint) (*enginemodel.Instance, error) {
	return r.findInstance(ctx, tenantID, instanceID, false)
}

func (r *runtimeInstances) FindInstanceByIDForUpdate(ctx context.Context, tenantID, instanceID uint) (*enginemodel.Instance, error) {
	return r.findInstance(ctx, tenantID, instanceID, true)
}

func (r *runtimeInstances) findInstance(ctx context.Context, tenantID, instanceID uint, lock bool) (*enginemodel.Instance, error) {
	query := infrastructure.ResolveDB(ctx, r.base)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row model.WfInstance
	if err := query.Where("id = ?", instanceID).First(&row).Error; err != nil {
		return nil, err
	}
	return instanceFromRow(&row), nil
}

func (r *runtimeInstances) SaveInstance(ctx context.Context, instance *enginemodel.Instance) error {
	row := instanceToRow(instance)
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfInstance{}).
		Where("id = ?", instance.ID).
		Updates(map[string]any{"status": row.Status}).Error
}

func (r *runtimeInstances) FindRunningInstanceByBusiness(ctx context.Context, tenantID uint, businessType, businessID string) (*enginemodel.Instance, error) {
	var row model.WfInstance
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("business_type = ? AND business_id = ? AND status = ?", businessType, businessID, "RUNNING").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return instanceFromRow(&row), nil
}

func (r *runtimeInstances) FindInstanceByIdempotencyKey(ctx context.Context, tenantID uint, key string) (*enginemodel.Instance, error) {
	var row model.WfInstance
	err := infrastructure.ResolveDB(ctx, r.base).
		Where("idempotency_key = ?", key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return instanceFromRow(&row), nil
}

func (r *runtimeInstances) HasRunningInstanceByDefinition(ctx context.Context, definitionID uint) (bool, error) {
	var count int64
	err := infrastructure.ResolveDB(ctx, r.base).Model(&model.WfInstance{}).
		Where("definition_id = ? AND status = ?", definitionID, "RUNNING").
		Count(&count).Error
	return count > 0, err
}

// ---- 执行路径 ----

type runtimeExecutions struct{ *runtimeRepository }

func (r *runtimeExecutions) CreateExecution(ctx context.Context, execution *enginemodel.Execution) error {
	row := &model.WfExecution{
		InstanceID:        execution.InstanceID,
		ParentExecutionID: execution.ParentExecutionID,
		Status:            string(execution.Status),
	}
	row.TenantID = execution.TenantID
	if err := infrastructure.ResolveDB(ctx, r.base).Create(row).Error; err != nil {
		return err
	}
	execution.ID = row.ID
	return nil
}

func (r *runtimeExecutions) FindExecutionByID(ctx context.Context, tenantID, executionID uint) (*enginemodel.Execution, error) {
	var row model.WfExecution
	if err := infrastructure.ResolveDB(ctx, r.base).Where("id = ?", executionID).First(&row).Error; err != nil {
		return nil, err
	}
	return executionFromRow(&row), nil
}

func (r *runtimeExecutions) ListExecutionsByInstance(ctx context.Context, instanceID uint) ([]enginemodel.Execution, error) {
	rows := make([]model.WfExecution, 0)
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]enginemodel.Execution, 0, len(rows))
	for i := range rows {
		out = append(out, *executionFromRow(&rows[i]))
	}
	return out, nil
}

func (r *runtimeExecutions) SaveExecution(ctx context.Context, execution *enginemodel.Execution) error {
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfExecution{}).
		Where("id = ?", execution.ID).
		Updates(map[string]any{"status": string(execution.Status)}).Error
}

// ---- 节点实例 ----

type runtimeNodes struct{ *runtimeRepository }

func (r *runtimeNodes) CreateNodeInstance(ctx context.Context, nodeInstance *enginemodel.NodeInstance) error {
	row := &model.WfNodeInstance{
		InstanceID:  nodeInstance.InstanceID,
		ExecutionID: nodeInstance.ExecutionID,
		NodeKey:     nodeInstance.NodeKey,
		Status:      string(nodeInstance.Status),
	}
	row.TenantID = nodeInstance.TenantID
	if err := infrastructure.ResolveDB(ctx, r.base).Create(row).Error; err != nil {
		return err
	}
	nodeInstance.ID = row.ID
	return nil
}

func (r *runtimeNodes) FindNodeInstanceByID(ctx context.Context, tenantID, nodeInstanceID uint) (*enginemodel.NodeInstance, error) {
	var row model.WfNodeInstance
	if err := infrastructure.ResolveDB(ctx, r.base).Where("id = ?", nodeInstanceID).First(&row).Error; err != nil {
		return nil, err
	}
	return nodeFromRow(&row), nil
}

func (r *runtimeNodes) ListNodeInstancesByInstance(ctx context.Context, instanceID uint) ([]enginemodel.NodeInstance, error) {
	rows := make([]model.WfNodeInstance, 0)
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]enginemodel.NodeInstance, 0, len(rows))
	for i := range rows {
		out = append(out, *nodeFromRow(&rows[i]))
	}
	return out, nil
}

func (r *runtimeNodes) SaveNodeInstance(ctx context.Context, nodeInstance *enginemodel.NodeInstance) error {
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfNodeInstance{}).
		Where("id = ?", nodeInstance.ID).
		Updates(map[string]any{"status": string(nodeInstance.Status)}).Error
}

// ---- 任务与参与人 ----

type runtimeTasks struct{ *runtimeRepository }

func (r *runtimeTasks) CreateTask(ctx context.Context, task *enginemodel.Task) error {
	row := &model.WfTask{
		InstanceID:            task.InstanceID,
		NodeInstanceID:        task.NodeInstanceID,
		NodeKey:               task.NodeKey,
		Status:                string(task.Status),
		TransferredFromTaskID: task.TransferredFromTaskID,
		TransferredToMemberID: task.TransferredToMemberID,
	}
	row.TenantID = task.TenantID
	if err := infrastructure.ResolveDB(ctx, r.base).Create(row).Error; err != nil {
		return err
	}
	task.ID = row.ID
	return nil
}

func (r *runtimeTasks) ReplaceActors(ctx context.Context, taskID uint, actors []enginemodel.Actor) error {
	// 全量替换：先清后插（快照在任务创建时一次性写入）
	db := infrastructure.ResolveDB(ctx, r.base)
	if err := db.Where("task_id = ?", taskID).Delete(&model.WfTaskActor{}).Error; err != nil {
		return err
	}
	if len(actors) == 0 {
		return nil
	}
	rows := make([]model.WfTaskActor, 0, len(actors))
	for _, actor := range actors {
		row := &model.WfTaskActor{TaskID: taskID, MemberID: actor.MemberID, DisplayName: actor.DisplayName, ActorRole: "assignee"}
		rows = append(rows, *row)
	}
	return db.Create(&rows).Error
}

func (r *runtimeTasks) FindTaskByIDForUpdate(ctx context.Context, tenantID, taskID uint) (*enginemodel.Task, error) {
	query := infrastructure.ResolveDB(ctx, r.base).Clauses(clause.Locking{Strength: "UPDATE"})
	var row model.WfTask
	if err := query.Where("id = ?", taskID).First(&row).Error; err != nil {
		return nil, err
	}
	return taskFromRow(&row), nil
}

func (r *runtimeTasks) ListTasksByInstance(ctx context.Context, instanceID uint) ([]enginemodel.Task, error) {
	rows := make([]model.WfTask, 0)
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]enginemodel.Task, 0, len(rows))
	for i := range rows {
		out = append(out, *taskFromRow(&rows[i]))
	}
	return out, nil
}

func (r *runtimeTasks) ListTasksByNodeInstance(ctx context.Context, nodeInstanceID uint) ([]enginemodel.Task, error) {
	rows := make([]model.WfTask, 0)
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("node_instance_id = ?", nodeInstanceID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]enginemodel.Task, 0, len(rows))
	for i := range rows {
		out = append(out, *taskFromRow(&rows[i]))
	}
	return out, nil
}

func (r *runtimeTasks) ListActorsOfTask(ctx context.Context, taskID uint) ([]enginemodel.Actor, error) {
	rows := make([]model.WfTaskActor, 0)
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("task_id = ? AND actor_role = ?", taskID, "assignee").Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]enginemodel.Actor, 0, len(rows))
	for i := range rows {
		out = append(out, enginemodel.Actor{MemberID: rows[i].MemberID, DisplayName: rows[i].DisplayName})
	}
	return out, nil
}

func (r *runtimeTasks) SaveTask(ctx context.Context, task *enginemodel.Task) error {
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfTask{}).
		Where("id = ?", task.ID).
		Updates(map[string]any{
			"status":                   string(task.Status),
			"transferred_from_task_id": task.TransferredFromTaskID,
			"transferred_to_member_id": task.TransferredToMemberID,
		}).Error
}

func (r *runtimeTasks) CancelPendingTasksByNode(ctx context.Context, nodeInstanceID uint) (int64, error) {
	result := infrastructure.ResolveDB(ctx, r.base).Model(&model.WfTask{}).
		Where("node_instance_id = ? AND status = ?", nodeInstanceID, "PENDING").
		Update("status", "CANCELLED")
	return result.RowsAffected, result.Error
}

func (r *runtimeTasks) CancelPendingTasksByInstance(ctx context.Context, instanceID uint) (int64, error) {
	result := infrastructure.ResolveDB(ctx, r.base).Model(&model.WfTask{}).
		Where("instance_id = ? AND status = ?", instanceID, "PENDING").
		Update("status", "CANCELLED")
	return result.RowsAffected, result.Error
}

// ---- 抄送记录（000051） ----

type runtimeCCRecords struct{ *runtimeRepository }

func (r *runtimeCCRecords) CreateCCRecords(ctx context.Context, records []enginemodel.CCRecord) error {
	if len(records) == 0 {
		return nil
	}
	rows := make([]model.WfCCRecord, 0, len(records))
	for _, record := range records {
		row := model.WfCCRecord{
			InstanceID:     record.InstanceID,
			NodeInstanceID: record.NodeInstanceID,
			NodeKey:        record.NodeKey,
			MemberID:       record.MemberID,
			DisplayName:    record.DisplayName,
		}
		row.TenantID = record.TenantID
		rows = append(rows, row)
	}
	return infrastructure.ResolveDB(ctx, r.base).Create(&rows).Error
}

// ---- 延时任务（000052，Phase 5） ----

type runtimeJobs struct{ *runtimeRepository }

func (r *runtimeJobs) CreateJob(ctx context.Context, job *enginemodel.Job) error {
	row, err := jobToRow(job)
	if err != nil {
		return err
	}
	if err := infrastructure.ResolveDB(ctx, r.base).Create(row).Error; err != nil {
		return err
	}
	job.ID = row.ID
	return nil
}

// ClaimDueJobs 领取到期任务（第 19.2 章）：单条 UPDATE ... WHERE id IN
// (SELECT ... FOR UPDATE SKIP LOCKED) 原子完成「锁定 + 置 PROCESSING」，
// 与后续执行共用调用方事务——crash 时整体回滚为 PENDING，天然无孤儿
// PROCESSING。Raw SQL 不经租户 Callback：Worker 全租户轮询属预期行为。
func (r *runtimeJobs) ClaimDueJobs(ctx context.Context, now time.Time, batch int) ([]enginemodel.Job, error) {
	rows := make([]model.WfJob, 0, batch)
	err := infrastructure.ResolveDB(ctx, r.base).Raw(
		`UPDATE wf_job SET status = 'PROCESSING', updated_at = now()
		 WHERE id IN (
		     SELECT id FROM wf_job
		     WHERE status = 'PENDING' AND execute_at <= ?
		     ORDER BY execute_at
		     LIMIT ?
		     FOR UPDATE SKIP LOCKED
		 ) RETURNING *`, now, batch).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]enginemodel.Job, 0, len(rows))
	for i := range rows {
		job, err := jobFromRow(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *job)
	}
	return out, nil
}

func (r *runtimeJobs) SaveJob(ctx context.Context, job *enginemodel.Job) error {
	row, err := jobToRow(job)
	if err != nil {
		return err
	}
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]any{
			"status":          row.Status,
			"retry_count":     row.RetryCount,
			"execute_at":      row.ExecuteAt,
			"payload":         row.Payload,
			"last_error":      row.LastError,
			"max_retry_count": row.MaxRetryCount,
		}).Error
}

func (r *runtimeJobs) FindJobByID(ctx context.Context, tenantID, jobID uint) (*enginemodel.Job, error) {
	var row model.WfJob
	if err := infrastructure.ResolveDB(ctx, r.base).Where("id = ?", jobID).First(&row).Error; err != nil {
		return nil, err
	}
	return jobFromRow(&row)
}

func (r *runtimeJobs) cancelJobs(ctx context.Context, condition string, args ...any) error {
	return infrastructure.ResolveDB(ctx, r.base).Model(&model.WfJob{}).
		Where("status IN ?", []string{"PENDING", "PROCESSING"}).
		Where(condition, args...).
		Update("status", "CANCELLED").Error
}

func (r *runtimeJobs) CancelJobsByTask(ctx context.Context, taskID uint) error {
	return r.cancelJobs(ctx, "task_id = ?", taskID)
}

func (r *runtimeJobs) CancelJobsByNodeInstance(ctx context.Context, nodeInstanceID uint) error {
	return r.cancelJobs(ctx, "node_instance_id = ?", nodeInstanceID)
}

func (r *runtimeJobs) CancelJobsByInstance(ctx context.Context, instanceID uint) error {
	return r.cancelJobs(ctx, "instance_id = ?", instanceID)
}

func (r *runtimeJobs) ListJobsByInstance(ctx context.Context, instanceID uint) ([]enginemodel.Job, error) {
	rows := make([]model.WfJob, 0)
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]enginemodel.Job, 0, len(rows))
	for i := range rows {
		job, err := jobFromRow(&rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *job)
	}
	return out, nil
}

func jobToRow(job *enginemodel.Job) (*model.WfJob, error) {
	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return nil, fmt.Errorf("encode job payload: %w", err)
	}
	return &model.WfJob{
		ID:             job.ID,
		JobType:        string(job.Type),
		InstanceID:     job.InstanceID,
		NodeInstanceID: job.NodeInstanceID,
		TaskID:         job.TaskID,
		ExecuteAt:      job.ExecuteAt,
		Status:         string(job.Status),
		RetryCount:     job.RetryCount,
		MaxRetryCount:  job.MaxRetryCount,
		Payload:        model.DSLContent(payload),
		LastError:      job.LastError,
		// TenantID 由 Create 路径租户 Callback 兜底填充
		TenantID: job.TenantID,
	}, nil
}

func jobFromRow(row *model.WfJob) (*enginemodel.Job, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		return nil, fmt.Errorf("decode job %d payload: %w", row.ID, err)
	}
	return &enginemodel.Job{
		ID:             row.ID,
		TenantID:       row.TenantID,
		Type:           enginemodel.JobType(row.JobType),
		InstanceID:     row.InstanceID,
		NodeInstanceID: row.NodeInstanceID,
		TaskID:         row.TaskID,
		ExecuteAt:      row.ExecuteAt,
		Status:         enginemodel.JobStatus(row.Status),
		RetryCount:     row.RetryCount,
		MaxRetryCount:  row.MaxRetryCount,
		Payload:        payload,
		LastError:      row.LastError,
	}, nil
}

// ---- 操作流水 ----

type runtimeOperations struct{ *runtimeRepository }

func (r *runtimeOperations) AppendOperation(ctx context.Context, operation *enginemodel.Operation) error {
	payload, err := json.Marshal(operation.Payload)
	if err != nil {
		return fmt.Errorf("encode operation payload: %w", err)
	}
	row := &model.WfOperation{
		InstanceID:       operation.InstanceID,
		TaskID:           operation.TaskID,
		OperatorMemberID: operation.OperatorMemberID,
		OperationType:    string(operation.Type),
		Payload:          model.DSLContent(payload),
	}
	row.TenantID = operation.TenantID
	return infrastructure.ResolveDB(ctx, r.base).Create(row).Error
}

func (r *runtimeOperations) ListOperationsByInstance(ctx context.Context, instanceID uint) ([]enginemodel.Operation, error) {
	rows := make([]model.WfOperation, 0)
	if err := infrastructure.ResolveDB(ctx, r.base).
		Where("instance_id = ?", instanceID).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]enginemodel.Operation, 0, len(rows))
	for i := range rows {
		var payload map[string]any
		_ = json.Unmarshal([]byte(rows[i].Payload), &payload)
		out = append(out, enginemodel.Operation{
			ID:               rows[i].ID,
			TenantID:         rows[i].TenantID,
			InstanceID:       rows[i].InstanceID,
			TaskID:           rows[i].TaskID,
			OperatorMemberID: rows[i].OperatorMemberID,
			Type:             enginemodel.OperationType(rows[i].OperationType),
			Payload:          payload,
		})
	}
	return out, nil
}

// ---- 行转换（引擎内核 model 是唯一语义事实源） ----

func instanceToRow(instance *enginemodel.Instance) *model.WfInstance {
	var idem *string
	if instance.IdempotencyKey != "" {
		key := instance.IdempotencyKey
		idem = &key
	}
	return &model.WfInstance{
		ID:                  instance.ID,
		DefinitionID:        instance.DefinitionID,
		DefinitionVersionID: instance.DefinitionVersionID,
		BusinessType:        instance.BusinessType,
		BusinessID:          instance.BusinessID,
		AppID:               instance.AppID,
		FormID:              instance.FormID,
		FormVersionID:       instance.FormVersionID,
		Status:              string(instance.Status),
		StarterMemberID:     instance.StarterMemberID,
		IdempotencyKey:      idem,
	}
}

func instanceFromRow(row *model.WfInstance) *enginemodel.Instance {
	var idem string
	if row.IdempotencyKey != nil {
		idem = *row.IdempotencyKey
	}
	return &enginemodel.Instance{
		ID:                  row.ID,
		TenantID:            row.TenantID,
		DefinitionID:        row.DefinitionID,
		DefinitionVersionID: row.DefinitionVersionID,
		BusinessType:        row.BusinessType,
		BusinessID:          row.BusinessID,
		AppID:               row.AppID,
		FormID:              row.FormID,
		FormVersionID:       row.FormVersionID,
		Status:              enginemodel.InstanceStatus(row.Status),
		StarterMemberID:     row.StarterMemberID,
		IdempotencyKey:      idem,
	}
}

func executionFromRow(row *model.WfExecution) *enginemodel.Execution {
	return &enginemodel.Execution{
		ID:                row.ID,
		TenantID:          row.TenantID,
		InstanceID:        row.InstanceID,
		ParentExecutionID: row.ParentExecutionID,
		Status:            enginemodel.ExecutionStatus(row.Status),
	}
}

func nodeFromRow(row *model.WfNodeInstance) *enginemodel.NodeInstance {
	return &enginemodel.NodeInstance{
		ID:          row.ID,
		TenantID:    row.TenantID,
		InstanceID:  row.InstanceID,
		ExecutionID: row.ExecutionID,
		NodeKey:     row.NodeKey,
		Status:      enginemodel.NodeInstanceStatus(row.Status),
	}
}

func taskFromRow(row *model.WfTask) *enginemodel.Task {
	return &enginemodel.Task{
		ID:                    row.ID,
		TenantID:              row.TenantID,
		InstanceID:            row.InstanceID,
		NodeInstanceID:        row.NodeInstanceID,
		NodeKey:               row.NodeKey,
		Status:                enginemodel.TaskStatus(row.Status),
		TransferredFromTaskID: row.TransferredFromTaskID,
		TransferredToMemberID: row.TransferredToMemberID,
	}
}
