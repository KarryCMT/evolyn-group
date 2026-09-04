import type {
  FieldPermission,
  FieldPermissionMap,
  MissingFieldPermissionStrategy,
} from './types.js';

const ALLOW: Readonly<FieldPermission> = { visible: true, editable: true };
const DENY: Readonly<FieldPermission> = { visible: false, editable: false };

/**
 * 将后端字段权限矩阵投影为可供任意运行时消费的单字段能力。
 * 当矩阵不存在时，由调用方明确选择预览场景的 allow 或业务场景的 deny 策略，
 * 禁止散落的空值回退导致权限语义漂移。
 */
export function resolveFieldPermission(
  permissions: FieldPermissionMap | undefined,
  field: string,
  missing: MissingFieldPermissionStrategy,
): Readonly<FieldPermission> {
  if (!permissions) return missing === 'allow' ? ALLOW : DENY;
  return permissions[field] ?? DENY;
}
