-- 000016: 应用菜单（docs/低代码平台/应用管理/应用菜单接口功能设计方案.md）。
-- applications.menu_revision 为菜单专用乐观并发口令：菜单结构/名称/资产
-- 绑定/重排变更在同事务内条件递增；与 definition_version（发布演进）语义
-- 完全独立，非菜单更新不递增。
-- application_menu_entries 为菜单节点表：分组/表单/仪表盘/页面统一建模，
-- parent_entry_id 组成树，(tenant_id, code) 为节点公开标识；资产外键
-- （form/dashboard 等）待资产域迁移落地后再建。
ALTER TABLE applications
    ADD COLUMN menu_revision BIGINT NOT NULL DEFAULT 1;

COMMENT ON COLUMN applications.menu_revision IS '菜单修订号（菜单结构乐观并发口令）：菜单写入在同事务内条件递增；与 definition_version（发布演进）独立，应用名称/图标/归档等非菜单更新不递增';

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

-- 节点编码租户内唯一（软删行释放；服务端生成 menu_ 前缀随机编码）
CREATE UNIQUE INDEX IF NOT EXISTS uk_application_menu_entries_tenant_code
    ON application_menu_entries (tenant_id, code)
    WHERE deleted_at IS NULL;

-- 同父节点读取序：sort_order ASC, code ASC（tiebreak 用 code，与出网
-- entryId 同源，保证服务端输出序与前端复现序一致）
CREATE INDEX IF NOT EXISTS idx_application_menu_entries_app_parent_sort
    ON application_menu_entries (tenant_id, application_id, parent_entry_id, sort_order, code)
    WHERE deleted_at IS NULL;

-- 资产反查（资产软删/移动时定位关联菜单节点；首期无资产域写入恒空）
CREATE INDEX IF NOT EXISTS idx_application_menu_entries_app_target
    ON application_menu_entries (tenant_id, application_id, target_type, target_id)
    WHERE deleted_at IS NULL AND target_id IS NOT NULL;

COMMENT ON TABLE application_menu_entries IS '应用菜单节点：分组/表单/仪表盘/页面的导航树（一资产一节点）；分组无 target，非分组节点 target_type=entry_type 且必须引用资产；租户/应用归属由服务层校验回填';
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
