-- 000012: 账号「密码是否由用户设置」标记（短信免密注册）
-- 手机号+验证码注册的账号不收集密码，服务端落随机密码且标记 false：
-- 首次设置/修改密码免旧密码校验，成功后置位恢复常规校验。
-- 存量账号默认 true（密码均由注册/OAuth 首登链路写入）
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS password_initialized boolean NOT NULL DEFAULT true;

COMMENT ON COLUMN accounts.password_initialized IS '密码是否由用户本人设置：短信免密注册为 false（存服务端随机密码），首次设置密码后置 true';
