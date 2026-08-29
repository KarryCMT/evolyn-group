-- 000053：流程变量表 + wf_job 类型约束扩展（流程引擎 Phase 7，ADR-012）
-- service 节点响应映射写流程变量（表达式 variables.* 数据源），并以
-- service.invoke Job 承载异步 HTTP 调用（业务事务内不发外部请求）。
CREATE TABLE IF NOT EXISTS wf_variable (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    var_key varchar(64) NOT NULL,
    var_type varchar(16) NOT NULL,
    var_value JSONB NOT NULL DEFAULT 'null',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_variable_type CHECK (var_type IN ('string', 'number', 'boolean')),
    CONSTRAINT uq_wf_variable_instance_key UNIQUE (instance_id, var_key)
);

-- 实例维度变量加载（推进上下文构造）
CREATE INDEX IF NOT EXISTS idx_wf_variable_instance
    ON wf_variable (tenant_id, instance_id);

COMMENT ON TABLE wf_variable IS '流程变量（实例作用域，(instance_id,var_key) 唯一）：service 节点响应映射等写入，表达式经 variables.* 白名单数据源读取；V1 冻结标量值域（string/number/boolean）';
COMMENT ON COLUMN wf_variable.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_variable.instance_id IS '归属流程实例 ID';
COMMENT ON COLUMN wf_variable.var_key IS '变量名（字母开头标识符，实例内唯一，校验器冻结命名规则）';
COMMENT ON COLUMN wf_variable.var_type IS '值类型（V1 冻结标量集合）：string/number/boolean';
COMMENT ON COLUMN wf_variable.var_value IS '变量值 JSONB（标量；提取失败且非 required 时不落行）';
COMMENT ON COLUMN wf_variable.created_at IS '创建时间';
COMMENT ON COLUMN wf_variable.updated_at IS '更新时间（同键覆盖写）';

-- wf_job 类型目录扩展：service.invoke（异步 HTTP 调用，失败重试经重试
-- 记账退避回队，即第 19.1 章预留的 service.retry 语义）
ALTER TABLE wf_job DROP CONSTRAINT IF EXISTS chk_wf_job_type;
ALTER TABLE wf_job ADD CONSTRAINT chk_wf_job_type
    CHECK (job_type IN ('task.reminder', 'task.timeout', 'service.invoke'));

COMMENT ON COLUMN wf_job.job_type IS '任务类型：task.reminder=待办提醒 / task.timeout=待办超时自动动作 / service.invoke=服务节点异步 HTTP 调用（000053 扩展）';
