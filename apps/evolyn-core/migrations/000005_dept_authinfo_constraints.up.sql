-- FIX-015/017：部门父引用与第三方身份唯一。

-- FIX-015：departments.parent_id 由 0=root 迁移为 NULL=root，并加自引用外键
UPDATE departments SET parent_id = NULL WHERE parent_id = 0;
-- 指向不存在父部门的行一并置 NULL（孤儿挂回根，与树组装逻辑一致）
UPDATE departments SET parent_id = NULL
WHERE parent_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM departments p WHERE p.id = departments.parent_id);
-- 跨租户父部门（历史数据）同样置 NULL：同租户约束由服务层校验
UPDATE departments d SET parent_id = NULL
WHERE parent_id IS NOT NULL
  AND parent_id IN (
      SELECT p.id FROM departments p WHERE p.tenant_id <> d.tenant_id
  );

ALTER TABLE departments ALTER COLUMN parent_id DROP NOT NULL;
ALTER TABLE departments ALTER COLUMN parent_id DROP DEFAULT;

ALTER TABLE departments DROP CONSTRAINT IF EXISTS fk_departments_parent;
ALTER TABLE departments
    ADD CONSTRAINT fk_departments_parent
    FOREIGN KEY (parent_id) REFERENCES departments(id);

-- FIX-017：第三方身份 (auth_type, auth_id) 唯一（软删友好）：
-- 同一外部身份不允许绑定多个账号
CREATE UNIQUE INDEX IF NOT EXISTS uk_auth_identity
    ON auth_infos (auth_type, auth_id)
    WHERE deleted_at IS NULL;
