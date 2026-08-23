<script setup lang="ts">
import { RiCloseFill, RiOrganizationChart, RiTeamFill, RiUserFill } from '@remixicon/vue';
import { computed } from 'vue';
import type { Component } from 'vue';
import type { EvolynMemberDepartmentRolePickerSelection } from './EvolynMemberDepartmentRolePicker.types';

defineOptions({ name: 'EvolynMemberDepartmentRolePickerSelectedItems' });

const props = defineProps<{
  selections: EvolynMemberDepartmentRolePickerSelection[];
}>();

const emit = defineEmits<{
  remove: [selection: EvolynMemberDepartmentRolePickerSelection];
}>();

const selectionIcon = computed<
  Record<EvolynMemberDepartmentRolePickerSelection['type'], Component>
>(() => ({
  department: RiOrganizationChart,
  role: RiTeamFill,
  member: RiUserFill,
}));
</script>

<template>
  <div class="evolyn-member-department-role-picker-selected-items" aria-label="已选择对象">
    <p
      v-if="!props.selections.length"
      class="evolyn-member-department-role-picker-selected-items__empty"
    >
      暂未选择成员、部门或角色
    </p>
    <div v-else class="evolyn-member-department-role-picker-selected-items__list">
      <span
        v-for="selection in props.selections"
        :key="`${selection.type}:${String(selection.id)}`"
        class="evolyn-member-department-role-picker-selected-items__item"
      >
        <img
          v-if="selection.type === 'member' && selection.avatarUrl"
          class="evolyn-member-department-role-picker-selected-items__avatar"
          :src="selection.avatarUrl"
          alt=""
        />
        <component
          v-else
          :is="selectionIcon[selection.type]"
          class="evolyn-member-department-role-picker-selected-items__icon"
          aria-hidden="true"
        />
        <span class="evolyn-member-department-role-picker-selected-items__label">{{
          selection.label
        }}</span>
        <button
          class="evolyn-member-department-role-picker-selected-items__remove"
          type="button"
          :aria-label="`移除${selection.label}`"
          @click="emit('remove', selection)"
        >
          <RiCloseFill aria-hidden="true" />
        </button>
      </span>
    </div>
  </div>
</template>
