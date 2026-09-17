<script setup lang="ts">
import type { TenantMembership } from '~/types';
import { ElDialog } from 'element-plus';

defineProps<{
  memberships: readonly TenantMembership[];
  switching: boolean;
}>();

const emit = defineEmits<{
  choose: [membership: TenantMembership];
}>();
const open = defineModel<boolean>({ required: true });
</script>

<template>
  <ElDialog v-model="open" title="选择进入的团队" width="420px" :close-on-click-modal="false">
    <div class="tenant-membership-dialog__list">
      <button
        v-for="membership in memberships"
        :key="membership.tenantId"
        class="tenant-membership-dialog__item"
        type="button"
        :disabled="switching"
        @click="emit('choose', membership)"
      >
        <span class="tenant-membership-dialog__name">{{ membership.name }}</span>
        <span class="tenant-membership-dialog__code">{{ membership.code }}</span>
        <el-tag v-if="membership.isOwner" size="small"> 所有者 </el-tag>
      </button>
    </div>
  </ElDialog>
</template>

<style scoped lang="scss">
.tenant-membership-dialog__list {
  display: flex;
  flex-direction: column;
  gap: var(--el-space-md);
}

.tenant-membership-dialog__item {
  display: flex;
  gap: var(--el-space-lg);
  align-items: center;
  padding: var(--el-space-lg) var(--el-space-xl);
  text-align: left;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  cursor: pointer;
  transition: border-color 0.2s;

  &:hover:not(:disabled) {
    border-color: var(--el-color-primary);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.tenant-membership-dialog__name {
  font-size: var(--el-font-size-medium);
  color: var(--el-text-color-primary);
  font-weight: 500;
}

.tenant-membership-dialog__code {
  flex: 1;
  color: var(--el-text-color-secondary);
  font-size: var(--el-font-size-base);
}
</style>
