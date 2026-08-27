-- 000037: 表单资产域 P1（ADR-010 / docs/低代码平台/表单设计器/表单资产域后端契约.md）。
-- forms 表单资产 + 草稿（目标保存协议全文，draft_revision 乐观锁口令）、
-- form_versions 不可变发布快照（version_no 表单内递增，schema_revision=行 id）。
-- 记录表 form_records 与 form-records 资源随 P2 迁移 000038 落地。
CREATE TABLE IF NOT EXISTS forms (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    application_id BIGINT NOT NULL,
    name varchar(128) NOT NULL,
    draft_content JSONB NOT NULL,
    draft_revision BIGINT NOT NULL DEFAULT 1,
    protocol_version INTEGER NOT NULL DEFAULT 1,
    latest_version_id BIGINT,
    published_version INTEGER NOT NULL DEFAULT 0,
    creator_member_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    CONSTRAINT chk_forms_content_object CHECK (jsonb_typeof(draft_content) = 'object')
);

-- 应用内列表（id 倒序，新表单靠前）；软删行排除
CREATE INDEX IF NOT EXISTS idx_forms_tenant_app
    ON forms (tenant_id, application_id, id DESC)
    WHERE deleted_at IS NULL;

-- 发布快照：不可变、追加写（无 updated/deleted 语义）；(form_id, version_no) 唯一
-- 保证并发发布不产生重号；schema_revision 在发布事务内回填为行 id。
CREATE TABLE IF NOT EXISTS form_versions (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    form_id BIGINT NOT NULL REFERENCES forms(id),
    version_no INTEGER NOT NULL,
    schema_revision BIGINT NOT NULL DEFAULT 0,
    content JSONB NOT NULL,
    field_keys JSONB NOT NULL DEFAULT '[]',
    protocol_version INTEGER NOT NULL DEFAULT 1,
    published_by_member_id BIGINT NOT NULL,
    published_at timestamp with time zone NOT NULL DEFAULT LOCALTIMESTAMP,
    created_at timestamp with time zone
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_form_versions_form_no
    ON form_versions (form_id, version_no);

CREATE INDEX IF NOT EXISTS idx_form_versions_tenant
    ON form_versions (tenant_id, id);

-- 基线管理员权限补授（口径同 000035「管理员规则签名」，与角色名无关）：forms 资源
-- 全量管理（创建/列表/详情/改名/草稿/发布/删除）。发布复用 forms:create 动词
--（POST /forms/:id/publish 的 URL 鉴权解析），一期不拆独立发布权。
UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'forms', 'operation', '*'))
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
      WHERE rule->>'resource' = 'forms'
  );

COMMENT ON TABLE forms IS '表单资产（租户内从属于应用，ADR-010）：draft_content 为目标保存协议草稿全文（content.items 两层结构，保存前经字段字典严格校验）；草稿与发布快照分表，删除只写 deleted_at，发布版本行保留';
COMMENT ON COLUMN forms.id IS '自增主键（菜单 target_id 引用值）';
COMMENT ON COLUMN forms.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN forms.application_id IS '所属应用 ID（同租户，服务层归属校验，禁止裸 ID 写入）';
COMMENT ON COLUMN forms.name IS '表单名称（trim 后 1–128 字符）；表单名称不进入协议 content';
COMMENT ON COLUMN forms.draft_content IS '目标保存协议草稿全文 JSONB：根结构 {content:{type:"form",items:[]}}，原样字节存取保证未编辑属性不丢失';
COMMENT ON COLUMN forms.draft_revision IS '草稿乐观锁口令：每次草稿保存条件递增，客户端原样回传，过期返回 FORM_REVISION_CONFLICT';
COMMENT ON COLUMN forms.protocol_version IS '协议版本外部承载（协议文档内不携带版本字段，字典 1.3），当前固定 1';
COMMENT ON COLUMN forms.latest_version_id IS '最新发布版本行 ID；NULL=从未发布';
COMMENT ON COLUMN forms.published_version IS '最新发布号（冗余自最新快照 version_no，0=未发布），供列表展示免 JOIN';
COMMENT ON COLUMN forms.creator_member_id IS '创建者（租户成员 ID），审计语义';
COMMENT ON COLUMN forms.created_at IS '创建时间';
COMMENT ON COLUMN forms.updated_at IS '更新时间';
COMMENT ON COLUMN forms.deleted_at IS '软删除时间，NULL=未删除；置位即释放 forms 配额，发布版本行保留';

COMMENT ON TABLE form_versions IS '不可变发布快照（追加写，无更新/删除路径）：发布时草稿全文固化，记录提交按 (publishedVersion, schemaRevision) 双口令定位并依据本快照终审；禁止以任何写路径覆盖历史快照';
COMMENT ON COLUMN form_versions.id IS '自增主键；即 schema_revision 口令的数值（出网字符串）';
COMMENT ON COLUMN form_versions.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN form_versions.form_id IS '所属表单 ID';
COMMENT ON COLUMN form_versions.version_no IS '表单内递增发布号 1,2,3…（即 publishedVersion）；与 form_id 联合唯一，并发发布兜底';
COMMENT ON COLUMN form_versions.schema_revision IS '修订口令（= 行 id，发布事务内回填）；与 version_no 共同构成提交定位双因子';
COMMENT ON COLUMN form_versions.content IS '发布时的目标保存协议全文快照 JSONB，写入后永不更新';
COMMENT ON COLUMN form_versions.field_keys IS '顶层字段键有序数组 JSONB（widgetName 序列），提交未知键快速拒绝与后续记录索引使用';
COMMENT ON COLUMN form_versions.protocol_version IS '快照协议版本（与发布时 forms 行一致）';
COMMENT ON COLUMN form_versions.published_by_member_id IS '发布人（租户成员 ID）';
COMMENT ON COLUMN form_versions.published_at IS '发布时间';
COMMENT ON COLUMN form_versions.created_at IS '创建时间';
