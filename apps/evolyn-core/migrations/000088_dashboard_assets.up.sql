-- 000088：业务仪表盘资产域一期。
-- 配额 seed 采用已确认口径：免费版 3、试用版 20、专业版不限量（-1）。

-- 套餐版本是权益事实源。仅为尚未声明 dashboards 的存量快照追加资源，
-- 保持迁移幂等，避免重复执行产生重复键。
UPDATE pf_edition_plan_versions pv
SET entitlements = jsonb_set(
    pv.entitlements::jsonb,
    '{resources}',
    COALESCE(pv.entitlements::jsonb->'resources', '[]'::jsonb) ||
      jsonb_build_array(jsonb_build_object(
        'key', 'dashboards',
        'category', 'stock',
        'limit', CASE pv.compatibility_plan_code
          WHEN 'free' THEN 3
          WHEN 'trial' THEN 20
          WHEN 'pro' THEN -1
          ELSE 0
        END,
        'unit', 'count'
      )),
    true
)
WHERE pv.compatibility_plan_code IN ('free', 'trial', 'pro')
  AND NOT EXISTS (
    SELECT 1
    FROM jsonb_array_elements(COALESCE(pv.entitlements::jsonb->'resources', '[]'::jsonb)) resource
    WHERE resource->>'key' = 'dashboards'
  );

CREATE TABLE IF NOT EXISTS tn_dashboards (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    app_id BIGINT NOT NULL REFERENCES tn_apps(id),
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    icon VARCHAR(32) NOT NULL DEFAULT '',
    color VARCHAR(32) NOT NULL DEFAULT '',
    protocol_version INTEGER NOT NULL DEFAULT 1,
    draft_content JSONB NOT NULL,
    draft_revision BIGINT NOT NULL DEFAULT 1,
    latest_version_id BIGINT,
    published_version INTEGER NOT NULL DEFAULT 0,
    creator_member_id BIGINT NOT NULL,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_tn_dashboards_draft_object CHECK (jsonb_typeof(draft_content) = 'object'),
    CONSTRAINT ck_tn_dashboards_protocol_version CHECK (protocol_version > 0),
    CONSTRAINT ck_tn_dashboards_draft_revision CHECK (draft_revision > 0),
    CONSTRAINT ck_tn_dashboards_published_version CHECK (published_version >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_dashboards_tenant_code
    ON tn_dashboards (tenant_id, code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tn_dashboards_tenant_app
    ON tn_dashboards (tenant_id, app_id, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tn_dashboard_versions (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    dashboard_id BIGINT NOT NULL REFERENCES tn_dashboards(id),
    version_no INTEGER NOT NULL,
    source_draft_revision BIGINT NOT NULL,
    protocol_version INTEGER NOT NULL,
    content JSONB NOT NULL,
    content_checksum CHAR(64) NOT NULL,
    published_by_member_id BIGINT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ,
    CONSTRAINT uk_tn_dashboard_versions_no UNIQUE (dashboard_id, version_no),
    CONSTRAINT ck_tn_dashboard_versions_content_object CHECK (jsonb_typeof(content) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_tn_dashboard_versions_tenant_dashboard
    ON tn_dashboard_versions (tenant_id, dashboard_id, version_no DESC);

ALTER TABLE tn_dashboards
    ADD CONSTRAINT fk_tn_dashboards_latest_version
    FOREIGN KEY (latest_version_id) REFERENCES tn_dashboard_versions(id);

CREATE TABLE IF NOT EXISTS tn_dashboard_version_subjects (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    dashboard_version_id BIGINT NOT NULL REFERENCES tn_dashboard_versions(id),
    subject_type VARCHAR(20) NOT NULL,
    subject_id BIGINT,
    created_at TIMESTAMPTZ,
    CONSTRAINT ck_tn_dashboard_version_subject_type
      CHECK (subject_type IN ('all', 'member', 'department', 'role', 'group')),
    CONSTRAINT ck_tn_dashboard_version_subject_value
      CHECK ((subject_type = 'all' AND subject_id IS NULL) OR
             (subject_type <> 'all' AND subject_id IS NOT NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_dashboard_version_subjects_identity
    ON tn_dashboard_version_subjects (
      dashboard_version_id,
      subject_type,
      COALESCE(subject_id, 0)
    );
CREATE INDEX IF NOT EXISTS idx_tn_dashboard_version_subjects_tenant_version
    ON tn_dashboard_version_subjects (tenant_id, dashboard_version_id);

-- dashboard_id 在请求取得幂等占位时允许为空；同一事务创建资产后必须回填。
-- 事务失败时占位与资产一起回滚，不会留下半成品绑定。
CREATE TABLE IF NOT EXISTS tn_dashboard_create_bindings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    dashboard_id BIGINT REFERENCES tn_dashboards(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tn_dashboard_create_bindings_request
      UNIQUE (tenant_id, member_id, request_id)
);

CREATE INDEX IF NOT EXISTS idx_tn_dashboard_create_bindings_tenant_dashboard
    ON tn_dashboard_create_bindings (tenant_id, dashboard_id);

-- 租户管理员获得管理与动作全量基线。
UPDATE tn_roles AS r
SET rules = (
    r.rules::jsonb || COALESCE((
        SELECT jsonb_agg(candidate.rule)
        FROM jsonb_array_elements('[
          {"resource":"dashboards","operation":"*"},
          {"resource":"dashboard-actions","operation":"*"}
        ]'::jsonb) AS candidate(rule)
        WHERE NOT EXISTS (
            SELECT 1 FROM jsonb_array_elements(r.rules::jsonb) existing(rule)
            WHERE existing.rule->>'resource' = candidate.rule->>'resource'
              AND existing.rule->>'operation' = candidate.rule->>'operation'
        )
    ), '[]'::jsonb)
)::json
WHERE r.deleted_at IS NULL
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) rule WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) rule WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*');

-- 全体已认证成员只获得运行查看动作；发布范围与数据权限仍会继续收口。
WITH authenticated_roles AS (
    SELECT DISTINCT r.id
    FROM tn_roles r
    INNER JOIN tn_group_roles gr ON gr.role_id = r.id
    INNER JOIN tn_groups g ON g.id = gr.group_id
    WHERE g.name = 'system:authenticated'
      AND g.kind = 'system'
      AND r.deleted_at IS NULL
)
UPDATE tn_roles r
SET rules = (r.rules::jsonb || '[{"resource":"dashboard-actions","operation":"view"}]'::jsonb)::json
WHERE r.id IN (SELECT id FROM authenticated_roles)
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
    SELECT 1 FROM json_array_elements(r.rules) rule
    WHERE rule->>'resource' = 'dashboard-actions' AND rule->>'operation' = 'view'
  );

COMMENT ON TABLE tn_dashboards IS '应用业务仪表盘资产：draft_content 为版本化草稿事实源，draft_revision 为乐观锁口令，公开 API 只使用 dashboard_ code';
COMMENT ON TABLE tn_dashboard_versions IS '仪表盘不可变发布快照；业务代码只追加和读取，不提供更新内容路径';
COMMENT ON TABLE tn_dashboard_version_subjects IS '发布版本冻结的成员范围主体定义；访问时按当前有效组织关系重新求值';
COMMENT ON TABLE tn_dashboard_create_bindings IS '仪表盘预创建幂等绑定：(tenant_id, member_id, request_id) 唯一，请求 hash 不同即冲突';
