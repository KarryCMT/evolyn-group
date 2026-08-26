import { http } from '@evolyn.do/utils';
import type { MemberListQuery, MemberPageDto } from './member';

export interface RoleGroupDto {
  id: number;
  name: string;
}

export interface OrganizationRoleDto {
  id: number;
  name: string;
  groupId: number;
}

interface RoleMutationDto {
  id: number;
  name: string;
  roleGroupId: number | null;
}

export interface OrganizationRoleTreeDto {
  groups: Array<RoleGroupDto & { roles: OrganizationRoleDto[] }>;
}

/** 查询内部组织页所需的角色组与角色树。 */
export function getOrganizationRoleTree(): Promise<OrganizationRoleTreeDto> {
  return http.get('/roles/tree');
}

/** 创建角色展示分组，不影响原有权限分组。 */
export function createRoleGroup(name: string): Promise<RoleGroupDto> {
  return http.post('/roles/groups', { name });
}

/** 修改角色展示分组名称。 */
export function renameRoleGroup(id: string, name: string): Promise<RoleGroupDto> {
  return http.put(`/roles/groups/${id}`, { name });
}

/** 删除角色展示分组，组内角色由服务端归回默认角色组。 */
export function deleteRoleGroup(id: string): Promise<null> {
  return http.delete(`/roles/groups/${id}`);
}

/** 保存角色展示分组的完整拖拽排序。 */
export function reorderRoleGroups(groupIds: string[]): Promise<null> {
  return http.put('/roles/groups/order', { groupIds: groupIds.map(Number) });
}

/** 在指定角色组内创建角色。 */
export function createOrganizationRole(input: {
  name: string;
  groupId: number;
}): Promise<RoleMutationDto> {
  return http.post('/roles/organization', input);
}

/** 从角色组操作菜单直接添加角色。 */
export function createRoleInGroup(groupId: string, name: string): Promise<RoleMutationDto> {
  return http.post(`/roles/groups/${groupId}/roles`, { name });
}

/** 修改角色名称。 */
export function renameRole(id: string, name: string): Promise<RoleMutationDto> {
  return http.put(`/roles/${id}/name`, { name });
}

/** 将角色移动到指定展示分组。 */
export function moveRoleToGroup(id: string, groupId: string): Promise<RoleMutationDto> {
  return http.put(`/roles/${id}/group`, { groupId });
}

/** 删除角色。 */
export function deleteOrganizationRole(id: string): Promise<null> {
  return http.delete(`/roles/${id}`);
}

/** 保存单个角色组内的完整角色拖拽排序。 */
export function reorderOrganizationRoles(groupId: string, roleIds: string[]): Promise<null> {
  return http.put(`/roles/groups/${groupId}/roles/order`, { roleIds: roleIds.map(Number) });
}

/** 分页查询直接绑定指定角色的成员。 */
export function listRoleMembers(
  roleId: string,
  query: Omit<MemberListQuery, 'departmentId'> = {},
): Promise<MemberPageDto> {
  return http.get(`/roles/${roleId}/members`, query);
}

/** 批量为角色添加成员。 */
export function addRoleMembers(roleId: string, memberIds: string[]): Promise<null> {
  return http.post(`/roles/${roleId}/members`, { memberIds: memberIds.map(Number) });
}

/** 解除成员与角色的直接绑定。 */
export function removeRoleMember(roleId: string, memberId: string): Promise<null> {
  return http.delete(`/roles/${roleId}/members/${memberId}`);
}
