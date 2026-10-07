WITH authenticated_roles AS (
    SELECT DISTINCT r.id
    FROM tn_roles r
    INNER JOIN tn_group_roles gr ON gr.role_id = r.id
    INNER JOIN tn_groups g ON g.id = gr.group_id
    WHERE g.name = 'system:authenticated' AND g.kind = 'system'
)
UPDATE tn_roles r
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(r.rules) rule
    WHERE rule->>'resource' <> 'dashboard-actions'
)::json
WHERE r.id IN (SELECT id FROM authenticated_roles)
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array';

UPDATE tn_roles r
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(r.rules) rule
    WHERE rule->>'resource' NOT IN ('dashboards', 'dashboard-actions')
)::json
WHERE r.deleted_at IS NULL
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array';

DROP TABLE IF EXISTS tn_dashboard_create_bindings;
DROP TABLE IF EXISTS tn_dashboard_version_subjects;
ALTER TABLE tn_dashboards DROP CONSTRAINT IF EXISTS fk_tn_dashboards_latest_version;
DROP TABLE IF EXISTS tn_dashboard_versions;
DROP TABLE IF EXISTS tn_dashboards;

UPDATE pf_edition_plan_versions pv
SET entitlements = jsonb_set(
    pv.entitlements::jsonb,
    '{resources}',
    COALESCE((
      SELECT jsonb_agg(resource)
      FROM jsonb_array_elements(COALESCE(pv.entitlements::jsonb->'resources', '[]'::jsonb)) resource
      WHERE resource->>'key' <> 'dashboards'
    ), '[]'::jsonb),
    true
)
WHERE EXISTS (
  SELECT 1
  FROM jsonb_array_elements(COALESCE(pv.entitlements::jsonb->'resources', '[]'::jsonb)) resource
  WHERE resource->>'key' = 'dashboards'
);
