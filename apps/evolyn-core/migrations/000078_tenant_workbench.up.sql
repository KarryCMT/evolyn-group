-- 000078：自定义工作台语义定版——企业级配置（租户级资产）。
-- 产品口径：工作台由企业管理员统一配置、全员共用；创建企业（租户）时
-- 开通事务内初始化一条默认工作台数据与该租户绑定（Go 侧 WorkbenchSeeder，
-- 口径同 NotificationSettingSeeder/ProductConfigSeeder）。
-- 000077 的「成员个人配置」模型废弃：tn_member_workbenches 无生产数据，
-- 直接物理删除换表。

-- 权限收口（对 000077 补授的调整）：
--   1) workbench:update 从 authenticated（全体成员）收回——仅企业管理员可保存；
--   2) workbench:view 保留授全体成员——首页渲染读取全员必需；
--   3) workbench:update 按管理员规则签名（members:* + roles:* + departments:*
--      与角色名无关，口径同 000035/000073）补授基线管理员角色，不经管理组放行。
UPDATE tn_roles
SET rules = (
    SELECT COALESCE(jsonb_agg(e), '[]'::jsonb)
    FROM json_array_elements(COALESCE(rules, '[]'::json)) AS e
    WHERE NOT (e::json->>'resource' = 'workbench' AND e::json->>'operation' = 'update')
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
)
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS e
      WHERE e::json->>'resource' = 'workbench' AND e::json->>'operation' = 'update'
  );

UPDATE tn_roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'workbench', 'operation', 'update'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'workbench' AND rule->>'operation' IN ('update', '*')
  );

-- 成员个人配置表废弃（000077 换表，无生产数据）
DROP TABLE IF EXISTS tn_member_workbenches;

-- 租户工作台：每租户一行（唯一约束兜底），content 承载与前端
-- @evolyn.do/dashboard 镜像的 DashboardSchema 单文档，revision 乐观锁
CREATE TABLE tn_workbenches (
    id         BIGSERIAL   PRIMARY KEY,
    tenant_id  BIGINT      NOT NULL DEFAULT 1,
    content    JSONB       NOT NULL,
    revision   BIGINT      NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tn_workbenches_tenant UNIQUE (tenant_id)
);

COMMENT ON TABLE tn_workbenches IS '企业自定义工作台（000078）：每租户一份 DashboardSchema 单文档 JSON，企业管理员配置、全员共用；开通事务种子默认布局';
COMMENT ON COLUMN tn_workbenches.id IS '自增主键';
COMMENT ON COLUMN tn_workbenches.tenant_id IS '工作台归属租户 ID（每租户一行，读写以 tenant_id 定位）';
COMMENT ON COLUMN tn_workbenches.content IS '工作台文档 JSON：{version, widgets[]}，结构由 workbench 域服务校验器（镜像前端 @evolyn.do/dashboard lifecycle.ts）保存时终审';
COMMENT ON COLUMN tn_workbenches.revision IS '乐观锁版本：种子初始化为 1，此后每次保存 +1；提交 revision 不匹配返回 409 WORKBENCH_REVISION_CONFLICT';
COMMENT ON COLUMN tn_workbenches.created_at IS '创建时间';
COMMENT ON COLUMN tn_workbenches.updated_at IS '最后保存时间';

-- 存量租户回填默认工作台（未注销清理的租户；注销保留期内租户保留行）。
-- 默认文档与前端 defaultWorkbench.ts、Go 侧 workbench.DefaultWorkbenchDocument
-- 三处逐字镜像，修改默认布局时三处同步。
INSERT INTO tn_workbenches (tenant_id, content, revision, created_at, updated_at)
SELECT t.id, $doc${"version":1,"widgets":[
  {"id":"onboarding-0-0","type":"onboarding","title":"新手引导","x":0,"y":0,"w":12,"h":2,"noResize":true},
  {"id":"greeting-0-2","type":"greeting","title":"问候语","x":0,"y":2,"w":3,"h":1,"minW":3,"minH":1,"maxH":1,"presetKey":"greeting"},
  {"id":"favorites-3-2","type":"favorites","title":"最近使用","x":3,"y":2,"w":9,"h":2,"minW":4,"minH":2,"presetKey":"recent"},
  {"id":"shortcut-0-4","type":"shortcut","title":"未命名快捷入口","x":0,"y":4,"w":12,"h":2,"minH":2,"presetKey":"shortcut"},
  {"id":"todo-0-6","type":"todo","title":"流程中心","x":0,"y":6,"w":3,"h":4,"minW":3,"minH":3,"presetKey":"todo"},
  {"id":"favorites-3-6","type":"favorites","title":"我的收藏","x":3,"y":6,"w":9,"h":2,"minW":4,"minH":2,"presetKey":"favorites"},
  {"id":"apps-3-8","type":"apps","title":"我的应用","x":3,"y":8,"w":9,"h":3,"minW":4,"minH":3,"presetKey":"apps"},
  {"id":"charts-3-11","type":"charts","title":"我的图表","x":3,"y":11,"w":9,"h":2,"minW":4,"minH":2,"presetKey":"my-charts"}
]}$doc$::jsonb, 1, NOW(), NOW()
FROM pf_tenants t
WHERE t.purged_at IS NULL
ON CONFLICT (tenant_id) DO NOTHING;
