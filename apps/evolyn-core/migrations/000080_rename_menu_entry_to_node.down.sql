-- 000080 down：完整逆操作——注释还原后，按索引/约束/序列/列/表的依赖逆序
-- 还原命名，保证回到 000079 终态。

-- ---------- 注释还原 ----------
COMMENT ON COLUMN tn_app_menu_nodes.target_type IS '资产引用类型：group 为 NULL，非分组节点等于 entry_type（CHECK 约束）';
COMMENT ON COLUMN tn_app_menu_nodes.menu_type IS '节点类型：group 分组 / form 表单 / dashboard 仪表盘 / page 页面';
COMMENT ON COLUMN tn_app_menu_nodes.parent_menu_id IS '父节点 ID，根节点为 NULL；父节点须同租户同应用且为 group（服务层校验，单列外键表达不了同应用约束）';
COMMENT ON COLUMN tn_app_menu_nodes.code IS '服务端生成的节点编码（menu_ 前缀），租户内唯一（uk_tn_app_menu_entries_tenant_code，软删行释放），出网即 entryId';
COMMENT ON TABLE tn_app_menu_nodes IS '应用菜单节点（000016，M2-菜单）：分组/表单/仪表盘/页面的导航树（一资产一节点）；分组无 target，非分组节点 target_type=entry_type 且必须引用资产；租户/应用归属由服务层校验回填';
COMMENT ON COLUMN tn_form_records.menu_code IS '触发提交的应用菜单节点公开编码快照；设计预览直提允许为空';
COMMENT ON COLUMN tn_app_menu_favorites.menu_id IS '收藏的菜单节点 ID（外键指向 tn_app_menu_entries；(member_id, entry_id) 唯一幂等）';

-- ---------- 序列还原 ----------
ALTER SEQUENCE tn_app_menu_nodes_id_seq RENAME TO tn_app_menu_entries_id_seq;

-- ---------- 索引还原 ----------
ALTER INDEX idx_tn_app_menu_nodes_app_target RENAME TO idx_tn_app_menu_entries_app_target;
ALTER INDEX idx_tn_app_menu_nodes_app_parent_sort RENAME TO idx_tn_app_menu_entries_app_parent_sort;
ALTER INDEX uk_tn_app_menu_nodes_tenant_code RENAME TO uk_tn_app_menu_entries_tenant_code;

-- ---------- 约束还原 ----------
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT uk_tn_app_menu_favorites_member_menu TO uk_tn_app_menu_favorites_member_entry;
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT tn_app_menu_favorites_menu_id_fkey TO tn_app_menu_favorites_entry_id_fkey;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT chk_tn_app_menu_nodes_target TO chk_tn_app_menu_target;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT chk_tn_app_menu_nodes_menu_type TO chk_tn_app_menu_entry_type;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT tn_app_menu_nodes_parent_menu_id_fkey TO tn_app_menu_entries_parent_entry_id_fkey;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT tn_app_menu_nodes_app_id_fkey TO tn_app_menu_entries_app_id_fkey;
ALTER TABLE tn_app_menu_nodes RENAME CONSTRAINT tn_app_menu_nodes_pkey TO tn_app_menu_entries_pkey;

-- ---------- 列还原 ----------
ALTER TABLE tn_form_records RENAME COLUMN menu_code TO entry_code;
ALTER TABLE tn_app_menu_favorites RENAME COLUMN menu_id TO entry_id;
ALTER TABLE tn_app_menu_nodes RENAME COLUMN parent_menu_id TO parent_entry_id;
ALTER TABLE tn_app_menu_nodes RENAME COLUMN menu_type TO entry_type;

-- ---------- 表还原 ----------
ALTER TABLE tn_app_menu_nodes RENAME TO tn_app_menu_entries;
