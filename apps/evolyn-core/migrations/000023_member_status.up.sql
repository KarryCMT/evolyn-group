-- 000023: 成员租户内生命周期。状态属于 users，不影响同一平台账号在其他租户的成员关系。
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS status varchar(16) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS resigned_at timestamp with time zone,
    ADD CONSTRAINT chk_users_status CHECK (status IN ('active', 'disabled', 'resigned'));

COMMENT ON COLUMN users.status IS '成员状态：active 启用、disabled 停用、resigned 离职；全部成员视图默认不含 resigned';
COMMENT ON COLUMN users.resigned_at IS '成员转为离职的时间；恢复启用或停用时清空';

CREATE INDEX IF NOT EXISTS idx_users_tenant_status_id
    ON users (tenant_id, status, id)
    WHERE deleted_at IS NULL;
