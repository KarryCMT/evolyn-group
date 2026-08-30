-- 000057：表单记录提交协议增加客户端幂等键与菜单入口快照。
-- 历史记录保持 NULL；新接口由服务层强制 data_op_id 非空且为 UUID。
ALTER TABLE form_records
    ADD COLUMN IF NOT EXISTS data_op_id varchar(36),
    ADD COLUMN IF NOT EXISTS entry_code varchar(64);

CREATE UNIQUE INDEX IF NOT EXISTS uk_form_records_tenant_data_op
    ON form_records (tenant_id, data_op_id);

COMMENT ON COLUMN form_records.data_op_id IS '客户端生成的单次提交幂等 UUID；同一租户内唯一，历史记录允许为空';
COMMENT ON COLUMN form_records.entry_code IS '触发提交的应用菜单节点公开编码快照；设计预览直提允许为空';
