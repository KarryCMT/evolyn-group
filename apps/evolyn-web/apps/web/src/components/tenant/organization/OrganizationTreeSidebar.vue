<script setup lang="ts">
import {
  RiAddFill,
  RiArrowDownSFill,
  RiFolder3Fill,
  RiGroup2Fill,
  RiSearch2Line,
  RiTeamFill,
} from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type {
  OrganizationDepartment,
  OrganizationMode,
  OrganizationRole,
  OrganizationRoleGroup,
  OrganizationSelection,
} from './organization.types';

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
  create: [kind: 'group' | 'role'];
}>();

const keyword = shallowRef('');
const createMenuVisible = shallowRef(false);

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

const departmentVisible = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase();
  if (!normalizedKeyword) return true;
  const includesDepartment = (department: OrganizationDepartment): boolean => {
    return (
      department.name.toLowerCase().includes(normalizedKeyword) ||
      Boolean(department.children?.some(includesDepartment))
    );
  };
  return includesDepartment(props.departments);
});

function changeMode(mode: OrganizationMode) {
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
      <button class="organization-tree-sidebar__quick-item" type="button">
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
      <div v-if="departmentVisible" class="organization-tree-sidebar__department-tree">
        <button
          class="organization-tree-sidebar__tree-item organization-tree-sidebar__tree-item--root"
          :class="{
            'organization-tree-sidebar__tree-item--active':
              props.selection.id === props.departments.id,
          }"
          type="button"
          @click="
            choose({ mode: 'department', id: props.departments.id, name: props.departments.name })
          "
        >
          <RiArrowDownSFill class="organization-tree-sidebar__expand-icon" />
          <RiGroup2Fill class="organization-tree-sidebar__tree-icon" />
          <span>{{ props.departments.name }}</span>
        </button>
        <button
          v-for="department in props.departments.children"
          :key="department.id"
          class="organization-tree-sidebar__tree-item organization-tree-sidebar__tree-item--child"
          :class="{
            'organization-tree-sidebar__tree-item--active': props.selection.id === department.id,
          }"
          type="button"
          @click="choose({ mode: 'department', id: department.id, name: department.name })"
        >
          <RiGroup2Fill class="organization-tree-sidebar__tree-icon" />
          <span>{{ department.name }}</span>
        </button>
      </div>
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
      <div class="organization-tree-sidebar__role-tree">
        <section
          v-for="group in filteredRoleGroups"
          :key="group.id"
          class="organization-tree-sidebar__role-group"
        >
          <button
            class="organization-tree-sidebar__role-group-title"
            type="button"
            @click="choose({ mode: 'role', id: group.id, name: group.name })"
          >
            <RiArrowDownSFill />
            <RiFolder3Fill />
            <span>{{ group.name }}</span>
          </button>
          <button
            v-for="role in props.roles.filter((item) => item.groupId === group.id)"
            :key="role.id"
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
        </section>
      </div>
    </div>
  </aside>
</template>

<style scoped lang="scss">
.organization-tree-sidebar {
  display: flex;
  box-sizing: border-box;
  width: 356px;
  min-width: 300px;
  height: 100%;
  padding: 28px 28px 20px;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
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
  &__tree-item,
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
  &__tree-item {
    height: 44px;
    gap: 8px;
    padding: 0 14px;
    font-size: 16px;
  }
  &__tree-item--root {
    padding-left: 12px;
  }
  &__tree-item--child {
    padding-left: 74px;
  }
  &__tree-item--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 6px;
  }
  &__expand-icon {
    width: 18px;
    height: 18px;
    color: var(--el-text-color-secondary);
  }
  &__tree-icon {
    width: 21px;
    height: 21px;
    flex: 0 0 21px;
    color: var(--el-color-primary);
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
  }
  &__role-group-title {
    width: 100%;
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
  &__role-item {
    width: 100%;
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
}
</style>
