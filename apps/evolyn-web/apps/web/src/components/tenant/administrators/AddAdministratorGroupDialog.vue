<script setup lang="ts">
import { RiCloseFill } from '@remixicon/vue';
import { shallowRef, watch } from 'vue';

defineOptions({ name: 'AddAdministratorGroupDialog' });

const visible = defineModel<boolean>({ default: false });
const props = defineProps<{
  /** 提交处理器：内部完成创建与列表刷新，返回 false 表示失败（保持弹窗）。 */
  submit: (name: string) => Promise<boolean>;
}>();
const groupName = shallowRef('');
const submitting = shallowRef(false);

async function submit() {
  const name = groupName.value.trim();
  if (!name) {
    return;
  }
  submitting.value = true;
  try {
    // 失败提示由提交方给出（业务码文案统一在组合层维护）
    if (await props.submit(name)) {
      visible.value = false;
    }
  } finally {
    submitting.value = false;
  }
}

watch(visible, (isVisible) => {
  if (isVisible) groupName.value = '';
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="add-administrator-group-dialog"
    width="604px"
    :show-close="false"
    append-to-body
  >
    <header class="add-administrator-group-dialog__header">
      <h2>添加管理组</h2>
      <button type="button" aria-label="关闭" @click="visible = false"><RiCloseFill /></button>
    </header>
    <el-input
      v-model="groupName"
      class="add-administrator-group-dialog__input"
      placeholder="请输入管理组名称"
      maxlength="30"
      show-word-limit: disabled="submitting"
      @keyup.enter="submit"
    />
    <footer class="add-administrator-group-dialog__footer">
      <el-button :disabled="submitting" @click="visible = false">取消</el-button>
      <el-button :loading="submitting" type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.add-administrator-group-dialog) {
  border-radius: var(--el-border-radius-large);
}
:global(.add-administrator-group-dialog .el-dialog__header) {
  display: none;
}
:global(.add-administrator-group-dialog .el-dialog__body) {
  padding: var(--el-space-3xl);
}
.add-administrator-group-dialog {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--el-space-3xl);
  }
  &__header h2 {
    margin: 0;
    color: #273142;
    font-size: var(--el-font-size-extra-large);
    line-height: 28px;
  }
  &__header button {
    display: inline-flex;
    border: 0;
    padding: var(--el-space-xs);
    background: transparent;
    color: #6f7886;
    cursor: pointer;
  }
  &__header button:hover {
    border-radius: var(--el-border-radius-base);
    background: var(--el-fill-color-light);
  }
  &__header svg {
    width: 24px;
    height: 24px;
  }
  &__input :deep(.el-input__wrapper) {
    min-height: 46px;
    box-shadow: 0 0 0 1px var(--el-color-primary) inset;
  }
  &__footer {
    display: flex;
    margin-top: var(--el-space-4xl);
    justify-content: flex-end;
    gap: var(--el-space-lg);
  }
  &__footer .el-button {
    min-width: 74px;
    height: 44px;
    font-size: var(--el-font-size-medium);
  }
}
</style>
