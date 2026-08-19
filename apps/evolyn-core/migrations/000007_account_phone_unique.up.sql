-- FIX-023（SEC-TENANT 集成验收发现）：accounts.phone 是非指针列，未填手机号
-- 的账号会写入空串 ''，第二个无手机号账号即触发 23505 唯一冲突——开通租户
-- 新建 owner（phone 常为空）必现。改为软删除友好的部分唯一索引：
-- 仅非空手机号参与唯一性，与 uk_* 系列索引同一风格。

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_phone_key;

CREATE UNIQUE INDEX IF NOT EXISTS uk_accounts_phone
    ON accounts (phone)
    WHERE phone <> '' AND deleted_at IS NULL;
