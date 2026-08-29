-- 000049 down：与 up 严格对称（先撤规则再撤表/索引，全部 IF EXISTS）。
UPDATE roles
SET rules = (
    (
        SELECT COALESCE(jsonb_agg(element), '[]'::jsonb)
        FROM json_array_elements(rules) AS element
        WHERE element::jsonb->>'resource' NOT IN ('workflow-instances', 'workflow-tasks')
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array';

DROP INDEX IF EXISTS idx_wf_operation_instance;
DROP TABLE IF EXISTS wf_operation;
DROP INDEX IF EXISTS idx_wf_task_actor_member;
DROP INDEX IF EXISTS uk_wf_task_actor;
DROP TABLE IF EXISTS wf_task_actor;
DROP INDEX IF EXISTS idx_wf_task_status;
DROP INDEX IF EXISTS idx_wf_task_node_instance;
DROP INDEX IF EXISTS idx_wf_task_instance;
DROP TABLE IF EXISTS wf_task;
DROP INDEX IF EXISTS idx_wf_node_instance_instance;
DROP TABLE IF EXISTS wf_node_instance;
DROP INDEX IF EXISTS idx_wf_execution_instance;
DROP TABLE IF EXISTS wf_execution;
DROP INDEX IF EXISTS idx_wf_instance_tenant;
DROP INDEX IF EXISTS idx_wf_instance_business;
DROP INDEX IF EXISTS uk_wf_instance_idempotency_key;
DROP INDEX IF EXISTS uk_wf_instance_running_business;
DROP TABLE IF EXISTS wf_instance;
