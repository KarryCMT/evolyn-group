-- 000046: 菜单按钮权限与节点状态（表单菜单动作批，ADR-011）。
-- application_menu_entries.hidden：对成员隐藏（导航隐藏）——普通成员读取
-- 菜单时按不存在裁剪，持 applications:create/patch 的菜单管理成员仍可见
-- （否则无法恢复显示）；仅导航语义，不拦截 runtime 直连（资产级访问控制
-- 属后续资产级授权批次）。
ALTER TABLE application_menu_entries
    ADD COLUMN hidden BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN application_menu_entries.hidden IS '对成员隐藏（000046，导航隐藏）：普通成员读取菜单时按不存在裁剪，持 applications:create/patch 的菜单管理成员仍可见以便恢复；仅导航语义，不拦截 runtime 直连';

-- 表单类型语义放宽（000046 批，ADR-011）：form_type 可经 form-actions:
-- switch-type 动作切换（服务层裁决，000044 注释中的「创建后不可变」不再
-- 成立），切换后原类型流程数据保留；列注释随语义同步。
COMMENT ON COLUMN forms.form_type IS '表单类型：standard 标准表单 / workflow 流程表单；可经 form-actions:switch-type 切换（ADR-011），切换后原类型流程数据保留，设计器能力以此字段为准';

-- application_menu_favorites：成员对菜单节点的个人收藏（个人状态而非授权
-- 对象，凡节点可见即可收藏，menu-favorites 资源授全体成员）。收藏挂菜单
-- 节点（导航位置）而非资产；节点软删时同事务硬删收藏行，不做软删。
CREATE TABLE IF NOT EXISTS application_menu_favorites (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    application_id BIGINT NOT NULL REFERENCES applications(id),
    entry_id BIGINT NOT NULL REFERENCES application_menu_entries(id),
    created_at timestamp with time zone,
    CONSTRAINT uk_application_menu_favorites_member_entry UNIQUE (member_id, entry_id)
);

COMMENT ON TABLE application_menu_favorites IS '应用菜单个人收藏（000046，ADR-011）：成员×菜单节点的个人状态，不参与菜单共享结构与修订号；节点软删时同事务硬删关联行';
COMMENT ON COLUMN application_menu_favorites.id IS '自增主键';
COMMENT ON COLUMN application_menu_favorites.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN application_menu_favorites.member_id IS '收藏成员 ID（租户成员，读写一律叠加本列双条件）';
COMMENT ON COLUMN application_menu_favorites.application_id IS '收藏节点所属应用 ID（外键指向 applications）';
COMMENT ON COLUMN application_menu_favorites.entry_id IS '收藏的菜单节点 ID（外键指向 application_menu_entries；(member_id, entry_id) 唯一幂等）';
COMMENT ON COLUMN application_menu_favorites.created_at IS '收藏时间';

-- 「我的收藏」按成员枚举（读侧列表入口）
CREATE INDEX IF NOT EXISTS idx_application_menu_favorites_member
    ON application_menu_favorites (tenant_id, member_id, application_id);
