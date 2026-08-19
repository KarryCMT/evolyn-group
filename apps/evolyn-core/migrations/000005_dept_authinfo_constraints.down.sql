-- 回滚：恢复 0=root 语义、移除自引用外键与第三方身份唯一索引
DROP INDEX IF EXISTS uk_auth_identity;
ALTER TABLE departments DROP CONSTRAINT IF EXISTS fk_departments_parent;
ALTER TABLE departments ALTER COLUMN parent_id SET DEFAULT 0;
ALTER TABLE departments ALTER COLUMN parent_id SET NOT NULL;
UPDATE departments SET parent_id = 0 WHERE parent_id IS NULL;
