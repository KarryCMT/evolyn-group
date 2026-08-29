-- 000052：延时任务表（流程引擎 Phase 5，ADR-012 第 19 章）
-- V1 不引入 Asynq：wf_job + Worker 轮询 FOR UPDATE SKIP LOCKED 承载
-- task.timeout / task.reminder；PostgreSQL 仍是流程状态唯一事实源，
-- DB Worker 挂掉不等于 Workflow 状态丢失。
CREATE TABLE IF NOT EXISTS wf_job (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    job_type varchar(32) NOT NULL,
    instance_id BIGINT NOT NULL DEFAULT 0,
    node_instance_id BIGINT NOT NULL DEFAULT 0,
    task_id BIGINT NOT NULL DEFAULT 0,
    execute_at timestamp with time zone NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    max_retry_count INT NOT NULL DEFAULT 3,
    payload JSONB NOT NULL DEFAULT '{}',
    last_error text NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_job_type CHECK (job_type IN ('task.reminder', 'task.timeout')),
    CONSTRAINT chk_wf_job_status CHECK (status IN
        ('PENDING', 'PROCESSING', 'SUCCEEDED', 'FAILED', 'CANCELLED'))
);

-- Worker 领取主查询（第 28 章）：status + execute_at（FOR UPDATE SKIP LOCKED）
CREATE INDEX IF NOT EXISTS idx_wf_job_claim
    ON wf_job (status, execute_at);

-- 任务终态联动取消（部分索引只覆盖在途 Job，写放大最小）
CREATE INDEX IF NOT EXISTS idx_wf_job_task_pending
    ON wf_job (task_id) WHERE status IN ('PENDING', 'PROCESSING');

-- 实例终态联动取消与诊断查询
CREATE INDEX IF NOT EXISTS idx_wf_job_instance
    ON wf_job (instance_id, id);

COMMENT ON TABLE wf_job IS '流程延时任务（追加语义 + 有限重试）：任务创建时按节点配置排期 task.timeout/task.reminder，Worker 单事务内领取（FOR UPDATE SKIP LOCKED）→ 执行 → 回写结果，claim+执行同事务故 crash 自动回滚为 PENDING（第 19 章）；超时自动动作必须经 Task Engine 正常执行路径，Worker 不得直改流程状态';
COMMENT ON COLUMN wf_job.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_job.job_type IS '任务类型：task.reminder=待办提醒 / task.timeout=待办超时自动动作（Phase 7 后可扩展 service.retry/webhook.retry）';
COMMENT ON COLUMN wf_job.instance_id IS '关联流程实例 ID（0=未关联）';
COMMENT ON COLUMN wf_job.node_instance_id IS '关联节点实例 ID（0=未关联）';
COMMENT ON COLUMN wf_job.task_id IS '关联人工任务 ID（0=未关联；任务终态时在途 Job 联动取消）';
COMMENT ON COLUMN wf_job.execute_at IS '计划执行时间（Worker 轮询窗口）';
COMMENT ON COLUMN wf_job.status IS 'Job 状态机：PENDING/PROCESSING/SUCCEEDED/FAILED/CANCELLED（迁移表 task 包唯一裁决）';
COMMENT ON COLUMN wf_job.retry_count IS '已重试次数（执行失败重试回队时递增）';
COMMENT ON COLUMN wf_job.max_retry_count IS '重试上限（超过即 FAILED 终态）';
COMMENT ON COLUMN wf_job.payload IS '任务载荷 JSONB（超时动作参数/提醒上下文等）';
COMMENT ON COLUMN wf_job.last_error IS '最近一次失败摘要（只入日志/诊断，不出网）';
COMMENT ON COLUMN wf_job.created_at IS '创建时间';
COMMENT ON COLUMN wf_job.updated_at IS '更新时间（领取/回写/重试回队）';
