-- 000079 down：完整逆操作——数据面先还原（app → application），再逆序还原
-- home_mode 约束、注释、序列、约束、索引、列名与表名，保证回到 000078 终态。

-- ---------- 数据面还原 ----------
UPDATE pf_edition_plan_versions
SET entitlements = replace(entitlements::text, '"app_management"', '"application_management"')::jsonb
WHERE entitlements::text LIKE '%app_management%';

UPDATE tn_admin_groups
SET scope_config = replace(
        replace(
            replace(
                replace(scope_config::text, '"appIds"', '"applicationIds"'),
                '"allApps"', '"allApplications"'
            ),
            '"app"', '"application"'
        ),
        'externalOrg/app/addressBook', 'externalOrg/application/addressBook'
    )::jsonb
WHERE scope_config::text LIKE '%app%';
UPDATE tn_admin_groups SET scope = 'application' WHERE scope = 'app';

-- 注意：tn_roles.rules 还原无法区分历史上本就是其他 "apps" 形态的规则，
-- 仅当规则串不含其它 app 资源时按本迁移引入的形态回退（幂等近似）。
UPDATE tn_roles
SET rules = replace(
        replace(rules::text, '"tn_apps"', '"tn_applications"'),
        '"apps"', '"applications"'
    )::json
WHERE rules::text LIKE '%apps%';

-- ---------- home_mode 还原 ----------
ALTER TABLE tn_apps DROP CONSTRAINT chk_tn_apps_home_mode;
UPDATE tn_apps SET home_mode = 'application' WHERE home_mode = 'app';
ALTER TABLE tn_apps ADD CONSTRAINT chk_tn_applications_home_mode
    CHECK (home_mode IN ('builder', 'application'));

-- ---------- 注释还原 ----------
COMMENT ON COLUMN tn_app_menu_favorites.app_id IS '收藏节点所属应用 ID（外键指向 tn_applications）';
COMMENT ON COLUMN tn_app_menu_favorites.entry_id IS '收藏的菜单节点 ID（外键指向 tn_application_menu_entries；(member_id, entry_id) 唯一幂等）';
COMMENT ON COLUMN tn_audit_logs.category_code IS '稳定日志范围码：member_management 成员管理 / organization 组织架构 / role_permission 角色权限 / tenant_settings 企业设置 / application 应用管理 / file_storage 文件管理 / account_security 账号安全 / log_export 日志导出';
COMMENT ON COLUMN tn_apps.code IS '服务端生成的应用编码，租户内唯一（uk_tn_applications_tenant_code，软删行释放），供 URL/外部引用/日志使用，创建后不可修改';
COMMENT ON COLUMN tn_apps.home_mode IS '应用首页形态：builder 显示首次构建引导 / application 进入运行时应用首页；由应用生命周期维护，不按当前成员可见菜单数量推导';
COMMENT ON COLUMN tn_app_menu_entries.app_id IS '所属应用 ID（外键指向 tn_applications），同应用约束由服务层在加载校验';
COMMENT ON COLUMN tn_app_menu_entries.code IS '服务端生成的节点编码（menu_ 前缀），租户内唯一（uk_tn_application_menu_entries_tenant_code，软删行释放），出网即 entryId';
COMMENT ON COLUMN tn_app_menu_entries.hidden IS '对成员隐藏（000046，导航隐藏）：普通成员读取菜单时按不存在裁剪，持 tn_applications:create/patch 的菜单管理成员仍可见以便恢复；仅导航语义，不拦截 runtime 直连';
COMMENT ON COLUMN tn_forms.app_id IS '所属应用 ID（同租户，服务层归属校验，禁止裸 ID 写入）';
COMMENT ON COLUMN tn_admin_groups.scope IS '管理组类型：system=系统管理员页（通讯录管理组），application=灵衍云管理员页（普通管理组）';
COMMENT ON COLUMN tn_admin_groups.scope_config IS '范围配置 JSONB：department/role/externalOrg/application/addressBook 区块，按 scope 适用性由服务层校验；ID 清单悬挂引用由读取侧丢弃';
COMMENT ON TABLE pf_product_catalogs IS '平台内置产品目录：平台提供、可被多个租户启用的产品（如灵衍云）；不是租户自建的 tn_applications 应用';

-- ---------- 序列还原 ----------
ALTER SEQUENCE tn_apps_id_seq RENAME TO tn_applications_id_seq;
ALTER SEQUENCE tn_app_installations_id_seq RENAME TO tn_application_installations_id_seq;
ALTER SEQUENCE tn_app_menu_entries_id_seq RENAME TO tn_application_menu_entries_id_seq;
ALTER SEQUENCE tn_app_menu_favorites_id_seq RENAME TO tn_application_menu_favorites_id_seq;

-- ---------- 约束还原 ----------
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT uk_tn_app_menu_favorites_member_entry TO uk_tn_application_menu_favorites_member_entry;
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT tn_app_menu_favorites_entry_id_fkey TO tn_application_menu_favorites_entry_id_fkey;
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT tn_app_menu_favorites_app_id_fkey TO tn_application_menu_favorites_application_id_fkey;
ALTER TABLE tn_app_menu_favorites RENAME CONSTRAINT tn_app_menu_favorites_pkey TO tn_application_menu_favorites_pkey;
ALTER TABLE tn_app_menu_entries RENAME CONSTRAINT chk_tn_app_menu_target TO chk_tn_application_menu_target;
ALTER TABLE tn_app_menu_entries RENAME CONSTRAINT chk_tn_app_menu_entry_type TO chk_tn_application_menu_entry_type;
ALTER TABLE tn_app_menu_entries RENAME CONSTRAINT tn_app_menu_entries_parent_entry_id_fkey TO tn_application_menu_entries_parent_entry_id_fkey;
ALTER TABLE tn_app_menu_entries RENAME CONSTRAINT tn_app_menu_entries_app_id_fkey TO tn_application_menu_entries_application_id_fkey;
ALTER TABLE tn_app_menu_entries RENAME CONSTRAINT tn_app_menu_entries_pkey TO tn_application_menu_entries_pkey;
ALTER TABLE tn_app_installations RENAME CONSTRAINT tn_app_installations_app_id_key TO tn_application_installations_application_id_key;
ALTER TABLE tn_app_installations RENAME CONSTRAINT tn_app_installations_app_id_fkey TO tn_application_installations_application_id_fkey;
ALTER TABLE tn_app_installations RENAME CONSTRAINT tn_app_installations_pkey TO tn_application_installations_pkey;
ALTER TABLE tn_apps RENAME CONSTRAINT chk_tn_apps_status TO chk_tn_applications_status;
ALTER TABLE tn_apps RENAME CONSTRAINT chk_tn_apps_source_type TO chk_tn_applications_source_type;
ALTER TABLE tn_apps RENAME CONSTRAINT chk_tn_apps_provision_status TO chk_tn_applications_provision_status;
ALTER TABLE tn_apps RENAME CONSTRAINT tn_apps_pkey TO tn_applications_pkey;

-- ---------- 索引还原 ----------
ALTER INDEX idx_tn_audit_logs_tenant_app_created RENAME TO idx_tn_audit_logs_tenant_application_created;
ALTER INDEX idx_tn_app_menu_favorites_member RENAME TO idx_tn_application_menu_favorites_member;
ALTER INDEX idx_tn_app_menu_entries_app_target RENAME TO idx_tn_application_menu_entries_app_target;
ALTER INDEX idx_tn_app_menu_entries_app_parent_sort RENAME TO idx_tn_application_menu_entries_app_parent_sort;
ALTER INDEX uk_tn_app_menu_entries_tenant_code RENAME TO uk_tn_application_menu_entries_tenant_code;
ALTER INDEX idx_tn_app_installations_tenant RENAME TO idx_tn_application_installations_tenant;
ALTER INDEX idx_tn_apps_tenant_owner RENAME TO idx_tn_applications_tenant_owner;
ALTER INDEX idx_tn_apps_tenant_status_sort RENAME TO idx_tn_applications_tenant_status_sort;
ALTER INDEX uk_tn_apps_tenant_code RENAME TO uk_tn_applications_tenant_code;

-- ---------- 列还原 ----------
ALTER TABLE tn_audit_logs RENAME COLUMN app_name_snapshot TO application_name_snapshot;
ALTER TABLE tn_audit_logs RENAME COLUMN app_code TO application_code;
ALTER TABLE tn_audit_logs RENAME COLUMN app_id TO application_id;
ALTER TABLE tn_asset_permission_groups RENAME COLUMN app_id TO application_id;
ALTER TABLE tn_forms RENAME COLUMN app_id TO application_id;
ALTER TABLE tn_app_menu_favorites RENAME COLUMN app_id TO application_id;
ALTER TABLE tn_app_menu_entries RENAME COLUMN app_id TO application_id;
ALTER TABLE tn_app_installations RENAME COLUMN app_id TO application_id;

-- ---------- 表还原 ----------
ALTER TABLE tn_app_menu_favorites RENAME TO tn_application_menu_favorites;
ALTER TABLE tn_app_menu_entries RENAME TO tn_application_menu_entries;
ALTER TABLE tn_app_installations RENAME TO tn_application_installations;
ALTER TABLE tn_apps RENAME TO tn_applications;
