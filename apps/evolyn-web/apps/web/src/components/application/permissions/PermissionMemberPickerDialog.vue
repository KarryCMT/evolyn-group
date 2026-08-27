<script setup lang="ts">
import { computed, shallowRef } from 'vue';
import type { CreatePermissionGroupPayload, PermissionSubject } from './permission.types';

defineOptions({ name: 'PermissionMemberPickerDialog' });

const props = defineProps<{
  modelValue: boolean;
  title: string;
  subjects: PermissionSubject[];
}>();

const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  confirm: [payload: CreatePermissionGroupPayload];
}>();

const groupName = shallowRef('');
const selectedSubjectIds = shallowRef<string[]>([]);
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (visible: boolean) => emit('update:modelValue', visible),
});

function toggleSubject(subjectId: string) {
  selectedSubjectIds.value = selectedSubjectIds.value.includes(subjectId)
    ? selectedSubjectIds.value.filter((id) => id !== subjectId)
    : [...selectedSubjectIds.value, subjectId];
}

function confirm() {
  if (!selectedSubjectIds.value.length) return;
  emit('confirm', {
    groupName: groupName.value.trim(),
    subjectIds: selectedSubjectIds.value,
  });
  groupName.value = '';
  selectedSubjectIds.value = [];
  dialogVisible.value = false;
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    class="permission-member-picker-dialog"
    :title="props.title"
    width="580px"
  >
    <div class="permission-member-picker-dialog__intro">
      选择授权对象；正式接口接入后，此处将展示当前租户的成员、部门和角色。
    </div>
    <label class="permission-member-picker-dialog__name-field">
      <span>权限组名称</span>
      <el-input v-model="groupName" placeholder="留空则使用预定义权限组名称" maxlength="30" />
    </label>
    <div class="permission-member-picker-dialog__subject-list" aria-label="可授权对象">
      <button
        v-for="subject in props.subjects"
        :key="subject.id"
        class="permission-member-picker-dialog__subject"
        :class="{
          'permission-member-picker-dialog__subject--selected': selectedSubjectIds.includes(
            subject.id,
          ),
        }"
        type="button"
        @click="toggleSubject(subject.id)"
      >
        <span class="permission-member-picker-dialog__subject-name">{{ subject.name }}</span>
        <span class="permission-member-picker-dialog__subject-type">{{
          subject.type === 'member' ? '成员' : subject.type === 'department' ? '部门' : '角色'
        }}</span>
      </button>
    </div>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :disabled="!selectedSubjectIds.length" @click="confirm"
        >确定</el-button
      >
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.permission-member-picker-dialog {
  &__intro {
    margin-bottom: var(--el-space-2xl);
    padding: var(--el-space-md) var(--el-space-lg);
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-small);
    line-height: 20px;
  }

  &__name-field {
    display: grid;
    align-items: center;
    grid-template-columns: 92px minmax(0, 1fr);
    gap: var(--el-space-lg);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
  }

  &__subject-list {
    display: grid;
    margin-top: var(--el-space-2xl);
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
  }

  &__subject {
    display: flex;
    min-height: 54px;
    padding: var(--el-space-md) var(--el-space-lg);
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-md);
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: var(--el-fill-color-blank);
    font: inherit;
    text-align: left;

    &:hover {
      border-color: var(--el-color-primary-light-5);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &--selected {
      border-color: var(--el-color-primary);
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__subject-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__subject-type {
    flex: 0 0 auto;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-extra-small);
  }
}

@media (max-width: 620px) {
  .permission-member-picker-dialog {
    &__subject-list {
      grid-template-columns: 1fr;
    }
  }
}
</style>
