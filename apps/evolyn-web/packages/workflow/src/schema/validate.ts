/**
 * 前端即时校验：镜像后端引擎严格校验器（engine/workflow/definition/validator.go）
 * 的核心规则子集，错误码逐字一致（SCHEMA_VERSION_INVALID / KEY_DUPLICATE /
 * EDGE_CONDITION_INVALID / NODE_CONFIG_INVALID …），供设计器在保存/发布前
 * 即时反馈。后端校验始终是发布口径的最终事实源；本层不做的表达式预编译、
 * 图可达性精细分析等由发布时后端兜底并回传 issues。
 */
import type { WorkflowAssigneeSpec, WorkflowDocument, WorkflowNode } from './types';
import {
  WORKFLOW_MAX_JOB_SECONDS,
  WORKFLOW_SERVICE_MAX_RETRIES,
  WORKFLOW_SERVICE_MAX_TIMEOUT_SECONDS,
} from './types';

/** 校验问题：与后端 ValidationErrors 出网负载（issues[{path,code,message}]）同形 */
export interface WorkflowIssue {
  path: string;
  code: string;
  message: string;
}

/** 校验问题定位结果：待高亮的节点 key / 边 key 集合 */
export interface WorkflowIssueTarget {
  nodeKeys: Set<string>;
  edgeKeys: Set<string>;
}

/** 后端错误码目录（与 engine/workflow/definition/validator.go 逐字一致） */
export const WORKFLOW_ISSUE_CODES = {
  SchemaVersion: 'SCHEMA_VERSION_INVALID',
  KeyInvalid: 'KEY_INVALID',
  KeyDuplicate: 'KEY_DUPLICATE',
  NodeTypeInvalid: 'NODE_TYPE_INVALID',
  EdgeRefMissing: 'EDGE_REF_MISSING',
  StartCardinality: 'START_CARDINALITY_INVALID',
  EndCardinality: 'END_CARDINALITY_INVALID',
  EdgeDirection: 'EDGE_DIRECTION_INVALID',
  NodeConfig: 'NODE_CONFIG_INVALID',
  ConditionEdge: 'EDGE_CONDITION_INVALID',
  ExpressionInvalid: 'EXPRESSION_INVALID',
  FieldPermission: 'FIELD_PERMISSION_INVALID',
  NodeUnreachable: 'NODE_UNREACHABLE',
  ParallelConfig: 'PARALLEL_CONFIG_INVALID',
} as const;

/** 汇总文档问题（一次尽量报全，与后端同口径便于画布一次标注） */
export function collectWorkflowIssues(document: WorkflowDocument): WorkflowIssue[] {
  const issues: WorkflowIssue[] = [];
  const nodeKeys = new Set<string>();
  const nodesByKey = new Map<string, WorkflowNode>();

  document.nodes.forEach((node, index) => {
    const base = `nodes[${index}]`;
    if (!node.key.trim()) {
      issues.push(issue(base, WORKFLOW_ISSUE_CODES.KeyInvalid, '节点 key 不能为空'));
    } else if (nodeKeys.has(node.key)) {
      issues.push(issue(base, WORKFLOW_ISSUE_CODES.KeyDuplicate, `节点 key 重复：${node.key}`));
    }
    nodeKeys.add(node.key);
    nodesByKey.set(node.key, node);
  });

  // start/end 基数：有且只有一个 start、至少一个 end
  const startNodes = document.nodes.filter((node) => node.type === 'start');
  if (startNodes.length !== 1) {
    issues.push(
      issue(
        'nodes',
        WORKFLOW_ISSUE_CODES.StartCardinality,
        startNodes.length === 0 ? '必须且只能存在一个 start 节点' : 'start 节点只允许存在一个',
      ),
    );
  }
  if (!document.nodes.some((node) => node.type === 'end')) {
    issues.push(issue('nodes', WORKFLOW_ISSUE_CODES.EndCardinality, '至少存在一个 end 节点'));
  }

  const edgeKeys = new Set<string>();
  document.edges.forEach((edge, index) => {
    const base = `edges[${index}]`;
    if (!edge.key.trim()) {
      issues.push(issue(base, WORKFLOW_ISSUE_CODES.KeyInvalid, '连线 key 不能为空'));
    } else if (edgeKeys.has(edge.key)) {
      issues.push(issue(base, WORKFLOW_ISSUE_CODES.KeyDuplicate, `连线 key 重复：${edge.key}`));
    }
    edgeKeys.add(edge.key);

    if (!nodesByKey.has(edge.source) || !nodesByKey.has(edge.target)) {
      issues.push(
        issue(base, WORKFLOW_ISSUE_CODES.EdgeRefMissing, '连线的 source/target 节点不存在'),
      );
      return;
    }
    if (edge.source === edge.target) {
      issues.push(issue(base, WORKFLOW_ISSUE_CODES.EdgeRefMissing, '不允许自环连线'));
    }
    if (nodesByKey.get(edge.source)?.type === 'start') {
      // start 不允许入边在下方统一检查；此处先校验 end 出边
    }
    if (nodesByKey.get(edge.source)?.type === 'end') {
      issues.push(issue(base, WORKFLOW_ISSUE_CODES.EdgeDirection, 'end 节点不允许出边'));
    }
    if (nodesByKey.get(edge.target)?.type === 'start') {
      issues.push(issue(base, WORKFLOW_ISSUE_CODES.EdgeDirection, 'start 节点不允许入边'));
    }

    // 条件语义：仅 condition 出边可携带条件；condition 出边必须有 default 分支
    const sourceNode = nodesByKey.get(edge.source);
    const isConditionSource = sourceNode?.type === 'condition';
    if (edge.condition) {
      if (!isConditionSource) {
        issues.push(
          issue(
            `${base}.condition`,
            WORKFLOW_ISSUE_CODES.ConditionEdge,
            '仅条件分支节点的出边允许携带条件',
          ),
        );
      } else if (!edge.condition.expression.trim()) {
        issues.push(
          issue(
            `${base}.condition.expression`,
            WORKFLOW_ISSUE_CODES.ExpressionInvalid,
            '条件表达式不能为空',
          ),
        );
      }
    }
    if (isConditionSource) {
      const siblings = document.edges.filter((item) => item.source === edge.source);
      const hasDefault = siblings.some((item) => !item.condition);
      const hasExpression = siblings.some((item) => item.condition);
      if (!hasDefault) {
        issues.push(
          issue(
            `nodes.${edge.source}`,
            WORKFLOW_ISSUE_CODES.ConditionEdge,
            '条件分支必须保留一条默认（无条件）出边',
          ),
        );
      } else if (siblings.length > 1 && !hasExpression) {
        issues.push(
          issue(
            `nodes.${edge.source}`,
            WORKFLOW_ISSUE_CODES.ConditionEdge,
            '条件分支的多条出边中，非默认出边必须携带条件表达式',
          ),
        );
      }
    }
  });

  // 从 start 可达性：不可达节点直接标注（后端同口径 NODE_UNREACHABLE）
  const reachable = reachableKeys(document);
  document.nodes.forEach((node, index) => {
    if (node.type !== 'start' && !reachable.has(node.key)) {
      issues.push(
        issue(
          `nodes[${index}]`,
          WORKFLOW_ISSUE_CODES.NodeUnreachable,
          `节点 ${node.name} 从发起节点不可达`,
        ),
      );
    }
  });

  // 按节点类型校验配置必填项
  document.nodes.forEach((node, index) => {
    const base = `nodes[${index}].config`;
    if (node.type === 'approval') {
      if (!node.config.approvalMode) {
        issues.push(issue(`${base}.approvalMode`, WORKFLOW_ISSUE_CODES.NodeConfig, '审批模式必填'));
      }
      if (node.config.approvalMode === 'countersign') {
        const ratio = node.config.passRatio;
        if (ratio === undefined || ratio <= 0 || ratio > 1) {
          issues.push(
            issue(
              `${base}.passRatio`,
              WORKFLOW_ISSUE_CODES.NodeConfig,
              '会签必须配置 (0,1] 内的通过比例',
            ),
          );
        }
      }
      collectAssigneeIssues(node.config.assignee, `${base}.assignee`, '审批人', issues);
    } else if (node.type === 'cc') {
      collectAssigneeIssues(node.config.recipients, `${base}.recipients`, '抄送对象', issues);
    } else if (node.type === 'service') {
      const service = node.config.service;
      if (!service || !/^https?:\/\//.test(service.url.trim())) {
        issues.push(
          issue(
            `${base}.service.url`,
            WORKFLOW_ISSUE_CODES.NodeConfig,
            '服务调用必须配置 http(s) 请求地址',
          ),
        );
      }
      if (service && (service.timeoutSeconds ?? 10) > WORKFLOW_SERVICE_MAX_TIMEOUT_SECONDS) {
        issues.push(
          issue(
            `${base}.service.timeoutSeconds`,
            WORKFLOW_ISSUE_CODES.NodeConfig,
            `请求超时不能超过 ${WORKFLOW_SERVICE_MAX_TIMEOUT_SECONDS} 秒`,
          ),
        );
      }
      if (service && (service.maxRetries ?? 0) > WORKFLOW_SERVICE_MAX_RETRIES) {
        issues.push(
          issue(
            `${base}.service.maxRetries`,
            WORKFLOW_ISSUE_CODES.NodeConfig,
            `重试上限不能超过 ${WORKFLOW_SERVICE_MAX_RETRIES} 次`,
          ),
        );
      }
    } else if (node.type === 'parallel') {
      if (!node.config.parallel || !['split', 'join'].includes(node.config.parallel.role)) {
        issues.push(
          issue(
            `${base}.parallel.role`,
            WORKFLOW_ISSUE_CODES.ParallelConfig,
            '并行网关必须声明 split/join 角色',
          ),
        );
      }
    }
    if (
      node.config.timeout &&
      (node.config.timeout.seconds < 1 || node.config.timeout.seconds > WORKFLOW_MAX_JOB_SECONDS)
    ) {
      issues.push(
        issue(
          `${base}.timeout.seconds`,
          WORKFLOW_ISSUE_CODES.NodeConfig,
          '超时时间必须在 1 秒 ~ 30 天之间',
        ),
      );
    }
    if (
      node.config.reminder &&
      (node.config.reminder.seconds < 1 || node.config.reminder.seconds > WORKFLOW_MAX_JOB_SECONDS)
    ) {
      issues.push(
        issue(
          `${base}.reminder.seconds`,
          WORKFLOW_ISSUE_CODES.NodeConfig,
          '提醒时间必须在 1 秒 ~ 30 天之间',
        ),
      );
    }
    // 字段权限值域（结构层已收敛为四档，这里防御脏数据）
    if (node.config.formPermissions) {
      for (const [fieldName, permission] of Object.entries(node.config.formPermissions)) {
        if (!['hidden', 'readonly', 'editable', 'required'].includes(permission)) {
          issues.push(
            issue(
              `${base}.formPermissions.${fieldName}`,
              WORKFLOW_ISSUE_CODES.FieldPermission,
              '字段权限值无效',
            ),
          );
        }
      }
    }
  });

  return issues;
}

/** 审批人规格校验：type 必填参数逐项检查（与后端按类型校验口径一致） */
function collectAssigneeIssues(
  spec: WorkflowAssigneeSpec | undefined,
  base: string,
  label: string,
  issues: WorkflowIssue[],
) {
  if (!spec || !spec.type) {
    issues.push(issue(base, WORKFLOW_ISSUE_CODES.NodeConfig, `${label}必填`));
    return;
  }
  switch (spec.type) {
    case 'user':
      if (!spec.userIds?.length) {
        issues.push(
          issue(`${base}.userIds`, WORKFLOW_ISSUE_CODES.NodeConfig, `${label}需选择至少一名成员`),
        );
      }
      break;
    case 'role':
      if (!spec.roleCode?.trim()) {
        issues.push(
          issue(`${base}.roleCode`, WORKFLOW_ISSUE_CODES.NodeConfig, `${label}需选择角色`),
        );
      }
      break;
    case 'form_field':
      if (!spec.formField?.trim()) {
        issues.push(
          issue(`${base}.formField`, WORKFLOW_ISSUE_CODES.NodeConfig, `${label}需选择表单用户字段`),
        );
      }
      break;
    case 'department':
    case 'department_manager':
      if (!spec.deptId) {
        issues.push(issue(`${base}.deptId`, WORKFLOW_ISSUE_CODES.NodeConfig, `${label}需选择部门`));
      }
      break;
    default:
      break;
  }
}

/**
 * 把校验问题（前端即时或后端回传）定位到画布元素：
 * path 形如 nodes[2].config.assignee / edges[0].condition.expression，
 * 解析出节点 key 与边 key 两个集合供画布高亮。
 */
export function resolveIssueTargets(
  document: WorkflowDocument,
  issues: readonly WorkflowIssue[],
): WorkflowIssueTarget {
  const nodeKeys = new Set<string>();
  const edgeKeys = new Set<string>();
  for (const item of issues) {
    const nodeMatch = /^nodes\[(\d+)\]/.exec(item.path);
    if (nodeMatch) {
      const node = document.nodes[Number(nodeMatch[1])];
      if (node) nodeKeys.add(node.key);
      continue;
    }
    const edgeMatch = /^edges\[(\d+)\]/.exec(item.path);
    if (edgeMatch) {
      const edge = document.edges[Number(edgeMatch[1])];
      if (edge) edgeKeys.add(edge.key);
    }
  }
  return { nodeKeys, edgeKeys };
}

/** 从 start 出发沿边可达的节点 key 集合（BFS；环安全） */
function reachableKeys(document: WorkflowDocument): Set<string> {
  const outgoing = new Map<string, string[]>();
  for (const edge of document.edges) {
    outgoing.set(edge.source, [...(outgoing.get(edge.source) ?? []), edge.target]);
  }
  const visited = new Set<string>();
  const queue = document.nodes.filter((node) => node.type === 'start').map((node) => node.key);
  while (queue.length) {
    const key = queue.shift()!;
    if (visited.has(key)) continue;
    visited.add(key);
    for (const target of outgoing.get(key) ?? []) {
      if (!visited.has(target)) queue.push(target);
    }
  }
  return visited;
}

function issue(path: string, code: string, message: string): WorkflowIssue {
  return { path, code, message };
}
