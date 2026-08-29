-- 000051：抄送记录表（流程引擎 Phase 4，ADR-012 第 10.6 章）
-- CC 不是审批 Task，不参与节点完成判定；「抄送我的」是审批中心高频查询，
-- 依 10.6 章「查询模型最简」原则落独立追加写记录表，而非 JSONB operation 检索。
CREATE TABLE IF NOT EXISTS wf_cc_record (
    id BIGSERIAL PRIMARY KEY NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    instance_id BIGINT NOT NULL,
    node_instance_id BIGINT NOT NULL DEFAULT 0,
    node_key varchar(64) NOT NULL DEFAULT '',
    member_id BIGINT NOT NULL,
    display_name varchar(100) NOT NULL DEFAULT '',
    created_at timestamp with time zone,
    CONSTRAINT fk_wf_cc_record_instance FOREIGN KEY (instance_id) REFERENCES wf_instance(id)
);

-- 抄送我的（第 28 章高频查询）：tenant_id + member + 游标
CREATE INDEX IF NOT EXISTS idx_wf_cc_record_member
    ON wf_cc_record (tenant_id, member_id, id DESC);

-- 实例维度抄送列表（详情页时间线联动）
CREATE INDEX IF NOT EXISTS idx_wf_cc_record_instance
    ON wf_cc_record (instance_id, id);

COMMENT ON TABLE wf_cc_record IS '流程抄送记录（追加写，禁止更新）：CC 节点执行时一次性解析并快照抄送对象（v1.1 审批人快照同语义）；cc 不是审批任务，不参与节点完成判定（ADR-012 第 10.6 章）';
COMMENT ON COLUMN wf_cc_record.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN wf_cc_record.instance_id IS '归属流程实例 ID（wf_instance 外键）';
COMMENT ON COLUMN wf_cc_record.node_instance_id IS '触发抄送的节点实例 ID';
COMMENT ON COLUMN wf_cc_record.node_key IS '抄送节点 key（设计态，配置从发布快照读取）';
COMMENT ON COLUMN wf_cc_record.member_id IS '抄送对象成员 ID（同租户有效成员，解析时校验）';
COMMENT ON COLUMN wf_cc_record.display_name IS '抄送对象显示名快照（仅历史展示，实时身份以成员 ID 为准）';
COMMENT ON COLUMN wf_cc_record.created_at IS '创建时间';
