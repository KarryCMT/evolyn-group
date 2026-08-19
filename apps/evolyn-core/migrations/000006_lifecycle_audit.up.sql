-- FIX-012/013：租户注销生命周期与业务审计日志。

-- FIX-012：注销请求/保留截止/清理完成时间线（Purge Worker 消费）
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS delete_requested_at timestamp with time zone;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS retention_until timestamp with time zone;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS purged_at timestamp with time zone;

-- 已处于 deleted 状态的存量租户补时间线（以迁移执行时刻起算保留期，30 天）
UPDATE tenants
SET delete_requested_at = COALESCE(delete_requested_at, LOCALTIMESTAMP),
    retention_until = COALESCE(retention_until, LOCALTIMESTAMP + INTERVAL '30 days')
WHERE status = 'deleted';

-- FIX-013：业务审计日志（追加写流水，谁在什么租户对什么资源做了什么）
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    account_id BIGINT NOT NULL DEFAULT 0,
    member_id BIGINT NOT NULL DEFAULT 0,
    module varchar(64) NOT NULL,
    action varchar(64) NOT NULL,
    resource_type varchar(64) NOT NULL,
    resource_id varchar(128),
    request_id varchar(64),
    ip varchar(64),
    user_agent varchar(256),
    before_data JSONB,
    after_data JSONB,
    created_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs (tenant_id, id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_module_action ON audit_logs (module, action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs (resource_type, resource_id);
