-- 000059: 统一业务主表的创建/更新操作者。
-- creator_id / updater_id 始终保存 accounts.id；租户内成员展示通过
-- tenant_id + users.account_id 反查。NULL 表示注册前、迁移或定时任务等系统操作。

-- groups 的同名历史列存的是 users.id，先转换为平台账号 ID，避免字段名称相同
-- 但语义不同。成员已软删时仍可由其保留记录完成转换。
UPDATE groups AS g
SET creator_id = u.account_id
FROM users AS u
WHERE g.creator_id IS NOT NULL
  AND u.id = g.creator_id
  AND u.tenant_id = g.tenant_id;

UPDATE groups AS g
SET updater_id = u.account_id
FROM users AS u
WHERE g.updater_id IS NOT NULL
  AND u.id = g.updater_id
  AND u.tenant_id = g.tenant_id;

-- files 与 role_groups 原有 creator_id 都是成员归属字段，不能改写为账号
-- 审计字段；先改列名保留既有业务语义，再由下方统一补齐账号审计列。
ALTER TABLE files RENAME COLUMN creator_id TO creator_member_id;
ALTER TABLE role_groups RENAME COLUMN creator_id TO creator_member_id;

DO $$
DECLARE
    table_name text;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'accounts', 'auth_infos', 'tenants',
        'users', 'departments', 'role_groups', 'roles',
        'member_invitations', 'tenant_public_invitation_links',
        'tenant_member_field_settings', 'member_profiles', 'admin_groups',
        'applications', 'application_menu_entries', 'forms',
        'asset_permission_groups', 'files',
        'tenant_notification_settings', 'tenant_notification_custom_recipients',
        'tenant_product_configs',
        'wf_definition', 'wf_instance', 'wf_execution', 'wf_node_instance',
        'wf_task', 'wf_task_actor'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS creator_id BIGINT', table_name);
        EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS updater_id BIGINT', table_name);
        EXECUTE format('COMMENT ON COLUMN %I.creator_id IS %L', table_name, '创建账号 ID（accounts.id）；NULL 表示系统或无认证操作');
        EXECUTE format('COMMENT ON COLUMN %I.updater_id IS %L', table_name, '最后更新账号 ID（accounts.id）；NULL 表示系统或尚未有认证更新');
    END LOOP;
END $$;

COMMENT ON COLUMN groups.creator_id IS '创建账号 ID（accounts.id）；由 000059 从历史成员 ID 转换';
COMMENT ON COLUMN groups.updater_id IS '最后更新账号 ID（accounts.id）；由 000059 从历史成员 ID 转换';
COMMENT ON COLUMN files.creator_member_id IS '文件归属成员 ID（users.id），用于上传者访问边界';
COMMENT ON COLUMN role_groups.creator_member_id IS '创建角色组的成员 ID（users.id）';
