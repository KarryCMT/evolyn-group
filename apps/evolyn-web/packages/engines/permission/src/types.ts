/** 后端授权结果在前端的字段级投影；仅控制交互，不构成授权边界。 */
export interface FieldPermission {
  visible: boolean;
  editable: boolean;
}

export type FieldPermissionMap = Readonly<Record<string, FieldPermission>>;

export type MissingFieldPermissionStrategy = 'allow' | 'deny';
