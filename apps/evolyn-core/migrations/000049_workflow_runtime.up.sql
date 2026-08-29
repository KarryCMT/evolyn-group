-- 000049: 流程引擎 Phase 2 最小 Runtime（ADR-012 / docs/低代码平台/流程引擎/）。
-- 运行态六表：wf_instance / wf_execution / wf_node_instance / wf_task /
-- wf_task_actor / wf_operation。PostgreSQL 是 Runtime 状态唯一事实源：
-- 运行实例幂等走 (tenant_id, business_type, business_id) WHERE RUNNING 部分唯一
-- 索引（第 14.1 章）；请求幂等走 idempotency_key 部分唯一索引（第 14.2 章）；
-- 并发推进靠 SELECT ... FOR UPDATE 行锁（仓储适配层 Clauses 承担）。
-- 运行态表一律追加写为主、状态机字段更新为辅，无软删（历史完整保留）。
CREATE TABLE IF NOT EXISTS wf_instance (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    definition_id BIGINT NOT NULL,
    definition_version_id BIGINT NOT NULL,
    business_type varchar(64) NOT NULL,
    business_id varchar(64) NOT NULL,
    app_id BIGINT NOT NULL DEFAULT 0,
    form_id BIGINT NOT NULL DEFAULT 0,
    form_version_id BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'RUNNING',
    starter_member_id BIGINT NOT NULL,
    idempotency_key varchar(64),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_instance_status CHECK (status IN ('DRAFT', 'RUNNING', 'COMPLETED', 'REJECTED', 'CANCELLED'))
);

-- 业务幂等：同一 tenant+business_type+business_id 同一时间至多一个 RUNNING
-- 实例（REJECTED/CANCELLED 后可修改业务数据重新发起，第 14.1 章）
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_instance_running_business
    ON wf_instance (tenant_id, business_type, business_id)
    WHERE status = 'RUNNING';

-- 请求幂等：双击提交/HTTP 超时重试/客户端重发返回同一实例（第 14.2 章）
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_instance_idempotency_key
    ON wf_instance (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_wf_instance_business
    ON wf_instance (tenant_id, business_type, business_id, id DESC);

CREATE INDEX IF NOT EXISTS idx_wf_instance_tenant
    ON wf_instance (tenant_id, id DESC);

CREATE TABLE IF NOT EXISTS wf_execution (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    parent_execution_id BIGINT NOT NULL DEFAULT 0,
    status varchar(16) NOT NULL DEFAULT 'RUNNING',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_execution_status CHECK (status IN ('RUNNING', 'COMPLETED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_wf_execution_instance
    ON wf_execution (instance_id, id);

CREATE TABLE IF NOT EXISTS wf_node_instance (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    execution_id BIGINT NOT NULL,
    node_key varchar(64) NOT NULL,
    status varchar(24) NOT NULL DEFAULT 'PENDING',
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_node_instance_status CHECK (status IN
        ('PENDING', 'RUNNING', 'WAITING', 'WAITING_RESUBMIT', 'COMPLETED', 'REJECTED', 'CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_wf_node_instance_instance
    ON wf_node_instance (instance_id, id);

CREATE TABLE IF NOT EXISTS wf_task (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    node_instance_id BIGINT NOT NULL,
    node_key varchar(64) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'PENDING',
    transferred_from_task_id BIGINT NOT NULL DEFAULT 0,
    transferred_to_member_id BIGINT NOT NULL DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_wf_task_status CHECK (status IN
        ('PENDING', 'APPROVED', 'REJECTED', 'TRANSFERRED', 'CANCELLED', 'EXPIRED'))
);

-- 我的待办主查询（第 28 章）：tenant_id + actor + task_status（经 task_actor 关联）
CREATE INDEX IF NOT EXISTS idx_wf_task_instance
    ON wf_task (instance_id, id);

CREATE INDEX IF NOT EXISTS idx_wf_task_node_instance
    ON wf_task (node_instance_id, status);

CREATE INDEX IF NOT EXISTS idx_wf_task_status
    ON wf_task (tenant_id, status, id DESC);

CREATE TABLE IF NOT EXISTS wf_task_actor (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    task_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    display_name varchar(100) NOT NULL DEFAULT '',
    actor_role varchar(16) NOT NULL DEFAULT 'assignee',
    created_at timestamp with time zone,
    CONSTRAINT chk_wf_task_actor_role CHECK (actor_role IN ('assignee', 'cc'))
);

-- 任务创建时一次性快照（v1.1 定版）；同人同任务不重复落参与人行
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_task_actor
    ON wf_task_actor (task_id, member_id);

CREATE INDEX IF NOT EXISTS idx_wf_task_actor_member
    ON wf_task_actor (tenant_id, member_id, id DESC);

-- 操作流水：追加写（无 updated_at，禁止任何更新路径），审批时间线唯一事实源
CREATE TABLE IF NOT EXISTS wf_operation (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    task_id BIGINT NOT NULL DEFAULT 0,
    operator_member_id BIGINT NOT NULL DEFAULT 0,
    operation_type varchar(32) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at timestamp with time zone
);

CREATE INDEX IF NOT EXISTS idx_wf_operation_instance
    ON wf_operation (instance_id, id);

-- 权限补授：workflows 为设计态管理资源（基线管理员）；发起/查看与待办审批是
-- 全体成员的业务动作（workflow-instances:create=发起、get=查看，
-- workflow-tasks:create=审批动作 POST、get=待办详情），授 authenticated
-- 系统分组关联角色（口径同 menu-favorites 先例）；具体某一 Task 能否审批
-- 仍由 TaskActor 实例级校验兜底（第 21 章）。
UPDATE roles
SET rules = (
    rules::jsonb
    || '[{"resource": "workflow-instances", "operation": "create"}, {"resource": "workflow-instances", "operation": "get"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM group_roles gr
      INNER JOIN groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'workflow-instances'
  );

UPDATE roles
SET rules = (
    rules::jsonb
    || '[{"resource": "workflow-tasks", "operation": "create"}, {"resource": "workflow-tasks", "operation": "get"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM group_roles gr
      INNER JOIN groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'workflow-tasks'
  );

-- 基线管理员权限补授（口径同 000035/000037/000048）：workflow-instances 与
-- workflow-tasks 资源全量（含后续 terminate 等管理动作）。
UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'workflow-instances', 'operation', '*'),
        jsonb_build_object('resource', 'workflow-tasks', 'operation', '*')
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'workflow-instances'
  );

COMMENT ON TABLE wf_instance IS '流程实例（ADR-012）：发起时一次性冻结绑定流程版本与表单版本（运行中永不自动升级）；状态 DRAFT/RUNNING/COMPLETED/REJECTED/CANCELLED 由状态机迁移表（task 包）唯一裁决；运行态历史完整保留，无软删';
COMMENT ON COLUMN wf_instance.id IS '自增主键（实例对外标识，API 以 id 定位）';
COMMENT ON COLUMN wf_instance.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_instance.definition_id IS '流程定义 ID（内部外键，不出网）';
COMMENT ON COLUMN wf_instance.definition_version_id IS '绑定的不可变发布快照行 ID（wf_definition_version），运行期间固定不变';
COMMENT ON COLUMN wf_instance.business_type IS '业务类型（如 form_record）；与 business_id 构成业务幂等键';
COMMENT ON COLUMN wf_instance.business_id IS '业务标识；同一 tenant+type+id 同一时间至多一个 RUNNING 实例（部分唯一索引兜底）';
COMMENT ON COLUMN wf_instance.app_id IS '业务归属应用 ID（0=未绑定；发起时记录）';
COMMENT ON COLUMN wf_instance.form_id IS '绑定表单 ID（0=未绑定；发起时经 FormDirectory 窄端口解析）';
COMMENT ON COLUMN wf_instance.form_version_id IS '绑定表单发布快照行 ID（0=未绑定）；表达式求值/字段权限/渲染均以此为准（Phase 3 接入 Form Domain）';
COMMENT ON COLUMN wf_instance.status IS '实例状态：DRAFT/RUNNING/COMPLETED/REJECTED/CANCELLED；迁移合法性由状态机迁移表校验';
COMMENT ON COLUMN wf_instance.starter_member_id IS '发起人（租户成员 ID）；Withdraw 资格与退回发起人的目标';
COMMENT ON COLUMN wf_instance.idempotency_key IS '请求幂等键（客户端生成）：同租户非空唯一，重发返回同一实例；NULL=未携带';
COMMENT ON COLUMN wf_instance.created_at IS '创建时间';
COMMENT ON COLUMN wf_instance.updated_at IS '更新时间';

COMMENT ON TABLE wf_execution IS '执行路径（为并行 Phase 8、子流程预留）：V1 仅根路径 RUNNING → COMPLETED/CANCELLED';
COMMENT ON COLUMN wf_execution.id IS '自增主键';
COMMENT ON COLUMN wf_execution.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_execution.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_execution.parent_execution_id IS '父路径 ID（0=根路径；并行 split 才非零）';
COMMENT ON COLUMN wf_execution.status IS '路径状态：RUNNING/COMPLETED/CANCELLED';
COMMENT ON COLUMN wf_execution.created_at IS '创建时间';
COMMENT ON COLUMN wf_execution.updated_at IS '更新时间';

COMMENT ON TABLE wf_node_instance IS '节点实例：某设计态 Node 在一次实例中的一次实际运行（Node ≠ NodeInstance ≠ Task，三人会签 = 1 节点实例 → 3 任务）；节点配置按 node_key 从发布快照读取';
COMMENT ON COLUMN wf_node_instance.id IS '自增主键';
COMMENT ON COLUMN wf_node_instance.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_node_instance.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_node_instance.execution_id IS '所属执行路径 ID';
COMMENT ON COLUMN wf_node_instance.node_key IS '对应设计态节点 key（快照内稳定标识）';
COMMENT ON COLUMN wf_node_instance.status IS '节点实例状态：PENDING/RUNNING/WAITING/WAITING_RESUBMIT/COMPLETED/REJECTED/CANCELLED；迁移合法性由状态机迁移表校验';
COMMENT ON COLUMN wf_node_instance.created_at IS '创建时间';
COMMENT ON COLUMN wf_node_instance.updated_at IS '更新时间';

COMMENT ON TABLE wf_task IS '人工任务（面向用户的待办）：原任务转办后关闭为 TRANSFERRED 并另建新任务，不直接修改原审批人；状态迁移由状态机迁移表唯一裁决';
COMMENT ON COLUMN wf_task.id IS '自增主键（任务对外标识）';
COMMENT ON COLUMN wf_task.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_task.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_task.node_instance_id IS '所属节点实例 ID';
COMMENT ON COLUMN wf_task.node_key IS '对应设计态节点 key';
COMMENT ON COLUMN wf_task.status IS '任务状态：PENDING/APPROVED/REJECTED/TRANSFERRED/CANCELLED/EXPIRED；PENDING 是唯一可执行动作状态';
COMMENT ON COLUMN wf_task.transferred_from_task_id IS '转办来源任务 ID（0=非转办产生）；任务历史链可追溯';
COMMENT ON COLUMN wf_task.transferred_to_member_id IS '转办目标成员 ID（仅 TRANSFERRED 任务记录去向）';
COMMENT ON COLUMN wf_task.created_at IS '创建时间';
COMMENT ON COLUMN wf_task.updated_at IS '更新时间';

COMMENT ON TABLE wf_task_actor IS '任务参与人快照（v1.1 定版）：Resolver 在任务创建时一次性解析落库，运行中不随组织变化重算；显示名为快照值，实时身份以成员 ID 为准；cc 为抄送参与人（不参与节点完成判定）';
COMMENT ON COLUMN wf_task_actor.id IS '自增主键';
COMMENT ON COLUMN wf_task_actor.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_task_actor.task_id IS '所属任务 ID';
COMMENT ON COLUMN wf_task_actor.member_id IS '参与人（租户成员 ID）；同任务同成员唯一';
COMMENT ON COLUMN wf_task_actor.display_name IS '解析时快照显示名（历史展示用）';
COMMENT ON COLUMN wf_task_actor.actor_role IS '参与角色：assignee=审批参与人 / cc=抄送对象';
COMMENT ON COLUMN wf_task_actor.created_at IS '创建时间';

COMMENT ON TABLE wf_operation IS '操作流水（追加写，禁止更新）：审批时间线唯一事实源；与状态变更同事务写入（第 13.2 章事务模板），operator_member_id=0 表示系统触发（如超时自动动作）';
COMMENT ON COLUMN wf_operation.id IS '自增主键';
COMMENT ON COLUMN wf_operation.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_operation.instance_id IS '所属实例 ID';
COMMENT ON COLUMN wf_operation.task_id IS '关联任务 ID（0=实例级操作，如 START/WITHDRAW）';
COMMENT ON COLUMN wf_operation.operator_member_id IS '操作人成员 ID（0=系统）';
COMMENT ON COLUMN wf_operation.operation_type IS '操作类型：START/APPROVE/REJECT/RETURN_TO_STARTER/RESUBMIT/WITHDRAW/TERMINATE/TRANSFER/CC/TIMEOUT/REMINDER';
COMMENT ON COLUMN wf_operation.payload IS '操作载荷 JSONB（节点 key、意见、转办去向等；敏感字段出网前脱敏）';
COMMENT ON COLUMN wf_operation.created_at IS '创建时间（追加写）';
