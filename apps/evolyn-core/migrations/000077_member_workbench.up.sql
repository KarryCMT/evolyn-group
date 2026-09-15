-- 000077：自定义工作台后端落库（成员个人配置）。
-- 成员端首页与设计器共用的 DashboardSchema 单文档 JSON（{version, widgets[]}，
-- 协议与前端 @evolyn.do/dashboard schema/lifecycle.ts 镜像，服务端校验器
-- 终审）整存整取于 tn_member_workbenches.content；每租户每成员一行，
-- revision 乐观锁防同成员多端/多窗口并发覆盖。无记录即默认布局（前端
-- 回退），个人配置不做软删，也不需要租户开通种子行。

CREATE TABLE IF NOT EXISTS tn_member_workbenches (
    id         BIGSERIAL   PRIMARY KEY,
    tenant_id  BIGINT      NOT NULL DEFAULT 1,
    member_id  BIGINT      NOT NULL,
    content    JSONB       NOT NULL,
    revision   BIGINT      NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tn_member_workbenches_tenant_member UNIQUE (tenant_id, member_id)
);

COMMENT ON TABLE tn_member_workbenches IS '成员自定义工作台（000077）：每租户每成员一份 DashboardSchema 单文档 JSON；无记录即默认布局，个人配置不做软删';
COMMENT ON COLUMN tn_member_workbenches.id IS '自增主键';
COMMENT ON COLUMN tn_member_workbenches.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tn_member_workbenches.member_id IS '工作台归属成员 ID（tn_users.id；读写一律叠加 tenant_id+member_id 双条件，不存在跨成员路径）';
COMMENT ON COLUMN tn_member_workbenches.content IS '工作台文档 JSON：{version, widgets[]}，结构由 workbench 域服务校验器（镜像前端 @evolyn.do/dashboard lifecycle.ts）保存时终审';
COMMENT ON COLUMN tn_member_workbenches.revision IS '乐观锁版本：首次保存为 1，此后每次保存 +1；提交 revision 不匹配返回 409 WORKBENCH_REVISION_CONFLICT';
COMMENT ON COLUMN tn_member_workbenches.created_at IS '创建时间';
COMMENT ON COLUMN tn_member_workbenches.updated_at IS '最后保存时间';

-- 权限补授：workbench:view 覆盖 GET（view 语义含 get/list）、update 覆盖
-- PUT，授 authenticated 系统分组关联角色（全体成员，口径同 menu-favorites/
-- notifications 先例）。数据范围恒为「当前租户 × 当前成员」，由 Service/
-- Repository 双条件兜底，资源级授权不可能放大为跨成员读写。
-- rules 列可空（json，无 NOT NULL）：COALESCE 兜底，防止 NULL || 规则
-- 数组得到 NULL 造成静默漏授（口径同 000040）。
UPDATE tn_roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || '[{"resource": "workbench", "operation": "view"}, {"resource": "workbench", "operation": "update"}]'::jsonb
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
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS e
      WHERE e::json->>'resource' = 'workbench'
  );
