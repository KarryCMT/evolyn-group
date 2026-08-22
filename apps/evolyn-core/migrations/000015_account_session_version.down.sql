-- 000015 回滚：移除账号会话版本列
ALTER TABLE accounts
    DROP COLUMN IF EXISTS session_version;
