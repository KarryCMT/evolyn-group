-- 000039 down：与 up 严格对称（先撤规则再撤表/索引，全部 IF EXISTS）。
UPDATE roles
SET rules = (
    (
        SELECT COALESCE(jsonb_agg(element), '[]'::jsonb)
        FROM json_array_elements(rules) AS element
        WHERE element::jsonb->>'resource' NOT IN ('notifications', 'notification-settings')
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array';

DROP INDEX IF EXISTS idx_notification_outbox_dispatch;
DROP INDEX IF EXISTS uk_notification_outbox_event_id;
DROP TABLE IF EXISTS notification_outbox_events;

DROP INDEX IF EXISTS idx_tenant_notification_recipients_tenant;
DROP TABLE IF EXISTS tenant_notification_custom_recipients;

DROP INDEX IF EXISTS idx_tenant_notif_pref_recipients_recipient;
DROP INDEX IF EXISTS idx_tenant_notif_pref_recipients_pref;
DROP TABLE IF EXISTS tenant_notification_preference_recipients;

DROP INDEX IF EXISTS idx_tenant_notification_prefs_setting;
DROP INDEX IF EXISTS uk_tenant_notification_prefs_event;
DROP TABLE IF EXISTS tenant_notification_preferences;

DROP INDEX IF EXISTS uk_tenant_notification_settings_tenant;
DROP TABLE IF EXISTS tenant_notification_settings;

DROP INDEX IF EXISTS idx_notification_inboxes_member_unread;
DROP INDEX IF EXISTS idx_notification_inboxes_member_list;
DROP INDEX IF EXISTS uk_notification_inboxes_unique;
DROP TABLE IF EXISTS notification_member_inboxes;

DROP INDEX IF EXISTS idx_notification_messages_tenant_category;
DROP INDEX IF EXISTS uk_notification_messages_tenant_event;
DROP TABLE IF EXISTS notification_messages;
