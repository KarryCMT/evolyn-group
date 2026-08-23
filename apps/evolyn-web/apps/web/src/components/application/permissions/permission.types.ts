/** 权限中心展示的应用资产类型；接口接入后与资产域的稳定类型一一对应。 */
export type PermissionAssetType = 'workflow-form' | 'form' | 'dashboard';

/** 左栏资产项：当前为前端交互演示数据，后续由资产列表接口替换。 */
export interface PermissionAsset {
  id: string;
  name: string;
  type: PermissionAssetType;
}

/** 权限组归属主体；成员、部门、角色统一以标签形式呈现。 */
export interface PermissionSubject {
  id: string;
  name: string;
  type: 'member' | 'department' | 'role';
}

/** 权限组卡片模型；后续应映射 asset_permission_groups 的读取模型。 */
export interface AssetPermissionGroup {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  subjects: PermissionSubject[];
}

/** 新增权限组弹窗的前端提交结构，保留给后续 API 接入。 */
export interface CreatePermissionGroupPayload {
  groupName: string;
  subjectIds: string[];
}
