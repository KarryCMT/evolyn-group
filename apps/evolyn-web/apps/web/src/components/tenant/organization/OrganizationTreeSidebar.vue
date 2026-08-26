<script setup lang="ts">
import {
  RiAddFill,
  RiArrowDownSFill,
  RiFolder3Fill,
  RiGroup2Fill,
  RiMore2Fill,
  RiSearch2Line,
  RiTeamFill,
} from '@remixicon/vue';
import { computed, onBeforeUnmount, onMounted, shallowRef } from 'vue';
import OrganizationDepartmentTreeNode from './OrganizationDepartmentTreeNode.vue';
import type {
  OrganizationDepartment,
  OrganizationMode,
  OrganizationRole,
  OrganizationRoleGroup,
  OrganizationSelection,
} from './organization.types';

type RoleGroupAction = 'rename' | 'add-role' | 'delete';
type RoleAction = 'rename' | 'adjust-group' | 'delete';
type OpenedOperationMenu =
  | {
      kind: 'group';
      group: OrganizationRoleGroup;
      top: number;
      left: number;
    }
  | {
      kind: 'role';
      role: OrganizationRole;
      top: number;
      left: number;
    };

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
const expandedGroupIds = shallowRef<string[]>([]);
const draggingGroupId = shallowRef<string | null>(null);
const draggingRole = shallowRef<{ groupId: string; roleId: string } | null>(null);
const openedOperationMenu = shallowRef<OpenedOperationMenu | null>(null);

const openedGroupMenuID = computed(() =>
  openedOperationMenu.value?.kind === 'group' ? openedOperationMenu.value.group.id : null,
);
const openedRoleMenuID = computed(() =>
  openedOperationMenu.value?.kind === 'role' ? openedOperationMenu.value.role.id : null,
);
const openedRoleGroup = computed(() =>
  openedOperationMenu.value?.kind === 'group' ? openedOperationMenu.value.group : null,
);
const openedRole = computed(() =>
  openedOperationMenu.value?.kind === 'role' ? openedOperationMenu.value.role : null,
);
const operationMenuStyle = computed(() => {
  const menu = openedOperationMenu.value;
  if (!menu) return {};
  return { top: `${menu.top}px`, left: `${menu.left}px` };
});

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

function changeMode(mode: OrganizationMode) {
  createMenuVisible.value = false;
  closeOperationMenu();
  emit('update:mode', mode);
}

function choose(selection: OrganizationSelection) {
  closeOperationMenu();
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

function groupRoles(groupID: string) {
  return rolesByGroup.value.get(groupID) ?? [];
}

function isGroupExpanded(groupID: string) {
  // 初次进入时角色组保持展开，用户可单独收起任意分组。
  return !expandedGroupIds.value.includes(groupID);
}

function toggleGroup(groupID: string) {
  const expanded = isGroupExpanded(groupID);
  expandedGroupIds.value = expanded
    ? [...expandedGroupIds.value, groupID]
    : expandedGroupIds.value.filter((id) => id !== groupID);
}

function resolveOperationMenuPosition(event: MouseEvent) {
  const reference = event.currentTarget;
  if (!(reference instanceof HTMLElement)) return null;
  const rect = reference.getBoundingClientRect();
  return {
    top: Math.min(rect.top, window.innerHeight - 180),
    left: Math.min(rect.right + 8, window.innerWidth - 180),
  };
}

function closeOperationMenu() {
  openedOperationMenu.value = null;
}

function toggleRoleGroupMenu(group: OrganizationRoleGroup, event: MouseEvent) {
  if (openedGroupMenuID.value === group.id) {
    closeOperationMenu();
    return;
  }
  const position = resolveOperationMenuPosition(event);
  if (!position) return;
  createMenuVisible.value = false;
  openedOperationMenu.value = { kind: 'group', group, ...position };
}

function toggleRoleMenu(role: OrganizationRole, event: MouseEvent) {
  if (openedRoleMenuID.value === role.id) {
    closeOperationMenu();
    return;
  }
  const position = resolveOperationMenuPosition(event);
  if (!position) return;
  createMenuVisible.value = false;
  openedOperationMenu.value = { kind: 'role', role, ...position };
}

function handleRoleGroupMenuAction(action: RoleGroupAction) {
  const group = openedRoleGroup.value;
  if (!group) return;
  closeOperationMenu();
  emit('roleGroupAction', action, group);
}

function handleRoleMenuAction(action: RoleAction) {
  const role = openedRole.value;
  if (!role) return;
  closeOperationMenu();
  emit('roleAction', action, role);
}

function startGroupDrag(groupID: string, event: DragEvent) {
  draggingGroupId.value = groupID;
  event.dataTransfer?.setData('text/plain', groupID);
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
}

function finishGroupDrag() {
  draggingGroupId.value = null;
}

function dropGroup(targetID: string) {
  const sourceID = draggingGroupId.value;
  finishGroupDrag();
  if (!sourceID || sourceID === targetID) return;
  const groupIDs = props.roleGroups.map((group) => group.id);
  const sourceIndex = groupIDs.indexOf(sourceID);
  const targetIndex = groupIDs.indexOf(targetID);
  if (sourceIndex < 0 || targetIndex < 0) return;
  groupIDs.splice(sourceIndex, 1);
  groupIDs.splice(targetIndex, 0, sourceID);
  emit('roleGroupReorder', groupIDs);
}

function startRoleDrag(groupId: string, roleId: string, event: DragEvent) {
  draggingRole.value = { groupId, roleId };
  event.dataTransfer?.setData('text/plain', roleId);
  if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
}

function finishRoleDrag() {
  draggingRole.value = null;
}

function dropRole(groupId: string, targetRoleId: string) {
  const source = draggingRole.value;
  finishRoleDrag();
  // 跨角色组移动有独立的“调整分组”操作，拖拽只改变当前组内的展示顺序。
  if (!source || source.groupId !== groupId || source.roleId === targetRoleId) return;
  const roleIDs = groupRoles(groupId).map((role) => role.id);
  const sourceIndex = roleIDs.indexOf(source.roleId);
  const targetIndex = roleIDs.indexOf(targetRoleId);
  if (sourceIndex < 0 || targetIndex < 0) return;
  roleIDs.splice(sourceIndex, 1);
  roleIDs.splice(targetIndex, 0, source.roleId);
  emit('roleReorder', groupId, roleIDs);
}

function handleOperationMenuKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') closeOperationMenu();
}

// 操作菜单使用固定坐标的单一 Teleport 容器，不再交由 Popper 在触发节点变更时重算位置。
// 角色树滚动、窗口缩放或点到菜单外时直接关闭，避免出现陈旧位置的浮层。
onMounted(() => {
  document.addEventListener('pointerdown', closeOperationMenu);
  document.addEventListener('keydown', handleOperationMenuKeydown);
  window.addEventListener('resize', closeOperationMenu);
  window.addEventListener('scroll', closeOperationMenu, true);
});

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', closeOperationMenu);
  document.removeEventListener('keydown', handleOperationMenuKeydown);
  window.removeEventListener('resize', closeOperationMenu);
  window.removeEventListener('scroll', closeOperationMenu, true);
});
</script>

<template>
  <aside class="organization-tree-sidebar" aria-label="内部组织导航">
    <div class="organization-tree-sidebar__mode-switch" role="tablist" aria-label="组织视图">
      <button
        class="organization-tree-sidebar__mode-button"
        :class="{ 'organization-tree-sidebar__mode-button--active': props.mode === 'department' }"
        type="button"
        role="tab"
        :aria-selected="props.mode === 'department'"
        @click="changeMode('department')"
      >
        部门
      </button>
      <button
        class="organization-tree-sidebar__mode-button"
        :class="{ 'organization-tree-sidebar__mode-button--active': props.mode === 'role' }"
        type="button"
        role="tab"
        :aria-selected="props.mode === 'role'"
        @click="changeMode('role')"
      >
        角色
      </button>
    </div>

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
        <OrganizationDepartmentTreeNode
          :department="filteredDepartments"
          :depth="0"
          :selection="props.selection"
          @select="choose"
          @action="handleDepartmentAction"
        />
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
        <section
          v-for="group in filteredRoleGroups"
          :key="group.id"
          class="organization-tree-sidebar__role-group"
          :class="{
            'organization-tree-sidebar__role-group--dragging': draggingGroupId === group.id,
          }"
          @dragover.prevent
          @drop.prevent="dropGroup(group.id)"
        >
          <div
            class="organization-tree-sidebar__role-group-row"
            :class="{
              'organization-tree-sidebar__role-group-row--menu-open':
                openedGroupMenuID === group.id,
            }"
          >
            <button
              class="organization-tree-sidebar__role-group-title"
              type="button"
              :aria-expanded="isGroupExpanded(group.id)"
              @click="toggleGroup(group.id)"
            >
              <RiArrowDownSFill
                :class="{
                  'organization-tree-sidebar__expand-icon--collapsed': !isGroupExpanded(group.id),
                }"
              />
              <RiFolder3Fill />
              <span>{{ group.name }}</span>
            </button>
            <div class="organization-tree-sidebar__role-group-actions">
              <button
                class="organization-tree-sidebar__drag-handle"
                type="button"
                draggable="true"
                aria-label="拖拽排序角色组"
                @dragend="finishGroupDrag"
                @dragstart="startGroupDrag(group.id, $event)"
              >
                ⠿
              </button>
              <button
                class="organization-tree-sidebar__more-button"
                type="button"
                aria-label="角色组操作"
                @click="toggleRoleGroupMenu(group, $event)"
                @pointerdown.stop
              >
                <RiMore2Fill />
              </button>
            </div>
          </div>
          <template v-if="isGroupExpanded(group.id)">
            <div
              v-for="role in groupRoles(group.id)"
              :key="role.id"
              class="organization-tree-sidebar__role-row"
              :class="{
                'organization-tree-sidebar__role-row--dragging': draggingRole?.roleId === role.id,
                'organization-tree-sidebar__role-row--active': props.selection.id === role.id,
                'organization-tree-sidebar__role-row--menu-open': openedRoleMenuID === role.id,
              }"
              @dragover.prevent
              @drop.prevent="dropRole(group.id, role.id)"
            >
              <button
                class="organization-tree-sidebar__role-item"
                :class="{
                  'organization-tree-sidebar__role-item--active': props.selection.id === role.id,
                }"
                type="button"
                @click="choose({ mode: 'role', id: role.id, name: role.name })"
              >
                <RiGroup2Fill />
                <span>{{ role.name }}</span>
              </button>
              <div class="organization-tree-sidebar__role-actions">
                <button
                  class="organization-tree-sidebar__drag-handle"
                  type="button"
                  draggable="true"
                  aria-label="拖拽排序角色"
                  @dragend="finishRoleDrag"
                  @dragstart="startRoleDrag(group.id, role.id, $event)"
                >
                  ⠿
                </button>
                <button
                  class="organization-tree-sidebar__more-button"
                  type="button"
                  aria-label="角色操作"
                  @click="toggleRoleMenu(role, $event)"
                  @pointerdown.stop
                >
                  <RiMore2Fill />
                </button>
              </div>
            </div>
          </template>
        </section>
      </el-scrollbar>
    </div>

    <Teleport to="body">
      <div
        v-if="openedOperationMenu"
        class="organization-tree-sidebar__operation-menu"
        :style="operationMenuStyle"
        @pointerdown.stop
      >
        <template v-if="openedRoleGroup">
          <button type="button" @click="handleRoleGroupMenuAction('rename')">修改名称</button>
          <button type="button" @click="handleRoleGroupMenuAction('add-role')">添加角色</button>
          <button
            class="organization-tree-sidebar__operation-menu-delete"
            type="button"
            @click="handleRoleGroupMenuAction('delete')"
          >
            删除
          </button>
        </template>
        <template v-else-if="openedRole">
          <button type="button" @click="handleRoleMenuAction('rename')">修改名称</button>
          <button type="button" @click="handleRoleMenuAction('adjust-group')">调整分组</button>
          <button
            class="organization-tree-sidebar__operation-menu-delete"
            type="button"
            @click="handleRoleMenuAction('delete')"
          >
            删除
          </button>
        </template>
      </div>
    </Teleport>
  </aside>
</template>

<style scoped lang="scss">
.organization-tree-sidebar {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  height: 100%;
  padding: 28px 28px 20px;
  flex-direction: column;
  background: #fff;

  &__mode-switch {
    display: grid;
    height: 44px;
    padding: 2px;
    grid-template-columns: 1fr 1fr;
    border-radius: 8px;
    background: var(--el-fill-color);
  }

  &__mode-button {
    border: 0;
    border-radius: 6px;
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    font-size: 16px;
    transition:
      background-color 0.18s ease,
      color 0.18s ease,
      box-shadow 0.18s ease;

    &:hover {
      background: var(--el-fill-color-light);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--active {
      color: var(--el-color-primary);
      background: #fff;
      box-shadow: 0 1px 3px rgb(31 41 55 / 14%);
    }
  }

  &__department-area,
  &__role-area {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
  }
  &__section-label {
    margin: 23px 0 12px;
    color: var(--el-text-color-secondary);
    font-size: 16px;
    line-height: 24px;
  }
  &__section-label--department {
    margin-top: 30px;
  }

  &__quick-item,
  &__role-group-title,
  &__role-item {
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
    gap: 13px;
    padding: 0 18px;
    border-radius: 8px;
    font-size: 16px;

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
    padding: 0 13px;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    align-items: center;
    gap: 10px;
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
      font-size: 16px;
    }
    input::placeholder {
      color: var(--el-text-color-placeholder);
    }
  }

  &__department-tree,
  &__role-tree {
    min-height: 0;
    margin-top: 22px;
    overflow-y: auto;
  }
  &__empty {
    margin: 22px 0 0;
    color: var(--el-text-color-placeholder);
    font-size: 14px;
    text-align: center;
  }

  &__roles-heading {
    display: flex;
    margin-top: 22px;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-secondary);
    font-size: 16px;
    line-height: 36px;
  }
  &__add-button {
    display: inline-flex;
    width: 36px;
    height: 36px;
    padding: 0;
    border: 0;
    border-radius: 8px;
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
  &__role-group {
    margin-top: 12px;
    border-radius: 8px;
  }
  &__role-group--dragging {
    opacity: 0.52;
  }
  &__role-group-row {
    display: flex;
    height: 46px;
    padding-right: 7px;
    align-items: center;
    border-radius: 8px;

    &:hover {
      background: var(--el-fill-color-light);
    }
  }
  &__role-group-title {
    min-width: 0;
    flex: 1;
    height: 46px;
    gap: 8px;
    padding: 0 14px;
    font-size: 16px;
  }
  &__role-group-title svg:first-child {
    width: 18px;
    height: 18px;
    color: var(--el-text-color-secondary);
  }
  &__role-group-title svg:nth-child(2) {
    width: 23px;
    height: 23px;
    color: var(--el-color-warning);
  }
  &__expand-icon--collapsed {
    transform: rotate(-90deg);
  }
  &__role-group-actions {
    display: none;
    align-items: center;
    gap: 2px;
  }
  &__role-group-row:hover &__role-group-actions,
  &__role-group-row--menu-open &__role-group-actions,
  &__role-group-actions:focus-within {
    display: flex;
  }
  &__drag-handle,
  &__more-button {
    display: inline-flex;
    width: 28px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: 6px;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: grab;
    font: inherit;
  }
  &__drag-handle:hover,
  &__more-button:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__drag-handle:active {
    cursor: grabbing;
  }
  &__more-button {
    cursor: pointer;
  }
  &__more-button svg {
    width: 20px;
    height: 20px;
  }
  &__role-item {
    min-width: 0;
    flex: 1;
    height: 56px;
    gap: 9px;
    padding: 0 24px 0 72px;
    border-radius: 8px;
    font-size: 16px;
  }
  &__role-item svg {
    width: 22px;
    height: 22px;
    color: var(--el-color-primary);
  }
  &__role-item--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__role-row {
    display: flex;
    height: 56px;
    padding-right: 7px;
    align-items: center;
    border-radius: 8px;

    &:hover {
      background: var(--el-fill-color-light);
    }
  }
  &__role-row--dragging {
    opacity: 0.52;
  }
  &__role-row--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__role-row .organization-tree-sidebar__role-item--active {
    background: transparent;
  }
  &__role-actions {
    display: none;
    align-items: center;
    gap: 2px;
  }
  &__role-row:hover &__role-actions,
  &__role-row--menu-open &__role-actions,
  &__role-actions:focus-within {
    display: flex;
  }

  &__create-menu {
    display: grid;
    gap: 2px;
  }
  &__create-menu button {
    padding: 8px 10px;
    border: 0;
    border-radius: 4px;
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    font-size: 15px;
    text-align: left;
  }
  &__create-menu button:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &__operation-menu {
    display: grid;
    position: fixed;
    z-index: 3000;
    width: 164px;
    padding: 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: #fff;
    box-shadow: var(--el-box-shadow-light);
    gap: 2px;
  }
  &__operation-menu::before {
    position: absolute;
    top: 20px;
    left: -5px;
    width: 9px;
    height: 9px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    border-left: 1px solid var(--el-border-color-lighter);
    background: #fff;
    content: '';
    transform: rotate(45deg);
  }
  &__operation-menu button {
    position: relative;
    padding: 8px 10px;
    border: 0;
    border-radius: 4px;
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
    font-size: 15px;
    text-align: left;
  }
  &__operation-menu button:hover {
    background: var(--el-fill-color-light);
  }
  &__operation-menu .organization-tree-sidebar__operation-menu-delete {
    color: var(--el-color-danger);
  }
  &__operation-menu .organization-tree-sidebar__operation-menu-delete:hover {
    background: var(--el-color-danger-light-9);
  }
}
</style>
