-- 000038: 表单资产域 P2 —— 记录提交。
-- form_records 记录行（追加写，P2 无更新/删除/列表管理，记录管理页随 P4 落地）；
-- form-records 资源授予全体成员（authenticated 基线按系统角色名补授，口径同 000014）：
-- 提交与表单设计权限（forms 资源）彻底分离，普通成员可提交但不可管理表单。
CREATE TABLE IF NOT EXISTS form_records (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    form_id BIGINT NOT NULL REFERENCES forms(id),
    form_version_id BIGINT NOT NULL REFERENCES form_versions(id),
    values JSONB NOT NULL,
    submitted_by_member_id BIGINT NOT NULL,
    submitted_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_form_records_tenant_form
    ON form_records (tenant_id, form_id, id DESC);

UPDATE roles
SET rules = (rules::jsonb || '[{"resource": "form-records", "operation": "create"}]'::jsonb)::json
WHERE name = 'authenticated'
  AND deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'form-records'
  );

COMMENT ON TABLE form_records IS '表单记录提交（追加写）：values 为服务端按发布快照校验通过后的值（键=widgetName，仅快照内可见字段）；form_version_id 固定受理时所依据的发布版本（历史版本合法）';
COMMENT ON COLUMN form_records.id IS '自增主键（记录 ID）';
COMMENT ON COLUMN form_records.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN form_records.form_id IS '表单 ID';
COMMENT ON COLUMN form_records.form_version_id IS '受理时依据的发布快照行 ID（任意历史版本均可受理，字段定义可复现）';
COMMENT ON COLUMN form_records.values IS '字段值 JSONB（键=widgetName）；服务端终审通过的清洗值，隐藏字段与布局字段不落库';
COMMENT ON COLUMN form_records.submitted_by_member_id IS '提交人（租户成员 ID）';
COMMENT ON COLUMN form_records.submitted_at IS '提交时间';
COMMENT ON COLUMN form_records.created_at IS '创建时间';
