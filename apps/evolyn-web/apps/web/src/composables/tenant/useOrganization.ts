import { computed, reactive, shallowRef } from 'vue';
import {
  createDepartment as createDepartmentApi,
  getDepartmentTree,
  updateDepartment as updateDepartmentApi,
} from '~/api/department';
import type { DepartmentDto } from '~/api/department';
import { listMembers, updateMemberStatus } from '~/api/member';
import type { MemberListItemDto, MemberStatus } from '~/api/member';
import {
  addRoleMembers as addRoleMembersApi,
  createOrganizationRole,
  createRoleInGroup,
  createRoleGroup as createRoleGroupApi,
  deleteOrganizationRole as deleteOrganizationRoleApi,
  deleteRoleGroup as deleteRoleGroupApi,
  getOrganizationRoleTree,
  listRoleMembers,
  moveRoleToGroup,
  renameRoleGroup as renameRoleGroupApi,
  removeRoleMember as removeRoleMemberApi,
  reorderRoleGroups as reorderRoleGroupsApi,
  reorderOrganizationRoles as reorderOrganizationRolesApi,
  renameRole as renameRoleApi,
} from '~/api/role';
import type {
  OrganizationDepartment,
  OrganizationMember,
  OrganizationMode,
  OrganizationRole,
  OrganizationRoleGroup,
  OrganizationSelection,
} from '~/components/tenant/organization/organization.types';

const rootDepartment: OrganizationDepartment = {
  id: 'tenant-root',
  name: '重庆万柯互联网科技有限责任公司',
  isTenantRoot: true,
  children: [
    { id: 'dept-rd', name: '研发部' },
    { id: 'dept-product', name: '产品部' },
  ],
};

/** 内部组织页面的状态与数据读取边界。部门、角色和成员均以接口数据为准。 */
export function useOrganization() {
  const mode = shallowRef<OrganizationMode>('department');
  const selection = shallowRef<OrganizationSelection>({
    mode: 'department',
    id: 'all-members',
    name: '全部成员',
  });
  const departments = shallowRef<OrganizationDepartment>(rootDepartment);
  const roleGroups = shallowRef<OrganizationRoleGroup[]>([]);
  const roles = shallowRef<OrganizationRole[]>([]);
  const members = shallowRef<OrganizationMember[]>([]);
  const availableMembers = shallowRef<OrganizationMember[]>([]);
  const memberTotal = shallowRef(0);
  const memberPage = shallowRef(1);
  const filters = reactive({
    departmentKeyword: '',
    roleKeyword: '',
    memberKeyword: '',
    status: 'all' as 'all' | Exclude<MemberStatus, 'resigned'>,
  });

  const currentRoles = computed(() => {
    return roles.value.filter((item) => item.groupId === selection.value.id);
  });

  const filteredMembers = computed(() => {
    const selected = selection.value;
    if (selected.mode === 'role')
      return members.value.filter((member) => member.roleIds.includes(selected.id));
    return members.value;
  });

  function switchMode(nextMode: OrganizationMode) {
    mode.value = nextMode;
    filters.memberKeyword = '';
    const firstRole = roles.value[0];
    selection.value =
      nextMode === 'department'
        ? { mode: 'department', id: 'all-members', name: '全部成员' }
        : firstRole
          ? { mode: 'role', id: firstRole.id, name: firstRole.name }
          : { mode: 'role', id: '', name: '角色' };
    memberPage.value = 1;
  }

  function selectItem(nextSelection: OrganizationSelection) {
    selection.value = nextSelection;
    memberPage.value = 1;
  }

  function toOrganizationMember(member: MemberListItemDto): OrganizationMember {
    return {
      id: String(member.id),
      accountId: String(member.accountId),
      name: member.name,
      phone: member.phone,
      email: member.email,
      avatar: member.avatar,
      department:
        member.departments.map((department) => department.name).join('、') || '未分配部门',
      departmentIds: member.departments.map((department) => String(department.id)),
      roleIds: member.roles.map((role) => String(role.id)),
      roleNames: member.roles.map((role) => role.name),
      status: member.status,
      resignedAt: member.resignedAt,
      employeeNo: String(member.id),
      alias: '',
      gender: '',
    };
  }

  /** 以当前左侧选中项和筛选栏参数查询成员，部门始终按稳定 ID 传递。 */
  async function loadMembers() {
    const selected = selection.value;
    const departmentID =
      selected.mode === 'department' &&
      selected.id !== 'all-members' &&
      selected.id !== 'resigned-members' &&
      selected.id !== departments.value.id
        ? Number(selected.id)
        : undefined;
    const memberStatus: MemberStatus | undefined =
      selected.id === 'resigned-members'
        ? 'resigned'
        : filters.status === 'all'
          ? undefined
          : filters.status;
    const query = {
      status: memberStatus,
      keyword: filters.memberKeyword.trim() || undefined,
      page: memberPage.value,
      pageSize: 20,
    };
    const page =
      selected.mode === 'role' && selected.id
        ? await listRoleMembers(selected.id, query)
        : await listMembers({
            ...query,
            departmentId: Number.isFinite(departmentID) ? departmentID : undefined,
          });
    members.value = page.items.map(toOrganizationMember);
    memberTotal.value = page.total;
  }

  /** 角色成员选择器需展示租户内全部可选成员，不复用当前角色的分页结果。 */
  async function loadAvailableMembers() {
    const page = await listMembers({ page: 1, pageSize: 100 });
    availableMembers.value = page.items.map(toOrganizationMember);
  }

  async function loadRoles() {
    const tree = await getOrganizationRoleTree();
    roleGroups.value = tree.groups.map((group) => ({ id: String(group.id), name: group.name }));
    roles.value = tree.groups.flatMap((group) =>
      group.roles.map((role) => ({
        id: String(role.id),
        name: role.name,
        groupId: String(role.groupId),
      })),
    );
  }

  async function changeMemberStatus(member: OrganizationMember, status: MemberStatus) {
    await updateMemberStatus(member.id, status);
    await loadMembers();
  }

  function setMemberPage(page: number) {
    memberPage.value = page;
  }

  function toOrganizationDepartment(department: DepartmentDto): OrganizationDepartment {
    return {
      id: String(department.id),
      name: department.name,
      parentId: department.parentId === null ? null : String(department.parentId),
      order: department.order,
      status: department.status,
      children: department.children?.map(toOrganizationDepartment),
    };
  }

  /** 组织根节点由租户资料构造，部门树只承载其下属节点。 */
  async function loadDepartments(tenantName: string) {
    const tree = await getDepartmentTree();
    departments.value = {
      id: 'tenant-root',
      name: tenantName,
      isTenantRoot: true,
      children: tree.map(toOrganizationDepartment),
    };
  }

  function setTenantRootName(name: string) {
    departments.value = { ...departments.value, name };
    if (selection.value.id === departments.value.id) {
      selection.value = { ...selection.value, name };
    }
  }

  /** 不直接修改树节点，始终沿父链复制，保持 shallowRef 的更新可追踪。 */
  function updateDepartment(
    current: OrganizationDepartment,
    departmentId: string,
    updater: (department: OrganizationDepartment) => OrganizationDepartment,
  ): OrganizationDepartment {
    if (current.id === departmentId) return updater(current);
    if (!current.children) return current;
    return {
      ...current,
      children: current.children.map((child) => updateDepartment(child, departmentId, updater)),
    };
  }

  async function renameDepartment(department: OrganizationDepartment, name: string) {
    const departmentId = Number(department.id);
    const updated = await updateDepartmentApi(departmentId, {
      name,
      parentId: department.parentId ? Number(department.parentId) : null,
      order: department.order ?? 0,
      status: department.status ?? 'active',
    });
    departments.value = updateDepartment(departments.value, department.id, (current) => ({
      ...current,
      ...toOrganizationDepartment(updated),
    }));
    if (selection.value.id === department.id) {
      selection.value = { ...selection.value, name: updated.name };
    }
  }

  async function createChildDepartment(parentId: string, name: string) {
    const created = await createDepartmentApi({
      name,
      parentId: parentId === departments.value.id ? null : Number(parentId),
      order: 0,
      status: 'active',
    });
    const child = toOrganizationDepartment(created);
    departments.value = updateDepartment(departments.value, parentId, (department) => ({
      ...department,
      children: [...(department.children ?? []), child],
    }));
    selection.value = { mode: 'department', id: child.id, name: child.name };
  }

  function roleName(roleId: string) {
    return roles.value.find((role) => role.id === roleId)?.name ?? '';
  }

  async function addMembers(memberIds: string[]) {
    if (selection.value.mode !== 'role' || !selection.value.id) return;
    await addRoleMembersApi(selection.value.id, memberIds);
    await loadMembers();
  }

  async function removeMember(memberId: string) {
    if (selection.value.mode !== 'role' || !selection.value.id) return;
    await removeRoleMemberApi(selection.value.id, memberId);
    await loadMembers();
  }

  function updateMember(member: OrganizationMember) {
    members.value = members.value.map((item) => (item.id === member.id ? member : item));
  }

  async function createRoleGroup(name: string) {
    const group = await createRoleGroupApi(name);
    roleGroups.value = [...roleGroups.value, { id: String(group.id), name: group.name }];
  }

  async function createRole(name: string, targetGroupId?: string) {
    const groupId = targetGroupId ?? roleGroups.value[0]?.id;
    if (!groupId) return;
    const created = targetGroupId
      ? await createRoleInGroup(targetGroupId, name)
      : await createOrganizationRole({ name, groupId: Number(groupId) });
    const role = { id: String(created.id), name: created.name, groupId };
    roles.value = [...roles.value, role];
    selection.value = { mode: 'role', id: role.id, name: role.name };
  }

  async function renameRole(name: string) {
    if (selection.value.mode !== 'role' || !selection.value.id) return;
    const updated = await renameRoleApi(selection.value.id, name);
    roles.value = roles.value.map((role) =>
      role.id === selection.value.id ? { ...role, name: updated.name } : role,
    );
    selection.value = { ...selection.value, name: updated.name };
  }

  async function renameRoleGroup(group: OrganizationRoleGroup, name: string) {
    const updated = await renameRoleGroupApi(group.id, name);
    roleGroups.value = roleGroups.value.map((item) =>
      item.id === group.id ? { ...item, name: updated.name } : item,
    );
  }

  async function deleteRoleGroup(group: OrganizationRoleGroup) {
    await deleteRoleGroupApi(group.id);
    // 服务端将组内角色迁回默认组，重读树以保持角色归属与排序完全一致。
    await loadRoles();
  }

  async function reorderRoleGroups(groupIds: string[]) {
    const previous = roleGroups.value;
    const groupByID = new Map(previous.map((group) => [group.id, group]));
    const next = groupIds
      .map((id) => groupByID.get(id))
      .filter((group): group is OrganizationRoleGroup => Boolean(group));
    if (next.length !== previous.length) return;
    roleGroups.value = next;
    try {
      await reorderRoleGroupsApi(groupIds);
    } catch (error) {
      roleGroups.value = previous;
      throw error;
    }
  }

  async function moveRole(targetGroupId: string) {
    if (selection.value.mode !== 'role' || !selection.value.id) return;
    await moveRoleToGroup(selection.value.id, targetGroupId);
    roles.value = roles.value.map((role) =>
      role.id === selection.value.id ? { ...role, groupId: targetGroupId } : role,
    );
  }

  async function deleteRole(role: OrganizationRole) {
    await deleteOrganizationRoleApi(role.id);
    await loadRoles();
    if (selection.value.id === role.id) {
      const nextRole = roles.value[0];
      selection.value = nextRole
        ? { mode: 'role', id: nextRole.id, name: nextRole.name }
        : { mode: 'role', id: '', name: '角色' };
    }
  }

  async function reorderRoles(groupId: string, roleIds: string[]) {
    const previous = roles.value;
    const grouped = previous.filter((role) => role.groupId === groupId);
    const roleByID = new Map(grouped.map((role) => [role.id, role]));
    const reordered = roleIds
      .map((id) => roleByID.get(id))
      .filter((role): role is OrganizationRole => Boolean(role));
    if (reordered.length !== grouped.length) return;
    let index = 0;
    roles.value = previous.map((role) => (role.groupId === groupId ? reordered[index++] : role));
    try {
      await reorderOrganizationRolesApi(groupId, roleIds);
    } catch (error) {
      roles.value = previous;
      throw error;
    }
  }

  return {
    mode,
    selection,
    departments,
    roleGroups,
    roles,
    members,
    availableMembers,
    memberTotal,
    memberPage,
    filters,
    currentRoles,
    filteredMembers,
    loadMembers,
    loadAvailableMembers,
    loadRoles,
    changeMemberStatus,
    setMemberPage,
    switchMode,
    selectItem,
    loadDepartments,
    setTenantRootName,
    renameDepartment,
    createChildDepartment,
    roleName,
    addMembers,
    removeMember,
    updateMember,
    createRoleGroup,
    createRole,
    renameRole,
    renameRoleGroup,
    deleteRoleGroup,
    reorderRoleGroups,
    moveRole,
    deleteRole,
    reorderRoles,
  };
}
