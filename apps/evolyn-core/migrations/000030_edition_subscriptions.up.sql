-- 000030: 版本信息一期（docs/低代码平台/版本信息/管理后台-版本信息两期功能设计方案.md）。
-- 套餐目录/不可变套餐版本快照/租户订阅/租户特批权益覆盖 四表 + 三档 seed 快照
-- + 存量租户订阅回填 + 旧 quotas 覆盖迁移为 legacy 覆盖 + 租户管理员角色补授 editions:get。
-- 建模要点：活动订阅及其套餐版本是当前权益的事实源；tenants.plan/tenants.quotas
-- 仅为 QuotaService 过渡期的兼容投影，订阅变更必须同事务同步两侧（服务层保证）。

-- 套餐目录：稳定套餐编码，不保存价格（价格属二期商品域）
CREATE TABLE IF NOT EXISTS edition_plans (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    code varchar(32) NOT NULL,
    name varchar(64) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    kind varchar(16) NOT NULL DEFAULT 'base',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT uk_edition_plans_code UNIQUE (code),
    CONSTRAINT chk_edition_plans_status CHECK (status IN ('active', 'retired')),
    CONSTRAINT chk_edition_plans_kind CHECK (kind IN ('base', 'addon'))
);

-- 套餐版本：不可变权益快照，已发布只能新增不能修改；一期仅经迁移新增
CREATE TABLE IF NOT EXISTS edition_plan_versions (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    plan_id BIGINT NOT NULL REFERENCES edition_plans(id),
    version INTEGER NOT NULL,
    display_name varchar(64) NOT NULL,
    billing_cycle varchar(16) NOT NULL DEFAULT 'none',
    compatibility_plan_code varchar(32) NOT NULL,
    entitlements JSONB NOT NULL,
    published_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    retired_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT uk_edition_plan_versions_plan_version UNIQUE (plan_id, version),
    CONSTRAINT chk_edition_plan_versions_cycle CHECK (billing_cycle IN ('none', 'monthly', 'yearly')),
    -- QuotaService 切换为统一权益解析器前，兼容投影仅允许三个历史套餐代码（设计 4.4.1）
    CONSTRAINT chk_edition_plan_versions_compat CHECK (compatibility_plan_code IN ('free', 'trial', 'pro'))
);

-- 租户订阅：同租户最多一条 active 基础订阅（部分唯一索引）；试用必须有到期时间
-- （服务层校验；legacy_pending_review 为存量试用的待补录态，不占用 active 槽位）
CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    plan_version_id BIGINT NOT NULL REFERENCES edition_plan_versions(id),
    status varchar(32) NOT NULL,
    grant_type varchar(16) NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone,
    operator_account_id BIGINT,
    remark varchar(512) NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_tenant_subscriptions_status CHECK (status IN ('active', 'expired', 'replaced', 'cancelled', 'legacy_pending_review')),
    CONSTRAINT chk_tenant_subscriptions_grant_type CHECK (grant_type IN ('system', 'manual', 'self_service', 'trial')),
    CONSTRAINT chk_tenant_subscriptions_ends CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_subscriptions_one_active
    ON tenant_subscriptions (tenant_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant
    ON tenant_subscriptions (tenant_id, status, starts_at DESC);

-- 到期降级任务扫描索引：只扫「活动且已到期」的订阅
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_expiring
    ON tenant_subscriptions (ends_at)
    WHERE status = 'active' AND ends_at IS NOT NULL;

-- 租户特批权益覆盖：manual（运营特批）/ trial（试用临时，与订阅同日到期）/
-- legacy（迁移期从旧 tenants.quotas 映射，只读）；无软删，替换即物理删除
CREATE TABLE IF NOT EXISTS tenant_entitlement_overrides (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    entitlement_key varchar(64) NOT NULL,
    value BIGINT NOT NULL,
    reason varchar(255) NOT NULL DEFAULT '',
    source varchar(16) NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone,
    operator_account_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_tenant_entitlement_overrides_source CHECK (source IN ('legacy', 'manual', 'trial')),
    CONSTRAINT chk_tenant_entitlement_overrides_ends CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_tenant_entitlement_overrides_tenant
    ON tenant_entitlement_overrides (tenant_id, entitlement_key);

-- ---- seed：三档基础套餐与其 version=1 已发布快照（幂等，不覆盖已发布快照）----
-- 数值与 tenant/model/plan.go DefaultQuotas 逐键一致；storage 单位为字节
-- （free 1 GiB、trial 5 GiB、pro 不限量），一期存储上限只允许 -1/0/整 GiB。
INSERT INTO edition_plans (code, name, status, kind)
VALUES ('free', '免费版', 'active', 'base'),
       ('trial', '试用版', 'active', 'base'),
       ('pro', '专业版', 'active', 'base')
ON CONFLICT (code) DO NOTHING;

INSERT INTO edition_plan_versions (plan_id, version, display_name, billing_cycle, compatibility_plan_code, entitlements, published_at)
SELECT p.id, 1, '免费版', 'none', 'free', $JSON${
  "resources": [
    {"key": "apps", "category": "stock", "limit": 3, "unit": "count"},
    {"key": "members", "category": "stock", "limit": 5, "unit": "person"},
    {"key": "forms", "category": "stock", "limit": 10, "unit": "count"},
    {"key": "storage_bytes", "category": "stock", "limit": 1073741824, "unit": "byte"},
    {"key": "workflow_runs_month", "category": "periodic", "limit": 100, "unit": "count", "resetCycle": "monthly"}
  ],
  "features": [
    {"key": "application_management", "group": "基础管理", "name": "应用管理", "available": true},
    {"key": "member_management", "group": "基础管理", "name": "成员管理", "available": true},
    {"key": "department_management", "group": "基础管理", "name": "部门管理", "available": true},
    {"key": "group_management", "group": "基础管理", "name": "分组管理", "available": true},
    {"key": "role_permission", "group": "基础管理", "name": "角色权限", "available": true},
    {"key": "file_upload", "group": "基础管理", "name": "附件上传", "available": true}
  ]
}$JSON$::jsonb, LOCALTIMESTAMP
FROM edition_plans p
WHERE p.code = 'free'
ON CONFLICT (plan_id, version) DO NOTHING;

INSERT INTO edition_plan_versions (plan_id, version, display_name, billing_cycle, compatibility_plan_code, entitlements, published_at)
SELECT p.id, 1, '试用版', 'none', 'trial', $JSON${
  "resources": [
    {"key": "apps", "category": "stock", "limit": 10, "unit": "count"},
    {"key": "members", "category": "stock", "limit": 30, "unit": "person"},
    {"key": "forms", "category": "stock", "limit": 50, "unit": "count"},
    {"key": "storage_bytes", "category": "stock", "limit": 5368709120, "unit": "byte"},
    {"key": "workflow_runs_month", "category": "periodic", "limit": 10000, "unit": "count", "resetCycle": "monthly"}
  ],
  "features": [
    {"key": "application_management", "group": "基础管理", "name": "应用管理", "available": true},
    {"key": "member_management", "group": "基础管理", "name": "成员管理", "available": true},
    {"key": "department_management", "group": "基础管理", "name": "部门管理", "available": true},
    {"key": "group_management", "group": "基础管理", "name": "分组管理", "available": true},
    {"key": "role_permission", "group": "基础管理", "name": "角色权限", "available": true},
    {"key": "file_upload", "group": "基础管理", "name": "附件上传", "available": true}
  ]
}$JSON$::jsonb, LOCALTIMESTAMP
FROM edition_plans p
WHERE p.code = 'trial'
ON CONFLICT (plan_id, version) DO NOTHING;

INSERT INTO edition_plan_versions (plan_id, version, display_name, billing_cycle, compatibility_plan_code, entitlements, published_at)
SELECT p.id, 1, '专业版', 'none', 'pro', $JSON${
  "resources": [
    {"key": "apps", "category": "stock", "limit": -1, "unit": "count"},
    {"key": "members", "category": "stock", "limit": -1, "unit": "person"},
    {"key": "forms", "category": "stock", "limit": -1, "unit": "count"},
    {"key": "storage_bytes", "category": "stock", "limit": -1, "unit": "byte"},
    {"key": "workflow_runs_month", "category": "periodic", "limit": -1, "unit": "count", "resetCycle": "monthly"}
  ],
  "features": [
    {"key": "application_management", "group": "基础管理", "name": "应用管理", "available": true},
    {"key": "member_management", "group": "基础管理", "name": "成员管理", "available": true},
    {"key": "department_management", "group": "基础管理", "name": "部门管理", "available": true},
    {"key": "group_management", "group": "基础管理", "name": "分组管理", "available": true},
    {"key": "role_permission", "group": "基础管理", "name": "角色权限", "available": true},
    {"key": "file_upload", "group": "基础管理", "name": "附件上传", "available": true}
  ]
}$JSON$::jsonb, LOCALTIMESTAMP
FROM edition_plans p
WHERE p.code = 'pro'
ON CONFLICT (plan_id, version) DO NOTHING;

-- ---- 存量租户订阅回填（幂等）----
-- free/pro：无历史到期信息，回填为长期兼容订阅（grant_type=system）；
-- trial：历史无到期日记录，回填 legacy_pending_review 待运营补录，
-- 不产生「active + ends_at NULL 的试用记录」（设计 4.4.3）。
INSERT INTO tenant_subscriptions (tenant_id, plan_version_id, status, grant_type, starts_at, ends_at, remark)
SELECT t.id,
       pv.id,
       CASE WHEN t.plan = 'trial' THEN 'legacy_pending_review' ELSE 'active' END,
       CASE WHEN t.plan = 'trial' THEN 'trial' ELSE 'system' END,
       COALESCE(t.created_at, LOCALTIMESTAMP),
       NULL,
       CASE WHEN t.plan = 'trial'
            THEN '存量试用：历史无到期日记录，待运营补录后转为活动订阅'
            ELSE '' END
FROM tenants t
JOIN edition_plans p ON p.code = t.plan
JOIN edition_plan_versions pv ON pv.plan_id = p.id AND pv.version = 1
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
  AND NOT EXISTS (SELECT 1 FROM tenant_subscriptions ts WHERE ts.tenant_id = t.id);

-- ---- 旧 tenants.quotas 覆盖迁移为 legacy 覆盖（幂等）----
-- storage_gb → storage_bytes 按 ×GiB 精确换算（旧值恒为整数 GB，无损）；
-- 其余键同名同单位。legacy 行只读，后续经服务层统一投影回写。
INSERT INTO tenant_entitlement_overrides (tenant_id, entitlement_key, value, reason, source, starts_at, ends_at)
SELECT t.id,
       CASE WHEN e.key = 'storage_gb' THEN 'storage_bytes' ELSE e.key END,
       CASE WHEN e.key = 'storage_gb'
            THEN (e.value #>> '{}')::bigint * 1073741824
            ELSE (e.value #>> '{}')::bigint END,
       '存量租户配额覆盖迁移（legacy）',
       'legacy',
       LOCALTIMESTAMP,
       NULL
FROM tenants t
CROSS JOIN LATERAL jsonb_each(COALESCE(t.quotas, '{}'::jsonb)) AS e(key, value)
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
  AND jsonb_typeof(e.value) = 'number'
  AND e.key IN ('apps', 'members', 'forms', 'workflow_runs_month', 'storage_gb')
  AND NOT EXISTS (
      SELECT 1 FROM tenant_entitlement_overrides o
      WHERE o.tenant_id = t.id
        AND o.entitlement_key = (CASE WHEN e.key = 'storage_gb' THEN 'storage_bytes' ELSE e.key END)
        AND o.source = 'legacy'
  );

-- ---- 权限：tenant-admin 基线角色补授 editions:get（幂等，口径同 000022）----
UPDATE roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'editions', 'operation', 'get'))
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'editions'
  );

-- ---- 数据字典注释 ----
COMMENT ON TABLE edition_plans IS '套餐目录：稳定套餐编码与展示名，区分基础套餐与附加能力；不保存价格，价格属二期商品域';
COMMENT ON COLUMN edition_plans.id IS '自增主键';
COMMENT ON COLUMN edition_plans.code IS '稳定套餐编码（free/trial/pro），全局唯一，发布后不变更';
COMMENT ON COLUMN edition_plans.name IS '套餐展示名称';
COMMENT ON COLUMN edition_plans.status IS '目录状态：active 可授予 / retired 已下架（历史订阅仍可引用其版本）';
COMMENT ON COLUMN edition_plans.kind IS '套餐类型：base 基础套餐 / addon 附加能力（二期附加包启用）';
COMMENT ON COLUMN edition_plans.created_at IS '创建时间';
COMMENT ON COLUMN edition_plans.updated_at IS '更新时间';

COMMENT ON TABLE edition_plan_versions IS '套餐版本：不可变权益快照（resources+features JSONB），已发布只能新增不能修改；一期仅经版本化迁移新增';
COMMENT ON COLUMN edition_plan_versions.id IS '自增主键，人工授予与订阅引用的版本 ID';
COMMENT ON COLUMN edition_plan_versions.plan_id IS '所属套餐，外键指向 edition_plans(id)';
COMMENT ON COLUMN edition_plan_versions.version IS '套餐内版本号，从 1 递增，(plan_id, version) 唯一';
COMMENT ON COLUMN edition_plan_versions.display_name IS '版本展示名称（如「免费版」）';
COMMENT ON COLUMN edition_plan_versions.billing_cycle IS '计费周期：none 不计费（一期）/ monthly / yearly（二期商品域启用）';
COMMENT ON COLUMN edition_plan_versions.compatibility_plan_code IS '兼容投影目标：同步 tenants.plan 时使用的旧套餐代码，仅允许 free/trial/pro';
COMMENT ON COLUMN edition_plan_versions.entitlements IS '权益快照 JSONB：resources[{key,category,limit,unit,resetCycle}] + features[{key,group,name,available,parameters}]；limit 语义 -1 不限量 / 0 不可用 / 正数上限，storage_bytes 一期只允许 -1/0/整 GiB';
COMMENT ON COLUMN edition_plan_versions.published_at IS '发布时间，非空即已发布';
COMMENT ON COLUMN edition_plan_versions.retired_at IS '下架时间，NULL 表示仍在授予；下架版本不可新授予但存量订阅继续有效';
COMMENT ON COLUMN edition_plan_versions.created_at IS '创建时间';
COMMENT ON COLUMN edition_plan_versions.updated_at IS '更新时间';

COMMENT ON TABLE tenant_subscriptions IS '租户订阅：权益的事实源。同租户最多一条 active 基础订阅；到期由任务降级为免费订阅，页面与写路径在任务落库前按免费版解析（expiry_fallback）';
COMMENT ON COLUMN tenant_subscriptions.id IS '自增主键';
COMMENT ON COLUMN tenant_subscriptions.tenant_id IS '所属租户 ID（平台侧与 worker 经显式条件读写，不走租户 Callback）';
COMMENT ON COLUMN tenant_subscriptions.plan_version_id IS '订阅的套餐版本快照，外键指向 edition_plan_versions(id)';
COMMENT ON COLUMN tenant_subscriptions.status IS '状态：active 活动 / expired 到期已降级 / replaced 被新订阅替换 / cancelled 人工取消 / legacy_pending_review 存量试用待补录';
COMMENT ON COLUMN tenant_subscriptions.grant_type IS '授予方式：system 系统初始 / manual 平台运营人工 / self_service 用户自助（二期）/ trial 试用';
COMMENT ON COLUMN tenant_subscriptions.starts_at IS '生效时间';
COMMENT ON COLUMN tenant_subscriptions.ends_at IS '到期时间，NULL 表示长期有效；试用（grant_type=trial）必须非空，服务层校验';
COMMENT ON COLUMN tenant_subscriptions.operator_account_id IS '平台人工操作的平台账号 ID，系统操作为 NULL';
COMMENT ON COLUMN tenant_subscriptions.remark IS '运营备注，仅平台运营面可见，不对租户普通成员泄露';
COMMENT ON COLUMN tenant_subscriptions.created_at IS '创建时间';
COMMENT ON COLUMN tenant_subscriptions.updated_at IS '更新时间';

COMMENT ON TABLE tenant_entitlement_overrides IS '租户特批权益覆盖：manual 运营特批 / trial 试用临时（与订阅同日到期）/ legacy 旧 quotas 迁移（只读）；降级时移除已到期或 trial 来源，保留仍有效的 manual';
COMMENT ON COLUMN tenant_entitlement_overrides.id IS '自增主键';
COMMENT ON COLUMN tenant_entitlement_overrides.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tenant_entitlement_overrides.entitlement_key IS '权益资源键（新键空间，存储为 storage_bytes 字节）';
COMMENT ON COLUMN tenant_entitlement_overrides.value IS '覆盖值：-1 不限量 / 0 不可用 / 正数上限；storage_bytes 须为整 GiB';
COMMENT ON COLUMN tenant_entitlement_overrides.reason IS '覆盖原因（运营填写）';
COMMENT ON COLUMN tenant_entitlement_overrides.source IS '来源：legacy 旧 quotas 迁移（只读）/ manual 运营特批 / trial 试用临时';
COMMENT ON COLUMN tenant_entitlement_overrides.starts_at IS '覆盖生效时间';
COMMENT ON COLUMN tenant_entitlement_overrides.ends_at IS '覆盖失效时间，NULL 表示长期；trial 来源必须与订阅同日到期';
COMMENT ON COLUMN tenant_entitlement_overrides.operator_account_id IS '操作的平台账号 ID，legacy/system 为 NULL';
COMMENT ON COLUMN tenant_entitlement_overrides.created_at IS '创建时间';
COMMENT ON COLUMN tenant_entitlement_overrides.updated_at IS '更新时间';
