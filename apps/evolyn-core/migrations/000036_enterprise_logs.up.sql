-- 000036: 企业日志（管理后台「企业日志」一期，docs/低代码平台/企业日志/）。
-- 1) login_logs / audit_logs 补企业日志展示投影字段与租户维度查询索引：
--    展示快照写时固化（成员改名/离职/删除后历史展示一致），存量历史行读取侧
--    以「历史操作记录」降级展示；
-- 2) 新增 enterprise_log_exports 导出任务表：一期同步生成、内容内联存储，
--    异步导出/对象存储文件引用/留存清理随后续批次接入；
-- 3) 按管理员规则签名补授 enterprise-logs 资源权限（与 000034/000035 同口径）。

-- ---------- login_logs：登录人显示名快照 + 租户维度查询索引 ----------
ALTER TABLE login_logs ADD COLUMN actor_name_snapshot varchar(128) NOT NULL DEFAULT '';

COMMENT ON COLUMN login_logs.actor_name_snapshot IS '登录人显示名快照（写时固化，成员改名/离职/删除后历史展示一致）；空为存量历史行，读取侧回查当前成员昵称兜底';

CREATE INDEX idx_login_logs_tenant_created ON login_logs (tenant_id, created_at DESC, id DESC);
CREATE INDEX idx_login_logs_tenant_member_created ON login_logs (tenant_id, member_id, created_at DESC, id DESC);

-- ---------- audit_logs：企业日志展示投影（事件码/分类/快照/摘要） ----------
ALTER TABLE audit_logs
    ADD COLUMN event_code varchar(100) NOT NULL DEFAULT '',
    ADD COLUMN category_code varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN actor_name_snapshot varchar(128) NOT NULL DEFAULT '',
    ADD COLUMN target_name_snapshot varchar(256) NOT NULL DEFAULT '',
    ADD COLUMN summary varchar(1000) NOT NULL DEFAULT '';

COMMENT ON COLUMN audit_logs.event_code IS '稳定事件码（模块.资源类型.动作，如 iam.member.update），由审计服务按事件注册表生成；空为存量历史行，展示降级为「历史操作记录」';
COMMENT ON COLUMN audit_logs.category_code IS '稳定日志范围码：member_management 成员管理 / organization 组织架构 / role_permission 角色权限 / tenant_settings 企业设置 / application 应用管理 / file_storage 文件管理 / account_security 账号安全 / log_export 日志导出';
COMMENT ON COLUMN audit_logs.actor_name_snapshot IS '操作人显示名快照（写时固化，成员资料变更不影响历史展示）';
COMMENT ON COLUMN audit_logs.target_name_snapshot IS '目标资源展示名快照（成员/部门/角色/应用等当时名称）';
COMMENT ON COLUMN audit_logs.summary IS '服务端生成并经脱敏的操作详情，可直接展示与导出；不含密码/验证码/令牌/私钥/完整手机号邮箱等敏感值';

CREATE INDEX idx_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC, id DESC);
CREATE INDEX idx_audit_logs_tenant_category_created ON audit_logs (tenant_id, category_code, created_at DESC, id DESC);
CREATE INDEX idx_audit_logs_tenant_member_created ON audit_logs (tenant_id, member_id, created_at DESC, id DESC);

-- ---------- enterprise_log_exports：导出任务表 ----------
CREATE TABLE enterprise_log_exports (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL DEFAULT 0,
    member_id BIGINT NOT NULL DEFAULT 0,
    kind varchar(16) NOT NULL,
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    total BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'pending',
    file_name varchar(128) NOT NULL DEFAULT '',
    file_data TEXT NOT NULL DEFAULT '',
    expires_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

COMMENT ON TABLE enterprise_log_exports IS '企业日志导出任务（000036）：固化提交时的筛选条件/申请人/租户/数据量/状态与过期时间；一期同步生成、内容内联存储，异步导出与对象存储文件引用随留存策略批次接入';
COMMENT ON COLUMN enterprise_log_exports.id IS '自增主键';
COMMENT ON COLUMN enterprise_log_exports.tenant_id IS '所属租户；任务状态读取与下载均复核归属，跨租户不可见';
COMMENT ON COLUMN enterprise_log_exports.account_id IS '申请人平台账号 ID';
COMMENT ON COLUMN enterprise_log_exports.member_id IS '申请人租户成员 ID，0=未解析到成员';
COMMENT ON COLUMN enterprise_log_exports.kind IS '导出日志类型：login 登录日志 / operation 操作日志';
COMMENT ON COLUMN enterprise_log_exports.filters IS '提交时固化的筛选条件快照 JSONB（成员/日志范围/事件码/时间范围），与列表查询参数同构';
COMMENT ON COLUMN enterprise_log_exports.total IS '导出数据量（行数）';
COMMENT ON COLUMN enterprise_log_exports.status IS '任务状态：pending 生成中 / ready 就绪 / failed 生成失败';
COMMENT ON COLUMN enterprise_log_exports.file_name IS '导出文件名（下载 Content-Disposition 用）';
COMMENT ON COLUMN enterprise_log_exports.file_data IS '导出文件内容（一期 CSV 内联存储；异步批改对象存储文件引用后本列退化为空串）';
COMMENT ON COLUMN enterprise_log_exports.expires_at IS '导出文件过期时间，过期后不可下载';
COMMENT ON COLUMN enterprise_log_exports.created_at IS '任务创建时间';

CREATE INDEX idx_enterprise_log_exports_tenant ON enterprise_log_exports (tenant_id, id DESC);

-- ---------- 基线管理员权限补授（与 000035 同口径） ----------
-- view 展开 get+list 覆盖登录/操作日志与筛选项查询接口；create 覆盖
-- POST /enterprise-logs/exports（enterprise-logs:export 语义），下载路径在
-- 控制器内复核 create 动词后放行
UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'enterprise-logs', 'operation', 'view'),
        jsonb_build_object('resource', 'enterprise-logs', 'operation', 'create')
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'enterprise-logs'
  );
