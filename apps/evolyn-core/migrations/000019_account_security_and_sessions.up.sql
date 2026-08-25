-- 000019: 账号会话体系与 MFA 数据层（ADR-009）。全部平台级表，归属
-- account_id（FK 级联删除），不含 tenant_id。会话/恢复码/因子均为
-- 状态语义列（revoked_at/used_at/disabled_at），不用 GORM 软删，
-- 避免与「可查询的活跃集合」过滤语义混淆。

-- 账号安全开关（一行一账号）
CREATE TABLE IF NOT EXISTS account_security_settings (
    account_id BIGINT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    single_session_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at timestamp with time zone
);

-- 已验证的 MFA 因子（一期仅 TOTP）；secret 仅密文保存，key_version 支持主密钥轮换
CREATE TABLE IF NOT EXISTS account_mfa_factors (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    type varchar(16) NOT NULL,
    secret_ciphertext varchar(1024) NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    verified_at timestamp with time zone,
    last_used_counter BIGINT NOT NULL DEFAULT 0,
    disabled_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_account_mfa_factors_type CHECK (type IN ('totp')),
    CONSTRAINT chk_account_mfa_factors_key_version CHECK (key_version >= 1),
    CONSTRAINT chk_account_mfa_factors_last_used_counter CHECK (last_used_counter >= 0)
);

-- 同账号同类型至多一个活跃因子（历史因子保留 disabled_at 审计轨迹）
CREATE UNIQUE INDEX IF NOT EXISTS uk_account_mfa_factors_active
    ON account_mfa_factors (account_id, type)
    WHERE disabled_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_account_mfa_factors_account
    ON account_mfa_factors (account_id);

-- 一次性恢复码：只存摘要，明文仅创建时展示一次；used_at 为消费标记
CREATE TABLE IF NOT EXISTS account_mfa_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    code_digest varchar(128) NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_account_mfa_recovery_codes_available
    ON account_mfa_recovery_codes (account_id)
    WHERE used_at IS NULL;

-- 设备级逻辑会话：可查询、可撤销、可判断并发登录。sid 为 JWT 携带的
-- 公开会话标识（服务端随机），token_version 随租户切换重签递增
CREATE TABLE IF NOT EXISTS account_sessions (
    id BIGSERIAL PRIMARY KEY,
    sid varchar(64) NOT NULL,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_version BIGINT NOT NULL DEFAULT 1,
    auth_method varchar(16) NOT NULL,
    mfa_method varchar(16),
    created_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    revoke_reason varchar(32),
    ip varchar(45),
    location varchar(128),
    user_agent varchar(512),
    CONSTRAINT chk_account_sessions_auth_method CHECK (auth_method IN ('password', 'sms', 'oauth', 'register')),
    CONSTRAINT chk_account_sessions_mfa_method CHECK (mfa_method IS NULL OR mfa_method IN ('totp', 'recovery')),
    CONSTRAINT chk_account_sessions_revoke_reason CHECK (revoke_reason IS NULL OR revoke_reason IN ('logout', 'replaced', 'password_changed', 'phone_changed', 'mfa_changed', 'admin_revoked')),
    CONSTRAINT chk_account_sessions_token_version CHECK (token_version >= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_account_sessions_sid
    ON account_sessions (sid);

-- 账号活跃会话集合（单会话挤出/会话列表）
CREATE INDEX IF NOT EXISTS idx_account_sessions_account_active
    ON account_sessions (account_id)
    WHERE revoked_at IS NULL;

-- 过期/撤销会话的清理 Worker 扫描口
CREATE INDEX IF NOT EXISTS idx_account_sessions_expires
    ON account_sessions (expires_at)
    WHERE revoked_at IS NULL;

-- 账号安全流水：MFA 开关、恢复码、会话挤出等事件（追加写，不改语义）
CREATE TABLE IF NOT EXISTS account_security_events (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    event_type varchar(32) NOT NULL,
    session_id varchar(64),
    request_id varchar(64),
    ip varchar(45),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_account_security_events_account
    ON account_security_events (account_id, created_at);
