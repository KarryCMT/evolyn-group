-- 000015: 账号会话版本（密码更新后全会话失效）
-- 密码找回或修改成功时，与密码散列在同一 UPDATE 内递增。JWT 携带签发时版本，
-- 认证中间件比对不一致即拒绝，避免 7 天无状态令牌在密码更新后继续可用。
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS session_version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN accounts.session_version IS '账号会话版本：密码重设/修改成功时递增，JWT 版本不一致即失效';
