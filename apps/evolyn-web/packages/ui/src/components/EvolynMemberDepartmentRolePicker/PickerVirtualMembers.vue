<script setup lang="ts">
import { RiUserFill } from '@remixicon/vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { computed, useTemplateRef } from 'vue';
import type { EvolynMemberDepartmentRolePickerMember } from './EvolynMemberDepartmentRolePicker.types';

defineOptions({ name: 'EvolynMemberDepartmentRolePickerVirtualMembers' });

const props = defineProps<{
  isDisabled: (member: EvolynMemberDepartmentRolePickerMember) => boolean;
  members: EvolynMemberDepartmentRolePickerMember[];
  multiple: boolean;
  selectedKeys: Set<string>;
}>();

const emit = defineEmits<{
  select: [member: EvolynMemberDepartmentRolePickerMember];
}>();

const scrollRef = useTemplateRef<HTMLElement>('scroll');
const virtualizer = useVirtualizer(
  computed(() => {
    // 在 computed 内读取模板 ref，挂载后才能让虚拟器重新绑定实际滚动容器。
    const scrollElement = scrollRef.value;
    return {
      count: props.members.length,
      estimateSize: () => 44,
      getItemKey: (index: number) => props.members[index]?.id ?? index,
      getScrollElement: () => scrollElement,
      initialRect: { height: 400, width: 600 },
      overscan: 8,
    };
  }),
);
const virtualRows = computed(() => virtualizer.value.getVirtualItems());

function isSelected(member: EvolynMemberDepartmentRolePickerMember) {
  return props.selectedKeys.has(`member:${String(member.id)}`);
}
</script>

<template>
  <div ref="scroll" class="evolyn-member-department-role-picker-virtual-members">
    <ul
      class="evolyn-member-department-role-picker-virtual-members__content"
      aria-label="成员列表"
      :style="{ height: `${virtualizer.getTotalSize()}px` }"
    >
      <li
        v-for="virtualRow in virtualRows"
        :key="virtualRow.key"
        class="evolyn-member-department-role-picker-virtual-members__item"
        :style="{ transform: `translateY(${virtualRow.start}px)` }"
      >
        <div
          v-if="members[virtualRow.index]"
          class="evolyn-member-department-role-picker-members__item"
          :class="{
            'evolyn-member-department-role-picker-members__item--disabled': isDisabled(
              members[virtualRow.index],
            ),
          }"
        >
          <button
            class="evolyn-member-department-role-picker-members__label"
            type="button"
            :disabled="isDisabled(members[virtualRow.index])"
            @click="emit('select', members[virtualRow.index])"
          >
            <img
              v-if="members[virtualRow.index].avatarUrl"
              class="evolyn-member-department-role-picker-members__avatar"
              :src="members[virtualRow.index].avatarUrl"
              alt=""
            />
            <span v-else class="evolyn-member-department-role-picker-members__avatar-placeholder">
              <RiUserFill aria-hidden="true" />
            </span>
            <span>{{ members[virtualRow.index].label }}</span>
          </button>
          <input
            class="evolyn-member-department-role-picker-members__checkbox"
            :type="multiple ? 'checkbox' : 'radio'"
            :aria-label="`选择${members[virtualRow.index].label}`"
            :checked="isSelected(members[virtualRow.index])"
            :disabled="isDisabled(members[virtualRow.index])"
            @change="emit('select', members[virtualRow.index])"
          />
        </div>
      </li>
    </ul>
  </div>
</template>
