<script setup lang="ts">
import { RiErrorWarningFill } from '@remixicon/vue';

defineOptions({ name: 'AccountCancellationDialog' });

const model = defineModel<boolean>({ required: true });
const emit = defineEmits<{
  confirm: [];
}>();

function close() {
  model.value = false;
}
</script>

<template>
  <el-dialog
    v-model="model"
    width="520px"
    class="account-cancellation-dialog"
    :show-close="false"
    :close-on-click-modal="false"
  >
    <section class="account-cancellation-dialog__content">
      <el-icon class="account-cancellation-dialog__icon"><RiErrorWarningFill /></el-icon>
      <div>
        <h2>你确定要注销当前账号吗？</h2>
        <p>注销后，账号将无法使用且无法恢复。账号下的成员身份、关联权限及第三方凭证会被删除。</p>
        <p class="account-cancellation-dialog__notice">
          如你仍是某个团队的创建人，请先转移创建人或注销团队。
        </p>
      </div>
    </section>

    <template #footer>
      <el-button @click="close">暂不注销</el-button>
      <el-button type="danger" @click="emit('confirm')">继续注销</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.account-cancellation-dialog) {
  --el-dialog-padding-primary: 0;

  border-radius: var(--el-border-radius-large);
}

:global(.account-cancellation-dialog .el-dialog__body) {
  padding: 0;
}

:global(.account-cancellation-dialog .el-dialog__footer) {
  display: flex;
  justify-content: flex-end;
  gap: var(--el-space-md);
  padding: 0 var(--el-space-3xl) var(--el-space-3xl);
}

:global(.account-cancellation-dialog .el-dialog__footer .el-button) {
  min-width: 96px;
  height: 38px;
  margin: 0;
}

.account-cancellation-dialog {
  &__content {
    display: flex;
    gap: var(--el-space-xl);
    padding: var(--el-space-4xl) var(--el-space-3xl) var(--el-space-3xl);
  }

  &__icon {
    flex: none;
    margin-top: var(--el-space-xs);
    color: var(--el-color-warning);
    font-size: 42px;
  }

  &__content h2 {
    margin: 0 0 var(--el-space-lg);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-extra-large);
    line-height: 28px;
  }

  &__content p {
    margin: 0;
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    line-height: 22px;
  }

  &__notice {
    margin-top: var(--el-space-lg) !important;
    color: var(--el-color-danger) !important;
  }
}

@media (max-width: 640px) {
  :global(.account-cancellation-dialog) {
    width: calc(100% - 32px) !important;
  }

  .account-cancellation-dialog {
    &__content {
      padding: var(--el-space-3xl) var(--el-space-2xl) var(--el-space-2xl);
    }
  }
}
</style>
