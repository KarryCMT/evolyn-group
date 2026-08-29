-- 000048: 流程引擎 Phase 1 Definition Engine（ADR-012 / docs/低代码平台/流程引擎/）。
-- wf_definition 流程定义 + 草稿（DSL v1 单文档 JSONB，draft_revision 乐观锁口令，
-- 与表单域 forms.draft_content 协议同构）；wf_definition_version 不可变发布快照
-- （version_no 定义内递增，DSL 整体冻结）。Node/Edge/Config 内嵌于 DSL 文档，
-- 不建 wf_node / wf_edge 规范化表（V1.1 定版：消除发布事务双写一致性负担）。
CREATE TABLE IF NOT EXISTS wf_definition (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    code varchar(64) NOT NULL,
    name varchar(128) NOT NULL,
    description varchar(512) NOT NULL DEFAULT '',
    draft_content JSONB NOT NULL,
    draft_revision BIGINT NOT NULL DEFAULT 1,
    latest_version_id BIGINT,
    published_version INTEGER NOT NULL DEFAULT 0,
    creator_member_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_wf_definition_content_object CHECK (jsonb_typeof(draft_content) = 'object')
);

-- 定义列表（id 倒序，新定义靠前）；软删行排除
CREATE INDEX IF NOT EXISTS idx_wf_definition_tenant
    ON wf_definition (tenant_id, id DESC)
    WHERE deleted_at IS NULL;

-- 稳定公开编码（wf_ 前缀）：路由/API/菜单 target 一律用 code，内部自增 ID
-- 不出网（对齐表单域 form_code 先例）；软删后编码可复用。
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_definition_tenant_code
    ON wf_definition (tenant_id, code)
    WHERE deleted_at IS NULL;

-- 发布快照：不可变、追加写（无 updated/deleted 语义）；(definition_id, version_no)
-- 唯一保证并发发布不产生重号。运行实例自 Phase 2 起固定绑定本表行。
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
-- workflows 资源全量管理（创建/列表/详情/改名/草稿/发布/删除）。发布复用
-- workflows:create 动词（POST /workflows/:code/publish 的 URL 鉴权解析），
-- 与表单域「发布复用 create」同口径；普通成员暂无 workflows 权限。
UPDATE roles
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
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
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
