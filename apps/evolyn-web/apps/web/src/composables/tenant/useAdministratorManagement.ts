import { computed, reactive, shallowRef } from 'vue';
import type {
  AdministratorGroup,
  AdministratorMember,
  AdministratorScope,
  ScopeMode,
} from '~/components/tenant/administrators/administrator.types';

function createGroup(
  id: string,
  name: string,
  members: AdministratorMember[] = [],
  builtIn = false,
): AdministratorGroup {
  return {
    id,
    name,
    builtIn,
    members,
    departmentEnabled: false,
    departmentMode: 'partial',
    departmentIds: ['department-rd'],
    roleVisible: false,
    roleManage: false,
    roleMode: 'partial',
    roleIds: ['role-test'],
    externalEnabled: false,
    applicationIds: ['app-demo'],
    applicationManage: false,
  };
}

/**
 * 管理员界面在接口尚未落地时维护本地可交互状态；数据结构刻意贴近后续权限接口，
 * 便于接入后以服务端读写替换初始化与更新动作。
 */
export function useAdministratorManagement(scope: AdministratorScope) {
  const systemGroups = reactive<AdministratorGroup[]>([
    createGroup('system-administrator', '系统管理员', [], true),
    createGroup('system-test', '测试', []),
  ]);
  const applicationGroups = reactive<AdministratorGroup[]>([createGroup('app-test', '测试')]);
  const selectedId = shallowRef(scope === 'system' ? 'system-administrator' : 'app-test');
  const groups = computed(() => (scope === 'system' ? systemGroups : applicationGroups));
  const selectedGroup = computed(
    () => groups.value.find((group) => group.id === selectedId.value) ?? groups.value[0],
  );

  function selectGroup(id: string) {
    selectedId.value = id;
  }

  function addGroup(name: string) {
    const group = createGroup(`${scope}-${Date.now()}`, name);
    groups.value.push(group);
    selectedId.value = group.id;
  }

  function updateMembers(members: AdministratorMember[]) {
    selectedGroup.value.members = members;
  }

  function setDepartmentMode(mode: ScopeMode) {
    selectedGroup.value.departmentMode = mode;
  }

  function setRoleMode(mode: ScopeMode) {
    selectedGroup.value.roleMode = mode;
  }

  return {
    groups,
    selectedGroup,
    selectedId,
    selectGroup,
    addGroup,
    updateMembers,
    setDepartmentMode,
    setRoleMode,
  };
}
