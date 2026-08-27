<script setup lang="ts">
import {
  RiAddFill,
  RiFolder3Fill,
  RiGroup2Fill,
  RiMore2Fill,
  RiSearch2Line,
  RiTeamFill,
} from '@remixicon/vue';
import type { AllowDropFunction, NodeDropType } from 'element-plus';
import { computed, shallowRef } from 'vue';
import type {
  OrganizationDepartment,
  OrganizationMode,
  OrganizationRole,
  OrganizationRoleGroup,
  OrganizationSelection,
} from './organization.types';

type RoleGroupAction = 'rename' | 'add-role' | 'delete';
type RoleAction = 'rename' | 'adjust-group' | 'delete';
interface RoleTreeNode {
  key: string;
  kind: 'group' | 'role';
  name: string;
  group: OrganizationRoleGroup;
  role?: OrganizationRole;
  children?: RoleTreeNode[];
}

type ElementTreeNode = Parameters<AllowDropFunction>[0];
type RoleTreeDropType = NodeDropType | 'prev' | 'next';

const props = defineProps<{
  mode: OrganizationMode;
  selection: OrganizationSelection;
  departments: OrganizationDepartment;
  roleGroups: OrganizationRoleGroup[];
  roles: OrganizationRole[];
}>();

const emit = defineEmits<{
  'update:mode': [mode: OrganizationMode];
  select: [selection: OrganizationSelection];
  departmentAction: [action: 'rename' | 'create-child', department: OrganizationDepartment];
  create: [kind: 'group' | 'role'];
  roleGroupAction: [action: RoleGroupAction, group: OrganizationRoleGroup];
  roleGroupReorder: [groupIds: string[]];
  roleAction: [action: RoleAction, role: OrganizationRole];
  roleReorder: [groupId: string, roleIds: string[]];
}>();

const keyword = shallowRef('');
const createMenuVisible = shallowRef(false);

// 分段控制器直接承载组织视图的唯一切换状态，避免维护一套自定义 tab 行为。
const modeOptions = [
  { label: '部门', value: 'department' },
  { label: '角色', value: 'role' },
] satisfies { label: string; value: OrganizationMode }[];
const departmentTreeProps = {
  children: 'children',
  label: 'name',
} as const;
const roleTreeProps = {
  children: 'children',
  label: 'name',
} as const;

const rolesByGroup = computed(() => {
  const result = new Map<string, OrganizationRole[]>();
  for (const role of props.roles) {
    const groupRoles = result.get(role.groupId) ?? [];
    groupRoles.push(role);
    result.set(role.groupId, groupRoles);
  }
  return result;
});

const filteredRoleGroups = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase();
  if (!normalizedKeyword) return props.roleGroups;
  return props.roleGroups.filter((group) => {
    const matchedRoles = props.roles.some(
      (role) => role.groupId === group.id && role.name.toLowerCase().includes(normalizedKeyword),
    );
    return group.name.toLowerCase().includes(normalizedKeyword) || matchedRoles;
  });
});

// el-tree 使用可序列化节点，业务对象则作为数据引用保留给选择和操作事件。
const roleTreeData = computed<RoleTreeNode[]>(() =>
  filteredRoleGroups.value.map((group) => ({
    key: `group:${group.id}`,
    kind: 'group',
    name: group.name,
    group,
    children: groupRoles(group.id).map((role) => ({
      key: `role:${role.id}`,
      kind: 'role',
      name: role.name,
      group,
      role,
    })),
  })),
);
const currentRoleTreeKey = computed(() =>
  props.selection.mode === 'role' && props.roles.some((role) => role.id === props.selection.id)
    ? `role:${props.selection.id}`
    : undefined,
);

const filteredDepartments = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase();
  if (!normalizedKeyword) return props.departments;
  const filterDepartment = (department: OrganizationDepartment): OrganizationDepartment | null => {
    const children = department.children
      ?.map(filterDepartment)
      .filter((item): item is OrganizationDepartment => item !== null);
    if (department.name.toLowerCase().includes(normalizedKeyword) || children?.length) {
      return { ...department, children };
    }
    return null;
  };
  return filterDepartment(props.departments);
});

function changeMode(mode: string | number | boolean) {
  if (mode !== 'department' && mode !== 'role') return;
  createMenuVisible.value = false;
  emit('update:mode', mode);
}

function choose(selection: OrganizationSelection) {
  emit('select', selection);
}

function handleCreate(kind: 'group' | 'role') {
  createMenuVisible.value = false;
  emit('create', kind);
}

function handleDepartmentAction(
  action: 'rename' | 'create-child',
  department: OrganizationDepartment,
) {
  emit('departmentAction', action, department);
}

function selectDepartment(department: OrganizationDepartment) {
  choose({
    mode: 'department',
    id: department.id,
    name: department.name,
  });
}

function groupRoles(groupID: string) {
  return rolesByGroup.value.get(groupID) ?? [];
}

function handleRoleGroupAction(action: RoleGroupAction, group: OrganizationRoleGroup) {
  emit('roleGroupAction', action, group);
}

function handleRoleAction(action: RoleAction, role: OrganizationRole) {
  emit('roleAction', action, role);
}

function selectRoleTreeNode(node: RoleTreeNode) {
  if (node.kind !== 'role' || !node.role) return;
  choose({ mode: 'role', id: node.role.id, name: node.role.name });
}

function canMoveRoleTreeNode(
  draggingData: RoleTreeNode,
  dropData: RoleTreeNode,
  dropType: RoleTreeDropType,
) {
  if (draggingData.kind !== dropData.kind || dropType === 'inner') return false;
  return draggingData.kind === 'group' || draggingData.group.id === dropData.group.id;
}

const canDropRoleTreeNode: AllowDropFunction = (draggingNode, dropNode, dropType) =>
  canMoveRoleTreeNode(draggingNode.data as RoleTreeNode, dropNode.data as RoleTreeNode, dropType);

function moveTreeNode(ids: string[], sourceID: string, targetID: string, dropType: NodeDropType) {
  if (sourceID === targetID || dropType === 'inner') return null;
  const sourceIndex = ids.indexOf(sourceID);
  const targetIndex = ids.indexOf(targetID);
  if (sourceIndex < 0 || targetIndex < 0) return null;
  ids.splice(sourceIndex, 1);
  const updatedTargetIndex = ids.indexOf(targetID);
  ids.splice(updatedTargetIndex + (dropType === 'after' ? 1 : 0), 0, sourceID);
  return ids;
}

function handleRoleTreeDrop(
  draggingNode: ElementTreeNode,
  dropNode: ElementTreeNode,
  dropType: NodeDropType,
) {
  const draggingData = draggingNode.data as RoleTreeNode;
  const dropData = dropNode.data as RoleTreeNode;
  if (!canMoveRoleTreeNode(draggingData, dropData, dropType)) return;
  if (draggingData.kind === 'group') {
    const groupIDs = moveTreeNode(
      props.roleGroups.map((group) => group.id),
      draggingData.group.id,
      dropData.group.id,
      dropType,
    );
    if (groupIDs) emit('roleGroupReorder', groupIDs);
    return;
  }
  if (!draggingData.role || !dropData.role) return;
  const roleIDs = moveTreeNode(
    groupRoles(draggingData.group.id).map((role) => role.id),
    draggingData.role.id,
    dropData.role.id,
    dropType,
  );
  if (roleIDs) emit('roleReorder', draggingData.group.id, roleIDs);
}
</script>

<template>
  <aside class="organization-tree-sidebar" aria-label="内部组织导航">
    <el-segmented
      :model-value="props.mode"
      :options="modeOptions"
      block
      size="large"
      class="organization-tree-sidebar__mode-switch"
      aria-label="组织视图"
      @update:model-value="changeMode"
    />

    <div v-if="props.mode === 'department'" class="organization-tree-sidebar__department-area">
      <p class="organization-tree-sidebar__section-label">成员</p>
      <button
        class="organization-tree-sidebar__quick-item"
        :class="{
          'organization-tree-sidebar__quick-item--active': props.selection.id === 'all-members',
        }"
        type="button"
        @click="choose({ mode: 'department', id: 'all-members', name: '全部成员' })"
      >
        <RiTeamFill />
        <span>全部成员</span>
      </button>
      <button
        class="organization-tree-sidebar__quick-item"
        :class="{
          'organization-tree-sidebar__quick-item--active':
            props.selection.id === 'resigned-members',
        }"
        type="button"
        @click="choose({ mode: 'department', id: 'resigned-members', name: '离职成员' })"
      >
        <RiGroup2Fill />
        <span>离职成员</span>
      </button>

      <p
        class="organization-tree-sidebar__section-label organization-tree-sidebar__section-label--department"
      >
        部门
      </p>
      <label class="organization-tree-sidebar__search">
        <RiSearch2Line />
        <input v-model="keyword" placeholder="搜索" aria-label="搜索部门" />
      </label>
      <el-scrollbar v-if="filteredDepartments" class="organization-tree-sidebar__department-tree">
        <el-tree
          :data="[filteredDepartments]"
          :props="departmentTreeProps"
          node-key="id"
          default-expand-all
          highlight-current
          :current-node-key="props.selection.id"
          :expand-on-click-node="false"
          :indent="28"
          @node-click="selectDepartment"
        >
          <template #default="{ data: department }">
            <div class="organization-tree-sidebar__department-node">
              <RiGroup2Fill class="organization-tree-sidebar__department-node-icon" />
              <span class="organization-tree-sidebar__department-node-name">{{
                department.name
              }}</span>
              <el-dropdown trigger="click" @command="handleDepartmentAction($event, department)">
                <button
                  class="organization-tree-sidebar__department-node-more"
                  type="button"
                  :aria-label="`${department.name} 操作`"
                  @click.stop
                >
                  <RiMore2Fill />
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="rename">修改名称</el-dropdown-item>
                    <el-dropdown-item command="create-child">添加子部门</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-tree>
      </el-scrollbar>
      <p v-else class="organization-tree-sidebar__empty">未找到匹配的部门</p>
    </div>

    <div v-else class="organization-tree-sidebar__role-area">
      <label class="organization-tree-sidebar__search">
        <RiSearch2Line />
        <input v-model="keyword" placeholder="搜索" aria-label="搜索角色" />
      </label>
      <div class="organization-tree-sidebar__roles-heading">
        <span>创建的角色</span>
        <el-popover
          v-model:visible="createMenuVisible"
          placement="bottom-end"
          :width="166"
          trigger="click"
        >
          <template #reference>
            <button
              class="organization-tree-sidebar__add-button"
              type="button"
              aria-label="新建角色"
            >
              <RiAddFill />
            </button>
          </template>
          <div class="organization-tree-sidebar__create-menu">
            <button type="button" @click="handleCreate('group')">创建角色组</button>
            <button type="button" @click="handleCreate('role')">创建角色</button>
          </div>
        </el-popover>
      </div>
      <el-scrollbar class="organization-tree-sidebar__role-tree">
        <el-tree
          :data="roleTreeData"
          :props="roleTreeProps"
          node-key="key"
          default-expand-all
          highlight-current
          draggable
          :current-node-key="currentRoleTreeKey"
          :allow-drop="canDropRoleTreeNode"
          :expand-on-click-node="true"
          :indent="28"
          @node-click="selectRoleTreeNode"
          @node-drop="handleRoleTreeDrop"
        >
          <template #default="{ data: node }">
            <div
              class="organization-tree-sidebar__role-node"
              :class="{
                'organization-tree-sidebar__role-node--group': node.kind === 'group',
                'organization-tree-sidebar__role-node--role': node.kind === 'role',
              }"
            >
              <RiFolder3Fill
                v-if="node.kind === 'group'"
                class="organization-tree-sidebar__role-node-icon"
              />
              <RiGroup2Fill v-else class="organization-tree-sidebar__role-node-icon" />
              <span class="organization-tree-sidebar__role-node-name">{{ node.name }}</span>
              <el-dropdown
                v-if="node.kind === 'group'"
                trigger="click"
                @command="handleRoleGroupAction($event, node.group)"
              >
                <button
                  class="organization-tree-sidebar__role-node-more"
                  type="button"
                  :aria-label="`${node.name} 操作`"
                  @click.stop
                >
                  <RiMore2Fill />
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="rename">修改名称</el-dropdown-item>
                    <el-dropdown-item command="add-role">添加角色</el-dropdown-item>
                    <el-dropdown-item command="delete">删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <el-dropdown
                v-else-if="node.role"
                trigger="click"
                @command="handleRoleAction($event, node.role)"
              >
                <button
                  class="organization-tree-sidebar__role-node-more"
                  type="button"
                  :aria-label="`${node.name} 操作`"
                  @click.stop
                >
                  <RiMore2Fill />
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="rename">修改名称</el-dropdown-item>
                    <el-dropdown-item command="adjust-group">调整分组</el-dropdown-item>
                    <el-dropdown-item command="delete">删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-tree>
      </el-scrollbar>
    </div>
  </aside>
</template>

<style scoped lang="scss">
.organization-tree-sidebar {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  height: 100%;
  padding: var(--el-space-3xl) var(--el-space-3xl) var(--el-space-2xl);
  flex-direction: column;
  background: #fff;

  &__mode-switch {
    width: 100%;
    min-height: 44px;
    --el-segmented-bg-color: var(--el-fill-color);
    --el-segmented-item-selected-color: var(--el-color-primary);
    --el-segmented-item-selected-bg-color: var(--el-bg-color);
    --el-segmented-item-hover-bg-color: var(--el-fill-color-light);
    --el-segmented-item-active-bg-color: var(--el-fill-color);
  }

  &__department-area,
  &__role-area {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
  }
  &__section-label {
    margin: var(--el-space-3xl) 0 var(--el-space-lg);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
    line-height: 24px;
  }
  &__section-label--department {
    margin-top: var(--el-space-4xl);
  }

  &__quick-item {
    display: flex;
    box-sizing: border-box;
    min-width: 0;
    border: 0;
    align-items: center;
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: pointer;
    font: inherit;
    text-align: left;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      background: var(--el-fill-color-light);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  &__quick-item {
    height: 56px;
    gap: var(--el-space-lg);
    padding: 0 var(--el-space-xl);
    border-radius: var(--el-border-radius-medium);
    font-size: var(--el-font-size-medium);

    svg {
      width: 22px;
      height: 22px;
      color: #a6afbd;
    }
    &--active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
    &--active svg {
      color: var(--el-color-primary);
    }
  }

  &__search {
    display: flex;
    box-sizing: border-box;
    height: 46px;
    padding: 0 var(--el-space-lg);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-secondary);

    &:focus-within {
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
    }
    svg {
      width: 20px;
      height: 20px;
      flex: 0 0 20px;
    }
    input {
      width: 100%;
      border: 0;
      outline: 0;
      color: var(--el-text-color-primary);
      background: transparent;
      font: inherit;
      font-size: var(--el-font-size-medium);
    }
    input::placeholder {
      color: var(--el-text-color-placeholder);
    }
  }
  &__role-area .organization-tree-sidebar__search {
    // 与分段控制器保留固定间距，避免角色搜索框紧贴切换控件。
    margin-top: var(--el-space-2xl);
  }

  &__department-tree,
  &__role-tree {
    min-height: 0;
    margin-top: var(--el-space-2xl);
    overflow-y: auto;
  }
  &__department-tree :deep(.el-tree) {
    --el-tree-node-content-height: 44px;
    background: transparent;
  }
  &__department-tree :deep(.el-tree-node__content) {
    padding-right: var(--el-space-sm);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-regular);
  }
  &__department-tree :deep(.el-tree-node__content:hover),
  &__department-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__department-node {
    display: flex;
    min-width: 0;
    flex: 1;
    align-items: center;
  }
  &__department-node-icon {
    width: 20px;
    height: 20px;
    margin-right: var(--el-space-md);
    flex: 0 0 20px;
    color: var(--el-color-primary);
  }
  &__department-node-name {
    overflow: hidden;
    flex: 1;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__department-node-more {
    display: inline-flex;
    width: 28px;
    height: 28px;
    margin-left: var(--el-space-xs);
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;
    opacity: 0;
    transition:
      opacity 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-8);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
      opacity: 1;
    }
  }
  &__department-node-more svg {
    width: 19px;
    height: 19px;
  }
  &__department-tree :deep(.el-tree-node__content:hover) &__department-node-more,
  &__department-tree
    :deep(.el-tree-node.is-current > .el-tree-node__content)
    &__department-node-more {
    opacity: 1;
  }
  &__empty {
    margin: var(--el-space-2xl) 0 0;
    color: var(--el-text-color-placeholder);
    font-size: var(--el-font-size-base);
    text-align: center;
  }

  &__roles-heading {
    display: flex;
    margin-top: var(--el-space-2xl);
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
    line-height: 36px;
  }
  &__add-button {
    display: inline-flex;
    width: 36px;
    height: 36px;
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    justify-content: center;
    color: #fff;
    background: var(--el-color-primary);
    cursor: pointer;
  }
  &__add-button:hover {
    background: var(--el-color-primary-light-3);
  }
  &__add-button:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 2px;
  }
  &__add-button svg {
    width: 22px;
    height: 22px;
  }
  &__role-tree :deep(.el-tree) {
    --el-tree-node-content-height: 46px;
    background: transparent;
  }
  &__role-tree :deep(.el-tree-node__content) {
    padding-right: var(--el-space-sm);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-regular);
  }
  &__role-tree :deep(.el-tree-node__content:hover),
  &__role-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__role-node {
    display: flex;
    min-width: 0;
    flex: 1;
    align-items: center;
  }
  &__role-node-icon {
    width: 22px;
    height: 22px;
    margin-right: var(--el-space-md);
    flex: 0 0 22px;
    color: var(--el-color-primary);
  }
  &__role-node--group .organization-tree-sidebar__role-node-icon {
    color: var(--el-color-warning);
  }
  &__role-node-name {
    overflow: hidden;
    flex: 1;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__role-node-more {
    display: inline-flex;
    width: 28px;
    height: 28px;
    margin-left: var(--el-space-xs);
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;
    opacity: 0;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-8);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
      opacity: 1;
    }
  }
  &__role-node-more svg {
    width: 19px;
    height: 19px;
  }
  &__role-tree :deep(.el-tree-node__content:hover) &__role-node-more,
  &__role-tree :deep(.el-tree-node.is-current > .el-tree-node__content) &__role-node-more {
    opacity: 1;
  }

  &__create-menu {
    display: grid;
    gap: var(--el-space-xs);
  }
  &__create-menu button {
    padding: var(--el-space-md) var(--el-space-md);
    border: 0;
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    font-size: var(--el-font-size-base);
    text-align: left;
  }
  &__create-menu button:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}
</style>
