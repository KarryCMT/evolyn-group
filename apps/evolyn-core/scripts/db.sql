-- evolyn-core 冷启动初始化（与 GORM AutoMigrate 口径保持一致）
-- 模型定版：ADR-006 账号×成员拆分 + ADR-007 域模块化；租户规范见架构文档第 26 章
CREATE DATABASE weave;

\c weave;

-- 租户（平台一级资源；tenants 表自身带 tenant_id 列，运营面查询须无租户上下文）
CREATE TABLE IF NOT EXISTS tenants (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    code varchar(64) NOT NULL UNIQUE,
    name varchar(128) NOT NULL,
    plan varchar(32) NOT NULL DEFAULT 'free',
    status varchar(16) NOT NULL DEFAULT 'active',
    owner_account_id BIGINT NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    quotas JSONB NOT NULL DEFAULT '{}',
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

INSERT INTO tenants (code, name, plan, status, created_at) VALUES
    ('default', '默认租户', 'free', 'active', LOCALTIMESTAMP) ON CONFLICT DO NOTHING;

-- 平台账号（登录身份，全局唯一；无 tenant_id——账号跨租户，ADR-006）
CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL UNIQUE,
    nickname varchar(100),
    phone varchar(32) UNIQUE,
    email varchar(256),
    password varchar(256),
    avatar varchar(256),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

INSERT INTO accounts (name, nickname, email, password, created_at) VALUES
    ('admin', 'admin', 'admin@weave.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP),
    ('demo', 'demo', 'admin@weave.com', '$2a$10$5whQjJqSdL18PrEP.z/gZOubMKhFB38K0CvHWdnaQodb/H3yeG4J2', LOCALTIMESTAMP)
    ON CONFLICT DO NOTHING;

-- 第三方登录凭证（归属账号，平台级无 tenant_id）
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

-- 成员（原 users 表；登录身份已拆至 accounts，ADR-006）
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    account_id BIGINT NOT NULL DEFAULT 0,
    nickname varchar(100),
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

INSERT INTO users (account_id, nickname, tenant_id, created_at)
    SELECT a.id, a.nickname, 1, a.created_at FROM accounts AS a
    WHERE a.name IN ('admin', 'demo') ON CONFLICT DO NOTHING;

-- 部门（租户内组织架构，邻接表树）
CREATE TABLE IF NOT EXISTS departments (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    parent_id BIGINT NOT NULL DEFAULT 0,
    name varchar(100) NOT NULL,
    "order" BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'active',
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE TABLE IF NOT EXISTS department_users(
    department_id BIGINT NOT NULL REFERENCES departments(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    PRIMARY KEY(department_id, user_id)
);

CREATE TABLE IF NOT EXISTS groups (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL UNIQUE,
    kind varchar(100),
    describe varchar(1024),
    creator_id BIGINT,
    updater_id BIGINT,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

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

CREATE TABLE IF NOT EXISTS resources (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(256) NOT NULL,
    scope varchar(100),
    kind varchar(100)
);

CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    name varchar(100) NOT NULL UNIQUE,
    scope varchar(100),
    namespace varchar(100),
    rules json,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

INSERT INTO roles (name, scope, rules) VALUES
    ('cluster-admin', 'cluster', '[{"resource": "*", "operation": "*"}]'),
    ('authenticated', 'cluster', '[{"resource": "users", "operation": "*"},{"resource": "auth", "operation": "*"}]'),
    ('unauthenticated', 'cluster', '[{"resource": "auth", "operation": "create"}]');

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
    ((SELECT id FROM groups WHERE name = 'system:unauthenticated'), (SELECT id FROM roles WHERE name = 'unauthenticated'));
