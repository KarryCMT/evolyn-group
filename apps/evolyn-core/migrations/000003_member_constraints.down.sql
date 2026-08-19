-- 回滚：移除成员唯一约束与账号外键（不恢复被清理的历史数据）
DROP INDEX IF EXISTS uk_users_tenant_account;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_account;
