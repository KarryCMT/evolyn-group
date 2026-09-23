-- 000087: 二维码短 Token。扫码必须在认证租户上下文中再次执行记录权限校验。
CREATE TABLE IF NOT EXISTS tn_label_qr_tokens (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    token VARCHAR(64) NOT NULL,
    app_id BIGINT NOT NULL REFERENCES tn_apps(id),
    form_id BIGINT NOT NULL REFERENCES tn_forms(id),
    record_id BIGINT NOT NULL REFERENCES tn_form_records(id),
    template_id BIGINT NOT NULL REFERENCES tn_label_templates(id),
    target_type VARCHAR(32) NOT NULL DEFAULT 'form_record',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    expire_at TIMESTAMPTZ,
    creator_member_id BIGINT NOT NULL,
    last_scan_at TIMESTAMPTZ,
    scan_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tn_label_qr_tokens_token UNIQUE (token),
    CONSTRAINT uk_tn_label_qr_tokens_target UNIQUE (tenant_id, form_id, record_id, template_id, target_type),
    CONSTRAINT ck_tn_label_qr_tokens_target_type CHECK (target_type = 'form_record'),
    CONSTRAINT ck_tn_label_qr_tokens_scan_count CHECK (scan_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_tn_label_qr_tokens_record
    ON tn_label_qr_tokens (tenant_id, form_id, record_id);

COMMENT ON TABLE tn_label_qr_tokens IS '二维码不可猜测短 Token；命中后仍按当前租户成员重新校验表单记录权限';
