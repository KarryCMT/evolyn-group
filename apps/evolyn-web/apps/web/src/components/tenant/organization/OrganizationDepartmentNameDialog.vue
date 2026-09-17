<script setup lang="ts">
import { ElDialog, ElInput } from 'element-plus';
import type { OrganizationDepartment } from './organization.types';

defineProps<{
  mode: 'rename' | 'create-child';
  target: OrganizationDepartment | null;
  submitting: boolean;
}>();

const emit = defineEmits<{
  closed: [];
  confirm: [];
}>();
const open = defineModel<boolean>({ required: true });
const name = defineModel<string>('name', { required: true });
</script>

<template>
  <ElDialog
    v-model="open"
    :title="mode === 'rename' ? '修改部门名称' : '添加子部门'"
    width="480px"
    align-center
    @closed="emit('closed')"
  >
    <p v-if="mode === 'create-child'" class="organization-department-name-dialog__prompt">
      将在「{{ target?.name }}」下创建子部门
    </p>
    <ElInput
      v-model="name"
      :placeholder="mode === 'rename' ? '请输入部门名称' : '请输入子部门名称'"
      maxlength="30"
      show-word-limit
      @keyup.enter="emit('confirm')"
    />
    <template #footer>
      <el-button @click="open = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="emit('confirm')">确定</el-button>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.organization-department-name-dialog__prompt {
  margin: 0 0 var(--el-space-lg);
  color: var(--el-text-color-secondary);
}
</style>
