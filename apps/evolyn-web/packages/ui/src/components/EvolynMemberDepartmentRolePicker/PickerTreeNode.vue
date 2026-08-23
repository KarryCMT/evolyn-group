<script setup lang="ts">
import {
  RiArrowDownSFill,
  RiArrowRightSFill,
  RiFolderFill,
  RiOrganizationChart,
} from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import type {
  EvolynMemberDepartmentRolePickerItemType,
  EvolynMemberDepartmentRolePickerTreeNode,
} from './EvolynMemberDepartmentRolePicker.types';

defineOptions({ name: 'EvolynMemberDepartmentRolePickerTreeNode' });

const props = defineProps<{
  activeId?: string | number;
  itemType: Exclude<EvolynMemberDepartmentRolePickerItemType, 'member'>;
  mode: 'filter' | 'select';
  multiple: boolean;
  node: EvolynMemberDepartmentRolePickerTreeNode;
  selectedKeys: Set<string>;
  isDisabled: (node: EvolynMemberDepartmentRolePickerTreeNode) => boolean;
}>();

const emit = defineEmits<{
  select: [node: EvolynMemberDepartmentRolePickerTreeNode];
}>();

const expanded = shallowRef(true);
const hasChildren = computed(() => Boolean(props.node.children?.length));
const isSelected = computed(() =>
  props.selectedKeys.has(`${props.itemType}:${String(props.node.id)}`),
);
const isActive = computed(() => String(props.activeId) === String(props.node.id));
const icon = computed(() => (props.itemType === 'department' ? RiOrganizationChart : RiFolderFill));

function toggleExpanded() {
  expanded.value = !expanded.value;
}

function selectNode() {
  if (props.mode === 'select' && props.isDisabled(props.node)) return;
  emit('select', props.node);
}
</script>

<template>
  <li class="evolyn-member-department-role-picker-tree-node">
    <div
      class="evolyn-member-department-role-picker-tree-node__row"
      :class="{
        'evolyn-member-department-role-picker-tree-node__row--active': isActive,
        'evolyn-member-department-role-picker-tree-node__row--disabled':
          mode === 'select' && isDisabled(node),
      }"
    >
      <button
        v-if="hasChildren"
        class="evolyn-member-department-role-picker-tree-node__expand"
        type="button"
        :aria-label="expanded ? `收起${node.label}` : `展开${node.label}`"
        @click="toggleExpanded"
      >
        <RiArrowDownSFill v-if="expanded" aria-hidden="true" />
        <RiArrowRightSFill v-else aria-hidden="true" />
      </button>
      <span v-else class="evolyn-member-department-role-picker-tree-node__spacer" />
      <button
        class="evolyn-member-department-role-picker-tree-node__label"
        type="button"
        :disabled="mode === 'select' && isDisabled(node)"
        @click="selectNode"
      >
        <component :is="icon" aria-hidden="true" />
        <span>{{ node.label }}</span>
      </button>
      <input
        v-if="mode === 'select'"
        class="evolyn-member-department-role-picker-tree-node__checkbox"
        :type="multiple ? 'checkbox' : 'radio'"
        :aria-label="`选择${node.label}`"
        :checked="isSelected"
        :disabled="isDisabled(node)"
        @change="selectNode"
      />
    </div>
    <ul
      v-if="hasChildren && expanded"
      class="evolyn-member-department-role-picker-tree-node__children"
    >
      <PickerTreeNode
        v-for="child in node.children"
        :key="child.id"
        :active-id="activeId"
        :item-type="itemType"
        :mode="mode"
        :multiple="multiple"
        :node="child"
        :selected-keys="selectedKeys"
        :is-disabled="isDisabled"
        @select="emit('select', $event)"
      />
    </ul>
  </li>
</template>
