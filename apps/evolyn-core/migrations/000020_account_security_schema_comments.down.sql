-- 000020 down：回滚账号安全数据字典注释，不影响表结构与业务数据。

COMMENT ON COLUMN account_security_events.created_at IS NULL;
COMMENT ON COLUMN account_security_events.metadata IS NULL;
COMMENT ON COLUMN account_security_events.ip IS NULL;
COMMENT ON COLUMN account_security_events.request_id IS NULL;
COMMENT ON COLUMN account_security_events.session_id IS NULL;
COMMENT ON COLUMN account_security_events.event_type IS NULL;
COMMENT ON COLUMN account_security_events.account_id IS NULL;
COMMENT ON COLUMN account_security_events.id IS NULL;
COMMENT ON TABLE account_security_events IS NULL;

COMMENT ON COLUMN account_sessions.user_agent IS NULL;
COMMENT ON COLUMN account_sessions.location IS NULL;
COMMENT ON COLUMN account_sessions.ip IS NULL;
COMMENT ON COLUMN account_sessions.revoke_reason IS NULL;
COMMENT ON COLUMN account_sessions.revoked_at IS NULL;
COMMENT ON COLUMN account_sessions.expires_at IS NULL;
COMMENT ON COLUMN account_sessions.last_seen_at IS NULL;
COMMENT ON COLUMN account_sessions.created_at IS NULL;
COMMENT ON COLUMN account_sessions.mfa_method IS NULL;
COMMENT ON COLUMN account_sessions.auth_method IS NULL;
COMMENT ON COLUMN account_sessions.token_version IS NULL;
COMMENT ON COLUMN account_sessions.account_id IS NULL;
COMMENT ON COLUMN account_sessions.sid IS NULL;
COMMENT ON COLUMN account_sessions.id IS NULL;
COMMENT ON TABLE account_sessions IS NULL;

COMMENT ON COLUMN account_mfa_recovery_codes.created_at IS NULL;
COMMENT ON COLUMN account_mfa_recovery_codes.used_at IS NULL;
COMMENT ON COLUMN account_mfa_recovery_codes.code_digest IS NULL;
COMMENT ON COLUMN account_mfa_recovery_codes.account_id IS NULL;
COMMENT ON COLUMN account_mfa_recovery_codes.id IS NULL;
COMMENT ON TABLE account_mfa_recovery_codes IS NULL;

COMMENT ON COLUMN account_mfa_factors.updated_at IS NULL;
COMMENT ON COLUMN account_mfa_factors.created_at IS NULL;
COMMENT ON COLUMN account_mfa_factors.disabled_at IS NULL;
COMMENT ON COLUMN account_mfa_factors.last_used_counter IS NULL;
COMMENT ON COLUMN account_mfa_factors.verified_at IS NULL;
COMMENT ON COLUMN account_mfa_factors.key_version IS NULL;
COMMENT ON COLUMN account_mfa_factors.secret_ciphertext IS NULL;
COMMENT ON COLUMN account_mfa_factors.type IS NULL;
COMMENT ON COLUMN account_mfa_factors.account_id IS NULL;
COMMENT ON COLUMN account_mfa_factors.id IS NULL;
COMMENT ON TABLE account_mfa_factors IS NULL;

COMMENT ON COLUMN account_security_settings.updated_at IS NULL;
COMMENT ON COLUMN account_security_settings.single_session_enabled IS NULL;
COMMENT ON COLUMN account_security_settings.mfa_enabled IS NULL;
COMMENT ON COLUMN account_security_settings.account_id IS NULL;
COMMENT ON TABLE account_security_settings IS NULL;
