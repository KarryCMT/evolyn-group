-- 000076: 流水号服务端计数器。计数范围固定为租户×表单×字段×周期键，
-- 由提交事务中的 UPSERT 原子推进；不复用已发号的计数。
CREATE TABLE tn_form_serial_counters (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    form_id BIGINT NOT NULL,
    field_id VARCHAR(10) NOT NULL,
    cycle_key VARCHAR(16) NOT NULL,
    next_value BIGINT NOT NULL CHECK (next_value > 0),
    created_at TIMESTAMP NOT NULL DEFAULT LOCALTIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT LOCALTIMESTAMP,
    UNIQUE (tenant_id, form_id, field_id, cycle_key)
);

CREATE INDEX idx_tn_form_serial_counters_form ON tn_form_serial_counters (tenant_id, form_id);
