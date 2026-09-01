-- 000064: 产品日志（管理后台「产品日志」一期，docs/低代码平台/产品日志/）。
-- 1) tn_audit_logs 补应用维度投影三字段与租户×应用查询索引：支撑产品日志
--    「所属应用」展示、应用筛选与应用删除后的历史快照展示（快照与 ID 均
--    写时固化）；非应用内操作 application_id 为 NULL；
-- 2) 新增 tn_product_log_exports 导出任务表：一期同步生成、内容内联存储，
--    异步导出/对象存储文件引用/留存清理随后续批次接入；
-- 3) 按管理员规则签名补授 product-logs 资源权限（与 000036 同口径）。

-- ---------- tn_audit_logs：应用维度投影 ----------
ALTER TABLE tn_audit_logs
    ADD COLUMN application_id BIGINT NULL,
    ADD COLUMN application_code varchar(128) NOT NULL DEFAULT '',
    ADD COLUMN application_name_snapshot varchar(256) NOT NULL DEFAULT '';

COMMENT ON COLUMN tn_audit_logs.application_id IS '应用内操作所属应用 ID（产品日志查询与租户归属校验维度）；NULL=非应用内操作或历史数据。应用维度查询必须以前置 tenant_id 为首列，禁止仅凭本列查询';
COMMENT ON COLUMN tn_audit_logs.application_code IS '应用稳定编码快照（写时固化，应用删除后历史展示不依赖当前应用行）';
COMMENT ON COLUMN tn_audit_logs.application_name_snapshot IS '应用名称快照（写时固化，应用改名/删除后历史展示一致）';

CREATE INDEX idx_tn_audit_logs_tenant_application_created
    ON tn_audit_logs (tenant_id, application_id, created_at DESC, id DESC);

-- ---------- tn_product_log_exports：导出任务表 ----------
CREATE TABLE tn_product_log_exports (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL DEFAULT 0,
    member_id BIGINT NOT NULL DEFAULT 0,
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    total BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'pending',
    file_name varchar(128) NOT NULL DEFAULT '',
    file_data TEXT NOT NULL DEFAULT '',
    expires_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

COMMENT ON TABLE tn_product_log_exports IS '产品日志导出任务（000064）：固化提交时的筛选条件/申请人/租户/数据量/状态与过期时间；一期同步生成、内容内联存储，异步导出与对象存储文件引用随留存策略批次接入';
COMMENT ON COLUMN tn_product_log_exports.id IS '自增主键';
COMMENT ON COLUMN tn_product_log_exports.tenant_id IS '所属租户；任务状态读取与下载均复核归属，跨租户不可见';
COMMENT ON COLUMN tn_product_log_exports.account_id IS '申请人平台账号 ID';
COMMENT ON COLUMN tn_product_log_exports.member_id IS '申请人租户成员 ID，0=未解析到成员';
COMMENT ON COLUMN tn_product_log_exports.filters IS '提交时固化的筛选条件快照 JSONB（成员/日志范围/事件码/应用/关键词/时间范围），与列表查询参数同构';
COMMENT ON COLUMN tn_product_log_exports.total IS '导出数据量（行数）';
COMMENT ON COLUMN tn_product_log_exports.status IS '任务状态：pending 生成中 / ready 就绪 / failed 生成失败';
COMMENT ON COLUMN tn_product_log_exports.file_name IS '导出文件名（下载 Content-Disposition 用）';
COMMENT ON COLUMN tn_product_log_exports.file_data IS '导出文件内容（一期 CSV 内联存储；异步批改对象存储文件引用后本列退化为空串）';
COMMENT ON COLUMN tn_product_log_exports.expires_at IS '导出文件过期时间，过期后不可下载';
COMMENT ON COLUMN tn_product_log_exports.created_at IS '任务创建时间';

CREATE INDEX idx_tn_product_log_exports_tenant ON tn_product_log_exports (tenant_id, id DESC);

-- ---------- 基线管理员权限补授（与 000036 同口径） ----------
-- view 展开 get+list 覆盖列表/筛选项/任务状态查询接口；create 覆盖
-- POST /product-logs/exports（product-logs:export 语义），下载路径在
-- 控制器内复核 create 动词后放行。该资源不接受管理组间接放行
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'product-logs', 'operation', 'view'),
        jsonb_build_object('resource', 'product-logs', 'operation', 'create')
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
      WHERE rule->>'resource' = 'product-logs'
  );
