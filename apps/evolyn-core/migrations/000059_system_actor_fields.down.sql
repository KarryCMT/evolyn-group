-- 回滚时恢复 groups 的历史成员 ID 语义；其他表移除本迁移新增字段。
UPDATE groups AS g
SET creator_id = u.id
FROM users AS u
WHERE g.creator_id IS NOT NULL
  AND u.account_id = g.creator_id
  AND u.tenant_id = g.tenant_id;

UPDATE groups AS g
SET updater_id = u.id
FROM users AS u
WHERE g.updater_id IS NOT NULL
  AND u.account_id = g.updater_id
  AND u.tenant_id = g.tenant_id;

UPDATE role_groups AS rg
SET creator_id = u.id
FROM users AS u
WHERE rg.creator_id IS NOT NULL
  AND u.account_id = rg.creator_id
  AND u.tenant_id = rg.tenant_id;

-- 旧列为 NOT NULL DEFAULT 0；本迁移期间产生的系统操作无账号时需还原为旧哨兵值。
UPDATE role_groups SET creator_id = 0 WHERE creator_id IS NULL;
ALTER TABLE role_groups ALTER COLUMN creator_id SET DEFAULT 0;
ALTER TABLE role_groups ALTER COLUMN creator_id SET NOT NULL;

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
        EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS creator_id', table_name);
        EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS updater_id', table_name);
    END LOOP;
END $$;
