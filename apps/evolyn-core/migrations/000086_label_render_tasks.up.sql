-- 000086: 二维码标签异步批量渲染任务与逐记录进度。
CREATE TABLE IF NOT EXISTS tn_label_render_tasks (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    code VARCHAR(64) NOT NULL,
    template_id BIGINT NOT NULL REFERENCES tn_label_templates(id),
    app_id BIGINT NOT NULL REFERENCES tn_apps(id),
    form_id BIGINT NOT NULL REFERENCES tn_forms(id),
    template_version_id BIGINT NOT NULL REFERENCES tn_label_template_versions(id),
    template_version_no INTEGER NOT NULL,
    output_format VARCHAR(12) NOT NULL DEFAULT 'pdf',
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    total_count INTEGER NOT NULL,
    success_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    progress INTEGER NOT NULL DEFAULT 0,
    file_code VARCHAR(64),
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message VARCHAR(500) NOT NULL DEFAULT '',
    requested_by_member_id BIGINT NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tn_label_render_tasks_code UNIQUE (tenant_id, code),
    CONSTRAINT ck_tn_label_render_tasks_format CHECK (output_format = 'pdf'),
    CONSTRAINT ck_tn_label_render_tasks_status CHECK (status IN ('pending', 'running', 'success', 'partial_success', 'failed', 'cancelled')),
    CONSTRAINT ck_tn_label_render_tasks_counts CHECK (total_count > 0 AND success_count >= 0 AND failed_count >= 0 AND success_count + failed_count <= total_count),
    CONSTRAINT ck_tn_label_render_tasks_progress CHECK (progress BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_tn_label_render_tasks_tenant_created
    ON tn_label_render_tasks (tenant_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS tn_label_render_task_items (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL REFERENCES tn_label_render_tasks(id) ON DELETE CASCADE,
    record_id BIGINT NOT NULL,
    sequence_no INTEGER NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message VARCHAR(500) NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tn_label_render_task_items_sequence UNIQUE (task_id, sequence_no),
    CONSTRAINT ck_tn_label_render_task_items_status CHECK (status IN ('pending', 'running', 'success', 'failed')),
    CONSTRAINT ck_tn_label_render_task_items_sequence CHECK (sequence_no > 0)
);

CREATE INDEX IF NOT EXISTS idx_tn_label_render_task_items_task
    ON tn_label_render_task_items (tenant_id, task_id, sequence_no);

COMMENT ON TABLE tn_label_render_tasks IS '二维码标签批量渲染任务：固定模板发布快照，任务事务提交后由 Asynq 消费';
COMMENT ON TABLE tn_label_render_task_items IS '批量标签逐记录进度与安全错误摘要，不保存业务字段快照';
