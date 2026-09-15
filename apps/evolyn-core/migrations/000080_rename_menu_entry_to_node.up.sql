-- 000080：菜单域命名统一 entry→menu / MenuEntry→MenuNode（前后端代码、API
-- 字段 menuId/menuCode/menuType/路由 /menu/nodes/:menuCode 已同步改名，本
-- 迁移收敛数据库层）：
--   1) 表 tn_app_menu_entries → tn_app_menu_nodes（菜单节点表，节点即菜单项）；
--   2) 列 entry_id→menu_id、parent_entry_id→parent_menu_id、entry_type→
--      menu_type、tn_form_records.entry_code→menu_code（提交入口编码快照）；
--   3) 约束/索引/序列随表与新列名统一改名；
--   4) 注释随新命名重发。
-- 历史迁移文件（000016/000046/000063/000079）按仓库规则保留旧名不改写。

-- ---------- 1. 表改名 ----------
ALTER TABLE tn_app_menu_entries RENAME TO tn_app_menu_nodes;

-- ---------- 2. 列改名 ----------
ALTER TABLE tn_app_menu_nodes RENAME COLUMN entry_type TO menu_type;
ALTER TABLE tn_app_menu_nodes RENAME COLUMN parent_entry_id TO parent_menu_id;
ALTER TABLE tn_app_menu_favorites RENAME COLUMN entry_id TO menu_id;
ALTER TABLE tn_form_records RENAME COLUMN entry_code TO menu_code;

-- ---------- 3. 约束改名 ----------
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT tn_app_menu_entries_pkey TO tn_app_menu_nodes_pkey;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT tn_app_menu_entries_app_id_fkey TO tn_app_menu_nodes_app_id_fkey;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT tn_app_menu_entries_parent_entry_id_fkey TO tn_app_menu_nodes_parent_menu_id_fkey;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT chk_tn_app_menu_entry_type TO chk_tn_app_menu_nodes_menu_type;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT chk_tn_app_menu_target TO chk_tn_app_menu_nodes_target;
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT tn_app_menu_favorites_entry_id_fkey TO tn_app_menu_favorites_menu_id_fkey;
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT uk_tn_app_menu_favorites_member_entry TO uk_tn_app_menu_favorites_member_menu;

-- ---------- 4. 索引改名 ----------
ALTER INDEX uk_tn_app_menu_entries_tenant_code RENAME TO uk_tn_app_menu_nodes_tenant_code;
ALTER INDEX idx_tn_app_menu_entries_app_parent_sort RENAME TO idx_tn_app_menu_nodes_app_parent_sort;
ALTER INDEX idx_tn_app_menu_entries_app_target RENAME TO idx_tn_app_menu_nodes_app_target;

-- ---------- 5. 序列改名 ----------
ALTER SEQUENCE tn_app_menu_entries_id_seq RENAME TO tn_app_menu_nodes_id_seq;

-- ---------- 6. 注释随新命名重发 ----------
COMMENT ON COLUMN tn_app_menu_favorites.menu_id IS '收藏的菜单节点 ID（外键指向 tn_app_menu_nodes；(member_id, menu_id) 唯一幂等）';
COMMENT ON COLUMN tn_form_records.menu_code IS '触发提交的应用菜单节点公开编码快照；设计预览直提允许为空';
COMMENT ON TABLE tn_app_menu_nodes IS '应用菜单节点（000016，M2-菜单）：分组/表单/仪表盘/页面的导航树（一资产一节点）；分组无 target，非分组节点 target_type=menu_type 且必须引用资产；租户/应用归属由服务层校验回填';
COMMENT ON COLUMN tn_app_menu_nodes.code IS '服务端生成的节点编码（menu_ 前缀），租户内唯一（uk_tn_app_menu_nodes_tenant_code，软删行释放），出网即 menuId';
COMMENT ON COLUMN tn_app_menu_nodes.parent_menu_id IS '父节点 ID，根节点为 NULL；父节点须同租户同应用且为 group（服务层校验，单列外键表达不了同应用约束）';
COMMENT ON COLUMN tn_app_menu_nodes.menu_type IS '节点类型：group 分组 / form 表单 / dashboard 仪表盘 / page 页面';
COMMENT ON COLUMN tn_app_menu_nodes.target_type IS '资产引用类型：group 为 NULL，非分组节点等于 menu_type（CHECK 约束）';
