-- evolyn-core 冷启动初始化（终态快照）
-- 本文件 = migrations/ 000001..000014 全链执行后的等价状态，仅作
-- make postgres 快速起库用；Schema 唯一事实来源是 migrations/（FIX-009），
-- 结构变更必须同时提交 Migration，并同步维护本快照。
-- 快照库上重放迁移链应当零副作用：表/索引/约束使用与迁移一致的名字，
-- 种子写入均带 ON CONFLICT DO NOTHING。
-- 模型定版：ADR-006 账号×成员拆分 + ADR-007 域模块化 + 第一期整改 FIX-001~017
CREATE DATABASE evolyn;

\c evolyn;

-- 平台账号（登录身份，全局唯一；无 tenant_id——账号跨租户，ADR-006）。
-- 先建账号再建租户：tenants.owner_account_id 外键依赖本表（FIX-016）
CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL UNIQUE,
    nickname varchar(100),
    phone varchar(32),
    email varchar(256),
    password varchar(256),
    password_initialized boolean DEFAULT true NOT NULL,
    session_version BIGINT DEFAULT 0 NOT NULL,
    avatar varchar(256),
    onboarding jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

-- 手机号唯一性（000007）：非空才参与，未填账号落 '' 不互斥；
-- 软删除友好，与迁移链 uk_accounts_phone 同名同构
CREATE UNIQUE INDEX IF NOT EXISTS uk_accounts_phone
    ON accounts (phone)
    WHERE phone <> '' AND deleted_at IS NULL;

INSERT INTO accounts (name, nickname, email, password, created_at) VALUES
    ('admin', 'admin', 'admin@evolyn.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP),
    ('demo', 'demo', 'admin@evolyn.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP)
    ON CONFLICT DO NOTHING;

-- 租户（平台一级资源，无 tenant_id——FIX-014；owner 可空外键——FIX-016；
-- 注销生命周期时间线——FIX-012）
CREATE TABLE IF NOT EXISTS tenants (
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
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT fk_tenants_owner FOREIGN KEY (owner_account_id) REFERENCES accounts(id)
);

INSERT INTO tenants (code, name, plan, status, created_at) VALUES
    ('default', '默认租户', 'free', 'active', LOCALTIMESTAMP) ON CONFLICT DO NOTHING;

-- 默认租户 Owner 指向 admin 账号
UPDATE tenants SET owner_account_id = (SELECT id FROM accounts WHERE name = 'admin')
WHERE code = 'default' AND owner_account_id IS NULL;

-- 第三方登录凭证（归属账号，平台级无 tenant_id；
-- (auth_type, auth_id) 软删友好唯一——FIX-017）
CREATE TABLE IF NOT EXISTS auth_infos (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    account_id BIGINT NOT NULL DEFAULT 0 REFERENCES accounts(id),
    url varchar(256),
    auth_type varchar(256),
    auth_id varchar(256),
    access_token varchar(256),
    refresh_token varchar(256),
    expiry timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_auth_identity
    ON auth_infos (auth_type, auth_id)
    WHERE deleted_at IS NULL;

-- 成员（原 users 表；登录身份已拆至 accounts，ADR-006）。
-- account_id 外键——FIX-005；(tenant_id, account_id) 软删友好唯一——FIX-004
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    account_id BIGINT NOT NULL,
    nickname varchar(100),
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT fk_users_account FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_tenant_account
    ON users (tenant_id, account_id)
    WHERE deleted_at IS NULL;

INSERT INTO users (account_id, nickname, tenant_id, created_at)
    SELECT a.id, a.nickname, 1, a.created_at FROM accounts AS a
    WHERE a.name IN ('admin', 'demo') ON CONFLICT DO NOTHING;

-- 部门（租户内组织架构，邻接表树；parent_id NULL=root + 自引用外键——FIX-015）
CREATE TABLE IF NOT EXISTS departments (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    parent_id BIGINT,
    name varchar(100) NOT NULL,
    "order" BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'active',
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT fk_departments_parent FOREIGN KEY (parent_id) REFERENCES departments(id)
);

CREATE TABLE IF NOT EXISTS department_users(
    department_id BIGINT NOT NULL REFERENCES departments(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    PRIMARY KEY(department_id, user_id)
);

-- 用户组（租户内唯一——FIX-003：部分唯一索引，软删后可重建同名）
CREATE TABLE IF NOT EXISTS groups (
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

CREATE UNIQUE INDEX IF NOT EXISTS uk_groups_tenant_name
    ON groups (tenant_id, name)
    WHERE deleted_at IS NULL;

INSERT INTO groups (name, kind, describe, created_at) VALUES
    ('root', 'system', 'evolyn system group', LOCALTIMESTAMP),
    ('system:authenticated', 'system', 'system group contains all authenticated user', LOCALTIMESTAMP),
    ('system:unauthenticated', 'system', 'system group contains all unauthenticated user', LOCALTIMESTAMP)  ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS user_groups(
    group_id BIGINT NOT NULL REFERENCES groups(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    PRIMARY KEY(group_id, user_id)
);

-- admin 成员（同 ID 复制策略下 account_id = 自身 ID）入 root 组
INSERT INTO user_groups (group_id, user_id)
    SELECT g.id, u.id FROM users AS u, groups AS g
    WHERE (u.account_id = (SELECT id FROM accounts WHERE name = 'admin') AND g.name = 'root') ON CONFLICT DO NOTHING;

-- 平台资源目录（平台级，不挂租户）
CREATE TABLE IF NOT EXISTS resources (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(256) NOT NULL,
    scope varchar(100),
    kind varchar(100)
);

-- 角色（租户内唯一——FIX-002；软删除/时间三件套与模型对齐——FIX-001）
CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL,
    scope varchar(100),
    namespace varchar(100),
    rules json,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_roles_tenant_name
    ON roles (tenant_id, name)
    WHERE deleted_at IS NULL;

INSERT INTO roles (name, scope, rules) VALUES
    ('cluster-admin', 'cluster', '[{"resource": "*", "operation": "*"}]'),
    ('authenticated', 'cluster', '[{"resource": "users", "operation": "*"},{"resource": "auth", "operation": "*"},{"resource": "accounts", "operation": "*"},{"resource": "applications", "operation": "view"}]'),
    ('unauthenticated', 'cluster', '[{"resource": "auth", "operation": "create"}]') ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS user_roles(
    user_id BIGINT NOT NULL REFERENCES users(id),
    role_id BIGINT NOT NULL REFERENCES roles(id),
    PRIMARY KEY(user_id, role_id)
);

CREATE TABLE IF NOT EXISTS group_roles(
    group_id BIGINT NOT NULL REFERENCES groups(id),
    role_id BIGINT NOT NULL REFERENCES roles(id),
    PRIMARY KEY(group_id, role_id)
);

INSERT INTO group_roles (group_id, role_id) VALUES
    ((SELECT id FROM groups WHERE name = 'root'), (SELECT id FROM roles WHERE name = 'cluster-admin')),
    ((SELECT id FROM groups WHERE name = 'system:authenticated'), (SELECT id FROM roles WHERE name = 'authenticated')),
    ((SELECT id FROM groups WHERE name = 'system:unauthenticated'), (SELECT id FROM roles WHERE name = 'unauthenticated'))
    ON CONFLICT DO NOTHING;

-- 业务审计日志（FIX-013）：追加写流水，无软删语义
CREATE TABLE IF NOT EXISTS audit_logs (
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
    created_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs (tenant_id, id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_module_action ON audit_logs (module, action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs (resource_type, resource_id);

-- 登录日志（000013）：会话建立事件流水，账号维度追加写；与 audit_logs 职责互斥
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

-- 应用实例（000014，M2-A）：status 仅 active/archived，删除只写 deleted_at；
-- provision_status 独立表达实例化进度（M2-A 空白应用同步创建即 ready）
CREATE TABLE IF NOT EXISTS applications (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    code varchar(64) NOT NULL,
    name varchar(128) NOT NULL,
    icon varchar(32) NOT NULL DEFAULT 'bookmark',
    color varchar(32) NOT NULL DEFAULT 'primary',
    owner_member_id BIGINT NOT NULL,
    creator_member_id BIGINT NOT NULL,
    source_type varchar(16) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'active',
    provision_status varchar(16) NOT NULL DEFAULT 'ready',
    definition_version INTEGER NOT NULL DEFAULT 1,
    menu_revision BIGINT NOT NULL DEFAULT 1,
    sort_order BIGINT NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_applications_source_type CHECK (source_type IN ('blank', 'template')),
    CONSTRAINT chk_applications_status CHECK (status IN ('active', 'archived')),
    CONSTRAINT chk_applications_provision_status CHECK (provision_status IN ('ready', 'pending', 'running', 'failed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_applications_tenant_code
    ON applications (tenant_id, code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_applications_tenant_status_sort
    ON applications (tenant_id, status, sort_order, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_applications_tenant_owner
    ON applications (tenant_id, owner_member_id)
    WHERE deleted_at IS NULL;

-- 安装记录（000014）：应用创建来源快照，一应用一条、追加写无软删；
-- template_* 两列 M2-A 恒为 NULL，M2-B 模板安装启用
CREATE TABLE IF NOT EXISTS application_installations (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    application_id BIGINT NOT NULL UNIQUE REFERENCES applications(id),
    source_type varchar(16) NOT NULL,
    template_id BIGINT,
    template_version_id BIGINT,
    channel varchar(32) NOT NULL,
    blueprint_checksum varchar(64),
    installed_by_member_id BIGINT NOT NULL,
    installed_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_application_installations_tenant
    ON application_installations (tenant_id, id);

-- 应用菜单节点（000016，M2-菜单）：分组/表单/仪表盘/页面统一为菜单节点；
-- 分组无 target，非分组节点 target_type=entry_type 且引用资产；读取序
-- sort_order ASC, code ASC（tiebreak 与出网 entryId 同源）
CREATE TABLE IF NOT EXISTS application_menu_entries (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    application_id BIGINT NOT NULL REFERENCES applications(id),
    code varchar(64) NOT NULL,
    parent_entry_id BIGINT NULL REFERENCES application_menu_entries(id),
    entry_type varchar(16) NOT NULL,
    name varchar(128) NOT NULL,
    icon varchar(32) NULL,
    color varchar(32) NULL,
    target_type varchar(16) NULL,
    target_id BIGINT NULL,
    sort_order BIGINT NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_application_menu_entry_type
      CHECK (entry_type IN ('group', 'form', 'dashboard', 'page')),
    CONSTRAINT chk_application_menu_target
      CHECK (
        (entry_type = 'group' AND target_type IS NULL AND target_id IS NULL)
        OR
        (entry_type <> 'group' AND target_type = entry_type AND target_id IS NOT NULL)
      )
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_application_menu_entries_tenant_code
    ON application_menu_entries (tenant_id, code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_application_menu_entries_app_parent_sort
    ON application_menu_entries (tenant_id, application_id, parent_entry_id, sort_order, code)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_application_menu_entries_app_target
    ON application_menu_entries (tenant_id, application_id, target_type, target_id)
    WHERE deleted_at IS NULL AND target_id IS NOT NULL;

-- ============ 表/字段注释（000008，与迁移链一致） ============
COMMENT ON TABLE accounts IS '平台账号（登录身份）：登录名/手机号/密码与第三方凭证挂账号；账号跨租户，无 tenant_id（ADR-006/FIX-014）';
COMMENT ON COLUMN accounts.id IS '自增主键';
COMMENT ON COLUMN accounts.name IS '登录名，全局唯一';
COMMENT ON COLUMN accounts.nickname IS '平台级展示昵称；租户内昵称见 users.nickname';
COMMENT ON COLUMN accounts.phone IS '手机号，非空时全局唯一（部分唯一索引 uk_accounts_phone，未填写落空串不参与唯一）';
COMMENT ON COLUMN accounts.email IS '邮箱';
COMMENT ON COLUMN accounts.password IS '登录密码（bcrypt 摘要）；纯 OAuth 账号可为空';
COMMENT ON COLUMN accounts.password_initialized IS '密码是否由用户本人设置：短信免密注册为 false（存服务端随机密码），首次设置密码后置 true';
COMMENT ON COLUMN accounts.session_version IS '账号会话版本：密码重设/修改成功时递增，JWT 版本不一致即失效';
COMMENT ON COLUMN accounts.avatar IS '头像 URL';
COMMENT ON COLUMN accounts.onboarding IS '账号注册引导画像：role 角色 / channel 了解渠道（注册向导第 3 步采集）';
COMMENT ON COLUMN accounts.created_at IS '创建时间';
COMMENT ON COLUMN accounts.updated_at IS '更新时间';
COMMENT ON COLUMN accounts.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE tenants IS '租户（平台一级资源）：套餐/状态/Owner/租户级配置与注销生命周期；无 tenant_id（FIX-014）';
COMMENT ON COLUMN tenants.id IS '自增主键';
COMMENT ON COLUMN tenants.code IS '租户编码，全局唯一，登录时识别目标租户';
COMMENT ON COLUMN tenants.name IS '租户名称';
COMMENT ON COLUMN tenants.plan IS '套餐标识（free 等，套餐定义在租户域代码 plan.go，暂无套餐表）';
COMMENT ON COLUMN tenants.status IS '租户状态：active=正常 / frozen=冻结 / deleted=已注销';
COMMENT ON COLUMN tenants.owner_account_id IS '开通者（Owner）平台账号 ID，NULL=未设置；外键指向 accounts(id)（FIX-016）';
COMMENT ON COLUMN tenants.config IS '租户级配置 JSONB：水印/品牌主题/时区/语言';
COMMENT ON COLUMN tenants.quotas IS '套餐配额覆盖 JSONB；空对象表示使用套餐默认值';
COMMENT ON COLUMN tenants.delete_requested_at IS '注销申请时间（FIX-012 生命周期时间线）';
COMMENT ON COLUMN tenants.retention_until IS '数据保留截止时间，到期由 Purge Worker 执行清理';
COMMENT ON COLUMN tenants.purged_at IS '最终清理完成时间（墓碑标记）';
COMMENT ON COLUMN tenants.created_at IS '创建时间';
COMMENT ON COLUMN tenants.updated_at IS '更新时间';
COMMENT ON COLUMN tenants.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE auth_infos IS '第三方登录凭证（OAuth）：归属平台账号，无租户归属（ADR-006）';
COMMENT ON COLUMN auth_infos.id IS '自增主键';
COMMENT ON COLUMN auth_infos.account_id IS '归属平台账号 ID，外键指向 accounts(id)；存量回填前可能为 0';
COMMENT ON COLUMN auth_infos.url IS '第三方资料页地址';
COMMENT ON COLUMN auth_infos.auth_type IS '第三方渠道类型：github / wechat';
COMMENT ON COLUMN auth_infos.auth_id IS '第三方用户唯一标识；与 auth_type 组成软删友好唯一键（FIX-017）';
COMMENT ON COLUMN auth_infos.access_token IS '访问令牌';
COMMENT ON COLUMN auth_infos.refresh_token IS '刷新令牌';
COMMENT ON COLUMN auth_infos.expiry IS '令牌过期时间';
COMMENT ON COLUMN auth_infos.created_at IS '创建时间';
COMMENT ON COLUMN auth_infos.updated_at IS '更新时间';
COMMENT ON COLUMN auth_infos.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE users IS '租户成员（租户内身份）：归属账号+租户内昵称；部门/分组/角色关系见各关联表（ADR-006）';
COMMENT ON COLUMN users.id IS '自增主键';
COMMENT ON COLUMN users.account_id IS '归属平台账号 ID，外键指向 accounts(id)；同一账号在每租户至多一条有效成员关系（FIX-004）';
COMMENT ON COLUMN users.nickname IS '租户内展示昵称；空则前端回落账号昵称';
COMMENT ON COLUMN users.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN users.created_at IS '创建时间';
COMMENT ON COLUMN users.updated_at IS '更新时间';
COMMENT ON COLUMN users.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE departments IS '部门（租户内组织架构，邻接表树）：承载组织结构与汇报关系，权限授权走分组/角色';
COMMENT ON COLUMN departments.id IS '自增主键';
COMMENT ON COLUMN departments.parent_id IS '父部门 ID，NULL=根节点；自引用外键，跨租户父节点被 FK 与服务层共同拦截（FIX-015）';
COMMENT ON COLUMN departments.name IS '部门名称';
COMMENT ON COLUMN departments."order" IS '同级排序值，小者在前';
COMMENT ON COLUMN departments.status IS '部门状态：active=正常 / disabled=停用';
COMMENT ON COLUMN departments.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN departments.created_at IS '创建时间';
COMMENT ON COLUMN departments.updated_at IS '更新时间';
COMMENT ON COLUMN departments.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE department_users IS '部门-成员关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN department_users.department_id IS '部门 ID，联合主键之一，外键指向 departments(id)';
COMMENT ON COLUMN department_users.user_id IS '成员 ID，联合主键之一，外键指向 users(id)';

COMMENT ON TABLE groups IS '用户分组（租户内权限分组）：系统组（root/system:*）与自定义组，角色可挂分组间接授权成员';
COMMENT ON COLUMN groups.id IS '自增主键';
COMMENT ON COLUMN groups.name IS '分组名，租户内唯一（软删友好部分唯一索引 uk_groups_tenant_name，FIX-003）';
COMMENT ON COLUMN groups.kind IS '分组类型：system=系统组 / custom=自定义组';
COMMENT ON COLUMN groups.describe IS '分组描述（列名沿用历史拼写）';
COMMENT ON COLUMN groups.creator_id IS '创建人成员 ID';
COMMENT ON COLUMN groups.updater_id IS '最后更新人成员 ID';
COMMENT ON COLUMN groups.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN groups.created_at IS '创建时间';
COMMENT ON COLUMN groups.updated_at IS '更新时间';
COMMENT ON COLUMN groups.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE user_groups IS '成员-分组关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN user_groups.group_id IS '分组 ID，联合主键之一，外键指向 groups(id)';
COMMENT ON COLUMN user_groups.user_id IS '成员 ID，联合主键之一，外键指向 users(id)';

COMMENT ON TABLE resources IS '平台资源目录（RBAC 鉴权资源清单）：平台级，不挂租户';
COMMENT ON COLUMN resources.id IS '自增主键';
COMMENT ON COLUMN resources.name IS '资源名，对应角色规则里的 resource（users/groups/auth 等）';
COMMENT ON COLUMN resources.scope IS '作用域：cluster=平台级';
COMMENT ON COLUMN resources.kind IS '资源种类：resource=API 资源 / menu=菜单';

COMMENT ON TABLE roles IS '角色（租户内）：rules 声明资源-操作授权规则，可挂成员或分组';
COMMENT ON COLUMN roles.id IS '自增主键';
COMMENT ON COLUMN roles.name IS '角色名，租户内唯一（软删友好部分唯一索引 uk_roles_tenant_name，FIX-002）';
COMMENT ON COLUMN roles.scope IS '作用域：cluster=平台级';
COMMENT ON COLUMN roles.namespace IS '命名空间，预留扩展';
COMMENT ON COLUMN roles.rules IS '授权规则 JSON 数组，元素形如 {"resource":"*","operation":"*"}；operation 支持 */edit/view';
COMMENT ON COLUMN roles.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN roles.created_at IS '创建时间';
COMMENT ON COLUMN roles.updated_at IS '更新时间';
COMMENT ON COLUMN roles.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE user_roles IS '成员-角色关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN user_roles.user_id IS '成员 ID，联合主键之一，外键指向 users(id)';
COMMENT ON COLUMN user_roles.role_id IS '角色 ID，联合主键之一，外键指向 roles(id)';

COMMENT ON TABLE group_roles IS '分组-角色关联表（多对多）：分组内成员经本表继承角色授权';
COMMENT ON COLUMN group_roles.group_id IS '分组 ID，联合主键之一，外键指向 groups(id)';
COMMENT ON COLUMN group_roles.role_id IS '角色 ID，联合主键之一，外键指向 roles(id)';

COMMENT ON TABLE audit_logs IS '业务审计日志（FIX-013）：追加写流水，记录谁在什么租户对什么资源做了什么；无更新/软删语义';
COMMENT ON COLUMN audit_logs.id IS '自增主键';
COMMENT ON COLUMN audit_logs.tenant_id IS '租户 ID，0=平台级操作（运营域）';
COMMENT ON COLUMN audit_logs.account_id IS '操作者平台账号 ID，0=系统或未知';
COMMENT ON COLUMN audit_logs.member_id IS '操作者租户成员 ID，0=平台级操作或未知';
COMMENT ON COLUMN audit_logs.module IS '业务域模块：tenant/iam/...';
COMMENT ON COLUMN audit_logs.action IS '动作：create/update/delete/bind/...';
COMMENT ON COLUMN audit_logs.resource_type IS '目标资源类型（users/groups/roles 等）';
COMMENT ON COLUMN audit_logs.resource_id IS '目标资源 ID，可空';
COMMENT ON COLUMN audit_logs.request_id IS '链路追踪请求 ID，可空';
COMMENT ON COLUMN audit_logs.ip IS '客户端 IP，可空';
COMMENT ON COLUMN audit_logs.user_agent IS '客户端 User-Agent，可空';
COMMENT ON COLUMN audit_logs.before_data IS '变更前数据快照 JSONB，可空';
COMMENT ON COLUMN audit_logs.after_data IS '变更后数据快照 JSONB，可空';
COMMENT ON COLUMN audit_logs.created_at IS '记录发生时间，默认当前时间';

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

COMMENT ON TABLE applications IS '应用实例（租户内一级资源，M2-A）：空白/模板安装创建的低代码应用；owner/creator 引用租户成员，删除只写 deleted_at（状态列不设 deleted）';
COMMENT ON COLUMN applications.id IS '自增主键';
COMMENT ON COLUMN applications.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN applications.code IS '服务端生成的应用编码，租户内唯一（uk_applications_tenant_code，软删行释放），供 URL/外部引用/日志使用，创建后不可修改';
COMMENT ON COLUMN applications.name IS '展示名称，允许同租户重名，不作业务主键';
COMMENT ON COLUMN applications.icon IS '稳定图标键（bookmark/briefcase/contacts/chart/check），不存前端组件名';
COMMENT ON COLUMN applications.color IS '稳定颜色键（primary），不存 CSS 字面值';
COMMENT ON COLUMN applications.owner_member_id IS '应用所有者（租户成员 ID），同租户约束由服务层校验';
COMMENT ON COLUMN applications.creator_member_id IS '创建者（租户成员 ID），成员删除后保留 ID 审计语义';
COMMENT ON COLUMN applications.source_type IS '创建来源：blank 空白创建 / template 模板安装（冗余于安装记录，便于列表查询）';
COMMENT ON COLUMN applications.status IS '可见业务状态：active 正常 / archived 归档；删除只写 deleted_at，不设 deleted 状态值';
COMMENT ON COLUMN applications.provision_status IS '实例化进度（与 status 独立）：ready 就绪 / pending 待处理 / running 处理中 / failed 失败；M2-A 空白应用同步创建即 ready';
COMMENT ON COLUMN applications.definition_version IS '应用定义版本（发布演进用），非数据库乐观锁';
COMMENT ON COLUMN applications.menu_revision IS '菜单修订号（菜单结构乐观并发口令）：菜单写入在同事务内条件递增；与 definition_version（发布演进）独立，应用名称/图标/归档等非菜单更新不递增';
COMMENT ON COLUMN applications.sort_order IS '列表排序值，小者在前，同值按 id 倒序';
COMMENT ON COLUMN applications.config IS '小型应用级配置 JSONB；严禁混入表单/页面/流程大定义';
COMMENT ON COLUMN applications.created_at IS '创建时间';
COMMENT ON COLUMN applications.updated_at IS '更新时间';
COMMENT ON COLUMN applications.deleted_at IS '软删除时间，NULL=未删除；置位即从常规列表隐藏并释放配额';

COMMENT ON TABLE application_installations IS '安装记录：应用创建来源快照（一应用一条），供升级、问题定位与审计；追加写无更新/软删语义';
COMMENT ON COLUMN application_installations.id IS '自增主键';
COMMENT ON COLUMN application_installations.tenant_id IS '所属租户 ID（与应用一致）';
COMMENT ON COLUMN application_installations.application_id IS '应用 ID，唯一，一个应用只有一条初始来源记录';
COMMENT ON COLUMN application_installations.source_type IS '来源类型：blank / template';
COMMENT ON COLUMN application_installations.template_id IS '模板 ID，空白应用为 NULL（M2-B 启用）';
COMMENT ON COLUMN application_installations.template_version_id IS '模板版本 ID，空白应用为 NULL（M2-B 启用）';
COMMENT ON COLUMN application_installations.channel IS '安装渠道：self 空白创建 / template_center 模板中心 / admin 运营 / api 开放接口';
COMMENT ON COLUMN application_installations.blueprint_checksum IS '安装时实际使用的蓝图校验值，空白应用为 NULL（M2-B 启用）';
COMMENT ON COLUMN application_installations.installed_by_member_id IS '发起安装的租户成员 ID';
COMMENT ON COLUMN application_installations.installed_at IS '安装时间';

COMMENT ON TABLE application_menu_entries IS '应用菜单节点（000016，M2-菜单）：分组/表单/仪表盘/页面的导航树（一资产一节点）；分组无 target，非分组节点 target_type=entry_type 且必须引用资产；租户/应用归属由服务层校验回填';
COMMENT ON COLUMN application_menu_entries.id IS '自增主键';
COMMENT ON COLUMN application_menu_entries.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN application_menu_entries.application_id IS '所属应用 ID（外键指向 applications），同应用约束由服务层在加载校验';
COMMENT ON COLUMN application_menu_entries.code IS '服务端生成的节点编码（menu_ 前缀），租户内唯一（uk_application_menu_entries_tenant_code，软删行释放），出网即 entryId';
COMMENT ON COLUMN application_menu_entries.parent_entry_id IS '父节点 ID，根节点为 NULL；父节点须同租户同应用且为 group（服务层校验，单列外键表达不了同应用约束）';
COMMENT ON COLUMN application_menu_entries.entry_type IS '节点类型：group 分组 / form 表单 / dashboard 仪表盘 / page 页面';
COMMENT ON COLUMN application_menu_entries.name IS '节点展示名';
COMMENT ON COLUMN application_menu_entries.icon IS '稳定图标键（可空），不存前端组件名；前端受控映射表转换为图标组件';
COMMENT ON COLUMN application_menu_entries.color IS '稳定颜色键（可空），不存 CSS 字面值';
COMMENT ON COLUMN application_menu_entries.target_type IS '资产引用类型：group 为 NULL，非分组节点等于 entry_type（CHECK 约束）';
COMMENT ON COLUMN application_menu_entries.target_id IS '资产域内部数字主键；出网时由资产查询投影为稳定公开编码，不直接暴露';
COMMENT ON COLUMN application_menu_entries.sort_order IS '同父节点排序值，仅同父内有意义；新增 1024 间隔，服务端重排写连续间隔值，不信任客户端排序值';
COMMENT ON COLUMN application_menu_entries.config IS '小型显示配置 JSONB（如页面打开方式）；严禁存放表单 Schema、流程定义、权限或前端组件名';
COMMENT ON COLUMN application_menu_entries.created_at IS '创建时间';
COMMENT ON COLUMN application_menu_entries.updated_at IS '更新时间';
COMMENT ON COLUMN application_menu_entries.deleted_at IS '软删除时间，NULL=未删除；资产软删时同事务软删关联节点，应用软删后的节点由清理任务处理';
