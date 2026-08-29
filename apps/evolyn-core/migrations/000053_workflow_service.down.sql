-- 000053 回滚：删除流程变量表 + wf_job 类型约束还原（Phase 7）
DROP TABLE IF EXISTS wf_variable;

ALTER TABLE wf_job DROP CONSTRAINT IF EXISTS chk_wf_job_type;
ALTER TABLE wf_job ADD CONSTRAINT chk_wf_job_type
    CHECK (job_type IN ('task.reminder', 'task.timeout'));
