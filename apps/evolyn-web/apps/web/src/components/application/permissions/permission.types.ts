import type {
  FormPermissionDataScope,
  FormPermissionField,
  FormPermissionFieldRule,
  FormPermissionGroup,
  FormPermissionOperation,
  FormPermissionSubject,
} from '~/api/form';

/** 当前权限页只消费已落地的表单资产；仪表盘权限在资产域落地后再扩展。 */
export type PermissionAssetType = 'workflow-form' | 'form' | 'group';

/** 左栏资产项：由应用菜单接口投影，叶子节点关联表单公开编码。 */
export interface PermissionAsset {
  id: string;
  name: string;
  type: PermissionAssetType;
  /** 叶子表单的公开编码；分组节点为空。 */
  formCode?: string;
  children?: PermissionAsset[];
}

/** 权限组归属主体；成员、部门、角色统一以标签形式呈现。 */
export type PermissionSubject = FormPermissionSubject;

/** 表单记录操作的稳定键；流程表单在普通操作之外可拥有流程专属操作。 */
export type PermissionOperation = FormPermissionOperation;

/** 单字段的可见、可编辑授权；正式接口接入后 field 对应表单 widgetName。 */
export interface PermissionFieldPermission extends FormPermissionFieldRule {
  label: string;
  required?: boolean;
}

/** 数据范围条件的组合方式；条件为空时表达全部数据。 */
export type PermissionDataScope = FormPermissionDataScope;

/** 权限组卡片模型：在表单权限组读取模型上附加字段展示信息。 */
export interface AssetPermissionGroup extends Omit<FormPermissionGroup, 'fieldPermissions'> {
  /** 字段权限编辑器需要字段展示信息；保存时映射回 fieldPermissions。 */
  fields: PermissionFieldPermission[];
}

export type PermissionField = FormPermissionField;

/** 兼容旧成员选择弹窗的提交结构。 */
export interface CreatePermissionGroupPayload {
  groupName: string;
  subjectIds: string[];
}
