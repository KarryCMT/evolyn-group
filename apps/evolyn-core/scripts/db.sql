-- evolyn-core 冷启动初始化（终态快照）
-- 本文件 = migrations/ 000001..000063 全链执行后的等价状态，仅作
-- make postgres 快速起库用；Schema 唯一事实来源是 migrations/（FIX-009），
-- 结构变更必须同时提交 Migration，并同步维护本快照。
-- 快照库上重放迁移链应当零副作用：表/索引/约束使用与迁移一致的名字，
-- 种子写入均带 ON CONFLICT DO NOTHING。
-- 模型定版：ADR-006 账号×成员拆分 + ADR-007 域模块化 + 第一期整改 FIX-001~017
-- 表命名空间（000063）：pf_ 平台 / sys_ 系统 / tn_ 租户 / wf_ 流程引擎；
-- 详细归类见 docs/低代码平台/数据库表命名空间前缀调整方案.md
CREATE DATABASE evolyn;

\c evolyn;

-- 平台账号（登录身份，全局唯一；无 tenant_id——账号跨租户，ADR-006）。
-- 先建账号再建租户：pf_tenants.owner_account_id 外键依赖本表（FIX-016）
CREATE TABLE IF NOT EXISTS pf_accounts (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL UNIQUE,
    nickname varchar(100),
    phone varchar(32),
    email varchar(256),
    password varchar(256),
    password_initialized boolean DEFAULT true NOT NULL,
    session_version BIGINT DEFAULT 0 NOT NULL,
    avatar text,
    onboarding jsonb DEFAULT '{}'::jsonb NOT NULL,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

-- 手机号唯一性（000007）：非空才参与，未填账号落 '' 不互斥；
-- 软删除友好，与迁移链 uk_pf_accounts_phone 同名同构
CREATE UNIQUE INDEX IF NOT EXISTS uk_pf_accounts_phone
    ON pf_accounts (phone)
    WHERE phone <> '' AND deleted_at IS NULL;

INSERT INTO pf_accounts (name, nickname, email, password, created_at) VALUES
    ('admin', 'admin', 'admin@evolyn.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP),
    ('demo', 'demo', 'admin@evolyn.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP)
    ON CONFLICT DO NOTHING;

-- 租户（平台一级资源，无 tenant_id——FIX-014；owner 可空外键——FIX-016；
-- 注销生命周期时间线——FIX-012）
CREATE TABLE IF NOT EXISTS pf_tenants (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    code varchar(64) NOT NULL UNIQUE,
    name varchar(128) NOT NULL,
    plan varchar(32) NOT NULL DEFAULT 'free',
    status varchar(16) NOT NULL DEFAULT 'active',
    owner_account_id BIGINT,
    config JSONB NOT NULL DEFAULT '{}',
    quotas JSONB NOT NULL DEFAULT '{}',
    delete_requested_at timestamp with time zone,
    retention_until timestamp with time zone,
    purged_at timestamp with time zone,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT fk_pf_tenants_owner FOREIGN KEY (owner_account_id) REFERENCES pf_accounts(id)
);

INSERT INTO pf_tenants (code, name, plan, status, created_at) VALUES
    ('default', '默认租户', 'free', 'active', LOCALTIMESTAMP) ON CONFLICT DO NOTHING;

-- 默认租户 Owner 指向 admin 账号
UPDATE pf_tenants SET owner_account_id = (SELECT id FROM pf_accounts WHERE name = 'admin')
WHERE code = 'default' AND owner_account_id IS NULL;

-- 第三方登录凭证（归属账号，平台级无 tenant_id；
-- (auth_type, auth_id) 软删友好唯一——FIX-017）
CREATE TABLE IF NOT EXISTS pf_auth_infos (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    account_id BIGINT NOT NULL DEFAULT 0 REFERENCES pf_accounts(id),
    url varchar(256),
    auth_type varchar(256),
    auth_id varchar(256),
    access_token varchar(256),
    refresh_token varchar(256),
    expiry timestamp with time zone,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pf_auth_identity
    ON pf_auth_infos (auth_type, auth_id)
    WHERE deleted_at IS NULL;

-- 成员（原 tn_users 表；登录身份已拆至 pf_accounts，ADR-006）。
-- account_id 外键——FIX-005；(tenant_id, account_id) 软删友好唯一——FIX-004
CREATE TABLE IF NOT EXISTS tn_users (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    account_id BIGINT NOT NULL,
    nickname varchar(100),
    status varchar(16) NOT NULL DEFAULT 'active',
    resigned_at timestamp with time zone,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT fk_tn_users_account FOREIGN KEY (account_id) REFERENCES pf_accounts(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_users_tenant_account
    ON tn_users (tenant_id, account_id)
    WHERE deleted_at IS NULL;

ALTER TABLE tn_users
    DROP CONSTRAINT IF EXISTS chk_tn_users_status,
    ADD CONSTRAINT chk_tn_users_status CHECK (status IN ('active', 'disabled', 'resigned'));

COMMENT ON COLUMN tn_users.status IS '成员状态：active 启用、disabled 停用、resigned 离职；全部成员视图默认不含 resigned';
COMMENT ON COLUMN tn_users.resigned_at IS '成员转为离职的时间；恢复启用或停用时清空';

CREATE INDEX IF NOT EXISTS idx_tn_users_tenant_status_id
    ON tn_users (tenant_id, status, id)
    WHERE deleted_at IS NULL;

INSERT INTO tn_users (account_id, nickname, tenant_id, created_at)
    SELECT a.id, a.nickname, 1, a.created_at FROM pf_accounts AS a
    WHERE a.name IN ('admin', 'demo') ON CONFLICT DO NOTHING;

-- 成员邀请在受邀人接受前保存完整档案草稿，不创建占位 tn_users 记录。
CREATE TABLE IF NOT EXISTS tn_member_invitations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    inviter_member_id BIGINT NOT NULL DEFAULT 0,
    name varchar(80) NOT NULL,
    identifier varchar(50),
    phone varchar(32),
    email varchar(256),
    profile jsonb NOT NULL DEFAULT '{}'::jsonb,
    invite_token varchar(64) NOT NULL,
    source varchar(16) NOT NULL DEFAULT 'manual',
    status varchar(16) NOT NULL DEFAULT 'pending',
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_tn_member_invitations_source CHECK (source IN ('manual', 'batch')),
    CONSTRAINT chk_tn_member_invitations_status CHECK (status IN ('pending', 'accepted', 'cancelled')),
    CONSTRAINT chk_tn_member_invitations_contact CHECK (phone <> '' OR email <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_member_invitations_token
    ON tn_member_invitations (invite_token) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_member_invitations_tenant_phone_pending
    ON tn_member_invitations (tenant_id, phone)
    WHERE phone <> '' AND status = 'pending' AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_member_invitations_tenant_email_pending
    ON tn_member_invitations (tenant_id, email)
    WHERE email <> '' AND status = 'pending' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tn_member_invitations_tenant_status
    ON tn_member_invitations (tenant_id, status, id DESC) WHERE deleted_at IS NULL;

-- 每个租户只有一条公开邀请链接；关闭后保留 token，重新开启无需再次生成。
CREATE TABLE IF NOT EXISTS tn_public_invitation_links (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    token varchar(64) NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    creator_member_id BIGINT NOT NULL DEFAULT 0,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_public_invitation_links_tenant
    ON tn_public_invitation_links (tenant_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_public_invitation_links_token
    ON tn_public_invitation_links (token) WHERE deleted_at IS NULL;

-- 部门（租户内组织架构，邻接表树；parent_id NULL=root + 自引用外键——FIX-015）
CREATE TABLE IF NOT EXISTS tn_departments (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    parent_id BIGINT,
    name varchar(100) NOT NULL,
    "order" BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'active',
    leader_member_id BIGINT,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT fk_tn_departments_parent FOREIGN KEY (parent_id) REFERENCES tn_departments(id)
);
CREATE INDEX IF NOT EXISTS idx_tn_departments_leader_member_id ON tn_departments (leader_member_id);

CREATE TABLE IF NOT EXISTS tn_department_users(
    department_id BIGINT NOT NULL REFERENCES tn_departments(id),
    user_id BIGINT NOT NULL REFERENCES tn_users(id),
    PRIMARY KEY(department_id, user_id)
);

-- 用户组（租户内唯一——FIX-003：部分唯一索引，软删后可重建同名）
CREATE TABLE IF NOT EXISTS tn_groups (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL,
    kind varchar(100),
    describe varchar(1024),
    creator_id BIGINT,
    updater_id BIGINT,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_groups_tenant_name
    ON tn_groups (tenant_id, name)
    WHERE deleted_at IS NULL;

-- 角色展示分组：仅供内部组织页归类角色，不影响分组角色授权关系。
CREATE TABLE IF NOT EXISTS tn_role_groups (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL,
    creator_member_id BIGINT,
    creator_id BIGINT,
    updater_id BIGINT,
    sort INTEGER NOT NULL DEFAULT 0,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_role_groups_tenant_name
    ON tn_role_groups (tenant_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tn_role_groups_tenant_sort
    ON tn_role_groups (tenant_id, sort, id)
    WHERE deleted_at IS NULL;

INSERT INTO tn_groups (name, kind, describe, created_at) VALUES
    ('root', 'system', 'evolyn system group', LOCALTIMESTAMP),
    ('system:authenticated', 'system', 'system group contains all authenticated user', LOCALTIMESTAMP),
    ('system:unauthenticated', 'system', 'system group contains all unauthenticated user', LOCALTIMESTAMP)  ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS tn_user_groups(
    group_id BIGINT NOT NULL REFERENCES tn_groups(id),
    user_id BIGINT NOT NULL REFERENCES tn_users(id),
    PRIMARY KEY(group_id, user_id)
);

-- admin 成员（同 ID 复制策略下 account_id = 自身 ID）入 root 组
INSERT INTO tn_user_groups (group_id, user_id)
    SELECT g.id, u.id FROM tn_users AS u, tn_groups AS g
    WHERE (u.account_id = (SELECT id FROM pf_accounts WHERE name = 'admin') AND g.name = 'root') ON CONFLICT DO NOTHING;

-- 平台资源目录（平台级，不挂租户）
CREATE TABLE IF NOT EXISTS sys_resources (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(256) NOT NULL,
    scope varchar(100),
    kind varchar(100)
);

-- 角色（租户内唯一——FIX-002；软删除/时间三件套与模型对齐——FIX-001）
CREATE TABLE IF NOT EXISTS tn_roles (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL,
    role_group_id BIGINT,
    sort INTEGER NOT NULL DEFAULT 0,
    scope varchar(100),
    namespace varchar(100),
    rules json,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_roles_tenant_name
    ON tn_roles (tenant_id, name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tn_roles_role_group_id ON tn_roles (role_group_id);
CREATE INDEX IF NOT EXISTS idx_tn_roles_role_group_sort
    ON tn_roles (tenant_id, role_group_id, sort, id)
    WHERE deleted_at IS NULL;
ALTER TABLE tn_roles DROP CONSTRAINT IF EXISTS fk_tn_roles_role_group;
ALTER TABLE tn_roles ADD CONSTRAINT fk_tn_roles_role_group
    FOREIGN KEY (role_group_id) REFERENCES tn_role_groups(id) ON DELETE SET NULL;

INSERT INTO tn_roles (name, scope, rules) VALUES
    ('平台管理员', 'cluster', '[{"resource": "*", "operation": "*"}]'),
    ('已认证用户', 'cluster', '[{"resource": "tn_users", "operation": "*"},{"resource": "auth", "operation": "*"},{"resource": "pf_accounts", "operation": "*"},{"resource": "tn_applications", "operation": "view"},{"resource": "tn_files", "operation": "edit"},{"resource": "form-records", "operation": "create"},{"resource": "notifications", "operation": "view"},{"resource": "notifications", "operation": "update"}]'),
    ('未认证用户', 'cluster', '[{"resource": "auth", "operation": "create"}]') ON CONFLICT DO NOTHING;

-- 租户管理员可更新组织根节点（租户名称）；存量数据库由迁移 000022 同步。
UPDATE tn_roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'tenant', 'operation', 'edit'))
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'tenant'
  );

-- 租户成员管理路由为 /members；创建者绑定的 tenant-admin 需拥有完整权限。
-- 存量数据库由迁移 000024 同步，避免资源名 tn_users 与 members 不一致导致 403。
UPDATE tn_roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'members', 'operation', '*'))
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'members'
  );

CREATE TABLE IF NOT EXISTS tn_user_roles(
    user_id BIGINT NOT NULL REFERENCES tn_users(id),
    role_id BIGINT NOT NULL REFERENCES tn_roles(id),
    PRIMARY KEY(user_id, role_id)
);

CREATE TABLE IF NOT EXISTS tn_group_roles(
    group_id BIGINT NOT NULL REFERENCES tn_groups(id),
    role_id BIGINT NOT NULL REFERENCES tn_roles(id),
    PRIMARY KEY(group_id, role_id)
);

INSERT INTO tn_group_roles (group_id, role_id) VALUES
    ((SELECT id FROM tn_groups WHERE name = 'root'), (SELECT id FROM tn_roles WHERE name = '平台管理员')),
    ((SELECT id FROM tn_groups WHERE name = 'system:authenticated'), (SELECT id FROM tn_roles WHERE name = '已认证用户')),
    ((SELECT id FROM tn_groups WHERE name = 'system:unauthenticated'), (SELECT id FROM tn_roles WHERE name = '未认证用户'))
    ON CONFLICT DO NOTHING;

-- 业务审计日志（FIX-013）：追加写流水，无软删语义
CREATE TABLE IF NOT EXISTS tn_audit_logs (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 0,
    account_id BIGINT NOT NULL DEFAULT 0,
    member_id BIGINT NOT NULL DEFAULT 0,
    module varchar(64) NOT NULL,
    action varchar(64) NOT NULL,
    resource_type varchar(64) NOT NULL,
    resource_id varchar(128),
    request_id varchar(64),
    ip varchar(64),
    user_agent varchar(256),
    before_data JSONB,
    after_data JSONB,
    event_code varchar(100) NOT NULL DEFAULT '',
    category_code varchar(64) NOT NULL DEFAULT '',
    actor_name_snapshot varchar(128) NOT NULL DEFAULT '',
    target_name_snapshot varchar(256) NOT NULL DEFAULT '',
    summary varchar(1000) NOT NULL DEFAULT '',
    application_id BIGINT NULL,
    application_code varchar(128) NOT NULL DEFAULT '',
    application_name_snapshot varchar(256) NOT NULL DEFAULT '',
    created_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tn_audit_logs_tenant ON tn_audit_logs (tenant_id, id);
CREATE INDEX IF NOT EXISTS idx_tn_audit_logs_module_action ON tn_audit_logs (module, action);
CREATE INDEX IF NOT EXISTS idx_tn_audit_logs_resource ON tn_audit_logs (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_tn_audit_logs_tenant_created ON tn_audit_logs (tenant_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_tn_audit_logs_tenant_category_created ON tn_audit_logs (tenant_id, category_code, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_tn_audit_logs_tenant_member_created ON tn_audit_logs (tenant_id, member_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_tn_audit_logs_tenant_application_created ON tn_audit_logs (tenant_id, application_id, created_at DESC, id DESC);

-- 登录日志（000013）：会话建立事件流水，账号维度追加写；与 tn_audit_logs 职责互斥
CREATE TABLE IF NOT EXISTS pf_login_logs (
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
    actor_name_snapshot varchar(128) NOT NULL DEFAULT '',
    created_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pf_login_logs_account ON pf_login_logs (account_id, id);
CREATE INDEX IF NOT EXISTS idx_pf_login_logs_tenant_created ON pf_login_logs (tenant_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_pf_login_logs_tenant_member_created ON pf_login_logs (tenant_id, member_id, created_at DESC, id DESC);

-- 企业日志导出任务（000036）：一期同步生成、内容内联存储，异步导出与对象
-- 存储文件引用随留存策略批次接入
CREATE TABLE IF NOT EXISTS tn_enterprise_log_exports (
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

CREATE INDEX IF NOT EXISTS idx_tn_enterprise_log_exports_tenant ON tn_enterprise_log_exports (tenant_id, id DESC);

-- 产品日志导出任务（000064）：一期同步生成、内容内联存储，异步导出与对象
-- 存储文件引用随留存策略批次接入
CREATE TABLE IF NOT EXISTS tn_product_log_exports (
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

CREATE INDEX IF NOT EXISTS idx_tn_product_log_exports_tenant ON tn_product_log_exports (tenant_id, id DESC);

-- 应用实例（000014，M2-A）：status 仅 active/archived，删除只写 deleted_at；
-- provision_status 独立表达实例化进度（M2-A 空白应用同步创建即 ready）
CREATE TABLE IF NOT EXISTS tn_applications (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    code varchar(64) NOT NULL,
    name varchar(128) NOT NULL,
    icon jsonb NOT NULL DEFAULT '{"type":"remix","name":"bookmark","background":"#f7be54,#eda426"}'::jsonb,
    color varchar(32) NOT NULL DEFAULT 'primary',
    owner_member_id BIGINT NOT NULL,
    creator_member_id BIGINT NOT NULL,
    source_type varchar(16) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    provision_status varchar(16) NOT NULL DEFAULT 'ready',
    home_mode varchar(16) NOT NULL DEFAULT 'builder',
    definition_version INTEGER NOT NULL DEFAULT 1,
    menu_revision BIGINT NOT NULL DEFAULT 1,
    sort_order BIGINT NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_tn_applications_source_type CHECK (source_type IN ('blank', 'template')),
    CONSTRAINT chk_tn_applications_status CHECK (status IN ('active', 'archived')),
    CONSTRAINT chk_tn_applications_provision_status CHECK (provision_status IN ('ready', 'pending', 'running', 'failed')),
    CONSTRAINT chk_tn_applications_home_mode CHECK (home_mode IN ('builder', 'application'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_applications_tenant_code
    ON tn_applications (tenant_id, code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tn_applications_tenant_status_sort
    ON tn_applications (tenant_id, status, sort_order, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tn_applications_tenant_owner
    ON tn_applications (tenant_id, owner_member_id)
    WHERE deleted_at IS NULL;

-- 安装记录（000014）：应用创建来源快照，一应用一条、追加写无软删；
-- template_* 两列 M2-A 恒为 NULL，M2-B 模板安装启用
CREATE TABLE IF NOT EXISTS tn_application_installations (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    application_id BIGINT NOT NULL UNIQUE REFERENCES tn_applications(id),
    source_type varchar(16) NOT NULL,
    template_id BIGINT,
    template_version_id BIGINT,
    channel varchar(32) NOT NULL,
    blueprint_checksum varchar(64),
    installed_by_member_id BIGINT NOT NULL,
    installed_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tn_application_installations_tenant
    ON tn_application_installations (tenant_id, id);

-- 应用菜单节点（000016，M2-菜单）：分组/表单/仪表盘/页面统一为菜单节点；
-- 分组无 target，非分组节点 target_type=entry_type 且引用资产；读取序
-- sort_order ASC, code ASC（tiebreak 与出网 entryId 同源）
CREATE TABLE IF NOT EXISTS tn_application_menu_entries (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    application_id BIGINT NOT NULL REFERENCES tn_applications(id),
    code varchar(64) NOT NULL,
    parent_entry_id BIGINT NULL REFERENCES tn_application_menu_entries(id),
    entry_type varchar(16) NOT NULL,
    name varchar(128) NOT NULL,
    icon varchar(32) NULL,
    color varchar(32) NULL,
    target_type varchar(16) NULL,
    target_id BIGINT NULL,
    sort_order BIGINT NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    hidden BOOLEAN NOT NULL DEFAULT false,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_tn_application_menu_entry_type
      CHECK (entry_type IN ('group', 'form', 'dashboard', 'page')),
    CONSTRAINT chk_tn_application_menu_target
      CHECK (
        (entry_type = 'group' AND target_type IS NULL AND target_id IS NULL)
        OR
        (entry_type <> 'group' AND target_type = entry_type AND target_id IS NOT NULL)
      )
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_application_menu_entries_tenant_code
    ON tn_application_menu_entries (tenant_id, code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tn_application_menu_entries_app_parent_sort
    ON tn_application_menu_entries (tenant_id, application_id, parent_entry_id, sort_order, code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tn_application_menu_entries_app_target
    ON tn_application_menu_entries (tenant_id, application_id, target_type, target_id)
    WHERE deleted_at IS NULL AND target_id IS NOT NULL;

-- 应用菜单个人收藏（000046，ADR-011）：成员×菜单节点的个人状态，不参与
-- 菜单共享结构与修订号；节点软删时同事务硬删关联行
CREATE TABLE IF NOT EXISTS tn_application_menu_favorites (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    application_id BIGINT NOT NULL REFERENCES tn_applications(id),
    entry_id BIGINT NOT NULL REFERENCES tn_application_menu_entries(id),
    created_at timestamp with time zone,
    CONSTRAINT uk_tn_application_menu_favorites_member_entry UNIQUE (member_id, entry_id)
);

CREATE INDEX IF NOT EXISTS idx_tn_application_menu_favorites_member
    ON tn_application_menu_favorites (tenant_id, member_id, application_id);

COMMENT ON TABLE tn_application_menu_favorites IS '应用菜单个人收藏（000046，ADR-011）：成员×菜单节点的个人状态，不参与菜单共享结构与修订号；节点软删时同事务硬删关联行';
COMMENT ON COLUMN tn_application_menu_favorites.id IS '自增主键';
COMMENT ON COLUMN tn_application_menu_favorites.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_application_menu_favorites.member_id IS '收藏成员 ID（租户成员，读写一律叠加本列双条件）';
COMMENT ON COLUMN tn_application_menu_favorites.application_id IS '收藏节点所属应用 ID（外键指向 tn_applications）';
COMMENT ON COLUMN tn_application_menu_favorites.entry_id IS '收藏的菜单节点 ID（外键指向 tn_application_menu_entries；(member_id, entry_id) 唯一幂等）';
COMMENT ON COLUMN tn_application_menu_favorites.created_at IS '收藏时间';

-- 表单资产与草稿（000037/000044/000045，ADR-010）：code 为稳定公开编码，
-- draft_content 为目标保存协议草稿全文
--（content.items 两层结构，保存前经字段字典严格校验），draft_revision 乐观锁
CREATE TABLE IF NOT EXISTS tn_forms (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    application_id BIGINT NOT NULL,
    code varchar(64) NOT NULL,
    name varchar(128) NOT NULL,
    form_type varchar(16) NOT NULL DEFAULT 'standard',
    draft_content JSONB NOT NULL,
    draft_revision BIGINT NOT NULL DEFAULT 1,
    protocol_version INTEGER NOT NULL DEFAULT 4,
    latest_version_id BIGINT,
    published_version INTEGER NOT NULL DEFAULT 0,
    creator_member_id BIGINT NOT NULL,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_tn_forms_content_object CHECK (jsonb_typeof(draft_content) = 'object'),
    CONSTRAINT chk_tn_forms_form_type CHECK (form_type IN ('standard', 'workflow'))
);

CREATE INDEX IF NOT EXISTS idx_tn_forms_tenant_app
    ON tn_forms (tenant_id, application_id, id DESC)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_forms_tenant_code
    ON tn_forms (tenant_id, code)
    WHERE deleted_at IS NULL;

-- 发布快照（000037）：不可变、追加写；(form_id, version_no) 唯一，
-- schema_revision=行 id（发布事务内回填），提交按双口令定位并依据快照终审
CREATE TABLE IF NOT EXISTS tn_form_versions (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    form_id BIGINT NOT NULL REFERENCES tn_forms(id),
    version_no INTEGER NOT NULL,
    schema_revision BIGINT NOT NULL DEFAULT 0,
    content JSONB NOT NULL,
    field_keys JSONB NOT NULL DEFAULT '[]',
    protocol_version INTEGER NOT NULL DEFAULT 4,
    published_by_member_id BIGINT NOT NULL,
    published_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    created_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_form_versions_form_no
    ON tn_form_versions (form_id, version_no);

CREATE INDEX IF NOT EXISTS idx_tn_form_versions_tenant
    ON tn_form_versions (tenant_id, id);

-- 表单记录提交（000038）：追加写；values 为按发布快照校验通过的值（键=widgetName），
-- form_version_id 固定受理时所依据的发布版本（历史版本合法）
CREATE TABLE IF NOT EXISTS tn_form_records (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    form_id BIGINT NOT NULL REFERENCES tn_forms(id),
    form_version_id BIGINT NOT NULL REFERENCES tn_form_versions(id),
    data_op_id varchar(36),
    entry_code varchar(64),
    values JSONB NOT NULL,
    submitted_by_member_id BIGINT NOT NULL,
    submitted_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_tn_form_records_tenant_form
    ON tn_form_records (tenant_id, form_id, id DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_form_records_tenant_data_op
    ON tn_form_records (tenant_id, data_op_id);

COMMENT ON COLUMN tn_form_records.data_op_id IS '客户端生成的单次提交幂等 UUID；同一租户内唯一，历史记录允许为空';
COMMENT ON COLUMN tn_form_records.entry_code IS '触发提交的应用菜单节点公开编码快照；设计预览直提允许为空';

-- 资产权限组（000058，表单权限 P1）：主体范围×操作集×字段矩阵×数据范围的
-- 整体授权单元；asset_type 现仅 form（类型白名单在 Service 注册表，预留
-- dashboard 扩展位）。表单存在任一权限组行（含禁用组）即进入授权模型（S5 收口）。
-- 索引策略定版：不建 values 根列 GIN（模板谓词作用于 values->'F'，GIN 无法命中）
CREATE TABLE IF NOT EXISTS tn_asset_permission_groups (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    application_id BIGINT NOT NULL,
    asset_type VARCHAR(16) NOT NULL DEFAULT 'form',
    asset_id BIGINT NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    description VARCHAR(200) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    operations JSONB NOT NULL DEFAULT '[]',
    field_permissions JSONB NOT NULL DEFAULT '[]',
    data_scope JSONB NOT NULL DEFAULT '{}',
    revision BIGINT NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    deleted_at timestamp with time zone,
    CONSTRAINT uq_tn_asset_permission_groups_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_tn_asset_permission_groups_tenant
    ON tn_asset_permission_groups (tenant_id, asset_type, asset_id);

COMMENT ON TABLE tn_asset_permission_groups IS '资产权限组（表单权限 P1）：主体范围×操作集×字段矩阵×数据范围的整体授权单元；表单存在任一权限组行（含禁用组）即进入授权模型（S5 收口）';
COMMENT ON COLUMN tn_asset_permission_groups.code IS '权限组公开编码（fpg_ 前缀服务端生成，出网稳定标识）';
COMMENT ON COLUMN tn_asset_permission_groups.enabled IS '启用状态；禁用组同样维持收口（S5）但不授权';
COMMENT ON COLUMN tn_asset_permission_groups.operations IS '操作键 JSONB 数组（view/add/copy/edit/delete/batch_print/batch_modify/import/export + 流程表单 workflow_*）';
COMMENT ON COLUMN tn_asset_permission_groups.field_permissions IS '字段矩阵 JSONB 数组 [{field,visible,editable}]；缺失字段 deny-by-default（S7）';
COMMENT ON COLUMN tn_asset_permission_groups.data_scope IS '数据范围 JSONB {match,conditions}（match: all/any；空条件=全部数据 S6）';
COMMENT ON COLUMN tn_asset_permission_groups.revision IS '整组乐观锁口令（PUT 全量提交，冲突返回 FORM_PERMISSION_REVISION_CONFLICT）';

-- 权限组主体（000058）：成员/部门/角色，判定侧按主体反查。外键 CASCADE 只对
-- 物理 DELETE 生效：组删除走软删，由 Service 同事务显式硬删本表行；subject_id
-- 不做外键，角色/部门删除时判定侧容错（解析不到的主体不命中）
CREATE TABLE IF NOT EXISTS tn_asset_permission_group_subjects (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    subject_type VARCHAR(16) NOT NULL,
    subject_id BIGINT NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tn_asset_permission_group_subjects UNIQUE (group_id, subject_type, subject_id),
    CONSTRAINT ck_tn_asset_permission_group_subjects_type
        CHECK (subject_type IN ('member', 'department', 'role')),
    CONSTRAINT fk_tn_asset_permission_group_subjects_group
        FOREIGN KEY (group_id) REFERENCES tn_asset_permission_groups (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tn_asset_permission_group_subjects_lookup
    ON tn_asset_permission_group_subjects (tenant_id, subject_type, subject_id);

COMMENT ON TABLE tn_asset_permission_group_subjects IS '资产权限组主体：成员/部门/角色；判定侧按主体反查命中组（部门含子部门，祖先链命中）';
COMMENT ON COLUMN tn_asset_permission_group_subjects.subject_type IS '主体类型：member/department/role（CHECK 约束）';
COMMENT ON COLUMN tn_asset_permission_group_subjects.subject_id IS '主体 ID（tn_users.id/tn_departments.id/tn_roles.id；无外键，判定侧容错解析不到的主体）';

-- 消息中心不可变逻辑消息（000039）：模板渲染后的展示快照固化存储，
-- 多成员经收件箱共享同一行；(tenant_id, event_id) 唯一承担 Worker 重试幂等
CREATE TABLE IF NOT EXISTS tn_notification_messages (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    event_id varchar(128) NOT NULL,
    category_code varchar(64) NOT NULL,
    event_code varchar(128) NOT NULL,
    severity varchar(16) NOT NULL DEFAULT 'info',
    title varchar(200) NOT NULL DEFAULT '',
    content varchar(2000) NOT NULL,
    action JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_ref JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT chk_tn_notification_messages_severity CHECK (severity IN ('info', 'success', 'warning', 'error')),
    CONSTRAINT chk_tn_notification_messages_action CHECK (jsonb_typeof(action) = 'object'),
    CONSTRAINT chk_tn_notification_messages_source_ref CHECK (jsonb_typeof(source_ref) = 'object'),
    CONSTRAINT chk_tn_notification_messages_content CHECK (length(btrim(content)) > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_notification_messages_tenant_event
    ON tn_notification_messages (tenant_id, event_id);

CREATE INDEX IF NOT EXISTS idx_tn_notification_messages_tenant_category
    ON tn_notification_messages (tenant_id, category_code, occurred_at DESC, id DESC);

-- 成员收件箱（000039）：站内信扇出与已读状态；查询/更新必须 tenant_id+member_id 双条件
CREATE TABLE IF NOT EXISTS tn_notification_member_inboxes (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    message_id BIGINT NOT NULL REFERENCES tn_notification_messages(id),
    member_id BIGINT NOT NULL,
    category_code varchar(64) NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    read_at timestamp with time zone,
    created_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_notification_inboxes_unique
    ON tn_notification_member_inboxes (tenant_id, message_id, member_id);

CREATE INDEX IF NOT EXISTS idx_tn_notification_inboxes_member_list
    ON tn_notification_member_inboxes (tenant_id, member_id, category_code, occurred_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_tn_notification_inboxes_member_unread
    ON tn_notification_member_inboxes (tenant_id, member_id, category_code, occurred_at DESC, id DESC)
    WHERE read_at IS NULL;

-- 租户通知设置聚合根（000039）：每租户一行有效记录，revision 覆盖整个聚合
CREATE TABLE IF NOT EXISTS tn_notification_settings (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    revision BIGINT NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_notification_settings_tenant
    ON tn_notification_settings (tenant_id)
    WHERE deleted_at IS NULL;

-- 事件偏好覆盖（000039）：只保存对注册表默认值的覆盖，无覆盖行投影默认
CREATE TABLE IF NOT EXISTS tn_notification_preferences (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    setting_id BIGINT NOT NULL REFERENCES tn_notification_settings(id),
    event_code varchar(128) NOT NULL,
    system_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    email_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    sms_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    recipients_overridden BOOLEAN NOT NULL DEFAULT FALSE,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_notification_prefs_event
    ON tn_notification_preferences (tenant_id, event_code);

CREATE INDEX IF NOT EXISTS idx_tn_notification_prefs_setting
    ON tn_notification_preferences (setting_id);

-- 事件偏好接收规则关联（000039）：CHECK 强制 target_kind 与联系人 ID 组合合法
CREATE TABLE IF NOT EXISTS tn_notification_preference_recipients (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    preference_id BIGINT NOT NULL REFERENCES tn_notification_preferences(id),
    target_kind varchar(32) NOT NULL,
    custom_recipient_id BIGINT,
    created_at timestamp with time zone,
    CONSTRAINT chk_tn_notif_pref_recipients_kind CHECK (target_kind IN ('event_actor', 'event_audience', 'tenant_admin', 'custom_recipient')),
    CONSTRAINT chk_tn_notif_pref_recipients_custom CHECK (
        (target_kind = 'custom_recipient' AND custom_recipient_id IS NOT NULL)
        OR (target_kind <> 'custom_recipient' AND custom_recipient_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_tn_notif_pref_recipients_pref
    ON tn_notification_preference_recipients (preference_id);

CREATE INDEX IF NOT EXISTS idx_tn_notif_pref_recipients_recipient
    ON tn_notification_preference_recipients (custom_recipient_id)
    WHERE custom_recipient_id IS NOT NULL;

-- 租户自定义外部提醒对象池（000039）：软删除保留关联历史；手机/邮箱至少一项由服务层校验
CREATE TABLE IF NOT EXISTS tn_notification_custom_recipients (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    name varchar(80) NOT NULL,
    mobile varchar(32) NOT NULL DEFAULT '',
    email varchar(254) NOT NULL DEFAULT '',
    revision BIGINT NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_tn_notification_recipients_tenant
    ON tn_notification_custom_recipients (tenant_id)
    WHERE deleted_at IS NULL;

-- 事务 Outbox（000039）：业务事务与消息物化之间的可靠边界，event_id 全局幂等
CREATE TABLE IF NOT EXISTS tn_notification_outbox_events (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    event_id varchar(128) NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    event_code varchar(128) NOT NULL,
    actor_member_id BIGINT NOT NULL DEFAULT 0,
    audience_member_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    status varchar(16) NOT NULL DEFAULT 'pending',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    last_error_code varchar(100) NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    processed_at timestamp with time zone,
    CONSTRAINT chk_tn_notification_outbox_status CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    CONSTRAINT chk_tn_notification_outbox_audience CHECK (jsonb_typeof(audience_member_ids) = 'array'),
    CONSTRAINT chk_tn_notification_outbox_parameters CHECK (jsonb_typeof(parameters) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_notification_outbox_event_id
    ON tn_notification_outbox_events (event_id);

CREATE INDEX IF NOT EXISTS idx_tn_notification_outbox_dispatch
    ON tn_notification_outbox_events (status, next_attempt_at, id);

-- RustFS 文件元数据（000017）：对象字节存于私有 bucket，数据库只保存
-- 租户归属、配额预留与访问控制所需字段。

CREATE TABLE IF NOT EXISTS tn_files (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    code varchar(64) NOT NULL,
    bucket varchar(128) NOT NULL,
    object_key varchar(768) NOT NULL,
    original_name varchar(255) NOT NULL,
    content_type varchar(255) NOT NULL,
    declared_size BIGINT NOT NULL,
    actual_size BIGINT NOT NULL DEFAULT 0,
    sha256 varchar(64) NULL,
    state varchar(16) NOT NULL,
    expires_at timestamp with time zone NULL,
    creator_member_id BIGINT NOT NULL,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_tn_files_state CHECK (state IN ('uploading', 'ready')),
    CONSTRAINT chk_tn_files_declared_size CHECK (declared_size > 0),
    CONSTRAINT chk_tn_files_actual_size CHECK (actual_size >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_files_tenant_code
    ON tn_files (tenant_id, code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tn_files_tenant_state
    ON tn_files (tenant_id, state)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tn_files_uploading_expiry
    ON tn_files (expires_at)
    WHERE deleted_at IS NULL AND state = 'uploading';

-- ============ 表/字段注释（000008，与迁移链一致） ============
COMMENT ON TABLE pf_accounts IS '平台账号（登录身份）：登录名/手机号/密码与第三方凭证挂账号；账号跨租户，无 tenant_id（ADR-006/FIX-014）';
COMMENT ON COLUMN pf_accounts.id IS '自增主键';
COMMENT ON COLUMN pf_accounts.name IS '登录名，全局唯一';
COMMENT ON COLUMN pf_accounts.nickname IS '平台级展示昵称；租户内昵称见 tn_users.nickname';
COMMENT ON COLUMN pf_accounts.phone IS '手机号，非空时全局唯一（部分唯一索引 uk_pf_accounts_phone，未填写落空串不参与唯一）';
COMMENT ON COLUMN pf_accounts.email IS '邮箱';
COMMENT ON COLUMN pf_accounts.password IS '登录密码（bcrypt 摘要）；纯 OAuth 账号可为空';
COMMENT ON COLUMN pf_accounts.password_initialized IS '密码是否由用户本人设置：短信免密注册为 false（存服务端随机密码），首次设置密码后置 true';
COMMENT ON COLUMN pf_accounts.session_version IS '账号会话版本：密码重设/修改成功时递增，JWT 版本不一致即失效';
COMMENT ON COLUMN pf_accounts.avatar IS '头像 URL 或裁剪压缩后的 data URL';
COMMENT ON COLUMN pf_accounts.onboarding IS '账号注册引导画像：role 角色 / channel 了解渠道（注册向导第 3 步采集）';
COMMENT ON COLUMN pf_accounts.created_at IS '创建时间';
COMMENT ON COLUMN pf_accounts.updated_at IS '更新时间';
COMMENT ON COLUMN pf_accounts.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE pf_tenants IS '租户（平台一级资源）：套餐/状态/Owner/租户级配置与注销生命周期；无 tenant_id（FIX-014）';
COMMENT ON COLUMN pf_tenants.id IS '自增主键';
COMMENT ON COLUMN pf_tenants.code IS '租户编码，全局唯一，登录时识别目标租户';
COMMENT ON COLUMN pf_tenants.name IS '租户名称';
COMMENT ON COLUMN pf_tenants.plan IS '套餐标识（free 等，套餐定义在租户域代码 plan.go，暂无套餐表）';
COMMENT ON COLUMN pf_tenants.status IS '租户状态：active=正常 / frozen=冻结 / deleted=已注销';
COMMENT ON COLUMN pf_tenants.owner_account_id IS '开通者（Owner）平台账号 ID，NULL=未设置；外键指向 pf_accounts(id)（FIX-016）';
COMMENT ON COLUMN pf_tenants.config IS '租户级配置 JSONB：水印/品牌主题/时区/语言';
COMMENT ON COLUMN pf_tenants.quotas IS '套餐配额覆盖 JSONB；空对象表示使用套餐默认值';
COMMENT ON COLUMN pf_tenants.delete_requested_at IS '注销申请时间（FIX-012 生命周期时间线）';
COMMENT ON COLUMN pf_tenants.retention_until IS '数据保留截止时间，到期由 Purge Worker 执行清理';
COMMENT ON COLUMN pf_tenants.purged_at IS '最终清理完成时间（墓碑标记）';
COMMENT ON COLUMN pf_tenants.created_at IS '创建时间';
COMMENT ON COLUMN pf_tenants.updated_at IS '更新时间';
COMMENT ON COLUMN pf_tenants.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE pf_auth_infos IS '第三方登录凭证（OAuth）：归属平台账号，无租户归属（ADR-006）';
COMMENT ON COLUMN pf_auth_infos.id IS '自增主键';
COMMENT ON COLUMN pf_auth_infos.account_id IS '归属平台账号 ID，外键指向 pf_accounts(id)；存量回填前可能为 0';
COMMENT ON COLUMN pf_auth_infos.url IS '第三方资料页地址';
COMMENT ON COLUMN pf_auth_infos.auth_type IS '第三方渠道类型：github / wechat';
COMMENT ON COLUMN pf_auth_infos.auth_id IS '第三方用户唯一标识；与 auth_type 组成软删友好唯一键（FIX-017）';
COMMENT ON COLUMN pf_auth_infos.access_token IS '访问令牌';
COMMENT ON COLUMN pf_auth_infos.refresh_token IS '刷新令牌';
COMMENT ON COLUMN pf_auth_infos.expiry IS '令牌过期时间';
COMMENT ON COLUMN pf_auth_infos.created_at IS '创建时间';
COMMENT ON COLUMN pf_auth_infos.updated_at IS '更新时间';
COMMENT ON COLUMN pf_auth_infos.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE tn_users IS '租户成员（租户内身份）：归属账号+租户内昵称；部门/分组/角色关系见各关联表（ADR-006）';
COMMENT ON COLUMN tn_users.id IS '自增主键';
COMMENT ON COLUMN tn_users.account_id IS '归属平台账号 ID，外键指向 pf_accounts(id)；同一账号在每租户至多一条有效成员关系（FIX-004）';
COMMENT ON COLUMN tn_users.nickname IS '租户内展示昵称；空则前端回落账号昵称';
COMMENT ON COLUMN tn_users.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_users.created_at IS '创建时间';
COMMENT ON COLUMN tn_users.updated_at IS '更新时间';
COMMENT ON COLUMN tn_users.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE tn_departments IS '部门（租户内组织架构，邻接表树）：承载组织结构与汇报关系，权限授权走分组/角色';
COMMENT ON COLUMN tn_departments.id IS '自增主键';
COMMENT ON COLUMN tn_departments.parent_id IS '父部门 ID，NULL=根节点；自引用外键，跨租户父节点被 FK 与服务层共同拦截（FIX-015）';
COMMENT ON COLUMN tn_departments.name IS '部门名称';
COMMENT ON COLUMN tn_departments."order" IS '同级排序值，小者在前';
COMMENT ON COLUMN tn_departments.status IS '部门状态：active=正常 / disabled=停用';
COMMENT ON COLUMN tn_departments.leader_member_id IS '部门负责人成员 ID，NULL=未设置；流程引擎 department_manager 审批人解析数据源（ADR-012 Phase 3），同租户有效性由部门服务写入时校验';
COMMENT ON COLUMN tn_departments.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_departments.created_at IS '创建时间';
COMMENT ON COLUMN tn_departments.updated_at IS '更新时间';
COMMENT ON COLUMN tn_departments.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE tn_department_users IS '部门-成员关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN tn_department_users.department_id IS '部门 ID，联合主键之一，外键指向 tn_departments(id)';
COMMENT ON COLUMN tn_department_users.user_id IS '成员 ID，联合主键之一，外键指向 tn_users(id)';

COMMENT ON TABLE tn_groups IS '用户分组（租户内权限分组）：系统组（root/system:*）与自定义组，角色可挂分组间接授权成员';
COMMENT ON COLUMN tn_groups.id IS '自增主键';
COMMENT ON COLUMN tn_groups.name IS '分组名，租户内唯一（软删友好部分唯一索引 uk_tn_groups_tenant_name，FIX-003）';
COMMENT ON COLUMN tn_groups.kind IS '分组类型：system=系统组 / custom=自定义组';
COMMENT ON COLUMN tn_groups.describe IS '分组描述（列名沿用历史拼写）';
COMMENT ON COLUMN tn_groups.creator_id IS '创建账号 ID（pf_accounts.id）';
COMMENT ON COLUMN tn_groups.updater_id IS '最后更新账号 ID（pf_accounts.id）';
COMMENT ON COLUMN tn_groups.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_groups.created_at IS '创建时间';
COMMENT ON COLUMN tn_groups.updated_at IS '更新时间';
COMMENT ON COLUMN tn_groups.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE tn_user_groups IS '成员-分组关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN tn_user_groups.group_id IS '分组 ID，联合主键之一，外键指向 tn_groups(id)';
COMMENT ON COLUMN tn_user_groups.user_id IS '成员 ID，联合主键之一，外键指向 tn_users(id)';

COMMENT ON TABLE sys_resources IS '平台资源目录（RBAC 鉴权资源清单）：平台级，不挂租户';
COMMENT ON COLUMN sys_resources.id IS '自增主键';
COMMENT ON COLUMN sys_resources.name IS '资源名，对应角色规则里的 resource（tn_users/tn_groups/auth 等）';
COMMENT ON COLUMN sys_resources.scope IS '作用域：cluster=平台级';
COMMENT ON COLUMN sys_resources.kind IS '资源种类：resource=API 资源 / menu=菜单';

COMMENT ON TABLE tn_roles IS '角色（租户内）：rules 声明资源-操作授权规则，可挂成员或分组';
COMMENT ON COLUMN tn_roles.id IS '自增主键';
COMMENT ON COLUMN tn_roles.name IS '角色名，租户内唯一（软删友好部分唯一索引 uk_tn_roles_tenant_name，FIX-002）';
COMMENT ON COLUMN tn_roles.role_group_id IS '角色所属展示分组 ID；为空表示未归类';
COMMENT ON COLUMN tn_roles.sort IS '角色在所属展示分组中的展示顺序，数值越小越靠前';
COMMENT ON COLUMN tn_roles.scope IS '作用域：cluster=平台级';
COMMENT ON COLUMN tn_roles.namespace IS '命名空间，预留扩展';
COMMENT ON COLUMN tn_roles.rules IS '授权规则 JSON 数组，元素形如 {"resource":"*","operation":"*"}；operation 支持 */edit/view';
COMMENT ON COLUMN tn_roles.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_roles.created_at IS '创建时间';
COMMENT ON COLUMN tn_roles.updated_at IS '更新时间';
COMMENT ON COLUMN tn_roles.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE tn_role_groups IS '角色展示分组：仅供内部组织页归类角色，不参与权限继承';
COMMENT ON COLUMN tn_role_groups.id IS '自增主键';
COMMENT ON COLUMN tn_role_groups.name IS '角色组名称，租户内未删除记录唯一';
COMMENT ON COLUMN tn_role_groups.creator_member_id IS '创建该角色组的成员 ID（tn_users.id）';
COMMENT ON COLUMN tn_role_groups.creator_id IS '创建账号 ID（pf_accounts.id）';
COMMENT ON COLUMN tn_role_groups.updater_id IS '最后更新账号 ID（pf_accounts.id）';
COMMENT ON COLUMN tn_role_groups.sort IS '角色组在内部组织左侧角色树中的展示顺序，数值越小越靠前';
COMMENT ON COLUMN tn_role_groups.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_role_groups.created_at IS '创建时间';
COMMENT ON COLUMN tn_role_groups.updated_at IS '更新时间';
COMMENT ON COLUMN tn_role_groups.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE tn_user_roles IS '成员-角色关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN tn_user_roles.user_id IS '成员 ID，联合主键之一，外键指向 tn_users(id)';
COMMENT ON COLUMN tn_user_roles.role_id IS '角色 ID，联合主键之一，外键指向 tn_roles(id)';

COMMENT ON TABLE tn_group_roles IS '分组-角色关联表（多对多）：分组内成员经本表继承角色授权';
COMMENT ON COLUMN tn_group_roles.group_id IS '分组 ID，联合主键之一，外键指向 tn_groups(id)';
COMMENT ON COLUMN tn_group_roles.role_id IS '角色 ID，联合主键之一，外键指向 tn_roles(id)';

COMMENT ON TABLE tn_audit_logs IS '业务审计日志（FIX-013）：追加写流水，记录谁在什么租户对什么资源做了什么；无更新/软删语义';
COMMENT ON COLUMN tn_audit_logs.id IS '自增主键';
COMMENT ON COLUMN tn_audit_logs.tenant_id IS '租户 ID，0=平台级操作（运营域）';
COMMENT ON COLUMN tn_audit_logs.account_id IS '操作者平台账号 ID，0=系统或未知';
COMMENT ON COLUMN tn_audit_logs.member_id IS '操作者租户成员 ID，0=平台级操作或未知';
COMMENT ON COLUMN tn_audit_logs.module IS '业务域模块：tenant/iam/...';
COMMENT ON COLUMN tn_audit_logs.action IS '动作：create/update/delete/bind/...';
COMMENT ON COLUMN tn_audit_logs.resource_type IS '目标资源类型（tn_users/tn_groups/tn_roles 等）';
COMMENT ON COLUMN tn_audit_logs.resource_id IS '目标资源 ID，可空';
COMMENT ON COLUMN tn_audit_logs.request_id IS '链路追踪请求 ID，可空';
COMMENT ON COLUMN tn_audit_logs.ip IS '客户端 IP，可空';
COMMENT ON COLUMN tn_audit_logs.user_agent IS '客户端 User-Agent，可空';
COMMENT ON COLUMN tn_audit_logs.before_data IS '变更前数据快照 JSONB，可空';
COMMENT ON COLUMN tn_audit_logs.after_data IS '变更后数据快照 JSONB，可空';
COMMENT ON COLUMN tn_audit_logs.event_code IS '稳定事件码（模块.资源类型.动作，如 iam.member.update），由审计服务按事件注册表生成；空为存量历史行，展示降级为「历史操作记录」';
COMMENT ON COLUMN tn_audit_logs.category_code IS '稳定日志范围码：member_management 成员管理 / organization 组织架构 / role_permission 角色权限 / tenant_settings 企业设置 / application 应用管理 / file_storage 文件管理 / account_security 账号安全 / log_export 日志导出';
COMMENT ON COLUMN tn_audit_logs.actor_name_snapshot IS '操作人显示名快照（写时固化，成员资料变更不影响历史展示）';
COMMENT ON COLUMN tn_audit_logs.target_name_snapshot IS '目标资源展示名快照（成员/部门/角色/应用等当时名称）';
COMMENT ON COLUMN tn_audit_logs.summary IS '服务端生成并经脱敏的操作详情，可直接展示与导出；不含密码/验证码/令牌/私钥/完整手机号邮箱等敏感值';
COMMENT ON COLUMN tn_audit_logs.application_id IS '应用内操作所属应用 ID（产品日志查询与租户归属校验维度）；NULL=非应用内操作或历史数据。应用维度查询必须以前置 tenant_id 为首列，禁止仅凭本列查询';
COMMENT ON COLUMN tn_audit_logs.application_code IS '应用稳定编码快照（写时固化，应用删除后历史展示不依赖当前应用行）';
COMMENT ON COLUMN tn_audit_logs.application_name_snapshot IS '应用名称快照（写时固化，应用改名/删除后历史展示一致）';
COMMENT ON COLUMN tn_audit_logs.created_at IS '记录发生时间，默认当前时间';

COMMENT ON TABLE pf_login_logs IS '登录日志：会话建立事件流水（登录/注册即登录），账号维度自查（ADR-006 平台级）；与 tn_audit_logs 职责互斥，登录不写业务审计';
COMMENT ON COLUMN pf_login_logs.id IS '自增主键';
COMMENT ON COLUMN pf_login_logs.account_id IS '登录的平台账号 ID（主查询维度）';
COMMENT ON COLUMN pf_login_logs.tenant_id IS '本次登录进入的租户 ID，0=无租户/平台场景';
COMMENT ON COLUMN pf_login_logs.member_id IS '本次登录绑定的租户成员 ID，0=未解析到成员';
COMMENT ON COLUMN pf_login_logs.method IS '登录方式：password 密码 / sms 短信验证码 / oauth_github、oauth_wechat 第三方 / register 注册即登录';
COMMENT ON COLUMN pf_login_logs.client IS '客户端形态（UA 解析）：web 电脑网页 / wap 手机网页 / unknown';
COMMENT ON COLUMN pf_login_logs.ip IS '客户端 IP，可空';
COMMENT ON COLUMN pf_login_logs.location IS 'IP 归属地（ip2region 离线库写时解析，如「广东省 深圳市」）；回环/私网为「内网地址」，解析失败为「未知」';
COMMENT ON COLUMN pf_login_logs.user_agent IS '客户端 User-Agent，可空';
COMMENT ON COLUMN pf_login_logs.request_id IS '链路追踪请求 ID，可空';
COMMENT ON COLUMN pf_login_logs.actor_name_snapshot IS '登录人显示名快照（写时固化，成员改名/离职/删除后历史展示一致）；空为存量历史行，读取侧回查当前成员昵称兜底';
COMMENT ON COLUMN pf_login_logs.created_at IS '登录时间';

COMMENT ON TABLE tn_enterprise_log_exports IS '企业日志导出任务（000036）：固化提交时的筛选条件/申请人/租户/数据量/状态与过期时间；一期同步生成、内容内联存储，异步导出与对象存储文件引用随留存策略批次接入';
COMMENT ON COLUMN tn_enterprise_log_exports.id IS '自增主键';
COMMENT ON COLUMN tn_enterprise_log_exports.tenant_id IS '所属租户；任务状态读取与下载均复核归属，跨租户不可见';
COMMENT ON COLUMN tn_enterprise_log_exports.account_id IS '申请人平台账号 ID';
COMMENT ON COLUMN tn_enterprise_log_exports.member_id IS '申请人租户成员 ID，0=未解析到成员';
COMMENT ON COLUMN tn_enterprise_log_exports.kind IS '导出日志类型：login 登录日志 / operation 操作日志';
COMMENT ON COLUMN tn_enterprise_log_exports.filters IS '提交时固化的筛选条件快照 JSONB（成员/日志范围/事件码/时间范围），与列表查询参数同构';
COMMENT ON COLUMN tn_enterprise_log_exports.total IS '导出数据量（行数）';
COMMENT ON COLUMN tn_enterprise_log_exports.status IS '任务状态：pending 生成中 / ready 就绪 / failed 生成失败';
COMMENT ON COLUMN tn_enterprise_log_exports.file_name IS '导出文件名（下载 Content-Disposition 用）';
COMMENT ON COLUMN tn_enterprise_log_exports.file_data IS '导出文件内容（一期 CSV 内联存储；异步批改对象存储文件引用后本列退化为空串）';
COMMENT ON COLUMN tn_enterprise_log_exports.expires_at IS '导出文件过期时间，过期后不可下载';
COMMENT ON COLUMN tn_enterprise_log_exports.created_at IS '任务创建时间';

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

COMMENT ON TABLE tn_applications IS '应用实例（租户内一级资源，M2-A）：空白/模板安装创建的低代码应用；owner/creator 引用租户成员，删除只写 deleted_at（状态列不设 deleted）';
COMMENT ON COLUMN tn_applications.id IS '自增主键';
COMMENT ON COLUMN tn_applications.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_applications.code IS '服务端生成的应用编码，租户内唯一（uk_tn_applications_tenant_code，软删行释放），供 URL/外部引用/日志使用，创建后不可修改';
COMMENT ON COLUMN tn_applications.name IS '展示名称，允许同租户重名，不作业务主键';
COMMENT ON COLUMN tn_applications.icon IS '应用图标 JSONB：remix 为 type/name/background，自定义图标为 type/name';
COMMENT ON COLUMN tn_applications.color IS '稳定颜色键（primary），不存 CSS 字面值';
COMMENT ON COLUMN tn_applications.owner_member_id IS '应用所有者（租户成员 ID），同租户约束由服务层校验';
COMMENT ON COLUMN tn_applications.creator_member_id IS '创建者（租户成员 ID），成员删除后保留 ID 审计语义';
COMMENT ON COLUMN tn_applications.source_type IS '创建来源：blank 空白创建 / template 模板安装（冗余于安装记录，便于列表查询）';
COMMENT ON COLUMN tn_applications.status IS '可见业务状态：active 正常 / archived 归档；删除只写 deleted_at，不设 deleted 状态值';
COMMENT ON COLUMN tn_applications.provision_status IS '实例化进度（与 status 独立）：ready 就绪 / pending 待处理 / running 处理中 / failed 失败；M2-A 空白应用同步创建即 ready';
COMMENT ON COLUMN tn_applications.home_mode IS '应用首页形态：builder 显示首次构建引导 / application 进入运行时应用首页；由应用生命周期维护，不按当前成员可见菜单数量推导';
COMMENT ON COLUMN tn_applications.definition_version IS '应用定义版本（发布演进用），非数据库乐观锁';
COMMENT ON COLUMN tn_applications.menu_revision IS '菜单修订号（菜单结构乐观并发口令）：菜单写入在同事务内条件递增；与 definition_version（发布演进）独立，应用名称/图标/归档等非菜单更新不递增';
COMMENT ON COLUMN tn_applications.sort_order IS '列表排序值，小者在前，同值按 id 倒序';
COMMENT ON COLUMN tn_applications.config IS '小型应用级配置 JSONB；严禁混入表单/页面/流程大定义';
COMMENT ON COLUMN tn_applications.created_at IS '创建时间';
COMMENT ON COLUMN tn_applications.updated_at IS '更新时间';
COMMENT ON COLUMN tn_applications.deleted_at IS '软删除时间，NULL=未删除；置位即从常规列表隐藏并释放配额';

COMMENT ON TABLE tn_application_installations IS '安装记录：应用创建来源快照（一应用一条），供升级、问题定位与审计；追加写无更新/软删语义';
COMMENT ON COLUMN tn_application_installations.id IS '自增主键';
COMMENT ON COLUMN tn_application_installations.tenant_id IS '所属租户 ID（与应用一致）';
COMMENT ON COLUMN tn_application_installations.application_id IS '应用 ID，唯一，一个应用只有一条初始来源记录';
COMMENT ON COLUMN tn_application_installations.source_type IS '来源类型：blank / template';
COMMENT ON COLUMN tn_application_installations.template_id IS '模板 ID，空白应用为 NULL（M2-B 启用）';
COMMENT ON COLUMN tn_application_installations.template_version_id IS '模板版本 ID，空白应用为 NULL（M2-B 启用）';
COMMENT ON COLUMN tn_application_installations.channel IS '安装渠道：self 空白创建 / template_center 模板中心 / admin 运营 / api 开放接口';
COMMENT ON COLUMN tn_application_installations.blueprint_checksum IS '安装时实际使用的蓝图校验值，空白应用为 NULL（M2-B 启用）';
COMMENT ON COLUMN tn_application_installations.installed_by_member_id IS '发起安装的租户成员 ID';
COMMENT ON COLUMN tn_application_installations.installed_at IS '安装时间';

COMMENT ON TABLE tn_application_menu_entries IS '应用菜单节点（000016，M2-菜单）：分组/表单/仪表盘/页面的导航树（一资产一节点）；分组无 target，非分组节点 target_type=entry_type 且必须引用资产；租户/应用归属由服务层校验回填';
COMMENT ON COLUMN tn_application_menu_entries.id IS '自增主键';
COMMENT ON COLUMN tn_application_menu_entries.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_application_menu_entries.application_id IS '所属应用 ID（外键指向 tn_applications），同应用约束由服务层在加载校验';
COMMENT ON COLUMN tn_application_menu_entries.code IS '服务端生成的节点编码（menu_ 前缀），租户内唯一（uk_tn_application_menu_entries_tenant_code，软删行释放），出网即 entryId';
COMMENT ON COLUMN tn_application_menu_entries.parent_entry_id IS '父节点 ID，根节点为 NULL；父节点须同租户同应用且为 group（服务层校验，单列外键表达不了同应用约束）';
COMMENT ON COLUMN tn_application_menu_entries.entry_type IS '节点类型：group 分组 / form 表单 / dashboard 仪表盘 / page 页面';
COMMENT ON COLUMN tn_application_menu_entries.name IS '节点展示名';
COMMENT ON COLUMN tn_application_menu_entries.icon IS '稳定图标键（可空），不存前端组件名；前端受控映射表转换为图标组件';
COMMENT ON COLUMN tn_application_menu_entries.color IS '稳定颜色键（可空），不存 CSS 字面值';
COMMENT ON COLUMN tn_application_menu_entries.target_type IS '资产引用类型：group 为 NULL，非分组节点等于 entry_type（CHECK 约束）';
COMMENT ON COLUMN tn_application_menu_entries.target_id IS '资产域内部数字主键；出网时由资产查询投影为稳定公开编码，不直接暴露';
COMMENT ON COLUMN tn_application_menu_entries.sort_order IS '同父节点排序值，仅同父内有意义；新增 1024 间隔，服务端重排写连续间隔值，不信任客户端排序值';
COMMENT ON COLUMN tn_application_menu_entries.config IS '小型显示配置 JSONB（如页面打开方式）；严禁存放表单 Schema、流程定义、权限或前端组件名';
COMMENT ON COLUMN tn_application_menu_entries.hidden IS '对成员隐藏（000046，导航隐藏）：普通成员读取菜单时按不存在裁剪，持 tn_applications:create/patch 的菜单管理成员仍可见以便恢复；仅导航语义，不拦截 runtime 直连';
COMMENT ON COLUMN tn_application_menu_entries.created_at IS '创建时间';
COMMENT ON COLUMN tn_application_menu_entries.updated_at IS '更新时间';
COMMENT ON COLUMN tn_application_menu_entries.deleted_at IS '软删除时间，NULL=未删除；资产软删时同事务软删关联节点，应用软删后的节点由清理任务处理';

COMMENT ON TABLE tn_forms IS '表单资产（租户内从属于应用，ADR-010）：draft_content 为目标保存协议草稿全文（content.items 两层结构，保存前经字段字典严格校验）；草稿与发布快照分表，删除只写 deleted_at，发布版本行保留';
COMMENT ON COLUMN tn_forms.id IS '自增主键（菜单 target_id 引用值）';
COMMENT ON COLUMN tn_forms.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_forms.application_id IS '所属应用 ID（同租户，服务层归属校验，禁止裸 ID 写入）';
COMMENT ON COLUMN tn_forms.code IS '表单稳定公开编码（form_ 前缀）；路由、API 与菜单 target 使用，禁止暴露自增主键';
COMMENT ON COLUMN tn_forms.name IS '表单名称（trim 后 1–128 字符）；表单名称不进入协议 content';
COMMENT ON COLUMN tn_forms.form_type IS '表单类型：standard 标准表单 / workflow 流程表单；可经 form-actions:switch-type 切换（ADR-011），切换后原类型流程数据保留，设计器能力以此字段为准';
COMMENT ON COLUMN tn_forms.draft_content IS '目标保存协议草稿全文 JSONB：v4 子表单含嵌套字段、行权限、快速填报、冻结列及移动端展示配置';
COMMENT ON COLUMN tn_forms.draft_revision IS '草稿乐观锁口令：每次草稿保存条件递增，客户端原样回传，过期返回 FORM_REVISION_CONFLICT';
COMMENT ON COLUMN tn_forms.protocol_version IS '表单保存协议版本（文档内不携带版本字段）；v4 增加子表单权限与端侧展示配置';
COMMENT ON COLUMN tn_forms.latest_version_id IS '最新发布版本行 ID；NULL=从未发布';
COMMENT ON COLUMN tn_forms.published_version IS '最新发布号（冗余自最新快照 version_no，0=未发布），供列表展示免 JOIN';
COMMENT ON COLUMN tn_forms.creator_member_id IS '创建者（租户成员 ID），审计语义';
COMMENT ON COLUMN tn_forms.created_at IS '创建时间';
COMMENT ON COLUMN tn_forms.updated_at IS '更新时间';
COMMENT ON COLUMN tn_forms.deleted_at IS '软删除时间，NULL=未删除；置位即释放 tn_forms 配额，发布版本行保留';
COMMENT ON TABLE tn_form_versions IS '不可变发布快照（追加写，无更新/删除路径）：发布时草稿全文固化，记录提交按 (publishedVersion, schemaRevision) 双口令定位并依据本快照终审；禁止以任何写路径覆盖历史快照';
COMMENT ON COLUMN tn_form_versions.id IS '自增主键；即 schema_revision 口令的数值（出网字符串）';
COMMENT ON COLUMN tn_form_versions.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_form_versions.form_id IS '所属表单 ID';
COMMENT ON COLUMN tn_form_versions.version_no IS '表单内递增发布号 1,2,3…（即 publishedVersion）；与 form_id 联合唯一，并发发布兜底';
COMMENT ON COLUMN tn_form_versions.schema_revision IS '修订口令（= 行 id，发布事务内回填）；与 version_no 共同构成提交定位双因子';
COMMENT ON COLUMN tn_form_versions.content IS '发布时的目标保存协议全文快照 JSONB，写入后永不更新';
COMMENT ON COLUMN tn_form_versions.field_keys IS '顶层字段键有序数组 JSONB（widgetName 序列），提交未知键快速拒绝与后续记录索引使用';
COMMENT ON COLUMN tn_form_versions.protocol_version IS '不可变发布快照协议版本；历史 v1-v3 快照读取时迁移为当前结构';
COMMENT ON COLUMN tn_form_versions.published_by_member_id IS '发布人（租户成员 ID）';
COMMENT ON COLUMN tn_form_versions.published_at IS '发布时间';
COMMENT ON COLUMN tn_form_versions.created_at IS '创建时间';
COMMENT ON TABLE tn_form_records IS '表单记录提交（追加写）：values 为服务端按发布快照校验通过后的值（键=widgetName，仅快照内可见字段）；form_version_id 固定受理时所依据的发布版本（历史版本合法）';
COMMENT ON COLUMN tn_form_records.id IS '自增主键（记录 ID）';
COMMENT ON COLUMN tn_form_records.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_form_records.form_id IS '表单 ID';
COMMENT ON COLUMN tn_form_records.form_version_id IS '受理时依据的发布快照行 ID（任意历史版本均可受理，字段定义可复现）';
COMMENT ON COLUMN tn_form_records.values IS '字段值 JSONB（键=widgetName）；服务端终审通过的清洗值，隐藏字段与布局字段不落库';
COMMENT ON COLUMN tn_form_records.submitted_by_member_id IS '提交人（租户成员 ID）';
COMMENT ON COLUMN tn_form_records.submitted_at IS '提交时间';
COMMENT ON COLUMN tn_form_records.created_at IS '创建时间';


-- 000019: 账号会话体系与 MFA 数据层（ADR-009），平台级表
CREATE TABLE IF NOT EXISTS pf_account_security_settings (
    account_id BIGINT PRIMARY KEY REFERENCES pf_accounts(id) ON DELETE CASCADE,
    mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    single_session_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS pf_account_mfa_factors (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES pf_accounts(id) ON DELETE CASCADE,
    type varchar(16) NOT NULL,
    secret_ciphertext varchar(1024) NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    verified_at timestamp with time zone,
    last_used_counter BIGINT NOT NULL DEFAULT 0,
    disabled_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_pf_account_mfa_factors_type CHECK (type IN ('totp')),
    CONSTRAINT chk_pf_account_mfa_factors_key_version CHECK (key_version >= 1),
    CONSTRAINT chk_pf_account_mfa_factors_last_used_counter CHECK (last_used_counter >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pf_account_mfa_factors_active
    ON pf_account_mfa_factors (account_id, type)
    WHERE disabled_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_pf_account_mfa_factors_account
    ON pf_account_mfa_factors (account_id);

CREATE TABLE IF NOT EXISTS pf_account_mfa_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES pf_accounts(id) ON DELETE CASCADE,
    code_digest varchar(128) NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_pf_account_mfa_recovery_codes_available
    ON pf_account_mfa_recovery_codes (account_id)
    WHERE used_at IS NULL;

CREATE TABLE IF NOT EXISTS pf_account_sessions (
    id BIGSERIAL PRIMARY KEY,
    sid varchar(64) NOT NULL,
    account_id BIGINT NOT NULL REFERENCES pf_accounts(id) ON DELETE CASCADE,
    token_version BIGINT NOT NULL DEFAULT 1,
    auth_method varchar(16) NOT NULL,
    mfa_method varchar(16),
    created_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    revoke_reason varchar(32),
    ip varchar(45),
    location varchar(128),
    user_agent varchar(512),
    CONSTRAINT chk_pf_account_sessions_auth_method CHECK (auth_method IN ('password', 'sms', 'oauth', 'register')),
    CONSTRAINT chk_pf_account_sessions_mfa_method CHECK (mfa_method IS NULL OR mfa_method IN ('totp', 'recovery')),
    CONSTRAINT chk_pf_account_sessions_revoke_reason CHECK (revoke_reason IS NULL OR revoke_reason IN ('logout', 'replaced', 'password_changed', 'phone_changed', 'mfa_changed', 'admin_revoked')),
    CONSTRAINT chk_pf_account_sessions_token_version CHECK (token_version >= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pf_account_sessions_sid
    ON pf_account_sessions (sid);

CREATE INDEX IF NOT EXISTS idx_pf_account_sessions_account_active
    ON pf_account_sessions (account_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_pf_account_sessions_expires
    ON pf_account_sessions (expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS pf_account_security_events (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES pf_accounts(id) ON DELETE CASCADE,
    event_type varchar(32) NOT NULL,
    session_id varchar(64),
    request_id varchar(64),
    ip varchar(45),
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_pf_account_security_events_account
    ON pf_account_security_events (account_id, created_at);

-- 000020: 账号安全表数据字典注释
COMMENT ON TABLE pf_account_security_settings IS '账号安全开关：平台账号维度，一行一账号；缺行等价于全部关闭';
COMMENT ON COLUMN pf_account_security_settings.account_id IS '平台账号 ID，主键并外键指向 pf_accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN pf_account_security_settings.mfa_enabled IS '是否启用登录二次验证（TOTP MFA）';
COMMENT ON COLUMN pf_account_security_settings.single_session_enabled IS '是否禁止同时登录；开启后新登录会撤销其他活跃会话';
COMMENT ON COLUMN pf_account_security_settings.updated_at IS '安全设置最后更新时间';

COMMENT ON TABLE pf_account_mfa_factors IS '账号 MFA 因子：一期仅 TOTP；密钥仅密文保存，历史停用因子保留审计轨迹';
COMMENT ON COLUMN pf_account_mfa_factors.id IS 'MFA 因子主键';
COMMENT ON COLUMN pf_account_mfa_factors.account_id IS '所属平台账号 ID，外键指向 pf_accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN pf_account_mfa_factors.type IS '因子类型：一期固定为 totp';
COMMENT ON COLUMN pf_account_mfa_factors.secret_ciphertext IS 'TOTP 密钥的 AES-GCM 密文，绝不存储或出网明文';
COMMENT ON COLUMN pf_account_mfa_factors.key_version IS '加密主密钥版本，按该版本解密以支持密钥轮换';
COMMENT ON COLUMN pf_account_mfa_factors.verified_at IS '首次验证成功并启用时间，NULL 表示待确认';
COMMENT ON COLUMN pf_account_mfa_factors.last_used_counter IS '最近已消费的 TOTP 时间步计数器，用于阻止动态码重放';
COMMENT ON COLUMN pf_account_mfa_factors.disabled_at IS '停用时间，NULL 表示活跃；部分唯一索引仅约束活跃因子';
COMMENT ON COLUMN pf_account_mfa_factors.created_at IS '创建时间';
COMMENT ON COLUMN pf_account_mfa_factors.updated_at IS '更新时间';

COMMENT ON TABLE pf_account_mfa_recovery_codes IS 'MFA 恢复码：明文仅在生成时展示一次，数据库仅保存不可逆摘要';
COMMENT ON COLUMN pf_account_mfa_recovery_codes.id IS '恢复码记录主键';
COMMENT ON COLUMN pf_account_mfa_recovery_codes.account_id IS '所属平台账号 ID，外键指向 pf_accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN pf_account_mfa_recovery_codes.code_digest IS '恢复码 SHA-256 摘要，绝不保存明文';
COMMENT ON COLUMN pf_account_mfa_recovery_codes.used_at IS '消费时间，NULL 表示可用；更新受 used_at IS NULL 条件保护';
COMMENT ON COLUMN pf_account_mfa_recovery_codes.created_at IS '创建时间';

COMMENT ON TABLE pf_account_sessions IS '设备级账号会话：sid 写入 JWT，用于设备管理、单会话挤出和服务端撤销';
COMMENT ON COLUMN pf_account_sessions.id IS '设备会话记录主键';
COMMENT ON COLUMN pf_account_sessions.sid IS '随机设备会话公开标识，写入 JWT sid 声明，全局唯一';
COMMENT ON COLUMN pf_account_sessions.account_id IS '所属平台账号 ID，外键指向 pf_accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN pf_account_sessions.token_version IS '会话令牌版本；租户切换重签时递增，旧令牌随即失效';
COMMENT ON COLUMN pf_account_sessions.auth_method IS '登录第一步认证方式：password、sms、oauth 或 register';
COMMENT ON COLUMN pf_account_sessions.mfa_method IS '完成第二步的 MFA 方法：totp 或 recovery；未启用 MFA 时为 NULL';
COMMENT ON COLUMN pf_account_sessions.created_at IS '会话创建时间';
COMMENT ON COLUMN pf_account_sessions.last_seen_at IS '最近请求时间，由认证中间件节流刷新';
COMMENT ON COLUMN pf_account_sessions.expires_at IS '会话绝对过期时间，与 JWT 有效期对齐';
COMMENT ON COLUMN pf_account_sessions.revoked_at IS '撤销时间，NULL 表示未撤销';
COMMENT ON COLUMN pf_account_sessions.revoke_reason IS '撤销原因：logout、replaced、password_changed、phone_changed、mfa_changed 或 admin_revoked';
COMMENT ON COLUMN pf_account_sessions.ip IS '登录来源 IP，支持 IPv4 和 IPv6';
COMMENT ON COLUMN pf_account_sessions.location IS '登录地离线解析结果';
COMMENT ON COLUMN pf_account_sessions.user_agent IS '登录设备 User-Agent 原文';

COMMENT ON TABLE pf_account_security_events IS '账号安全事件流水：记录 MFA、恢复码和会话撤销等事件；不替代登录日志或业务审计';
COMMENT ON COLUMN pf_account_security_events.id IS '安全事件记录主键';
COMMENT ON COLUMN pf_account_security_events.account_id IS '所属平台账号 ID，外键指向 pf_accounts(id)，删除账号时级联删除';
COMMENT ON COLUMN pf_account_security_events.event_type IS '安全事件类型，如 mfa_enabled、session_revoked、mfa_recovery_used';
COMMENT ON COLUMN pf_account_security_events.session_id IS '关联设备会话 sid；无关联时为空字符串';
COMMENT ON COLUMN pf_account_security_events.request_id IS '关联请求追踪 ID；无关联时为空字符串';
COMMENT ON COLUMN pf_account_security_events.ip IS '操作来源 IP';
COMMENT ON COLUMN pf_account_security_events.metadata IS '非敏感扩展元数据 JSONB，禁止写入密钥、验证码、恢复码或令牌';
COMMENT ON COLUMN pf_account_security_events.created_at IS '创建时间';

-- 000030: 版本信息一期——套餐目录/套餐版本/租户订阅/特批覆盖 四表。
-- 活动订阅及其套餐版本是权益事实源；pf_tenants.plan/quotas 为过渡兼容投影。
CREATE TABLE IF NOT EXISTS pf_edition_plans (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    code varchar(32) NOT NULL,
    name varchar(64) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    kind varchar(16) NOT NULL DEFAULT 'base',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT uk_pf_edition_plans_code UNIQUE (code),
    CONSTRAINT chk_pf_edition_plans_status CHECK (status IN ('active', 'retired')),
    CONSTRAINT chk_pf_edition_plans_kind CHECK (kind IN ('base', 'addon'))
);

CREATE TABLE IF NOT EXISTS pf_edition_plan_versions (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    plan_id BIGINT NOT NULL REFERENCES pf_edition_plans(id),
    version INTEGER NOT NULL,
    display_name varchar(64) NOT NULL,
    billing_cycle varchar(16) NOT NULL DEFAULT 'none',
    compatibility_plan_code varchar(32) NOT NULL,
    entitlements JSONB NOT NULL,
    published_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    retired_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT uk_pf_edition_plan_versions_plan_version UNIQUE (plan_id, version),
    CONSTRAINT chk_pf_edition_plan_versions_cycle CHECK (billing_cycle IN ('none', 'monthly', 'yearly')),
    CONSTRAINT chk_pf_edition_plan_versions_compat CHECK (compatibility_plan_code IN ('free', 'trial', 'pro'))
);

CREATE TABLE IF NOT EXISTS tn_subscriptions (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    plan_version_id BIGINT NOT NULL REFERENCES pf_edition_plan_versions(id),
    status varchar(32) NOT NULL,
    grant_type varchar(16) NOT NULL,
    starts_at timestamp with time zone NOT NULL,
    ends_at timestamp with time zone,
    operator_account_id BIGINT,
    remark varchar(512) NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_tn_subscriptions_status CHECK (status IN ('active', 'expired', 'replaced', 'cancelled', 'legacy_pending_review')),
    CONSTRAINT chk_tn_subscriptions_grant_type CHECK (grant_type IN ('system', 'manual', 'self_service', 'trial')),
    CONSTRAINT chk_tn_subscriptions_ends CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_subscriptions_one_active
    ON tn_subscriptions (tenant_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_tn_subscriptions_tenant
    ON tn_subscriptions (tenant_id, status, starts_at DESC);

CREATE INDEX IF NOT EXISTS idx_tn_subscriptions_expiring
    ON tn_subscriptions (ends_at)
    WHERE status = 'active' AND ends_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS tn_entitlement_overrides (
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
    CONSTRAINT chk_tn_entitlement_overrides_source CHECK (source IN ('legacy', 'manual', 'trial')),
    CONSTRAINT chk_tn_entitlement_overrides_ends CHECK (ends_at IS NULL OR ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_tn_entitlement_overrides_tenant
    ON tn_entitlement_overrides (tenant_id, entitlement_key);

-- seed：三档基础套餐与 version=1 已发布快照（幂等；数值对齐 DefaultQuotas）
INSERT INTO pf_edition_plans (code, name, status, kind)
VALUES ('free', '免费版', 'active', 'base'),
       ('trial', '试用版', 'active', 'base'),
       ('pro', '专业版', 'active', 'base')
ON CONFLICT (code) DO NOTHING;

INSERT INTO pf_edition_plan_versions (plan_id, version, display_name, billing_cycle, compatibility_plan_code, entitlements, published_at)
SELECT p.id, 1, '免费版', 'none', 'free', $JSON${
  "sys_resources": [
    {"key": "apps", "category": "stock", "limit": 3, "unit": "count"},
    {"key": "members", "category": "stock", "limit": 5, "unit": "person"},
    {"key": "tn_forms", "category": "stock", "limit": 10, "unit": "count"},
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
FROM pf_edition_plans p
WHERE p.code = 'free'
ON CONFLICT (plan_id, version) DO NOTHING;

INSERT INTO pf_edition_plan_versions (plan_id, version, display_name, billing_cycle, compatibility_plan_code, entitlements, published_at)
SELECT p.id, 1, '试用版', 'none', 'trial', $JSON${
  "sys_resources": [
    {"key": "apps", "category": "stock", "limit": 10, "unit": "count"},
    {"key": "members", "category": "stock", "limit": 30, "unit": "person"},
    {"key": "tn_forms", "category": "stock", "limit": 50, "unit": "count"},
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
FROM pf_edition_plans p
WHERE p.code = 'trial'
ON CONFLICT (plan_id, version) DO NOTHING;

INSERT INTO pf_edition_plan_versions (plan_id, version, display_name, billing_cycle, compatibility_plan_code, entitlements, published_at)
SELECT p.id, 1, '专业版', 'none', 'pro', $JSON${
  "sys_resources": [
    {"key": "apps", "category": "stock", "limit": -1, "unit": "count"},
    {"key": "members", "category": "stock", "limit": -1, "unit": "person"},
    {"key": "tn_forms", "category": "stock", "limit": -1, "unit": "count"},
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
FROM pf_edition_plans p
WHERE p.code = 'pro'
ON CONFLICT (plan_id, version) DO NOTHING;

-- 存量租户订阅回填（幂等）：free/pro 长期兼容订阅；trial 待补录
INSERT INTO tn_subscriptions (tenant_id, plan_version_id, status, grant_type, starts_at, ends_at, remark)
SELECT t.id,
       pv.id,
       CASE WHEN t.plan = 'trial' THEN 'legacy_pending_review' ELSE 'active' END,
       CASE WHEN t.plan = 'trial' THEN 'trial' ELSE 'system' END,
       COALESCE(t.created_at, LOCALTIMESTAMP),
       NULL,
       CASE WHEN t.plan = 'trial'
            THEN '存量试用：历史无到期日记录，待运营补录后转为活动订阅'
            ELSE '' END
FROM pf_tenants t
JOIN pf_edition_plans p ON p.code = t.plan
JOIN pf_edition_plan_versions pv ON pv.plan_id = p.id AND pv.version = 1
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
  AND NOT EXISTS (SELECT 1 FROM tn_subscriptions ts WHERE ts.tenant_id = t.id);

-- 旧 quotas 覆盖迁移为 legacy 覆盖（幂等；storage_gb ×GiB 精确换算）
INSERT INTO tn_entitlement_overrides (tenant_id, entitlement_key, value, reason, source, starts_at, ends_at)
SELECT t.id,
       CASE WHEN e.key = 'storage_gb' THEN 'storage_bytes' ELSE e.key END,
       CASE WHEN e.key = 'storage_gb'
            THEN (e.value #>> '{}')::bigint * 1073741824
            ELSE (e.value #>> '{}')::bigint END,
       '存量租户配额覆盖迁移（legacy）',
       'legacy',
       LOCALTIMESTAMP,
       NULL
FROM pf_tenants t
CROSS JOIN LATERAL jsonb_each(COALESCE(t.quotas, '{}'::jsonb)) AS e(key, value)
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
  AND jsonb_typeof(e.value) = 'number'
  AND e.key IN ('apps', 'members', 'tn_forms', 'workflow_runs_month', 'storage_gb')
  AND NOT EXISTS (
      SELECT 1 FROM tn_entitlement_overrides o
      WHERE o.tenant_id = t.id
        AND o.entitlement_key = (CASE WHEN e.key = 'storage_gb' THEN 'storage_bytes' ELSE e.key END)
        AND o.source = 'legacy'
  );

-- 「租户管理员」基线角色补授 editions:get（幂等；存量数据库由迁移 000030 同步）
UPDATE tn_roles
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

-- 000030: 数据字典注释
COMMENT ON TABLE pf_edition_plans IS '套餐目录：稳定套餐编码与展示名，区分基础套餐与附加能力；不保存价格，价格属二期商品域';
COMMENT ON COLUMN pf_edition_plans.id IS '自增主键';
COMMENT ON COLUMN pf_edition_plans.code IS '稳定套餐编码（free/trial/pro），全局唯一，发布后不变更';
COMMENT ON COLUMN pf_edition_plans.name IS '套餐展示名称';
COMMENT ON COLUMN pf_edition_plans.status IS '目录状态：active 可授予 / retired 已下架（历史订阅仍可引用其版本）';
COMMENT ON COLUMN pf_edition_plans.kind IS '套餐类型：base 基础套餐 / addon 附加能力（二期附加包启用）';
COMMENT ON COLUMN pf_edition_plans.created_at IS '创建时间';
COMMENT ON COLUMN pf_edition_plans.updated_at IS '更新时间';

COMMENT ON TABLE pf_edition_plan_versions IS '套餐版本：不可变权益快照（sys_resources+features JSONB），已发布只能新增不能修改；一期仅经版本化迁移新增';
COMMENT ON COLUMN pf_edition_plan_versions.id IS '自增主键，人工授予与订阅引用的版本 ID';
COMMENT ON COLUMN pf_edition_plan_versions.plan_id IS '所属套餐，外键指向 pf_edition_plans(id)';
COMMENT ON COLUMN pf_edition_plan_versions.version IS '套餐内版本号，从 1 递增，(plan_id, version) 唯一';
COMMENT ON COLUMN pf_edition_plan_versions.display_name IS '版本展示名称（如「免费版」）';
COMMENT ON COLUMN pf_edition_plan_versions.billing_cycle IS '计费周期：none 不计费（一期）/ monthly / yearly（二期商品域启用）';
COMMENT ON COLUMN pf_edition_plan_versions.compatibility_plan_code IS '兼容投影目标：同步 pf_tenants.plan 时使用的旧套餐代码，仅允许 free/trial/pro';
COMMENT ON COLUMN pf_edition_plan_versions.entitlements IS '权益快照 JSONB：sys_resources[{key,category,limit,unit,resetCycle}] + features[{key,group,name,available,parameters}]；limit 语义 -1 不限量 / 0 不可用 / 正数上限，storage_bytes 一期只允许 -1/0/整 GiB';
COMMENT ON COLUMN pf_edition_plan_versions.published_at IS '发布时间，非空即已发布';
COMMENT ON COLUMN pf_edition_plan_versions.retired_at IS '下架时间，NULL 表示仍在授予；下架版本不可新授予但存量订阅继续有效';
COMMENT ON COLUMN pf_edition_plan_versions.created_at IS '创建时间';
COMMENT ON COLUMN pf_edition_plan_versions.updated_at IS '更新时间';

COMMENT ON TABLE tn_subscriptions IS '租户订阅：权益的事实源。同租户最多一条 active 基础订阅；到期由任务降级为免费订阅，页面与写路径在任务落库前按免费版解析（expiry_fallback）';
COMMENT ON COLUMN tn_subscriptions.id IS '自增主键';
COMMENT ON COLUMN tn_subscriptions.tenant_id IS '所属租户 ID（平台侧与 worker 经显式条件读写，不走租户 Callback）';
COMMENT ON COLUMN tn_subscriptions.plan_version_id IS '订阅的套餐版本快照，外键指向 pf_edition_plan_versions(id)';
COMMENT ON COLUMN tn_subscriptions.status IS '状态：active 活动 / expired 到期已降级 / replaced 被新订阅替换 / cancelled 人工取消 / legacy_pending_review 存量试用待补录';
COMMENT ON COLUMN tn_subscriptions.grant_type IS '授予方式：system 系统初始 / manual 平台运营人工 / self_service 用户自助（二期）/ trial 试用';
COMMENT ON COLUMN tn_subscriptions.starts_at IS '生效时间';
COMMENT ON COLUMN tn_subscriptions.ends_at IS '到期时间，NULL 表示长期有效；试用（grant_type=trial）必须非空，服务层校验';
COMMENT ON COLUMN tn_subscriptions.operator_account_id IS '平台人工操作的平台账号 ID，系统操作为 NULL';
COMMENT ON COLUMN tn_subscriptions.remark IS '运营备注，仅平台运营面可见，不对租户普通成员泄露';
COMMENT ON COLUMN tn_subscriptions.created_at IS '创建时间';
COMMENT ON COLUMN tn_subscriptions.updated_at IS '更新时间';

COMMENT ON TABLE tn_entitlement_overrides IS '租户特批权益覆盖：manual 运营特批 / trial 试用临时（与订阅同日到期）/ legacy 旧 quotas 迁移（只读）；降级时移除已到期或 trial 来源，保留仍有效的 manual';
COMMENT ON COLUMN tn_entitlement_overrides.id IS '自增主键';
COMMENT ON COLUMN tn_entitlement_overrides.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_entitlement_overrides.entitlement_key IS '权益资源键（新键空间，存储为 storage_bytes 字节）';
COMMENT ON COLUMN tn_entitlement_overrides.value IS '覆盖值：-1 不限量 / 0 不可用 / 正数上限；storage_bytes 须为整 GiB';
COMMENT ON COLUMN tn_entitlement_overrides.reason IS '覆盖原因（运营填写）';
COMMENT ON COLUMN tn_entitlement_overrides.source IS '来源：legacy 旧 quotas 迁移（只读）/ manual 运营特批 / trial 试用临时';
COMMENT ON COLUMN tn_entitlement_overrides.starts_at IS '覆盖生效时间';
COMMENT ON COLUMN tn_entitlement_overrides.ends_at IS '覆盖失效时间，NULL 表示长期；trial 来源必须与订阅同日到期';
COMMENT ON COLUMN tn_entitlement_overrides.operator_account_id IS '操作的平台账号 ID，legacy/system 为 NULL';
COMMENT ON COLUMN tn_entitlement_overrides.created_at IS '创建时间';
COMMENT ON COLUMN tn_entitlement_overrides.updated_at IS '更新时间';

-- 000031: 成员信息管理——租户字段显示策略与正式成员扩展档案
CREATE TABLE IF NOT EXISTS tn_member_field_settings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    field_key varchar(64) NOT NULL,
    personal_visible boolean NOT NULL DEFAULT true,
    personal_editable boolean NOT NULL DEFAULT false,
    card_visible boolean NOT NULL DEFAULT false,
    revision bigint NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_member_field_settings_tenant_field
    ON tn_member_field_settings (tenant_id, field_key) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tn_member_field_settings_tenant_updated
    ON tn_member_field_settings (tenant_id, updated_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tn_member_profiles (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    identifier varchar(50),
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_member_profiles_tenant_member
    ON tn_member_profiles (tenant_id, member_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_member_profiles_tenant_identifier
    ON tn_member_profiles (tenant_id, identifier) WHERE identifier <> '' AND deleted_at IS NULL;

-- 存量租户默认配置回填（幂等）：15 个预置字段与服务端注册表逐项一致
INSERT INTO tn_member_field_settings (tenant_id, field_key, personal_visible, personal_editable, card_visible, revision)
SELECT t.id, f.field_key, f.personal_visible, f.personal_editable, f.card_visible, 1
FROM pf_tenants t
CROSS JOIN (VALUES
    ('name',        true,  false, true),
    ('code',        false, false, true),
    ('mobile',      true,  true,  true),
    ('email',       true,  true,  true),
    ('department',  false, false, true),
    ('role',        false, false, false),
    ('alias',       false, false, false),
    ('employeeId',  false, false, false),
    ('gender',      false, false, false),
    ('position',    false, false, false),
    ('employment',  false, false, false),
    ('hireDate',    false, false, false),
    ('workplace',   false, false, false),
    ('birthday',    false, false, false),
    ('education',   false, false, false)
) AS f(field_key, personal_visible, personal_editable, card_visible)
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
ON CONFLICT DO NOTHING;

-- 租户管理员基线角色补授 member-field-settings（幂等）
UPDATE tn_roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'member-field-settings', 'operation', '*'))
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'member-field-settings'
  );

-- 000031: 数据字典注释
COMMENT ON TABLE tn_member_field_settings IS '租户成员字段显示策略：字段设置与卡片展示页签的租户级配置，每租户每字段一行';
COMMENT ON COLUMN tn_member_field_settings.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_member_field_settings.field_key IS '预置字段 key（服务端字段注册表校验合法性，如 mobile/hireDate）';
COMMENT ON COLUMN tn_member_field_settings.personal_visible IS '成员在个人设置页可见该字段';
COMMENT ON COLUMN tn_member_field_settings.personal_editable IS '成员在个人设置页可编辑该字段（仅对扩展字段生效，手机/邮箱走绑定流程）';
COMMENT ON COLUMN tn_member_field_settings.card_visible IS '成员资料卡片展示该字段（服务端按此裁剪卡片数据）';
COMMENT ON COLUMN tn_member_field_settings.revision IS '租户配置快照版本号：同租户所有行同步递增，PATCH 以整页 revision 做乐观锁';
COMMENT ON COLUMN tn_member_field_settings.created_at IS '创建时间';
COMMENT ON COLUMN tn_member_field_settings.updated_at IS '最后更新时间';
COMMENT ON COLUMN tn_member_field_settings.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE tn_member_profiles IS '正式成员扩展档案：别名/工号/性别/职务/日期等租户内资料，邀请接受时从邀请草稿迁入；不重复存储手机号、邮箱、部门和角色';
COMMENT ON COLUMN tn_member_profiles.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_member_profiles.member_id IS '对应租户成员 tn_users.id，每成员一份有效档案';
COMMENT ON COLUMN tn_member_profiles.identifier IS '企业内编号（编号字段 code 的数据来源），租户内有效记录唯一，空值不参与唯一';
COMMENT ON COLUMN tn_member_profiles.attributes IS '扩展字段 JSONB：alias/employeeId/gender/position/employment/hireDate/workplace/birthday/education，日期统一 YYYY-MM-DD、文本最长 50 字符（服务层校验）';
COMMENT ON COLUMN tn_member_profiles.created_at IS '创建时间';
COMMENT ON COLUMN tn_member_profiles.updated_at IS '最后更新时间';
COMMENT ON COLUMN tn_member_profiles.deleted_at IS '软删除时间，NULL 表示有效';

-- 000032: 权限中心-管理员模块（管理组）
CREATE TABLE IF NOT EXISTS tn_admin_groups (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name varchar(30) NOT NULL,
    scope varchar(16) NOT NULL,
    built_in boolean NOT NULL DEFAULT false,
    scope_config jsonb NOT NULL DEFAULT '{}'::jsonb,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_admin_groups_tenant_name
    ON tn_admin_groups (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tn_admin_groups_tenant_scope
    ON tn_admin_groups (tenant_id, scope) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tn_admin_group_members (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    admin_group_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT uk_tn_admin_group_member UNIQUE (admin_group_id, member_id)
);
CREATE INDEX IF NOT EXISTS idx_tn_admin_group_members_member
    ON tn_admin_group_members (tenant_id, member_id);

-- 存量租户回填内置系统管理员组（幂等）：成员不落表，由 tenant-admin 角色绑定实时推导
INSERT INTO tn_admin_groups (tenant_id, name, scope, built_in, scope_config, created_at, updated_at)
SELECT t.id, '系统管理员', 'system', true, '{}'::jsonb, now(), now()
FROM pf_tenants t
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
ON CONFLICT DO NOTHING;

-- 租户管理员基线角色补授 admin-tn_groups（幂等）
UPDATE tn_roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'admin-tn_groups', 'operation', '*'))
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'admin-tn_groups'
  );

-- 000032: 数据字典注释
COMMENT ON TABLE tn_admin_groups IS '租户管理组（权限中心-管理员模块）：一组成员 + 对部门/角色/应用/互联组织的带范围委托管理权；内置组（built_in）成员由 tenant-admin 角色绑定推导';
COMMENT ON COLUMN tn_admin_groups.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_admin_groups.name IS '管理组名称，租户内有效记录唯一，最长 30 字符（服务层校验）';
COMMENT ON COLUMN tn_admin_groups.scope IS '管理组类型：system=系统管理员页（通讯录管理组），application=灵衍云管理员页（普通管理组）';
COMMENT ON COLUMN tn_admin_groups.built_in IS '是否内置组：true 为系统管理员组，不可改名/删除/改配置，成员读写代理到 tenant-admin 角色绑定';
COMMENT ON COLUMN tn_admin_groups.scope_config IS '范围配置 JSONB：department/role/externalOrg/application/addressBook 区块，按 scope 适用性由服务层校验；ID 清单悬挂引用由读取侧丢弃';
COMMENT ON COLUMN tn_admin_groups.created_at IS '创建时间';
COMMENT ON COLUMN tn_admin_groups.updated_at IS '最后更新时间';
COMMENT ON COLUMN tn_admin_groups.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE tn_admin_group_members IS '管理组成员绑定（自定义组）：移除即删行，变更流水走审计域；内置系统管理员组不使用本表';
COMMENT ON COLUMN tn_admin_group_members.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_admin_group_members.admin_group_id IS '管理组 tn_admin_groups.id（同租户）';
COMMENT ON COLUMN tn_admin_group_members.member_id IS '租户成员 tn_users.id（同租户，服务层校验）';
COMMENT ON COLUMN tn_admin_group_members.created_at IS '加入时间';

-- 000033: 产品中心一期（平台产品目录/租户产品配置/范围关联 + lingyanyun seed + 存量回填）
CREATE TABLE IF NOT EXISTS pf_product_catalogs (
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
    CONSTRAINT chk_pf_product_catalogs_status CHECK (status IN ('active', 'inactive'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pf_product_catalogs_code
    ON pf_product_catalogs (code) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tn_product_configs (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL REFERENCES pf_product_catalogs(id),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    scope_mode varchar(16) NOT NULL DEFAULT 'all',
    revision BIGINT NOT NULL DEFAULT 1,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT ck_tn_product_configs_scope_mode CHECK (scope_mode IN ('all', 'partial'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_product_configs_tenant_product_active
    ON tn_product_configs (tenant_id, product_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_tn_product_configs_tenant
    ON tn_product_configs (tenant_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tn_product_departments (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_product_config_id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL,
    department_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT uk_tn_product_departments UNIQUE (tenant_product_config_id, department_id)
);

CREATE INDEX IF NOT EXISTS idx_tn_product_departments_tenant
    ON tn_product_departments (tenant_id);

CREATE TABLE IF NOT EXISTS tn_product_members (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_product_config_id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT uk_tn_product_members UNIQUE (tenant_product_config_id, member_id)
);

CREATE INDEX IF NOT EXISTS idx_tn_product_members_tenant
    ON tn_product_members (tenant_id);

-- 内置产品目录 seed（幂等）
INSERT INTO pf_product_catalogs (code, name, icon, entry_path, status, sort_order)
VALUES ('lingyanyun', '灵衍云', 'product', '/workspace', 'active', 100)
ON CONFLICT DO NOTHING;

-- 存量租户配置回填（幂等）：active 目录默认启用、范围 all、不建关联行
INSERT INTO tn_product_configs (tenant_id, product_id, enabled, scope_mode, revision)
SELECT t.id, p.id, TRUE, 'all', 1
FROM pf_tenants t
JOIN pf_product_catalogs p ON p.status = 'active'
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
  AND NOT EXISTS (
      SELECT 1 FROM tn_product_configs c
      WHERE c.tenant_id = t.id AND c.product_id = p.id
  )
ON CONFLICT DO NOTHING;

-- 租户管理员基线角色补授 tenant-products（幂等）：view=get+list，update 覆盖两条 PUT 子资源路径
UPDATE tn_roles
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

-- 000033: 数据字典注释
COMMENT ON TABLE pf_product_catalogs IS '平台内置产品目录：平台提供、可被多个租户启用的产品（如灵衍云）；不是租户自建的 tn_applications 应用';
COMMENT ON COLUMN pf_product_catalogs.id IS '自增主键';
COMMENT ON COLUMN pf_product_catalogs.code IS '产品稳定机器码（如 lingyanyun），创建后不可修改，租户侧接口以 code 定位产品';
COMMENT ON COLUMN pf_product_catalogs.name IS '产品展示名称';
COMMENT ON COLUMN pf_product_catalogs.icon IS '稳定图标键（前端按键映射图标组件），不存前端组件名';
COMMENT ON COLUMN pf_product_catalogs.entry_path IS '站内产品入口路径；外部跳转地址能力后续单独设计';
COMMENT ON COLUMN pf_product_catalogs.status IS '目录状态：active 可用 / inactive 平台已停用（停用后所有租户不可访问）';
COMMENT ON COLUMN pf_product_catalogs.sort_order IS '产品中心卡片展示排序，升序';
COMMENT ON COLUMN pf_product_catalogs.created_at IS '创建时间';
COMMENT ON COLUMN pf_product_catalogs.updated_at IS '更新时间';
COMMENT ON COLUMN pf_product_catalogs.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE tn_product_configs IS '租户产品配置：某产品在一个租户中的启用状态与分发范围；每租户每产品一条有效记录（部分唯一索引保证）';
COMMENT ON COLUMN tn_product_configs.id IS '自增主键';
COMMENT ON COLUMN tn_product_configs.tenant_id IS '所属租户 ID（服务层显式条件定位，不依赖请求租户上下文）';
COMMENT ON COLUMN tn_product_configs.product_id IS '平台产品目录 pf_product_catalogs.id';
COMMENT ON COLUMN tn_product_configs.enabled IS '租户是否启用该产品：false 时保留范围配置，重新启用后恢复原范围';
COMMENT ON COLUMN tn_product_configs.scope_mode IS '可用范围模式：all 全部有效成员 / partial 仅选中部门（含子部门）与成员；CHECK 约束保证取值';
COMMENT ON COLUMN tn_product_configs.revision IS '配置乐观锁版本：每次成功写入（启停/范围替换）递增，客户端提交时携带读取到的版本号';
COMMENT ON COLUMN tn_product_configs.created_at IS '创建时间';
COMMENT ON COLUMN tn_product_configs.updated_at IS '最后更新时间';
COMMENT ON COLUMN tn_product_configs.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE tn_product_departments IS '产品可用范围-部门关联：仅 scope_mode=partial 时有记录；全量替换（先删后插）无软删；tenant_id 用于租户归属校验与查询';
COMMENT ON COLUMN tn_product_departments.id IS '自增主键';
COMMENT ON COLUMN tn_product_departments.tenant_product_config_id IS '租户产品配置 tn_product_configs.id（同租户）';
COMMENT ON COLUMN tn_product_departments.tenant_id IS '所属租户 ID（冗余存储，服务层校验部门同租户后写入）';
COMMENT ON COLUMN tn_product_departments.department_id IS '租户部门 tn_departments.id；选中部门的子部门经读时递归展开命中，不在此复制子部门 ID';
COMMENT ON COLUMN tn_product_departments.created_at IS '创建时间';

COMMENT ON TABLE tn_product_members IS '产品可用范围-成员关联：仅 scope_mode=partial 时有记录；全量替换（先删后插）无软删；成员离职/禁用后读取与访问判定均忽略';
COMMENT ON COLUMN tn_product_members.id IS '自增主键';
COMMENT ON COLUMN tn_product_members.tenant_product_config_id IS '租户产品配置 tn_product_configs.id（同租户）';
COMMENT ON COLUMN tn_product_members.tenant_id IS '所属租户 ID（冗余存储，服务层校验成员同租户且 active 后写入）';
COMMENT ON COLUMN tn_product_members.member_id IS '租户成员 tn_users.id，写入时必须为同租户 active 成员';
COMMENT ON COLUMN tn_product_members.created_at IS '创建时间';

-- 000034: 产品中心权限补授兜底（角色名可改，按管理员规则签名补授：members:* + tn_roles:* + tn_departments:*）
UPDATE tn_roles
SET rules = (
    rules::jsonb
    || jsonb_build_array(
        jsonb_build_object('resource', 'tenant-products', 'operation', 'view'),
        jsonb_build_object('resource', 'tenant-products', 'operation', 'update')
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
      WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'tenant-products'
  );

-- 000035: 基线管理员权限补授兜底（editions/member-field-settings/admin-tn_groups，
-- 与 000034 同口径：角色名可改，按管理员规则签名补授）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'editions', 'operation', 'get'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'editions');

UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'member-field-settings', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'member-field-settings');

UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'admin-tn_groups', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'admin-tn_groups');

-- 企业日志（000036）：基线管理员按签名补授 enterprise-logs 资源
--（view 覆盖查询、create 覆盖导出任务创建，与迁移 000036 同口径）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'enterprise-logs', 'operation', 'view'),
        jsonb_build_object('resource', 'enterprise-logs', 'operation', 'create')
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'enterprise-logs');

-- 产品日志（000064）：基线管理员按签名补授 product-logs 资源
--（view 覆盖查询、create 覆盖导出任务创建，与迁移 000064 同口径）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'product-logs', 'operation', 'view'),
        jsonb_build_object('resource', 'product-logs', 'operation', 'create')
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'product-logs');

-- 表单资产（000037，ADR-010）：基线管理员按签名补授 tn_forms 资源（全量管理；
-- 发布复用 create 动词，一期不拆独立发布权）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'tn_forms', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_forms');

-- 表单记录提交（000038）：全体成员（authenticated 基线按系统角色名补授）
-- 授 form-records:create；填写提交与表单设计权限（tn_forms）彻底分离
UPDATE tn_roles
SET rules = (rules::jsonb || '[{"resource": "form-records", "operation": "create"}]'::jsonb)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'form-records'
  );

-- 消息中心（000039）：基线管理员按规则签名补授 notification-settings 资源
--（口径同 000035/000037，与可改名的角色名无关）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'notification-settings', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'notification-settings');

-- 消息中心（000039）：全体成员读写自己的收件箱（view 覆盖摘要/列表，
-- update 覆盖单条/批量已读；数据范围由 Repository 双租户条件兜底）
UPDATE tn_roles
SET rules = (
    rules::jsonb
    || '[{"resource": "notifications", "operation": "view"}, {"resource": "notifications", "operation": "update"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'notifications'
  );


-- 表单菜单按钮动作（000047，ADR-011）：基线管理员按规则签名补授
-- form-actions 资源（切换类型/复制/隐藏的动作授权键；不对应 URL 首段，
-- 动作键仅由各域 Service 复核与菜单读取投影消费）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'form-actions', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'form-actions');

-- 菜单节点个人收藏（000047，ADR-011）：全体成员（authenticated 系统分组
-- 关联角色）授 menu-favorites create/delete——凡节点可见即可收藏
UPDATE tn_roles
SET rules = (
    rules::jsonb
    || '[{"resource": "menu-favorites", "operation": "create"}, {"resource": "menu-favorites", "operation": "delete"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'menu-favorites'
  );


-- ============================================================
-- 000048: 流程引擎 Phase 1 Definition Engine（ADR-012）
-- ============================================================

-- 流程定义（租户级资产）：draft_content 为 Workflow DSL v1 全文 JSONB 单文档
-- 事实源（不建 wf_node/wf_edge 表）；发布快照另存 wf_definition_version；
-- 软删仅允许无运行中实例时删除，发布版本行与运行态历史保留
CREATE TABLE IF NOT EXISTS wf_definition (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    code varchar(64) NOT NULL,
    name varchar(128) NOT NULL,
    description varchar(512) NOT NULL DEFAULT '',
    form_code varchar(64) NOT NULL DEFAULT '',
    draft_content JSONB NOT NULL,
    draft_revision BIGINT NOT NULL DEFAULT 1,
    latest_version_id BIGINT,
    published_version INTEGER NOT NULL DEFAULT 0,
    creator_member_id BIGINT NOT NULL,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_wf_definition_content_object CHECK (jsonb_typeof(draft_content) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_wf_definition_tenant
    ON wf_definition (tenant_id, id DESC)
    WHERE deleted_at IS NULL;

-- 稳定公开编码（wf_ 前缀）：路由/API/菜单 target 一律用 code，内部自增 ID 不出网
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_definition_tenant_code
    ON wf_definition (tenant_id, code)
    WHERE deleted_at IS NULL;

-- 表单绑定（000060）：流程型表单的流程设计页定位口令，一条表单租户内至多
-- 绑定一条未删除定义；空串=独立定义
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_definition_form_code
    ON wf_definition (tenant_id, form_code)
    WHERE deleted_at IS NULL AND form_code <> '';

CREATE INDEX IF NOT EXISTS idx_wf_definition_form_code
    ON wf_definition (tenant_id, form_code)
    WHERE deleted_at IS NULL AND form_code <> '';

-- 不可变发布快照（追加写）：发布时 DSL 全文整体冻结，发布前必须通过严格校验
-- 与 Expr 预编译；(definition_id, version_no) 唯一保证并发发布不重号
CREATE TABLE IF NOT EXISTS wf_definition_version (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    definition_id BIGINT NOT NULL REFERENCES wf_definition(id),
    version_no INTEGER NOT NULL,
    dsl_snapshot JSONB NOT NULL,
    published_by_member_id BIGINT NOT NULL,
    published_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    created_at timestamp with time zone,
    CONSTRAINT chk_wf_definition_version_snapshot_object CHECK (jsonb_typeof(dsl_snapshot) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_definition_version_no
    ON wf_definition_version (definition_id, version_no);

CREATE INDEX IF NOT EXISTS idx_wf_definition_version_tenant
    ON wf_definition_version (tenant_id, id);

-- 基线管理员权限补授（口径同 000035/000037「管理员规则签名」，与角色名无关）：
-- workflows 资源全量管理；发布复用 workflows:create 动词（POST URL 鉴权解析）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'workflows', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'workflows'
  );

COMMENT ON TABLE wf_definition IS '流程定义（ADR-012，租户级资产）：draft_content 为 Workflow DSL v1 全文 JSONB（schemaVersion/nodes/edges/settings 单文档事实源，不建 wf_node/wf_edge 表）；发布快照另存 wf_definition_version，删除只写 deleted_at 且仅允许无运行中实例时删除（Phase 2 起由服务层复核），发布版本行与运行态历史保留';
COMMENT ON COLUMN wf_definition.id IS '自增主键（内部标识，不出网；对外一律用 code）';
COMMENT ON COLUMN wf_definition.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_definition.code IS '稳定公开编码（wf_ 前缀 + 16 位随机 hex）：路由/API/菜单 target 使用；租户内未删除行唯一';
COMMENT ON COLUMN wf_definition.name IS '流程名称（trim 后 1–128 字符）；不进入 DSL 文档';
COMMENT ON COLUMN wf_definition.description IS '流程描述（≤512 字符）；不进入 DSL 文档';
COMMENT ON COLUMN wf_definition.form_code IS '绑定的表单公开编码（form_ 前缀，tn_forms.code）：流程型表单的工作区流程设计页定位口令，一条表单租户内至多绑定一条未删除定义；空串=独立定义（不属于任何表单）';
COMMENT ON COLUMN wf_definition.draft_content IS 'Workflow DSL v1 草稿全文 JSONB：{schemaVersion:"1.0",nodes:[],edges:[],settings:{}}，原样字节存取；保存与发布前经引擎严格校验器校验';
COMMENT ON COLUMN wf_definition.draft_revision IS '草稿乐观锁口令：每次草稿保存条件递增，客户端原样回传，过期返回 WORKFLOW_REVISION_CONFLICT';
COMMENT ON COLUMN wf_definition.latest_version_id IS '最新发布版本行 ID；NULL=从未发布';
COMMENT ON COLUMN wf_definition.published_version IS '最新发布号（冗余自最新快照 version_no，0=未发布），供列表展示免 JOIN';
COMMENT ON COLUMN wf_definition.creator_member_id IS '创建者（租户成员 ID），审计语义';
COMMENT ON COLUMN wf_definition.created_at IS '创建时间';
COMMENT ON COLUMN wf_definition.updated_at IS '更新时间';
COMMENT ON COLUMN wf_definition.deleted_at IS '软删除时间，NULL=未删除；发布版本行保留';

COMMENT ON TABLE wf_definition_version IS '不可变发布快照（追加写，无更新/删除路径）：发布时 DSL 全文整体冻结（Node/Edge/Config 内嵌其中，运行时 Navigator 按 node key 从本快照读取配置）；发布前必须通过 DSL 严格校验与 Expr 预编译，禁止以任何写路径覆盖历史快照';
COMMENT ON COLUMN wf_definition_version.id IS '自增主键';
COMMENT ON COLUMN wf_definition_version.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_definition_version.definition_id IS '所属流程定义 ID（内部外键，不出网）';
COMMENT ON COLUMN wf_definition_version.version_no IS '定义内递增发布号 1,2,3…；与 definition_id 联合唯一，并发发布兜底；运行实例以 (code, version_no) 定位版本';
COMMENT ON COLUMN wf_definition_version.dsl_snapshot IS '发布时的 Workflow DSL v1 全文快照 JSONB，写入后永不更新；不可变是「运行实例永不自动升级」的物质基础';
COMMENT ON COLUMN wf_definition_version.published_by_member_id IS '发布人（租户成员 ID）';
COMMENT ON COLUMN wf_definition_version.published_at IS '发布时间';
COMMENT ON COLUMN wf_definition_version.created_at IS '创建时间';


-- ============================================================
-- 000049: 流程引擎 Phase 2 最小 Runtime（ADR-012）
-- ============================================================

-- 流程实例：发起时一次性冻结绑定流程版本与表单版本；运行实例幂等走部分唯一索引
CREATE TABLE IF NOT EXISTS wf_instance (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    definition_id BIGINT NOT NULL,
    definition_version_id BIGINT NOT NULL,
    business_type varchar(64) NOT NULL,
    business_id varchar(64) NOT NULL,
    app_id BIGINT NOT NULL DEFAULT 0,
    form_id BIGINT NOT NULL DEFAULT 0,
    form_version_id BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'RUNNING',
    starter_member_id BIGINT NOT NULL,
    idempotency_key varchar(64),
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_instance_status CHECK (status IN ('DRAFT', 'RUNNING', 'COMPLETED', 'REJECTED', 'CANCELLED'))
);

-- 业务幂等：同一 tenant+type+id 同一时间至多一个 RUNNING 实例
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_instance_running_business
    ON wf_instance (tenant_id, business_type, business_id)
    WHERE status = 'RUNNING';

-- 请求幂等：同租户幂等键非空唯一，重发返回同一实例
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_instance_idempotency_key
    ON wf_instance (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wf_instance_business
    ON wf_instance (tenant_id, business_type, business_id, id DESC);

CREATE INDEX IF NOT EXISTS idx_wf_instance_tenant
    ON wf_instance (tenant_id, id DESC);

-- 执行路径（Phase 8 并行执行树：根路径 + split 扇出的子分支路径）
CREATE TABLE IF NOT EXISTS wf_execution (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    parent_execution_id BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'RUNNING',
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_execution_status CHECK (status IN ('RUNNING', 'COMPLETED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_wf_execution_instance
    ON wf_execution (instance_id, id);

-- 节点实例：设计态 Node 的一次实际运行；配置按 node_key 从发布快照读取
CREATE TABLE IF NOT EXISTS wf_node_instance (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    execution_id BIGINT NOT NULL,
    node_key varchar(64) NOT NULL,
    status varchar(24) NOT NULL DEFAULT 'PENDING',
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_node_instance_status CHECK (status IN
        ('PENDING', 'RUNNING', 'WAITING', 'WAITING_RESUBMIT', 'COMPLETED', 'REJECTED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_wf_node_instance_instance
    ON wf_node_instance (instance_id, id);

-- 人工任务：PENDING 唯一可执行动作状态；转办另建新任务
CREATE TABLE IF NOT EXISTS wf_task (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    node_instance_id BIGINT NOT NULL,
    node_key varchar(64) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'PENDING',
    transferred_from_task_id BIGINT NOT NULL DEFAULT 0,
    transferred_to_member_id BIGINT NOT NULL DEFAULT 0,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_task_status CHECK (status IN
        ('PENDING', 'APPROVED', 'REJECTED', 'TRANSFERRED', 'CANCELLED', 'EXPIRED'))
);

CREATE INDEX IF NOT EXISTS idx_wf_task_instance
    ON wf_task (instance_id, id);

CREATE INDEX IF NOT EXISTS idx_wf_task_node_instance
    ON wf_task (node_instance_id, status);

CREATE INDEX IF NOT EXISTS idx_wf_task_status
    ON wf_task (tenant_id, status, id DESC);

-- 任务参与人快照（任务创建时一次性解析落库，不随组织变化重算）
CREATE TABLE IF NOT EXISTS wf_task_actor (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    task_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    display_name varchar(100) NOT NULL DEFAULT '',
    actor_role varchar(16) NOT NULL DEFAULT 'assignee',
    creator_id BIGINT,
    updater_id BIGINT,
    created_at timestamp with time zone,
    CONSTRAINT chk_wf_task_actor_role CHECK (actor_role IN ('assignee', 'cc'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_task_actor
    ON wf_task_actor (task_id, member_id);

CREATE INDEX IF NOT EXISTS idx_wf_task_actor_member
    ON wf_task_actor (tenant_id, member_id, id DESC);

-- 操作流水（追加写，禁止更新）：审批时间线唯一事实源
CREATE TABLE IF NOT EXISTS wf_operation (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL DEFAULT 0,
    operator_member_id BIGINT NOT NULL DEFAULT 0,
    operation_type varchar(32) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_wf_operation_instance
    ON wf_operation (instance_id, id);

-- 抄送记录（000051，追加写）：CC 不是审批任务，不参与节点完成判定；
-- 「抄送我的」高频查询落独立记录表（10.6 章查询模型最简原则）
CREATE TABLE IF NOT EXISTS wf_cc_record (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    node_instance_id BIGINT NOT NULL DEFAULT 0,
    node_key varchar(64) NOT NULL DEFAULT '',
    member_id BIGINT NOT NULL,
    display_name varchar(100) NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    CONSTRAINT fk_wf_cc_record_instance FOREIGN KEY (instance_id) REFERENCES wf_instance(id)
);

CREATE INDEX IF NOT EXISTS idx_wf_cc_record_member
    ON wf_cc_record (tenant_id, member_id, id DESC);

CREATE INDEX IF NOT EXISTS idx_wf_cc_record_instance
    ON wf_cc_record (instance_id, id);

-- 延时任务（000052）：超时自动动作/待办提醒；Worker 单事务内领取
--（FOR UPDATE SKIP LOCKED）→ 执行 → 回写，claim+执行同事务 crash 自动回滚
CREATE TABLE IF NOT EXISTS wf_job (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    job_type varchar(32) NOT NULL,
    instance_id BIGINT NOT NULL DEFAULT 0,
    node_instance_id BIGINT NOT NULL DEFAULT 0,
    task_id BIGINT NOT NULL DEFAULT 0,
    execute_at timestamp with time zone NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    max_retry_count INT NOT NULL DEFAULT 3,
    payload JSONB NOT NULL DEFAULT '{}',
    last_error text NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_job_type CHECK (job_type IN ('task.reminder', 'task.timeout', 'service.invoke')),
    CONSTRAINT chk_wf_job_status CHECK (status IN
        ('PENDING', 'PROCESSING', 'SUCCEEDED', 'FAILED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_wf_job_claim
    ON wf_job (status, execute_at);

CREATE INDEX IF NOT EXISTS idx_wf_job_task_pending
    ON wf_job (task_id) WHERE status IN ('PENDING', 'PROCESSING');

CREATE INDEX IF NOT EXISTS idx_wf_job_instance
    ON wf_job (instance_id, id);

COMMENT ON TABLE wf_job IS '流程延时任务（追加语义 + 有限重试）：任务创建时按节点配置排期 task.timeout/task.reminder，Worker 单事务内领取（FOR UPDATE SKIP LOCKED）→ 执行 → 回写结果，claim+执行同事务故 crash 自动回滚为 PENDING（第 19 章）；超时自动动作必须经 Task Engine 正常执行路径，Worker 不得直改流程状态';
COMMENT ON COLUMN wf_job.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_job.job_type IS '任务类型：task.reminder=待办提醒 / task.timeout=待办超时自动动作 / service.invoke=服务节点异步 HTTP 调用（000053 扩展）';
COMMENT ON COLUMN wf_job.instance_id IS '关联流程实例 ID（0=未关联）';
COMMENT ON COLUMN wf_job.node_instance_id IS '关联节点实例 ID（0=未关联）';
COMMENT ON COLUMN wf_job.task_id IS '关联人工任务 ID（0=未关联；任务终态时在途 Job 联动取消）';
COMMENT ON COLUMN wf_job.execute_at IS '计划执行时间（Worker 轮询窗口）';
COMMENT ON COLUMN wf_job.status IS 'Job 状态机：PENDING/PROCESSING/SUCCEEDED/FAILED/CANCELLED（迁移表 task 包唯一裁决）';
COMMENT ON COLUMN wf_job.retry_count IS '已重试次数（执行失败重试回队时递增）';
COMMENT ON COLUMN wf_job.max_retry_count IS '重试上限（超过即 FAILED 终态）';
COMMENT ON COLUMN wf_job.payload IS '任务载荷 JSONB（超时动作参数/提醒上下文等）';
COMMENT ON COLUMN wf_job.last_error IS '最近一次失败摘要（只入日志/诊断，不出网）';
COMMENT ON COLUMN wf_job.created_at IS '创建时间';
COMMENT ON COLUMN wf_job.updated_at IS '更新时间（领取/回写/重试回队）';

-- 000053：流程变量表（流程引擎 Phase 7，ADR-012）
-- service 节点响应映射写流程变量（表达式 variables.* 数据源），后续节点
-- 条件/审批人/模板经 variables.* 读取；V1 冻结标量值域。
CREATE TABLE IF NOT EXISTS wf_variable (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    var_key varchar(64) NOT NULL,
    var_type varchar(16) NOT NULL,
    var_value JSONB NOT NULL DEFAULT 'null',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_variable_type CHECK (var_type IN ('string', 'number', 'boolean')),
    CONSTRAINT uq_wf_variable_instance_key UNIQUE (instance_id, var_key)
);

-- 实例维度变量加载（推进上下文构造）
CREATE INDEX IF NOT EXISTS idx_wf_variable_instance
    ON wf_variable (tenant_id, instance_id);

COMMENT ON TABLE wf_variable IS '流程变量（实例作用域，(instance_id,var_key) 唯一）：service 节点响应映射等写入，表达式经 variables.* 白名单数据源读取；V1 冻结标量值域（string/number/boolean）';
COMMENT ON COLUMN wf_variable.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_variable.instance_id IS '归属流程实例 ID';
COMMENT ON COLUMN wf_variable.var_key IS '变量名（字母开头标识符，实例内唯一，校验器冻结命名规则）';
COMMENT ON COLUMN wf_variable.var_type IS '值类型（V1 冻结标量集合）：string/number/boolean';
COMMENT ON COLUMN wf_variable.var_value IS '变量值 JSONB（标量；提取失败且非 required 时不落行）';
COMMENT ON COLUMN wf_variable.created_at IS '创建时间';
COMMENT ON COLUMN wf_variable.updated_at IS '更新时间（同键覆盖写）';

-- 权限补授：发起/查看与待办审批授全体成员（authenticated 系统分组），
-- 具体任务能否审批由 TaskActor 实例级校验兜底
UPDATE tn_roles
SET rules = (
    rules::jsonb
    || '[{"resource": "workflow-instances", "operation": "create"}, {"resource": "workflow-instances", "operation": "get"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'workflow-instances'
  );

UPDATE tn_roles
SET rules = (
    rules::jsonb
    || '[{"resource": "workflow-tasks", "operation": "create"}, {"resource": "workflow-tasks", "operation": "get"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'workflow-tasks'
  );

-- 基线管理员按规则签名补授 workflow-instances / workflow-tasks 全量
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'workflow-instances', 'operation', '*'),
        jsonb_build_object('resource', 'workflow-tasks', 'operation', '*')
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
      WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'workflow-instances'
  );

COMMENT ON TABLE wf_instance IS '流程实例（ADR-012）：发起时一次性冻结绑定流程版本与表单版本（运行中永不自动升级）；状态 DRAFT/RUNNING/COMPLETED/REJECTED/CANCELLED 由状态机迁移表（task 包）唯一裁决；运行态历史完整保留，无软删';
COMMENT ON COLUMN wf_instance.id IS '自增主键（实例对外标识，API 以 id 定位）';
COMMENT ON COLUMN wf_instance.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_instance.definition_id IS '流程定义 ID（内部外键，不出网）';
COMMENT ON COLUMN wf_instance.definition_version_id IS '绑定的不可变发布快照行 ID（wf_definition_version），运行期间固定不变';
COMMENT ON COLUMN wf_instance.business_type IS '业务类型（如 form_record）；与 business_id 构成业务幂等键';
COMMENT ON COLUMN wf_instance.business_id IS '业务标识；同一 tenant+type+id 同一时间至多一个 RUNNING 实例（部分唯一索引兜底）';
COMMENT ON COLUMN wf_instance.app_id IS '业务归属应用 ID（0=未绑定；发起时记录）';
COMMENT ON COLUMN wf_instance.form_id IS '绑定表单 ID（0=未绑定；发起时经 FormDirectory 窄端口解析）';
COMMENT ON COLUMN wf_instance.form_version_id IS '绑定表单发布快照行 ID（0=未绑定）；表达式求值/字段权限/渲染均以此为准（Phase 3 接入 Form Domain）';
COMMENT ON COLUMN wf_instance.status IS '实例状态：DRAFT/RUNNING/COMPLETED/REJECTED/CANCELLED；迁移合法性由状态机迁移表校验';
COMMENT ON COLUMN wf_instance.starter_member_id IS '发起人（租户成员 ID）；Withdraw 资格与退回发起人的目标';
COMMENT ON COLUMN wf_instance.idempotency_key IS '请求幂等键（客户端生成）：同租户非空唯一，重发返回同一实例；NULL=未携带';
COMMENT ON COLUMN wf_instance.created_at IS '创建时间';
COMMENT ON COLUMN wf_instance.updated_at IS '更新时间';

COMMENT ON TABLE wf_execution IS '执行路径（Phase 8 并行执行树）：根路径由发起创建，并行 split 扇出子分支路径，分支到 join 收口 COMPLETED；RUNNING → COMPLETED/CANCELLED';
COMMENT ON COLUMN wf_execution.id IS '自增主键';
COMMENT ON COLUMN wf_execution.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_execution.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_execution.parent_execution_id IS '父路径 ID（0=根路径；并行 split 扇出的子分支挂父路径 ID）';
COMMENT ON COLUMN wf_execution.status IS '路径状态：RUNNING/COMPLETED/CANCELLED';
COMMENT ON COLUMN wf_execution.created_at IS '创建时间';
COMMENT ON COLUMN wf_execution.updated_at IS '更新时间';

COMMENT ON TABLE wf_node_instance IS '节点实例：某设计态 Node 在一次实例中的一次实际运行（Node ≠ NodeInstance ≠ Task，三人会签 = 1 节点实例 → 3 任务）；节点配置按 node_key 从发布快照读取';
COMMENT ON COLUMN wf_node_instance.id IS '自增主键';
COMMENT ON COLUMN wf_node_instance.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_node_instance.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_node_instance.execution_id IS '所属执行路径 ID';
COMMENT ON COLUMN wf_node_instance.node_key IS '对应设计态节点 key（快照内稳定标识）';
COMMENT ON COLUMN wf_node_instance.status IS '节点实例状态：PENDING/RUNNING/WAITING/WAITING_RESUBMIT/COMPLETED/REJECTED/CANCELLED；迁移合法性由状态机迁移表校验';
COMMENT ON COLUMN wf_node_instance.created_at IS '创建时间';
COMMENT ON COLUMN wf_node_instance.updated_at IS '更新时间';

COMMENT ON TABLE wf_task IS '人工任务（面向用户的待办）：原任务转办后关闭为 TRANSFERRED 并另建新任务，不直接修改原审批人；状态迁移由状态机迁移表唯一裁决';
COMMENT ON COLUMN wf_task.id IS '自增主键（任务对外标识）';
COMMENT ON COLUMN wf_task.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_task.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_task.node_instance_id IS '所属节点实例 ID';
COMMENT ON COLUMN wf_task.node_key IS '对应设计态节点 key';
COMMENT ON COLUMN wf_task.status IS '任务状态：PENDING/APPROVED/REJECTED/TRANSFERRED/CANCELLED/EXPIRED；PENDING 是唯一可执行动作状态';
COMMENT ON COLUMN wf_task.transferred_from_task_id IS '转办来源任务 ID（0=非转办产生）；任务历史链可追溯';
COMMENT ON COLUMN wf_task.transferred_to_member_id IS '转办目标成员 ID（仅 TRANSFERRED 任务记录去向）';
COMMENT ON COLUMN wf_task.created_at IS '创建时间';
COMMENT ON COLUMN wf_task.updated_at IS '更新时间';

COMMENT ON TABLE wf_task_actor IS '任务参与人快照（v1.1 定版）：Resolver 在任务创建时一次性解析落库，运行中不随组织变化重算；显示名为快照值，实时身份以成员 ID 为准；cc 为抄送参与人（不参与节点完成判定）';
COMMENT ON COLUMN wf_task_actor.id IS '自增主键';
COMMENT ON COLUMN wf_task_actor.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_task_actor.task_id IS '所属任务 ID';
COMMENT ON COLUMN wf_task_actor.member_id IS '参与人（租户成员 ID）；同任务同成员唯一';
COMMENT ON COLUMN wf_task_actor.display_name IS '解析时快照显示名（历史展示用）';
COMMENT ON COLUMN wf_task_actor.actor_role IS '参与角色：assignee=审批参与人 / cc=抄送对象';
COMMENT ON COLUMN wf_task_actor.created_at IS '创建时间';

COMMENT ON TABLE wf_operation IS '操作流水（追加写，禁止更新）：审批时间线唯一事实源；与状态变更同事务写入（第 13.2 章事务模板），operator_member_id=0 表示系统触发（如超时自动动作）';
COMMENT ON COLUMN wf_operation.id IS '自增主键';
COMMENT ON COLUMN wf_operation.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_operation.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_operation.task_id IS '关联任务 ID（0=实例级操作，如 START/WITHDRAW）';
COMMENT ON COLUMN wf_operation.operator_member_id IS '操作人成员 ID（0=系统）';
COMMENT ON COLUMN wf_operation.operation_type IS '操作类型：START/APPROVE/REJECT/RETURN_TO_STARTER/RESUBMIT/WITHDRAW/TERMINATE/TRANSFER/CC/TIMEOUT/REMINDER';
COMMENT ON TABLE wf_cc_record IS '流程抄送记录（追加写，禁止更新）：CC 节点执行时一次性解析并快照抄送对象（v1.1 审批人快照同语义）；cc 不是审批任务，不参与节点完成判定（ADR-012 第 10.6 章）';
COMMENT ON COLUMN wf_cc_record.id IS '自增主键';
COMMENT ON COLUMN wf_cc_record.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_cc_record.instance_id IS '归属流程实例 ID（wf_instance 外键）';
COMMENT ON COLUMN wf_cc_record.node_instance_id IS '触发抄送的节点实例 ID';
COMMENT ON COLUMN wf_cc_record.node_key IS '抄送节点 key（设计态，配置从发布快照读取）';
COMMENT ON COLUMN wf_cc_record.member_id IS '抄送对象成员 ID（同租户有效成员，解析时校验）';
COMMENT ON COLUMN wf_cc_record.display_name IS '抄送对象显示名快照（仅历史展示，实时身份以成员 ID 为准）';
COMMENT ON COLUMN wf_cc_record.created_at IS '创建时间';
COMMENT ON COLUMN wf_operation.payload IS '操作载荷 JSONB（节点 key、意见、转办去向等；敏感字段出网前脱敏）';
COMMENT ON COLUMN wf_operation.created_at IS '创建时间（追加写）';

-- 权限补授（000058，表单权限 P1）：基线管理员（规则签名 members:* + tn_roles:*
-- + tn_departments:*，与可改名的角色名无关）补授 form-permissions（配置面 CRUD，
-- 权限组管理接口 Service 复核键，不经管理组放行）与 form-data（数据面旁路
-- 键资源，form-data:* 经动作资源注册表展开产出 form-data:admin）
UPDATE tn_roles
SET rules = (
    rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'form-permissions', 'operation', '*'),
        jsonb_build_object('resource', 'form-data', 'operation', '*')
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
      WHERE rule->>'resource' = 'tn_roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'tn_departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' IN ('form-permissions', 'form-data')
  );

-- 统一账号审计字段（000059）：creator_id / updater_id 始终引用 pf_accounts.id。
DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'pf_accounts', 'pf_auth_infos', 'pf_tenants',
        'tn_users', 'tn_departments', 'tn_groups', 'tn_role_groups', 'tn_roles',
        'tn_member_invitations', 'tn_public_invitation_links',
        'tn_member_field_settings', 'tn_member_profiles', 'tn_admin_groups',
        'tn_applications', 'tn_application_menu_entries', 'tn_forms',
        'tn_asset_permission_groups', 'tn_files',
        'tn_notification_settings', 'tn_notification_custom_recipients',
        'tn_product_configs',
        'wf_definition', 'wf_instance', 'wf_execution', 'wf_node_instance',
        'wf_task', 'wf_task_actor'
    ] LOOP
        EXECUTE format('COMMENT ON COLUMN %I.creator_id IS %L', table_name, '创建账号 ID（pf_accounts.id）；NULL 表示系统或无认证操作');
        EXECUTE format('COMMENT ON COLUMN %I.updater_id IS %L', table_name, '最后更新账号 ID（pf_accounts.id）；NULL 表示系统或尚未有认证更新');
    END LOOP;
END $$;

COMMENT ON COLUMN tn_files.creator_member_id IS '文件归属成员 ID（tn_users.id），用于上传者访问边界';
COMMENT ON COLUMN tn_role_groups.creator_member_id IS '创建角色组的成员 ID（tn_users.id）';
