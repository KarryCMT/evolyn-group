-- FIX-014/016：租户模型分层与 Owner 外键。
-- tenants 表移除自身的 tenant_id 列（「租户属于哪个租户」的语义问题）；
-- owner_account_id 由 0 哨兵改为可空真实外键（NULL = 暂无 Owner）。
-- FIX-023（MIGRATE-INT-001）：先解除 NOT NULL 再转 NULL——000001 的种子
-- 租户 owner 落 0 哨兵，空库全链执行时「先 UPDATE 后 DROP NOT NULL」会触发
-- 23502，导致迁移链在干净数据库上必然失败

-- 1. 列先改为可空、去默认值（为哨兵转 NULL 扫清约束）
ALTER TABLE tenants ALTER COLUMN owner_account_id DROP NOT NULL;
ALTER TABLE tenants ALTER COLUMN owner_account_id DROP DEFAULT;

-- 2. Owner 哨兵值转 NULL（不存在的账号引用一并置空，交给 FK 兜底后续写入）
UPDATE tenants SET owner_account_id = NULL
WHERE owner_account_id = 0
   OR NOT EXISTS (SELECT 1 FROM accounts a WHERE a.id = tenants.owner_account_id);

-- 3. Owner 正式外键（FIX-016）
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS fk_tenants_owner;
ALTER TABLE tenants
    ADD CONSTRAINT fk_tenants_owner
    FOREIGN KEY (owner_account_id) REFERENCES accounts(id);

-- 4. 移除 tenants 自身的 tenant_id 列（FIX-014）
ALTER TABLE tenants DROP COLUMN IF EXISTS tenant_id;

