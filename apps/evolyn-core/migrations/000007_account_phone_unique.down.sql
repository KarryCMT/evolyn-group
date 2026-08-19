-- 回滚 000007：恢复约束形态的全局唯一（含空串，weave 时代口径）
DROP INDEX IF EXISTS uk_accounts_phone;

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_phone_key;
ALTER TABLE accounts ADD CONSTRAINT accounts_phone_key UNIQUE (phone);
