-- 回滚物理表存储 Phase 1：删除记录信封投影列与元数据三表（动态物理表
-- tn_fd_*/tn_fc_* 不由本迁移管理——存在动态表时应先人工清理，见方案 §17）。
UPDATE tn_form_records SET values = '{}'::jsonb WHERE values IS NULL;
ALTER TABLE tn_form_records ALTER COLUMN values SET NOT NULL;
COMMENT ON COLUMN tn_form_records.values IS NULL;
DROP INDEX IF EXISTS ux_tn_form_records_tenant_id;
COMMENT ON COLUMN tn_form_records.workflow_updated_at IS NULL;
COMMENT ON COLUMN tn_form_records.workflow_status IS NULL;
ALTER TABLE tn_form_records DROP COLUMN IF EXISTS workflow_updated_at;
ALTER TABLE tn_form_records DROP COLUMN IF EXISTS workflow_status;
DROP INDEX IF EXISTS ix_tn_form_storage_versions_storage;
DROP INDEX IF EXISTS ix_tn_form_ddl_jobs_claim;
DROP TABLE IF EXISTS tn_form_ddl_jobs;
DROP TABLE IF EXISTS tn_form_storage_schema_versions; -- 含 updated_at
DROP TABLE IF EXISTS tn_form_storages;
