import { http } from '@evolyn.do/utils';

/**
 * 成员信息管理 API（管理后台「成员信息管理」两页签共用）：
 * 字段配置快照读取与单字段即时保存（勾选即生效，无整页保存按钮）。
 * 字段元数据（key/label/type/锁定规则）由服务端字段注册表固定维护，
 * 前端只消费配置值，不能提交字段名称或类型。
 */

/** 字段配置快照中的单字段视图：注册表元数据 + 租户配置值。 */
export interface MemberFieldSettingDto {
  key: string;
  label: string;
  type: string;
  /** 成员在「个人设置」页可见。 */
  personalVisible: boolean;
  /** 成员在「个人设置」页可编辑（仅扩展字段生效，手机/邮箱走绑定流程）。 */
  personalEditable: boolean;
  /** 成员资料卡片展示该字段（服务端按此裁剪卡片数据）。 */
  cardVisible: boolean;
  /** 服务端注册表锁定：可见性固定，管理员不可调整。 */
  visibilityLocked: boolean;
  /** 服务端注册表锁定：编辑权限固定。 */
  editableLocked: boolean;
  /** 服务端注册表锁定：卡片展示固定（如姓名为卡片固定信息）。 */
  cardLocked: boolean;
}

/** 字段配置整页快照：revision 为租户配置版本号，PATCH 时携带做乐观锁。 */
export interface MemberFieldConfigSnapshotDto {
  revision: number;
  fields: MemberFieldSettingDto[];
}

/** 单字段即时更新请求：只提交本次变更的开关与页面读取到的版本号。 */
export interface MemberFieldSettingUpdatePayload {
  personalVisible?: boolean;
  personalEditable?: boolean;
  cardVisible?: boolean;
  revision: number;
}

/** 读取当前租户的完整字段配置快照（15 个预置字段恒完整）。 */
export function getMemberFieldSettings(): Promise<MemberFieldConfigSnapshotDto> {
  return http.get('/member-field-settings');
}

/**
 * 即时更新一个字段的配置：响应返回最新整页快照，调用方以响应覆盖本地状态；
 * revision 过期时后端返回 MEMBER_FIELD_CONFIG_CONFLICT，应刷新后重试。
 */
export function updateMemberFieldSetting(
  fieldKey: string,
  payload: MemberFieldSettingUpdatePayload,
): Promise<MemberFieldConfigSnapshotDto> {
  return http.patch(`/member-field-settings/${fieldKey}`, payload);
}

// ---- 成员扩展档案（成员信息管理二期）----

/** 本人资料视图：values 按个人可见性裁剪，editableKeys 为可提交的扩展字段。 */
export interface MemberProfileViewDto {
  values: Record<string, string>;
  editableKeys: string[];
}

/** 管理员视角的成员资料：全量值 + cardVisible 裁剪的卡片视图 + 字段元数据。 */
export interface MemberProfileAdminViewDto {
  values: Record<string, string>;
  cardValues: Record<string, string>;
  fieldConfig: MemberFieldSettingDto[];
}

/** 读取本人在当前租户的成员资料（个人设置页数据源）。 */
export function getMyMemberProfile(): Promise<MemberProfileViewDto> {
  return http.get('/accounts/me/member-profile');
}

/** 更新本人扩展资料：仅接受 editableKeys 中的字段，其余一律被服务端拒绝。 */
export function updateMyMemberProfile(
  values: Record<string, string>,
): Promise<MemberProfileViewDto> {
  return http.put('/accounts/me/member-profile', { values });
}

/** 管理员读取指定成员的完整资料与卡片裁剪视图。 */
export function getMemberProfile(memberId: number | string): Promise<MemberProfileAdminViewDto> {
  return http.get(`/members/${memberId}/profile`);
}

/** 管理员维护成员扩展资料与企业内编号；identifier 缺省表示不变更。 */
export function updateMemberProfile(
  memberId: number | string,
  payload: { identifier?: string; values?: Record<string, string> },
): Promise<MemberProfileAdminViewDto> {
  return http.put(`/members/${memberId}/profile`, payload);
}
