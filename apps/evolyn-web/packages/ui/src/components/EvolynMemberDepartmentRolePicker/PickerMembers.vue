<script setup lang="ts">
import { RiUserFill } from '@remixicon/vue';
import type { EvolynMemberDepartmentRolePickerMember } from './EvolynMemberDepartmentRolePicker.types';

defineOptions({ name: 'EvolynMemberDepartmentRolePickerMembers' });

const props = defineProps<{
  isDisabled: (member: EvolynMemberDepartmentRolePickerMember) => boolean;
  members: EvolynMemberDepartmentRolePickerMember[];
  multiple: boolean;
  selectedKeys: Set<string>;
}>();

const emit = defineEmits<{
  select: [member: EvolynMemberDepartmentRolePickerMember];
}>();

function isSelected(member: EvolynMemberDepartmentRolePickerMember) {
  return props.selectedKeys.has(`member:${String(member.id)}`);
}
</script>

<template>
  <ul class="evolyn-member-department-role-picker-members" aria-label="成员列表">
    <li
      v-for="member in props.members"
      :key="member.id"
      class="evolyn-member-department-role-picker-members__item"
      :class="{
        'evolyn-member-department-role-picker-members__item--disabled': isDisabled(member),
      }"
    >
      <button
        class="evolyn-member-department-role-picker-members__label"
        type="button"
        :disabled="isDisabled(member)"
        @click="emit('select', member)"
      >
        <img
          v-if="member.avatarUrl"
          class="evolyn-member-department-role-picker-members__avatar"
          :src="member.avatarUrl"
          alt=""
        />
        <span v-else class="evolyn-member-department-role-picker-members__avatar-placeholder">
          <RiUserFill aria-hidden="true" />
        </span>
        <span>{{ member.label }}</span>
      </button>
      <input
        class="evolyn-member-department-role-picker-members__checkbox"
        :type="multiple ? 'checkbox' : 'radio'"
        :aria-label="`选择${member.label}`"
        :checked="isSelected(member)"
        :disabled="isDisabled(member)"
        @change="emit('select', member)"
      />
    </li>
  </ul>
</template>
