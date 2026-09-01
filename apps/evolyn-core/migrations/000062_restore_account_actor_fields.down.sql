-- 回滚 000062：移除本版本补建的账号审计字段，成员归属字段保持不变。

ALTER TABLE files DROP COLUMN IF EXISTS creator_id;
ALTER TABLE role_groups DROP COLUMN IF EXISTS creator_id;
