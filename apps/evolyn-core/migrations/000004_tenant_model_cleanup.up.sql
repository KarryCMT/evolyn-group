-- FIX-014/016：租户模型分层与 Owner 外键。
-- tenants 表移除自身的 tenant_id 列（「租户属于哪个租户」的语义问题）；
-- owner_account_id 由 0 哨兵改为可空真实外键（NULL = 暂无 Owner）。

-- 1. Owner 哨兵值转 NULL（不存在的账号引用一并置空，交给 FK 兜底后续写入）
UPDATE tenants SET owner_account_id = NULL
WHERE owner_account_id = 0
   OR NOT EXISTS (SELECT 1 FROM accounts a WHERE a.id = tenants.owner_account_id);

-- 2. 列改为可空、去默认值
ALTER TABLE tenants ALTER COLUMN owner_account_id DROP NOT NULL;
ALTER TABLE tenants ALTER COLUMN owner_account_id DROP DEFAULT;

-- 3. Owner 正式外键（FIX-016）
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS fk_tenants_owner;
ALTER TABLE tenants
    ADD CONSTRAINT fk_tenants_owner
    FOREIGN KEY (owner_account_id) REFERENCES accounts(id);

-- 4. 移除 tenants 自身的 tenant_id 列（FIX-014）
ALTER TABLE tenants DROP COLUMN IF EXISTS tenant_id;
