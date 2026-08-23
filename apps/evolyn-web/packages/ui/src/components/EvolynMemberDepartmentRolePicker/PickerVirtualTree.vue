<script setup lang="ts">
import {
  RiArrowDownSFill,
  RiArrowRightSFill,
  RiFolderFill,
  RiOrganizationChart,
} from '@remixicon/vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { computed, useTemplateRef } from 'vue';
import type {
  EvolynMemberDepartmentRolePickerItemType,
  EvolynMemberDepartmentRolePickerTreeNode,
} from './EvolynMemberDepartmentRolePicker.types';
import type { EvolynMemberDepartmentRolePickerTreeRow } from './useVirtualTree';

defineOptions({ name: 'EvolynMemberDepartmentRolePickerVirtualTree' });

const props = defineProps<{
  activeId?: string | number;
  itemType: Exclude<EvolynMemberDepartmentRolePickerItemType, 'member'>;
  mode: 'filter' | 'select';
  multiple: boolean;
  rows: EvolynMemberDepartmentRolePickerTreeRow[];
  selectedKeys: Set<string>;
  isDisabled: (node: EvolynMemberDepartmentRolePickerTreeNode) => boolean;
}>();

const emit = defineEmits<{
  select: [node: EvolynMemberDepartmentRolePickerTreeNode];
  toggleExpand: [key: string];
}>();

const scrollRef = useTemplateRef<HTMLElement>('scroll');
const icon = computed(() => (props.itemType === 'department' ? RiOrganizationChart : RiFolderFill));
const virtualizer = useVirtualizer(
  computed(() => {
    // 在 computed 内读取模板 ref，挂载后才能让虚拟器重新绑定实际滚动容器。
    const scrollElement = scrollRef.value;
    return {
      count: props.rows.length,
      estimateSize: () => 40,
      getItemKey: (index: number) => props.rows[index]?.key ?? index,
      getScrollElement: () => scrollElement,
      initialRect: { height: 400, width: 600 },
      overscan: 8,
    };
  }),
);
const virtualRows = computed(() => virtualizer.value.getVirtualItems());

function isSelected(node: EvolynMemberDepartmentRolePickerTreeNode) {
  return props.selectedKeys.has(`${props.itemType}:${String(node.id)}`);
}

function isActive(node: EvolynMemberDepartmentRolePickerTreeNode) {
  return String(props.activeId) === String(node.id);
}

function selectNode(node: EvolynMemberDepartmentRolePickerTreeNode) {
  if (props.mode === 'select' && props.isDisabled(node)) return;
  emit('select', node);
}
</script>

<template>
  <div ref="scroll" class="evolyn-member-department-role-picker-virtual-tree" role="tree">
    <ul
      class="evolyn-member-department-role-picker-virtual-tree__content"
      :style="{ height: `${virtualizer.getTotalSize()}px` }"
    >
      <li
        v-for="virtualRow in virtualRows"
        :key="virtualRow.key"
        class="evolyn-member-department-role-picker-virtual-tree__item"
        :style="{ transform: `translateY(${virtualRow.start}px)` }"
      >
        <div
          v-if="rows[virtualRow.index]"
          class="evolyn-member-department-role-picker-tree-node__row"
          :class="{
            'evolyn-member-department-role-picker-tree-node__row--active': isActive(
              rows[virtualRow.index].node,
            ),
            'evolyn-member-department-role-picker-tree-node__row--disabled':
              mode === 'select' && isDisabled(rows[virtualRow.index].node),
          }"
          :style="{ paddingInlineStart: `${rows[virtualRow.index].depth * 26}px` }"
          role="treeitem"
          :aria-expanded="
            rows[virtualRow.index].hasChildren ? rows[virtualRow.index].expanded : undefined
          "
        >
          <button
            v-if="rows[virtualRow.index].hasChildren"
            class="evolyn-member-department-role-picker-tree-node__expand"
            type="button"
            :aria-label="
              rows[virtualRow.index].expanded
                ? `收起${rows[virtualRow.index].node.label}`
                : `展开${rows[virtualRow.index].node.label}`
            "
            @click="emit('toggleExpand', rows[virtualRow.index].key)"
          >
            <RiArrowDownSFill v-if="rows[virtualRow.index].expanded" aria-hidden="true" />
            <RiArrowRightSFill v-else aria-hidden="true" />
          </button>
          <span v-else class="evolyn-member-department-role-picker-tree-node__spacer" />
          <button
            class="evolyn-member-department-role-picker-tree-node__label"
            type="button"
            :disabled="mode === 'select' && isDisabled(rows[virtualRow.index].node)"
            @click="selectNode(rows[virtualRow.index].node)"
          >
            <component :is="icon" aria-hidden="true" />
            <span>{{ rows[virtualRow.index].node.label }}</span>
          </button>
          <input
            v-if="mode === 'select'"
            class="evolyn-member-department-role-picker-tree-node__checkbox"
            :type="multiple ? 'checkbox' : 'radio'"
            :aria-label="`选择${rows[virtualRow.index].node.label}`"
            :checked="isSelected(rows[virtualRow.index].node)"
            :disabled="isDisabled(rows[virtualRow.index].node)"
            @change="selectNode(rows[virtualRow.index].node)"
          />
        </div>
      </li>
    </ul>
  </div>
</template>
