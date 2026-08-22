-- 000013: 登录日志（个人中心「账号设置-登录日志」自查数据源）。
-- 记录会话建立事件（登录/注册即登录），平台级账号维度（ADR-006 账号×成员
-- 拆分：登录身份挂账号），追加写流水无更新/软删语义；与 audit_logs（业务
-- 操作审计）职责互斥：登录动作只落本表，不写业务审计。tenant_id/member_id
-- 为本次会话绑定的租户上下文快照，仅供扩展分析，不做租户 Callback 过滤
CREATE TABLE IF NOT EXISTS login_logs (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    account_id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    member_id BIGINT NOT NULL DEFAULT 0,
    method varchar(32) NOT NULL,
    client varchar(32) NOT NULL DEFAULT 'unknown',
    ip varchar(64),
    location varchar(128),
    user_agent varchar(256),
    request_id varchar(64),
    created_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_login_logs_account ON login_logs (account_id, id);

COMMENT ON TABLE login_logs IS '登录日志：会话建立事件流水（登录/注册即登录），账号维度自查（ADR-006 平台级）；与 audit_logs 职责互斥，登录不写业务审计';
COMMENT ON COLUMN login_logs.id IS '自增主键';
COMMENT ON COLUMN login_logs.account_id IS '登录的平台账号 ID（主查询维度）';
COMMENT ON COLUMN login_logs.tenant_id IS '本次登录进入的租户 ID，0=无租户/平台场景';
COMMENT ON COLUMN login_logs.member_id IS '本次登录绑定的租户成员 ID，0=未解析到成员';
COMMENT ON COLUMN login_logs.method IS '登录方式：password 密码 / sms 短信验证码 / oauth_github、oauth_wechat 第三方 / register 注册即登录';
COMMENT ON COLUMN login_logs.client IS '客户端形态（UA 解析）：web 电脑网页 / wap 手机网页 / unknown';
COMMENT ON COLUMN login_logs.ip IS '客户端 IP，可空';
COMMENT ON COLUMN login_logs.location IS 'IP 归属地（ip2region 离线库写时解析，如「广东省 深圳市」）；回环/私网为「内网地址」，解析失败为「未知」';
COMMENT ON COLUMN login_logs.user_agent IS '客户端 User-Agent，可空';
COMMENT ON COLUMN login_logs.request_id IS '链路追踪请求 ID，可空';
COMMENT ON COLUMN login_logs.created_at IS '登录时间';
