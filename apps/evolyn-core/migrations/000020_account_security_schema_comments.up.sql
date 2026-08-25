-- 000020：为 000019 新增的账号安全表补充数据字典注释。
-- 该迁移只更新 PostgreSQL 元数据，确保已应用 000019 的环境也能获得注释。

COMMENT ON TABLE account_security_settings IS '账号安全开关：平台账号维度，一行一账号；缺行等价于全部关闭';
COMMENT ON COLUMN account_security_settings.account_id IS '平台账号 ID，主键并外键指向 accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN account_security_settings.mfa_enabled IS '是否启用登录二次验证（TOTP MFA）';
COMMENT ON COLUMN account_security_settings.single_session_enabled IS '是否禁止同时登录；开启后新登录会撤销其他活跃会话';
COMMENT ON COLUMN account_security_settings.updated_at IS '安全设置最后更新时间';

COMMENT ON TABLE account_mfa_factors IS '账号 MFA 因子：一期仅 TOTP；密钥仅密文保存，历史停用因子保留审计轨迹';
COMMENT ON COLUMN account_mfa_factors.id IS 'MFA 因子主键';
COMMENT ON COLUMN account_mfa_factors.account_id IS '所属平台账号 ID，外键指向 accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN account_mfa_factors.type IS '因子类型：一期固定为 totp';
COMMENT ON COLUMN account_mfa_factors.secret_ciphertext IS 'TOTP 密钥的 AES-GCM 密文，绝不存储或出网明文';
COMMENT ON COLUMN account_mfa_factors.key_version IS '加密主密钥版本，按该版本解密以支持密钥轮换';
COMMENT ON COLUMN account_mfa_factors.verified_at IS '首次验证成功并启用时间，NULL 表示待确认';
COMMENT ON COLUMN account_mfa_factors.last_used_counter IS '最近已消费的 TOTP 时间步计数器，用于阻止动态码重放';
COMMENT ON COLUMN account_mfa_factors.disabled_at IS '停用时间，NULL 表示活跃；部分唯一索引仅约束活跃因子';
COMMENT ON COLUMN account_mfa_factors.created_at IS '创建时间';
COMMENT ON COLUMN account_mfa_factors.updated_at IS '更新时间';

COMMENT ON TABLE account_mfa_recovery_codes IS 'MFA 恢复码：明文仅在生成时展示一次，数据库仅保存不可逆摘要';
COMMENT ON COLUMN account_mfa_recovery_codes.id IS '恢复码记录主键';
COMMENT ON COLUMN account_mfa_recovery_codes.account_id IS '所属平台账号 ID，外键指向 accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN account_mfa_recovery_codes.code_digest IS '恢复码 SHA-256 摘要，绝不保存明文';
COMMENT ON COLUMN account_mfa_recovery_codes.used_at IS '消费时间，NULL 表示可用；更新受 used_at IS NULL 条件保护';
COMMENT ON COLUMN account_mfa_recovery_codes.created_at IS '创建时间';

COMMENT ON TABLE account_sessions IS '设备级账号会话：sid 写入 JWT，用于设备管理、单会话挤出和服务端撤销';
COMMENT ON COLUMN account_sessions.id IS '设备会话记录主键';
COMMENT ON COLUMN account_sessions.sid IS '随机设备会话公开标识，写入 JWT sid 声明，全局唯一';
COMMENT ON COLUMN account_sessions.account_id IS '所属平台账号 ID，外键指向 accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN account_sessions.token_version IS '会话令牌版本；租户切换重签时递增，旧令牌随即失效';
COMMENT ON COLUMN account_sessions.auth_method IS '登录第一步认证方式：password、sms、oauth 或 register';
COMMENT ON COLUMN account_sessions.mfa_method IS '完成第二步的 MFA 方法：totp 或 recovery；未启用 MFA 时为 NULL';
COMMENT ON COLUMN account_sessions.created_at IS '会话创建时间';
COMMENT ON COLUMN account_sessions.last_seen_at IS '最近请求时间，由认证中间件节流刷新';
COMMENT ON COLUMN account_sessions.expires_at IS '会话绝对过期时间，与 JWT 有效期对齐';
COMMENT ON COLUMN account_sessions.revoked_at IS '撤销时间，NULL 表示未撤销';
COMMENT ON COLUMN account_sessions.revoke_reason IS '撤销原因：logout、replaced、password_changed、phone_changed、mfa_changed 或 admin_revoked';
COMMENT ON COLUMN account_sessions.ip IS '登录来源 IP，支持 IPv4 和 IPv6';
COMMENT ON COLUMN account_sessions.location IS '登录地离线解析结果';
COMMENT ON COLUMN account_sessions.user_agent IS '登录设备 User-Agent 原文';

COMMENT ON TABLE account_security_events IS '账号安全事件流水：记录 MFA、恢复码和会话撤销等事件；不替代登录日志或业务审计';
COMMENT ON COLUMN account_security_events.id IS '安全事件记录主键';
COMMENT ON COLUMN account_security_events.account_id IS '所属平台账号 ID，外键指向 accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN account_security_events.event_type IS '安全事件类型，如 mfa_enabled、session_revoked、mfa_recovery_used';
COMMENT ON COLUMN account_security_events.session_id IS '关联设备会话 sid；无关联时为空字符串';
COMMENT ON COLUMN account_security_events.request_id IS '关联请求追踪 ID；无关联时为空字符串';
COMMENT ON COLUMN account_security_events.ip IS '操作来源 IP';
COMMENT ON COLUMN account_security_events.metadata IS '非敏感扩展元数据 JSONB，禁止写入密钥、验证码、恢复码或令牌';
COMMENT ON COLUMN account_security_events.created_at IS '创建时间';
