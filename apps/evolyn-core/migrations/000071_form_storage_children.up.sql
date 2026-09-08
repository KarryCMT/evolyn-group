-- 子表单物理子表永久映射（方案 §5.4，Phase 3）：storage_id + parent_field_id
-- → table_name。表名唯一约束是子表名防重边界（与 tn_form_storages.table_name
-- 对父表的作用一致）；发布事务内先插元数据、唯一冲突换名重试，绝不先查
-- PostgreSQL catalog 再建表。
CREATE TABLE tn_form_storage_children (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    storage_id BIGINT NOT NULL REFERENCES tn_form_storages(id),
    parent_field_id VARCHAR(16) NOT NULL,
    table_name VARCHAR(63) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT LOCALTIMESTAMP,
    CONSTRAINT uq_tn_form_storage_children_field UNIQUE (storage_id, parent_field_id),
    CONSTRAINT uq_tn_form_storage_children_table UNIQUE (table_name)
);
COMMENT ON TABLE tn_form_storage_children IS '子表单物理子表永久映射：fieldId→表名一次性分配后永不变更';
COMMENT ON COLUMN tn_form_storage_children.storage_id IS '所属存储绑定（tn_form_storages.id）';
COMMENT ON COLUMN tn_form_storage_children.parent_field_id IS '子表单字段的不可变 fieldId';
COMMENT ON COLUMN tn_form_storage_children.table_name IS '子表名（tn_fc_ 前缀，服务端分配，全局唯一约束防重）';
