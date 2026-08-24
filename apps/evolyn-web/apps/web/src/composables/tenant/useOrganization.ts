import { computed, reactive, shallowRef } from 'vue';
import type {
  OrganizationDepartment,
  OrganizationMember,
  OrganizationMode,
  OrganizationRole,
  OrganizationRoleGroup,
  OrganizationSelection,
} from '~/components/tenant/organization/organization.types';

const rootDepartment: OrganizationDepartment = {
  id: 'dept-root',
  name: '重庆万柯互联网科技有限责任公司',
  children: [
    { id: 'dept-rd', name: '研发部' },
    { id: 'dept-product', name: '产品部' },
  ],
};

const initialGroups: OrganizationRoleGroup[] = [
  { id: 'role-group-default', name: '默认' },
  { id: 'role-group-crm', name: 'CRM角色组' },
];

const initialRoles: OrganizationRole[] = [
  { id: 'role-test', name: '测试', groupId: 'role-group-default' },
  { id: 'role-sales-director', name: '销售总监', groupId: 'role-group-crm' },
  { id: 'role-sales-manager', name: '销售主管', groupId: 'role-group-crm' },
  { id: 'role-sales', name: '销售人员', groupId: 'role-group-crm' },
];

const initialMembers: OrganizationMember[] = [
  {
    id: 'member-li',
    name: '李同学',
    phone: '+86-15355381414',
    email: '',
    department: '重庆万柯互联网科技有限责任公司',
    roleIds: ['role-test'],
    status: '已启用',
    employeeNo: 'sys_6a7f3132e6a0aba27cd…',
    alias: '',
    gender: '',
  },
];

/** 内部组织演示页的状态边界；后端接口落地后仅需替换这里的数据读写。 */
export function useOrganization() {
  const mode = shallowRef<OrganizationMode>('department');
  const selection = shallowRef<OrganizationSelection>({
    mode: 'department',
    id: 'all-members',
    name: '全部成员',
  });
  const departments = shallowRef<OrganizationDepartment>(rootDepartment);
  const roleGroups = shallowRef<OrganizationRoleGroup[]>(initialGroups);
  const roles = shallowRef<OrganizationRole[]>(initialRoles);
  const members = shallowRef<OrganizationMember[]>(initialMembers);
  const filters = reactive({
    departmentKeyword: '',
    roleKeyword: '',
    memberKeyword: '',
    status: '全部',
  });

  const currentRoles = computed(() => {
    return roles.value.filter((item) => item.groupId === selection.value.id);
  });

  const filteredMembers = computed(() => {
    const keyword = filters.memberKeyword.trim().toLowerCase();
    const selected = selection.value;
    return members.value.filter((member) => {
      const matchesKeyword =
        !keyword ||
        [member.name, member.phone, member.email].join(' ').toLowerCase().includes(keyword);
      const matchesStatus = filters.status === '全部' || member.status === filters.status;
      if (selected.mode === 'department') {
        return (
          matchesKeyword &&
          matchesStatus &&
          (selected.id === 'all-members' || member.department === selected.name)
        );
      }
      return matchesKeyword && matchesStatus && member.roleIds.includes(selected.id);
    });
  });

  function switchMode(nextMode: OrganizationMode) {
    mode.value = nextMode;
    filters.memberKeyword = '';
    selection.value =
      nextMode === 'department'
        ? { mode: 'department', id: 'all-members', name: '全部成员' }
        : { mode: 'role', id: 'role-test', name: '测试' };
  }

  function selectItem(nextSelection: OrganizationSelection) {
    selection.value = nextSelection;
  }

  function roleName(roleId: string) {
    return roles.value.find((role) => role.id === roleId)?.name ?? '';
  }

  function addMembers(memberIds: string[]) {
    if (selection.value.mode !== 'role') return;
    members.value = members.value.map((member) =>
      memberIds.includes(member.id) && !member.roleIds.includes(selection.value.id)
        ? { ...member, roleIds: [...member.roleIds, selection.value.id] }
        : member,
    );
  }

  function removeMember(memberId: string) {
    if (selection.value.mode !== 'role') return;
    members.value = members.value.map((member) =>
      member.id === memberId
        ? { ...member, roleIds: member.roleIds.filter((roleId) => roleId !== selection.value.id) }
        : member,
    );
  }

  function updateMember(member: OrganizationMember) {
    members.value = members.value.map((item) => (item.id === member.id ? member : item));
  }

  function createRoleGroup(name: string) {
    const group: OrganizationRoleGroup = { id: `role-group-${Date.now()}`, name };
    roleGroups.value = [...roleGroups.value, group];
  }

  function createRole(name: string) {
    const groupId =
      selection.value.mode === 'role' &&
      roleGroups.value.some((group) => group.id === selection.value.id)
        ? selection.value.id
        : roleGroups.value[0].id;
    const role = { id: `role-${Date.now()}`, name, groupId };
    roles.value = [...roles.value, role];
    selection.value = { mode: 'role', id: role.id, name: role.name };
  }

  function renameRole(name: string) {
    if (selection.value.mode !== 'role') return;
    roles.value = roles.value.map((role) =>
      role.id === selection.value.id ? { ...role, name } : role,
    );
    selection.value = { ...selection.value, name };
  }

  function moveRole(targetGroupId: string) {
    if (selection.value.mode !== 'role') return;
    roles.value = roles.value.map((role) =>
      role.id === selection.value.id ? { ...role, groupId: targetGroupId } : role,
    );
  }

  return {
    mode,
    selection,
    departments,
    roleGroups,
    roles,
    members,
    filters,
    currentRoles,
    filteredMembers,
    switchMode,
    selectItem,
    roleName,
    addMembers,
    removeMember,
    updateMember,
    createRoleGroup,
    createRole,
    renameRole,
    moveRole,
  };
}
