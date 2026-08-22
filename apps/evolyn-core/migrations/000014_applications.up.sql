-- 000014: 应用管理域 M2-A（docs/低代码平台/应用管理/开发文档.md）。
-- applications 应用实例主表 + application_installations 来源安装快照。
-- 状态模型：status 仅表达可见业务状态（active/archived），删除只写
-- deleted_at（软删单轨，避免与状态列双轨失真）；provision_status 独立表达
-- 实例化进度，M2-A 空白应用同步创建即 ready。
-- 模板三表（application_templates/application_template_versions/
-- application_provision_jobs）属 M2-B/M2-C，后续迁移再落地，本期不预建。
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
    sort_order BIGINT NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_applications_source_type CHECK (source_type IN ('blank', 'template')),
    CONSTRAINT chk_applications_status CHECK (status IN ('active', 'archived')),
    CONSTRAINT chk_applications_provision_status CHECK (provision_status IN ('ready', 'pending', 'running', 'failed'))
);

-- code 服务端生成、租户内唯一（软删行释放编码可被复用）
CREATE UNIQUE INDEX IF NOT EXISTS uk_applications_tenant_code
    ON applications (tenant_id, code)
    WHERE deleted_at IS NULL;

-- 工作台列表默认序：sort_order 升序、id 倒序（新应用靠前）
CREATE INDEX IF NOT EXISTS idx_applications_tenant_status_sort
    ON applications (tenant_id, status, sort_order, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_applications_tenant_owner
    ON applications (tenant_id, owner_member_id)
    WHERE deleted_at IS NULL;

-- 安装记录：应用创建来源快照（blank/template），一应用一条、追加写无软删；
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

-- 基线角色补授 applications 资源（与 000011 订正口径一致，幂等）：
-- tenant-admin 全量管理；authenticated 只读——工作台「我的应用」对全体
-- 成员可见，创建/编辑/删除仍由租户管理员按角色授权
UPDATE roles
SET rules = (rules::jsonb || '[{"resource": "applications", "operation": "*"}]'::jsonb)::json
WHERE name = 'tenant-admin'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'applications'
  );

UPDATE roles
SET rules = (rules::jsonb || '[{"resource": "applications", "operation": "view"}]'::jsonb)::json
WHERE name = 'authenticated'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'applications'
  );

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
