/** 应用管理组中的成员、部门、角色均以统一的标签模型在界面中展示。 */
export interface ManagementGroupSubject {
  id: string;
  name: string;
  type: 'member' | 'department' | 'role';
}

/** 管理组的权限模式：可覆盖全应用，或只覆盖指定表单、流程和仪表盘。 */
export type ManagementGroupPermissionMode = 'all' | 'partial';

/** 页面本地预览使用的管理组模型；后续接入接口时可直接映射为读取模型。 */
export interface ManagementGroup {
  id: string;
  name: string;
  description: string;
  managers: ManagementGroupSubject[];
  permissionMode: ManagementGroupPermissionMode;
  assetIds: string[];
  departmentIds: string[];
  roleIds: string[];
}

/** 编辑弹窗提交给页面的草稿，页面负责保存并刷新列表。 */
export type ManagementGroupDraft = Omit<ManagementGroup, 'id'>;
