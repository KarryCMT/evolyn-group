-- 000012 回滚：移除密码初始化标记列
ALTER TABLE accounts
    DROP COLUMN IF EXISTS password_initialized;
