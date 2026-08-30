/** 权限中心展示的应用资产类型；接口接入后与资产域的稳定类型一一对应。 */
export type PermissionAssetType = 'workflow-form' | 'form' | 'dashboard' | 'group';

/** 左栏资产项：当前为前端交互演示数据，后续由资产列表接口替换。 */
export interface PermissionAsset {
  id: string;
  name: string;
  type: PermissionAssetType;
  children?: PermissionAsset[];
}

/** 权限组归属主体；成员、部门、角色统一以标签形式呈现。 */
export interface PermissionSubject {
  id: string;
  name: string;
  type: 'member' | 'department' | 'role';
}

/** 表单记录操作的稳定键；流程表单在普通操作之外可拥有流程专属操作。 */
export type PermissionOperation =
  | 'view'
  | 'add'
  | 'copy'
  | 'edit'
  | 'delete'
  | 'batch_print'
  | 'batch_modify'
  | 'import'
  | 'export'
  | 'workflow_owner_transfer'
  | 'workflow_terminate'
  | 'workflow_activate';

/** 单字段的可见、可编辑授权；正式接口接入后 field 对应表单 widgetName。 */
export interface PermissionFieldPermission {
  field: string;
  label: string;
  required?: boolean;
  visible: boolean;
  editable: boolean;
}

/** 数据范围条件的组合方式；条件为空时表达全部数据。 */
export interface PermissionDataScope {
  match: 'all' | 'any';
}

/** 权限组卡片模型；后续应映射 asset_permission_groups 的读取模型。 */
export interface AssetPermissionGroup {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  subjects: PermissionSubject[];
  operations?: PermissionOperation[];
  fields?: PermissionFieldPermission[];
  dataScope?: PermissionDataScope;
}

/** 新增权限组弹窗的前端提交结构，保留给后续 API 接入。 */
export interface CreatePermissionGroupPayload {
  groupName: string;
  subjectIds: string[];
}
