-- 000033: 产品中心一期（docs/低代码平台/产品中心/管理后台-产品中心开发文档.md）。
-- 平台产品目录 + 租户产品配置 + 部门/成员范围关联 三组表 + lingyanyun 目录 seed
-- + 存量租户配置回填（默认启用、范围 all）+ 租户管理员角色补授 tenant-products。
-- 建模要点：产品目录是平台级资源（无 tenant_id）；租户配置是租户级资源；
-- 范围关联只在 scope_mode=partial 时有记录，all 不物化全部成员（文档 5.4）。

-- 平台产品目录：稳定机器码 + 展示信息 + 站内入口；平台停用后租户不可访问
CREATE TABLE IF NOT EXISTS product_catalogs (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    code varchar(64) NOT NULL,
    name varchar(64) NOT NULL,
    icon varchar(64) NOT NULL,
    entry_path varchar(255) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    sort_order BIGINT NOT NULL DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_product_catalogs_status CHECK (status IN ('active', 'inactive'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_product_catalogs_code
    ON product_catalogs (code) WHERE deleted_at IS NULL;

-- 租户产品主配置：每租户每产品一条有效记录；revision 为配置乐观锁版本
CREATE TABLE IF NOT EXISTS tenant_product_configs (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL REFERENCES product_catalogs(id),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    scope_mode varchar(16) NOT NULL DEFAULT 'all',
    revision BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT ck_tenant_product_configs_scope_mode CHECK (scope_mode IN ('all', 'partial'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tenant_product_configs_tenant_product_active
    ON tenant_product_configs (tenant_id, product_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tenant_product_configs_tenant
    ON tenant_product_configs (tenant_id) WHERE deleted_at IS NULL;

-- 部门范围关联：scope_mode=partial 时才有记录，全量替换（先删后插），无软删
CREATE TABLE IF NOT EXISTS tenant_product_departments (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_product_config_id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL,
    department_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT uk_tenant_product_departments UNIQUE (tenant_product_config_id, department_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_product_departments_tenant
    ON tenant_product_departments (tenant_id);

-- 成员范围关联：写入前由服务层校验成员属于当前租户且状态 active
CREATE TABLE IF NOT EXISTS tenant_product_members (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_product_config_id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT uk_tenant_product_members UNIQUE (tenant_product_config_id, member_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_product_members_tenant
    ON tenant_product_members (tenant_id);

-- ---- seed：内置产品目录 lingyanyun（幂等）----
INSERT INTO product_catalogs (code, name, icon, entry_path, status, sort_order)
VALUES ('lingyanyun', '灵衍云', 'product', '/workspace', 'active', 100)
ON CONFLICT DO NOTHING;

-- ---- 存量租户配置回填（幂等，文档 8.2）：active 目录默认启用、范围 all、不建关联行 ----
INSERT INTO tenant_product_configs (tenant_id, product_id, enabled, scope_mode, revision)
SELECT t.id, p.id, TRUE, 'all', 1
FROM tenants t
JOIN product_catalogs p ON p.status = 'active'
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
  AND NOT EXISTS (
      SELECT 1 FROM tenant_product_configs c
      WHERE c.tenant_id = t.id AND c.product_id = p.id
  )
ON CONFLICT DO NOTHING;

-- ---- 权限：tenant-admin 基线角色补授 tenant-products（幂等，口径同 000030/000032）----
-- view 展开 get+list（集合读取 GET /tenant-products 为 list 动词），update 覆盖
-- 启停与范围替换两条 PUT 子资源路径；仅授予租户管理员，不授予应用管理员
UPDATE roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(
        jsonb_build_object('resource', 'tenant-products', 'operation', 'view'),
        jsonb_build_object('resource', 'tenant-products', 'operation', 'update')
    )
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'tenant-products'
  );

-- ---- 数据字典注释 ----
COMMENT ON TABLE product_catalogs IS '平台内置产品目录：平台提供、可被多个租户启用的产品（如灵衍云）；不是租户自建的 applications 应用';
COMMENT ON COLUMN product_catalogs.id IS '自增主键';
COMMENT ON COLUMN product_catalogs.code IS '产品稳定机器码（如 lingyanyun），创建后不可修改，租户侧接口以 code 定位产品';
COMMENT ON COLUMN product_catalogs.name IS '产品展示名称';
COMMENT ON COLUMN product_catalogs.icon IS '稳定图标键（前端按键映射图标组件），不存前端组件名';
COMMENT ON COLUMN product_catalogs.entry_path IS '站内产品入口路径；外部跳转地址能力后续单独设计';
COMMENT ON COLUMN product_catalogs.status IS '目录状态：active 可用 / inactive 平台已停用（停用后所有租户不可访问）';
COMMENT ON COLUMN product_catalogs.sort_order IS '产品中心卡片展示排序，升序';
COMMENT ON COLUMN product_catalogs.created_at IS '创建时间';
COMMENT ON COLUMN product_catalogs.updated_at IS '更新时间';
COMMENT ON COLUMN product_catalogs.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE tenant_product_configs IS '租户产品配置：某产品在一个租户中的启用状态与分发范围；每租户每产品一条有效记录（部分唯一索引保证）';
COMMENT ON COLUMN tenant_product_configs.id IS '自增主键';
COMMENT ON COLUMN tenant_product_configs.tenant_id IS '所属租户 ID（服务层显式条件定位，不依赖请求租户上下文）';
COMMENT ON COLUMN tenant_product_configs.product_id IS '平台产品目录 product_catalogs.id';
COMMENT ON COLUMN tenant_product_configs.enabled IS '租户是否启用该产品：false 时保留范围配置，重新启用后恢复原范围';
COMMENT ON COLUMN tenant_product_configs.scope_mode IS '可用范围模式：all 全部有效成员 / partial 仅选中部门（含子部门）与成员；CHECK 约束保证取值';
COMMENT ON COLUMN tenant_product_configs.revision IS '配置乐观锁版本：每次成功写入（启停/范围替换）递增，客户端提交时携带读取到的版本号';
COMMENT ON COLUMN tenant_product_configs.created_at IS '创建时间';
COMMENT ON COLUMN tenant_product_configs.updated_at IS '最后更新时间';
COMMENT ON COLUMN tenant_product_configs.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE tenant_product_departments IS '产品可用范围-部门关联：仅 scope_mode=partial 时有记录；全量替换（先删后插）无软删；tenant_id 用于租户归属校验与查询';
COMMENT ON COLUMN tenant_product_departments.id IS '自增主键';
COMMENT ON COLUMN tenant_product_departments.tenant_product_config_id IS '租户产品配置 tenant_product_configs.id（同租户）';
COMMENT ON COLUMN tenant_product_departments.tenant_id IS '所属租户 ID（冗余存储，服务层校验部门同租户后写入）';
COMMENT ON COLUMN tenant_product_departments.department_id IS '租户部门 departments.id；选中部门的子部门经读时递归展开命中，不在此复制子部门 ID';
COMMENT ON COLUMN tenant_product_departments.created_at IS '创建时间';

COMMENT ON TABLE tenant_product_members IS '产品可用范围-成员关联：仅 scope_mode=partial 时有记录；全量替换（先删后插）无软删；成员离职/禁用后读取与访问判定均忽略';
COMMENT ON COLUMN tenant_product_members.id IS '自增主键';
COMMENT ON COLUMN tenant_product_members.tenant_product_config_id IS '租户产品配置 tenant_product_configs.id（同租户）';
COMMENT ON COLUMN tenant_product_members.tenant_id IS '所属租户 ID（冗余存储，服务层校验成员同租户且 active 后写入）';
COMMENT ON COLUMN tenant_product_members.member_id IS '租户成员 users.id，写入时必须为同租户 active 成员';
COMMENT ON COLUMN tenant_product_members.created_at IS '创建时间';
