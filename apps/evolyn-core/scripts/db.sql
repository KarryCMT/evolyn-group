-- evolyn-core 冷启动初始化（终态快照）
-- 本文件 = migrations/ 000001..000006 全链执行后的等价状态，仅作
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
    avatar varchar(256),
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
    ('admin', 'admin', 'admin@weave.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP),
    ('demo', 'demo', 'admin@weave.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP)
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
    ('root', 'system', 'weave system group', LOCALTIMESTAMP),
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
    ('authenticated', 'cluster', '[{"resource": "users", "operation": "*"},{"resource": "auth", "operation": "*"}]'),
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
