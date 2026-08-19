-- FIX-004/005：Member 身份唯一性与 Account 外键。
-- 先清理历史异常数据，再加约束，保证迁移可重复执行（幂等）。

-- 1. 孤儿/无效成员：account_id = 0 或账号不存在的成员行删除（软删语义）
DELETE FROM users
WHERE account_id = 0
   OR NOT EXISTS (SELECT 1 FROM accounts a WHERE a.id = users.account_id);

-- 2. 同租户重复成员：每组 (tenant_id, account_id) 保留最小 ID，其余删除
DELETE FROM users
WHERE id NOT IN (
    SELECT MIN(id) FROM users
    WHERE deleted_at IS NULL
    GROUP BY tenant_id, account_id
)
AND deleted_at IS NULL;

-- 3. 关系表中指向被清理成员的残留行（无软删语义，物理清理）
DELETE FROM user_roles WHERE user_id NOT IN (SELECT id FROM users);
DELETE FROM user_groups WHERE user_id NOT IN (SELECT id FROM users);
DELETE FROM department_users WHERE user_id NOT IN (SELECT id FROM users);

-- 4. account_id 正式外键：阻止写入不存在的账号（FIX-005）
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_account;
ALTER TABLE users
    ADD CONSTRAINT fk_users_account
    FOREIGN KEY (account_id) REFERENCES accounts(id);

-- 5. (tenant_id, account_id) 租户内唯一（软删友好，FIX-004）：
--    一个账号在一个租户仅一个有效成员；软删后可重新加入
CREATE UNIQUE INDEX IF NOT EXISTS uk_users_tenant_account
    ON users (tenant_id, account_id)
    WHERE deleted_at IS NULL;
