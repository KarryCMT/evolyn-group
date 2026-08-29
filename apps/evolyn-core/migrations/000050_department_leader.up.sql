-- 000050：部门负责人（IAM 前置能力，流程引擎 Phase 3 开工前置，ADR-012 第 17/33 章）
-- departments 增加 leader_member_id：流程引擎 department_manager 审批人 Resolver
-- 依赖 IAM 的明确 leader 语义，不在流程域私建第二套组织模型。
-- 不加外键：成员注销（PurgeByAccount 物理清理）时避免 FK 阻断；同租户与
-- 有效性由 IAM 部门服务在写入时校验（与 000035 角色改名风险同口径，宽列仅存 ID）。
ALTER TABLE departments
    ADD COLUMN IF NOT EXISTS leader_member_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_departments_leader_member_id ON departments (leader_member_id);

COMMENT ON COLUMN departments.leader_member_id IS '部门负责人成员 ID，NULL=未设置；流程引擎 department_manager 审批人解析数据源（ADR-012 Phase 3），同租户有效性由部门服务写入时校验';
