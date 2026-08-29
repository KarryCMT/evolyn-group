-- 000050 回滚：移除部门负责人字段
DROP INDEX IF EXISTS idx_departments_leader_member_id;
ALTER TABLE departments DROP COLUMN IF EXISTS leader_member_id;
