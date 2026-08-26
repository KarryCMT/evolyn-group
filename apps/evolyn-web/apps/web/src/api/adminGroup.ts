import { http } from '@evolyn.do/utils';

/**
 * 权限中心-管理员模块 API（管理组）：
 * 系统管理员页（scope=system）与灵衍云管理员页（scope=application）共用。
 * 交互语义为「勾选/选择确认即保存」：PATCH 每次只提交一个配置区块，
 * 区块整体替换；响应返回最新详情，调用方以响应覆盖本地状态。
 */

/** 管理组类型：system=系统管理员页（通讯录管理组），application=灵衍云管理员页（普通管理组）。 */
export type AdminGroupScope = 'system' | 'application';

/** 范围模式：all=全部，partial=部分（ID 清单）。 */
export type AdminScopeMode = 'all' | 'partial';

/** 管理组概要（列表项）：内置系统管理员组恒排最前。 */
export interface AdminGroupSummaryDto {
  id: number;
  name: string;
  scope: AdminGroupScope;
  builtIn: boolean;
  memberCount: number;
}

/** 管理组成员展示项：name 为租户内展示名，department 为首个部门名。 */
export interface AdminGroupMemberDto {
  id: number;
  name: string;
  department: string;
}

/** 部门范围：system 组为通讯录管理范围，application 组为使用权分发范围。 */
export interface AdminDepartmentScopeDto {
  enabled: boolean;
  mode: AdminScopeMode;
  departmentIds: number[];
}

/** 角色范围：可见/可管理为两项独立授权（可管理必然隐含可见）。 */
export interface AdminRoleScopeDto {
  visible: boolean;
  manage: boolean;
  mode: AdminScopeMode;
  roleIds: number[];
}

/** 互联组织范围：互联组织域未上线前仅保存配置。 */
export interface AdminExternalOrgScopeDto {
  enabled: boolean;
}

/** 应用范围（仅 application 组）：allApplications=true 为语义全量，新建应用自动纳入。 */
export interface AdminApplicationScopeDto {
  allApplications: boolean;
  applicationIds: number[];
  manage: boolean;
}

/** 通讯录管理子配置（仅 application 组的设置抽屉），与主行分发范围解耦。 */
export interface AdminAddressBookScopeDto {
  departmentEnabled: boolean;
  roleVisible: boolean;
  roleManage: boolean;
  externalEnabled: boolean;
}

/**
 * 管理组详情（权限面板读模型）：字段与页面本地状态一一对应；
 * 各类 ID 清单为原始 ID，展示名由前端结合部门树/角色树/应用列表映射。
 */
export interface AdminGroupDetailDto {
  id: number;
  name: string;
  scope: AdminGroupScope;
  builtIn: boolean;
  members: AdminGroupMemberDto[];
  departmentEnabled: boolean;
  departmentMode: AdminScopeMode;
  departmentIds: number[];
  roleVisible: boolean;
  roleManage: boolean;
  roleMode: AdminScopeMode;
  roleIds: number[];
  externalEnabled: boolean;
  /** 以下仅 application 组返回。 */
  applicationIds?: number[];
  allApplications: boolean;
  applicationManage: boolean;
  addressBook?: AdminAddressBookScopeDto;
}

/**
 * 分区块即时保存请求：至多携带一个区块，区块整体替换。
 * 内置系统管理员组仅允许 members（且至少保留一名管理员）。
 */
export interface AdminGroupPatchPayload {
  name?: string;
  members?: number[];
  departmentScope?: AdminDepartmentScopeDto;
  roleScope?: AdminRoleScopeDto;
  externalOrg?: AdminExternalOrgScopeDto;
  applicationScope?: AdminApplicationScopeDto;
  addressBook?: AdminAddressBookScopeDto;
}

/** 当前成员的管理组身份聚合（页面入口/菜单控制数据源）。 */
export interface MemberAdminScopesDto {
  /** 是否系统管理员（内置系统管理员组，即租户管理员）。 */
  systemAdmin: boolean;
  /** 所属自定义管理组概要。 */
  groups: AdminGroupSummaryDto[];
}

/** 按类型查询管理组列表；缺省 scope 返回全部（内置组在最前）。 */
export function listAdminGroups(scope?: AdminGroupScope): Promise<AdminGroupSummaryDto[]> {
  return http.get('/admin-groups', scope ? { scope } : {});
}

/** 管理组详情：成员展示视图 + 范围配置展开。 */
export function getAdminGroup(id: number | string): Promise<AdminGroupDetailDto> {
  return http.get(`/admin-groups/${id}`);
}

/** 创建自定义管理组（仅名称与类型，范围配置随后即时保存）。 */
export function createAdminGroup(payload: {
  scope: AdminGroupScope;
  name: string;
}): Promise<AdminGroupDetailDto> {
  return http.post('/admin-groups', payload);
}

/**
 * 即时更新一个配置区块：响应返回最新详情，调用方以响应覆盖本地状态。
 * 内置组改名/删改配置返回 ADMIN_GROUP_BUILTIN_IMMUTABLE；
 * 清空系统管理员返回 ADMIN_GROUP_LAST_ADMIN。
 */
export function updateAdminGroup(
  id: number | string,
  payload: AdminGroupPatchPayload,
): Promise<AdminGroupDetailDto> {
  return http.patch(`/admin-groups/${id}`, payload);
}

/** 删除自定义管理组（内置组不可删除）。 */
export function deleteAdminGroup(id: number | string): Promise<null> {
  return http.delete(`/admin-groups/${id}`);
}

/** 查询当前成员的管理组身份（管理后台菜单/页面入口控制）。 */
export function getMyAdminScopes(): Promise<MemberAdminScopesDto> {
  return http.get('/auth/admin-scopes');
}
