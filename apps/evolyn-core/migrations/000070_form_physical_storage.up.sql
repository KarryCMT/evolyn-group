-- 表单物理表存储（方案：docs/低代码平台/表单设计器/物理表存储后端实施方案.md）：
-- Phase 1 元数据三表 + 记录信封流程状态投影。新创建表单的业务值只有
-- physical 一种存储方式；存量 JSONB 记录仅作历史兼容读取，values 不再作为
-- 新记录的写入目标（不双写）。

-- 表单存储绑定：每表单一行，backend 固定 PHYSICAL（LEGACY_JSONB 仅为存量
-- 兼容枚举，不对新表单开放）；table_name 创建时分配、永久不变。
CREATE TABLE tn_form_storages (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    form_id BIGINT NOT NULL,
    backend VARCHAR(16) NOT NULL,
    table_name VARCHAR(63) NOT NULL,
    state VARCHAR(16) NOT NULL,
    applied_schema_version_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    CONSTRAINT uq_tn_form_storages_form UNIQUE (tenant_id, form_id),
    CONSTRAINT uq_tn_form_storages_table UNIQUE (table_name),
    CONSTRAINT chk_tn_form_storages_backend CHECK (backend IN ('PHYSICAL', 'LEGACY_JSONB')),
    CONSTRAINT chk_tn_form_storages_state CHECK (state IN ('READY', 'PUBLISHING', 'FAILED'))
);
COMMENT ON TABLE tn_form_storages IS '表单存储绑定：新表单固定 PHYSICAL；表名服务端分配且永久不变';
COMMENT ON COLUMN tn_form_storages.tenant_id IS '租户 ID（pf_tenants.id）';
COMMENT ON COLUMN tn_form_storages.form_id IS '表单资产 ID（tn_forms.id）';
COMMENT ON COLUMN tn_form_storages.backend IS '存储后端：PHYSICAL=物理值表；LEGACY_JSONB 仅存量兼容';
COMMENT ON COLUMN tn_form_storages.table_name IS '物理父表名（tn_fd_ 前缀，服务端分配，防重由唯一约束兜底）';
COMMENT ON COLUMN tn_form_storages.state IS '存储状态：READY=可用；PUBLISHING=有未完成 DDL；FAILED=最近发布失败';
COMMENT ON COLUMN tn_form_storages.applied_schema_version_id IS '当前已应用的物理模型版本（tn_form_storage_schema_versions.id）';

-- 物理模型版本：某次发布快照对应的完整列/类型/子表/弃用元数据与 DDL 计划，
-- 与不可变发布快照一一对应。
CREATE TABLE tn_form_storage_schema_versions (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    storage_id BIGINT NOT NULL REFERENCES tn_form_storages(id),
    form_version_id BIGINT NOT NULL REFERENCES tn_form_versions(id),
    model JSONB NOT NULL,
    plan JSONB NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    state VARCHAR(16) NOT NULL,
    applied_at TIMESTAMPTZ,
    error_code VARCHAR(64),
    error_detail TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    CONSTRAINT uq_tn_form_storage_schema_version UNIQUE (storage_id, form_version_id),
    CONSTRAINT chk_tn_form_storage_schema_state CHECK (state IN ('PENDING', 'APPLIED', 'FAILED'))
);
COMMENT ON TABLE tn_form_storage_schema_versions IS '物理模型版本：发布快照的完整物理结构与 DDL 计划（不可变，仅状态推进）';
COMMENT ON COLUMN tn_form_storage_schema_versions.storage_id IS '存储绑定（tn_form_storages.id）';
COMMENT ON COLUMN tn_form_storage_schema_versions.form_version_id IS '发布快照（tn_form_versions.id）';
COMMENT ON COLUMN tn_form_storage_schema_versions.model IS '合并弃用列后的完整物理模型（engine/data/storage.StorageModel）';
COMMENT ON COLUMN tn_form_storage_schema_versions.plan IS '结构计划动作序列（engine/data/storage.Plan）';
COMMENT ON COLUMN tn_form_storage_schema_versions.checksum IS '模型规范化序列化 sha256（领取执行前复核防篡改）';
COMMENT ON COLUMN tn_form_storage_schema_versions.state IS '版本状态：PENDING 待执行；APPLIED 已应用；FAILED 终态失败';
COMMENT ON COLUMN tn_form_storage_schema_versions.error_code IS '失败稳定错误码（仅 FAILED 时非空）';
COMMENT ON COLUMN tn_form_storage_schema_versions.error_detail IS '失败内部详情（只入运维排查，不出网）';
COMMENT ON COLUMN tn_form_storage_schema_versions.updated_at IS '状态推进时间（APPLIED/FAILED 回写时刷新）';

-- DDL Job：异步结构变更执行单元；同一存储同一时刻至多一个未完成 Job
--（并发发布由 FORM_STORAGE_BUSY 拒绝），领取与执行同事务保证 crash 回滚。
CREATE TABLE tn_form_ddl_jobs (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    storage_schema_version_id BIGINT NOT NULL REFERENCES tn_form_storage_schema_versions(id),
    status VARCHAR(16) NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    last_error_code VARCHAR(64),
    last_error_detail TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    CONSTRAINT uq_tn_form_ddl_jobs_schema UNIQUE (storage_schema_version_id),
    CONSTRAINT chk_tn_form_ddl_jobs_status CHECK (status IN ('PENDING', 'PROCESSING', 'SUCCEEDED', 'FAILED'))
);
COMMENT ON TABLE tn_form_ddl_jobs IS '物理表 DDL 异步执行 Job：FOR UPDATE SKIP LOCKED 领取，claim+执行+回写同事务';
COMMENT ON COLUMN tn_form_ddl_jobs.tenant_id IS '租户 ID（Worker 执行时以此重建租户上下文）';
COMMENT ON COLUMN tn_form_ddl_jobs.storage_schema_version_id IS '目标物理模型版本（tn_form_storage_schema_versions.id）';
COMMENT ON COLUMN tn_form_ddl_jobs.status IS '执行状态：PENDING/PROCESSING/SUCCEEDED/FAILED';
COMMENT ON COLUMN tn_form_ddl_jobs.retry_count IS '重试次数（退避后回队，超限转 FAILED）';
COMMENT ON COLUMN tn_form_ddl_jobs.next_attempt_at IS '下次可执行时间（退避调度）';
COMMENT ON COLUMN tn_form_ddl_jobs.last_error_code IS '最近失败稳定错误码';
COMMENT ON COLUMN tn_form_ddl_jobs.last_error_detail IS '最近失败内部详情（截断保留，不出网）';

CREATE INDEX ix_tn_form_ddl_jobs_claim ON tn_form_ddl_jobs(status, next_attempt_at) WHERE status = 'PENDING';
CREATE INDEX ix_tn_form_storage_versions_storage ON tn_form_storage_schema_versions(storage_id, state);

-- 记录信封流程状态投影（方案 §5.1/§10）：状态唯一事实源是 wf_instance.status，
-- 本列仅是高频筛选/导出投影，随实例状态变更同事务更新；普通表单恒 NONE。
ALTER TABLE tn_form_records ADD COLUMN workflow_status VARCHAR(16) NOT NULL DEFAULT 'NONE';
ALTER TABLE tn_form_records ADD COLUMN workflow_updated_at TIMESTAMPTZ;

-- 存量回填：按已绑定实例的单号回填状态与最后状态变更时间（无实例记录保持
-- NONE；以 wf_instance 现状为源，不反向修改流程实例）。
UPDATE tn_form_records r SET
    workflow_status = i.status,
    workflow_updated_at = COALESCE(i.updated_at, i.created_at)
FROM wf_instance i
WHERE r.workflow_instance_no <> ''
  AND i.tenant_id = r.tenant_id
  AND i.instance_no = r.workflow_instance_no;

COMMENT ON COLUMN tn_form_records.workflow_status IS '流程实例状态投影：DRAFT/RUNNING/COMPLETED/REJECTED/CANCELLED；普通表单恒 NONE（事实源 wf_instance.status）';
COMMENT ON COLUMN tn_form_records.workflow_updated_at IS '流程投影最后更新时间：随实例状态变更同事务刷新';

-- 物理表复合外键锚点：(tenant_id, id) 唯一，供 tn_fd_* 以复合外键防止
-- 跨租户错误绑定（方案 §5.1）。
CREATE UNIQUE INDEX ux_tn_form_records_tenant_id ON tn_form_records(tenant_id, id);

-- values 收口：新记录（physical 存储）不再写 JSONB 业务值；存量行保持原值
-- 供兼容读取与离线迁移。
COMMENT ON COLUMN tn_form_records.values IS '业务值 JSONB：仅存量历史记录；新记录（physical 存储）恒为 NULL，业务值在 tn_fd_* 物理表';
ALTER TABLE tn_form_records ALTER COLUMN values DROP NOT NULL;
