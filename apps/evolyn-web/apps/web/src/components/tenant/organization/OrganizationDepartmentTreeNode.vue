<script setup lang="ts">
import { RiArrowDownSFill, RiGroup2Fill, RiMore2Fill } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type { OrganizationDepartment, OrganizationSelection } from './organization.types';

const props = defineProps<{
  department: OrganizationDepartment;
  depth: number;
  selection: OrganizationSelection;
}>();

const emit = defineEmits<{
  select: [selection: OrganizationSelection];
  action: [action: 'rename' | 'create-child', department: OrganizationDepartment];
}>();

const expanded = shallowRef(true);
const hasChildren = computed(() => Boolean(props.department.children?.length));
const itemStyle = computed(() => ({ paddingLeft: `${12 + props.depth * 28}px` }));

function selectDepartment() {
  emit('select', {
    mode: 'department',
    id: props.department.id,
    name: props.department.name,
  });
}

function toggleExpanded() {
  if (hasChildren.value) expanded.value = !expanded.value;
}

function handleAction(command: 'rename' | 'create-child') {
  emit('action', command, props.department);
}
</script>

<template>
  <div class="organization-department-tree-node">
    <div
      class="organization-department-tree-node__item"
      :class="{
        'organization-department-tree-node__item--active':
          props.selection.id === props.department.id,
      }"
      :style="itemStyle"
    >
      <button
        v-if="hasChildren"
        class="organization-department-tree-node__expand"
        type="button"
        :aria-label="expanded ? '收起子部门' : '展开子部门'"
        @click="toggleExpanded"
      >
        <RiArrowDownSFill
          :class="{ 'organization-department-tree-node__expand--collapsed': !expanded }"
        />
      </button>
      <span
        v-else
        class="organization-department-tree-node__expand-placeholder"
        aria-hidden="true"
      />
      <button
        class="organization-department-tree-node__select"
        type="button"
        @click="selectDepartment"
      >
        <RiGroup2Fill class="organization-department-tree-node__icon" />
        <span class="organization-department-tree-node__name">{{ props.department.name }}</span>
      </button>
      <el-dropdown trigger="click" @command="handleAction">
        <button
          class="organization-department-tree-node__more"
          type="button"
          :aria-label="`${props.department.name} 操作`"
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
    <div v-if="hasChildren && expanded" class="organization-department-tree-node__children">
      <OrganizationDepartmentTreeNode
        v-for="child in props.department.children"
        :key="child.id"
        :department="child"
        :depth="props.depth + 1"
        :selection="props.selection"
        @select="emit('select', $event)"
        @action="(action, department) => emit('action', action, department)"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.organization-department-tree-node {
  &__item {
    display: flex;
    box-sizing: border-box;
    height: 44px;
    padding-right: 6px;
    border-radius: 6px;
    align-items: center;
    color: var(--el-text-color-regular);

    &:hover,
    &--active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
    &:hover .organization-department-tree-node__more,
    &--active .organization-department-tree-node__more,
    .organization-department-tree-node__more:focus-visible {
      opacity: 1;
    }
  }

  &__select {
    display: flex;
    min-width: 0;
    height: 100%;
    padding: 0;
    border: 0;
    flex: 1;
    align-items: center;
    color: inherit;
    background: transparent;
    cursor: pointer;
    font: inherit;
    text-align: left;
  }
  &__expand,
  &__more {
    display: inline-flex;
    width: 28px;
    height: 28px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    flex: 0 0 28px;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-8);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }
    svg {
      width: 19px;
      height: 19px;
    }
  }
  &__expand-placeholder {
    width: 28px;
    flex: 0 0 28px;
  }
  &__expand--collapsed {
    transform: rotate(-90deg);
  }
  &__icon {
    width: 20px;
    height: 20px;
    margin-right: 8px;
    flex: 0 0 20px;
    color: var(--el-color-primary);
  }
  &__name {
    overflow: hidden;
    flex: 1;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__more {
    margin-left: 4px;
    opacity: 0;
    transition:
      opacity 0.18s ease,
      background-color 0.18s ease;
  }
}
</style>
