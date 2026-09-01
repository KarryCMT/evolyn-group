/**
 * Workflow DSL v1 TypeScript 协议层：与后端
 * evolyn-core internal/engine/workflow/model/dsl.go 逐字对齐（字段名/值域/
 * 上限全部一致），是前后端共同协议的唯一前端事实源。设计器编辑与持久化
 * 都只操作这份文档结构；LogicFlow 图数据（含画布坐标）不是运行时模型。
 */

/** DSL 协议版本号；校验器只接受本值（后端 DSLSchemaVersion） */
export const WORKFLOW_DSL_SCHEMA_VERSION = '1.0';

/** V1 节点类型目录（与后端 V1NodeTypes 冻结口径一致） */
export type WorkflowNodeType =
  | 'start'
  | 'approval'
  | 'condition'
  | 'cc'
  | 'service'
  | 'parallel'
  | 'end';

/** 审批模式（后端 ApprovalMode）：单人 / 或签 / 会签 */
export type WorkflowApprovalMode = 'single' | 'or-sign' | 'countersign';

/** 驳回策略（V1 仅 terminate：终止型驳回） */
export type WorkflowRejectStrategy = 'terminate';

/** 字段权限值域（后端 FieldPermission，V1 冻结四档） */
export type WorkflowFieldPermission = 'hidden' | 'readonly' | 'editable' | 'required';

/** 审批人解析类型（后端 AssigneeType） */
export type WorkflowAssigneeType =
  | 'user'
  | 'role'
  | 'form_field'
  | 'department'
  | 'department_manager'
  | 'starter_manager';

/** 超时自动动作（V1 仅同意/驳回，经 Task Engine 正常路径触发） */
export type WorkflowTimeoutAction = 'approve' | 'reject';

/** 并行网关角色（Phase 8）：split 分流 / join 汇聚 */
export type WorkflowParallelRole = 'split' | 'join';

/**
 * 审批人/抄送对象规格：type 决定哪些参数字段生效（后端 AssigneeSpec）。
 * roleCode 语义 = 角色名称（租户内唯一，与 IAM 现行模型一致）。
 */
export interface WorkflowAssigneeSpec {
  type: WorkflowAssigneeType;
  /** 指定用户成员 ID 列表（type=user 必填非空） */
  userIds?: number[];
  /** 角色名称（type=role 必填） */
  roleCode?: string;
  /** 表单用户字段 widgetName（type=form_field 必填） */
  formField?: string;
  /** 部门 ID（type=department / department_manager 必填） */
  deptId?: number;
}

/** 审批节点超时配置（Phase 5）：到期自动动作，1 秒 ~ 30 天 */
export interface WorkflowTimeoutConfig {
  seconds: number;
  action: WorkflowTimeoutAction;
}

/** 审批节点提醒配置（Phase 5）：单次提醒，1 秒 ~ 30 天 */
export interface WorkflowReminderConfig {
  seconds: number;
}

/** service 节点响应映射：JSON 响应取值写入流程变量（标量收敛） */
export interface WorkflowServiceResponseMapping {
  /** 变量名（实例内唯一） */
  variable: string;
  /** 响应 JSON 内点分路径（空 = 整个响应体） */
  path?: string;
  /** 提取失败是否整体失败（缺省 false） */
  required?: boolean;
}

/** service 节点配置（Phase 7，V1 仅出站 HTTP） */
export interface WorkflowServiceConfig {
  /** 动作类型（V1 仅 http） */
  action: string;
  /** HTTP 方法（缺省 POST） */
  method?: string;
  /** 请求地址模板，支持 {{expr}} 插值（form、starter、variables 上下文） */
  url: string;
  /** 请求头（值支持插值；禁止明文敏感头，密钥由平台侧注入） */
  headers?: Record<string, string>;
  /** 请求体模板（JSON 形态，GET/DELETE 忽略；≤64KB） */
  body?: string;
  /** 单次请求超时秒数（缺省 10，1~120） */
  timeoutSeconds?: number;
  /** 失败重试上限（缺省 3，0~8） */
  maxRetries?: number;
  /** 响应 → 流程变量映射（2xx 时执行） */
  responseMapping?: WorkflowServiceResponseMapping[];
}

/** 并行网关配置（Phase 8） */
export interface WorkflowParallelConfig {
  role: WorkflowParallelRole;
}

/** 节点配置：扁平承载各类型配置项，运行时按节点类型裁剪（后端 NodeConfig） */
export interface WorkflowNodeConfig {
  /** 审批模式（approval 必填） */
  approvalMode?: WorkflowApprovalMode;
  /** 审批人规格（approval 必填） */
  assignee?: WorkflowAssigneeSpec;
  /** 驳回策略（可选，缺省 terminate） */
  rejectStrategy?: WorkflowRejectStrategy;
  /** 会签通过比例 (0,1]（countersign 必填），门槛 = ceil(人数 × 比例) */
  passRatio?: number;
  /** 字段权限：widgetName → 权限（approval 可选；未配置字段走运行时默认） */
  formPermissions?: Record<string, WorkflowFieldPermission>;
  /** 超时配置（可选） */
  timeout?: WorkflowTimeoutConfig;
  /** 提醒配置（可选） */
  reminder?: WorkflowReminderConfig;
  /** 抄送对象（cc 必填） */
  recipients?: WorkflowAssigneeSpec;
  /** 服务节点配置（service 必填） */
  service?: WorkflowServiceConfig;
  /** 并行网关配置（parallel 必填） */
  parallel?: WorkflowParallelConfig;
}

/** 出边条件：仅 condition 节点的出边允许携带（后端 EdgeCondition） */
export interface WorkflowEdgeCondition {
  expression: string;
}

/** 设计器画布坐标（settings.designer 私有数据，运行时完全不读取） */
export interface WorkflowPosition {
  x: number;
  y: number;
}

/** 设计器私有设置：节点 key → 画布坐标；随 DSL 文档整体持久化 */
export interface WorkflowDesignerSettings {
  layout?: Record<string, WorkflowPosition>;
}

/** 顶层流程设置：不开放自由键值，字段随里程碑显式追加（后端 Settings） */
export interface WorkflowSettings {
  /** 设计器私有数据（Phase 9）：引擎校验器与运行时忽略，仅编辑体验 */
  designer?: WorkflowDesignerSettings;
}

/** 设计态节点：key 为文档内稳定标识（唯一、发布后不可变），边靠 key 连接 */
export interface WorkflowNode {
  key: string;
  type: WorkflowNodeType;
  name: string;
  config: WorkflowNodeConfig;
}

/** 设计态连线：condition 为空即 default 分支（仅 condition 出边允许条件） */
export interface WorkflowEdge {
  key: string;
  source: string;
  target: string;
  condition?: WorkflowEdgeCondition;
}

/** Workflow DSL v1 顶层结构：草稿与发布快照的唯一事实形态 */
export interface WorkflowDocument {
  schemaVersion: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  settings: WorkflowSettings;
}

/**
 * 表单字段契约：由应用层从表单协议注入（工作流包不依赖表单包）。
 * widgetName 是 formPermissions 的键与 form.* 表达式的取数键；
 * userField 标记成员选择类字段，供「表单用户字段」审批人类型筛选。
 */
export interface WorkflowField {
  widgetName: string;
  label: string;
  required?: boolean;
  userField?: boolean;
}

/** 审批人配置的可选对象目录：应用层从组织/成员 API 注入 */
export interface WorkflowActorOptions {
  /** 可选成员（id = 租户成员 ID，即 AssigneeSpec.userIds 取值域） */
  members: Array<{ id: number; label: string }>;
  /** 可选角色（code = 角色名称，AssigneeSpec.roleCode 语义） */
  roles: Array<{ code: string; label: string }>;
  /** 部门树（id 供 department / department_manager 选择） */
  departments: WorkflowDepartmentOption[];
}

/** 部门选项（树形，children 为子部门） */
export interface WorkflowDepartmentOption {
  id: number;
  label: string;
  children?: WorkflowDepartmentOption[];
}

/** 超时/提醒排期秒数上限（30 天，与后端 MaxJobSeconds 同口径） */
export const WORKFLOW_MAX_JOB_SECONDS = 30 * 24 * 3600;

/** service 单次请求超时上限（秒）与重试上限（与后端冻结口径一致） */
export const WORKFLOW_SERVICE_MAX_TIMEOUT_SECONDS = 120;
export const WORKFLOW_SERVICE_MAX_RETRIES = 8;
