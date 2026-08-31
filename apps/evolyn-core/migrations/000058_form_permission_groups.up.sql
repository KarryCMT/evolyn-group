-- 000058: 表单资产权限组（docs/低代码平台/表单权限/表单权限组后端设计方案.md §6）。
-- 资产权限组 = 主体范围（成员/部门/角色）× 操作集 × 字段矩阵 × 数据范围的
-- 整体授权单元；表名泛化为 asset_permission_groups（asset_type 现仅 form，
-- 类型白名单由 Service 层资产类型注册表承载，不设数据库 CHECK——仪表盘
-- 扩展无需 DDL，约束与代码同源）。
--
-- 定版语义（S2–S8）：组绑定判定不跨组扁平化并集；数据面旁路仅认显式动作键
-- form-data:admin（服务层判定专用，不挂 URL 门）；禁用组同样维持收口；
-- deny-by-default 字段矩阵由判定侧按当前字段清单合并。
--
-- 索引策略（§5.2 定版）：不建 values 根列 GIN——模板谓词均作用于
-- values->'F' 表达式，GIN 无法命中，ne/not_in/empty 等负向谓词本就不可用；
-- P1 记录过滤在 (tenant_id, form_id, id DESC) 既有索引范围内取行集后行内
-- 过滤，性能演进收敛到投影表方案。

CREATE TABLE asset_permission_groups (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       BIGINT       NOT NULL,
    application_id  BIGINT       NOT NULL,                    -- 冗余归属，Service 校验与资产一致
    asset_type      VARCHAR(16)  NOT NULL DEFAULT 'form',     -- form（类型白名单在 Service 注册表）
    asset_id        BIGINT       NOT NULL,                    -- form → forms.id（内部主键）
    code            VARCHAR(64)  NOT NULL,                    -- fpg_ 前缀服务端生成，出网稳定标识
    name            VARCHAR(64)  NOT NULL,
    description     VARCHAR(200) NOT NULL DEFAULT '',
    enabled         BOOLEAN      NOT NULL DEFAULT TRUE,       -- 禁用组维持收口（S5）但不授权
    operations      JSONB        NOT NULL DEFAULT '[]',       -- 操作键数组（设计 §3 字典）
    field_permissions JSONB      NOT NULL DEFAULT '[]',       -- [{field,visible,editable}]（设计 §4）
    data_scope      JSONB        NOT NULL DEFAULT '{}',       -- {match,conditions}（设计 §5）
    revision        BIGINT       NOT NULL DEFAULT 1,          -- 整组乐观锁（PUT 全量提交口令）
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT uq_asset_permission_groups_code UNIQUE (code)
);

CREATE INDEX idx_asset_permission_groups_tenant
    ON asset_permission_groups (tenant_id, asset_type, asset_id);

-- 权限组主体：成员 / 部门 / 角色，判定侧按主体反查。
-- 外键 ON DELETE CASCADE 只对物理 DELETE 生效：组删除走软删（deleted_at），
-- 由 DELETE Service 同事务显式硬删 subjects 行；外键仅作为未来物理清理路径
-- （租户注销清库、数据迁移）的兜底，保证任何物理删除不产生孤儿授权行。
-- subject_id 不做外键：角色/部门删除时判定侧容错（解析不到的主体不命中），
-- 写入时经 Service 校验同租户存在性。
CREATE TABLE asset_permission_group_subjects (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    BIGINT      NOT NULL,
    group_id     BIGINT      NOT NULL,
    subject_type VARCHAR(16) NOT NULL,
    subject_id   BIGINT      NOT NULL,          -- users.id / departments.id / roles.id
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_asset_permission_group_subjects UNIQUE (group_id, subject_type, subject_id),
    CONSTRAINT ck_asset_permission_group_subjects_type
        CHECK (subject_type IN ('member', 'department', 'role')),
    CONSTRAINT fk_asset_permission_group_subjects_group
        FOREIGN KEY (group_id) REFERENCES asset_permission_groups (id) ON DELETE CASCADE
);

CREATE INDEX idx_asset_permission_group_subjects_lookup
    ON asset_permission_group_subjects (tenant_id, subject_type, subject_id);

COMMENT ON TABLE asset_permission_groups IS '资产权限组（表单权限 P1）：主体范围×操作集×字段矩阵×数据范围的整体授权单元；表单存在任一权限组行（含禁用组）即进入授权模型（S5 收口）';
COMMENT ON COLUMN asset_permission_groups.id IS '自增主键（不出网）';
COMMENT ON COLUMN asset_permission_groups.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN asset_permission_groups.application_id IS '所属应用 ID（冗余归属，Service 校验与资产一致）';
COMMENT ON COLUMN asset_permission_groups.asset_type IS '资产类型：form（类型白名单由 Service 层注册表承载，预留 dashboard 扩展位）';
COMMENT ON COLUMN asset_permission_groups.asset_id IS '资产内部 ID（form → forms.id）';
COMMENT ON COLUMN asset_permission_groups.code IS '权限组公开编码（fpg_ 前缀服务端生成，出网稳定标识）';
COMMENT ON COLUMN asset_permission_groups.name IS '权限组名称（1–64 字符）';
COMMENT ON COLUMN asset_permission_groups.description IS '权限组描述';
COMMENT ON COLUMN asset_permission_groups.enabled IS '启用状态；禁用组同样维持收口（S5）但不授权';
COMMENT ON COLUMN asset_permission_groups.operations IS '操作键 JSONB 数组（view/add/copy/edit/delete/batch_print/batch_modify/import/export + 流程表单 workflow_*，设计 §3 字典）';
COMMENT ON COLUMN asset_permission_groups.field_permissions IS '字段矩阵 JSONB 数组 [{field,visible,editable}]；缺失字段 deny-by-default（S7）';
COMMENT ON COLUMN asset_permission_groups.data_scope IS '数据范围 JSONB {match,conditions}（match: all/any；空条件=全部数据 S6）';
COMMENT ON COLUMN asset_permission_groups.revision IS '整组乐观锁口令（PUT 全量提交，冲突返回 FORM_PERMISSION_REVISION_CONFLICT）';
COMMENT ON COLUMN asset_permission_groups.created_at IS '创建时间';
COMMENT ON COLUMN asset_permission_groups.updated_at IS '更新时间';
COMMENT ON COLUMN asset_permission_groups.deleted_at IS '软删时间（组删除走软删，subjects 同事务显式硬删）';

COMMENT ON TABLE asset_permission_group_subjects IS '资产权限组主体：成员/部门/角色；判定侧按主体反查命中组（部门含子部门，祖先链命中）';
COMMENT ON COLUMN asset_permission_group_subjects.id IS '自增主键';
COMMENT ON COLUMN asset_permission_group_subjects.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN asset_permission_group_subjects.group_id IS '所属权限组 ID（物理删除级联；软删由 Service 同事务硬删本表）';
COMMENT ON COLUMN asset_permission_group_subjects.subject_type IS '主体类型：member/department/role（CHECK 约束）';
COMMENT ON COLUMN asset_permission_group_subjects.subject_id IS '主体 ID（users.id/departments.id/roles.id；无外键，判定侧容错解析不到的主体）';
COMMENT ON COLUMN asset_permission_group_subjects.created_at IS '创建时间';

-- 基线管理员补授（规则签名 members:* + roles:* + departments:*，与可改名的
-- 角色名无关，口径同 000034/000035/000047）：
-- 1) form-permissions（配置面 CRUD 动词，权限组管理接口的 Service 层复核键；
--    URL 门仍走 forms:* 首段解析）。该资源不经管理组放行。
-- 2) form-data（数据面旁路键资源，admin 动作码）：form-data:* 经动作资源
--    注册表通配展开产出 form-data:admin（S3 数据面旁路仅认该键）。
UPDATE roles
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
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' IN ('form-permissions', 'form-data')
  );
