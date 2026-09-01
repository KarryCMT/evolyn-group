-- 回滚时恢复 groups 的历史成员 ID 语义；其他表移除本迁移新增字段。
-- 账号已不再是该租户成员时，旧语义没有可用的 users.id；安全回落为 NULL，
-- 不允许将 accounts.id 错当成成员 ID 遗留到回滚后的列中。
UPDATE groups AS g
SET creator_id = (
    SELECT u.id FROM users AS u
    WHERE u.account_id = g.creator_id AND u.tenant_id = g.tenant_id
    ORDER BY u.id DESC
    LIMIT 1
)
WHERE g.creator_id IS NOT NULL;

UPDATE groups AS g
SET updater_id = (
    SELECT u.id FROM users AS u
    WHERE u.account_id = g.updater_id AND u.tenant_id = g.tenant_id
    ORDER BY u.id DESC
    LIMIT 1
)
WHERE g.updater_id IS NOT NULL;

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

ALTER TABLE files RENAME COLUMN creator_member_id TO creator_id;
ALTER TABLE role_groups RENAME COLUMN creator_member_id TO creator_id;
