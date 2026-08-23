-- 000016 down：先删节点表（含 parent_entry_id 自引用），再释放菜单修订号列
DROP TABLE IF EXISTS application_menu_entries;
ALTER TABLE applications DROP COLUMN IF EXISTS menu_revision;
