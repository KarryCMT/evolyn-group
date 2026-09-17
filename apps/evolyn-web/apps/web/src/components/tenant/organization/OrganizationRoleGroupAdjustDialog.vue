<script setup lang="ts">
import { ElDialog } from 'element-plus';
import type { OrganizationRoleGroup } from './organization.types';

const open = defineModel<boolean>({ required: true });
const groupId = defineModel<string>('groupId', { required: true });
const props = defineProps<{
  groups: OrganizationRoleGroup[];
}>();
const emit = defineEmits<{ confirm: [] }>();
</script>

<template>
  <ElDialog v-model="open" title="调整分组" width="620px" align-center>
    <p class="organization-role-group-adjust-dialog__prompt">请选择目标分组</p>
    <div class="organization-role-group-adjust-dialog__list">
      <button
        v-for="group in props.groups"
        :key="group.id"
        :class="{ 'organization-role-group-adjust-dialog__item--active': groupId === group.id }"
        type="button"
        @click="groupId = group.id"
      >
        <span>{{ group.name }}</span
        ><span v-if="groupId === group.id">●</span>
      </button>
    </div>
    <template #footer>
      <el-button @click="open = false">取消</el-button>
      <el-button type="primary" @click="emit('confirm')">确定</el-button>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.organization-role-group-adjust-dialog__prompt {
  margin: 0 0 var(--el-space-lg);
  color: var(--el-text-color-secondary);
}

.organization-role-group-adjust-dialog__list {
  min-height: 260px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-medium);
  overflow: hidden;
}

.organization-role-group-adjust-dialog__list button {
  display: flex;
  box-sizing: border-box;
  width: 100%;
  height: 46px;
  padding: 0 var(--el-space-xl);
  border: 0;
  align-items: center;
  justify-content: space-between;
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.organization-role-group-adjust-dialog__list button:hover {
  background: var(--el-fill-color-light);
}

.organization-role-group-adjust-dialog__item--active {
  color: var(--el-color-primary) !important;
  background: var(--el-color-primary-light-9) !important;
}
</style>
