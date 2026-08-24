<script setup lang="ts">
import { RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { shallowRef, watch } from 'vue';

defineOptions({ name: 'AddAdministratorGroupDialog' });

const visible = defineModel<boolean>({ default: false });
const emit = defineEmits<{ confirm: [name: string] }>();
const groupName = shallowRef('');

function submit() {
  const name = groupName.value.trim();
  if (!name) {
    ElMessage.warning('请输入管理组名称');
    return;
  }
  emit('confirm', name);
  visible.value = false;
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
      @keyup.enter="submit"
    />
    <footer class="add-administrator-group-dialog__footer">
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="submit">确定</el-button>
    </footer>
  </el-dialog>
</template>

<style scoped lang="scss">
:global(.add-administrator-group-dialog) {
  border-radius: 14px;
}
:global(.add-administrator-group-dialog .el-dialog__header) {
  display: none;
}
:global(.add-administrator-group-dialog .el-dialog__body) {
  padding: 28px;
}
.add-administrator-group-dialog {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
  }
  &__header h2 {
    margin: 0;
    color: #273142;
    font-size: 21px;
    line-height: 28px;
  }
  &__header button {
    display: inline-flex;
    border: 0;
    padding: 3px;
    background: transparent;
    color: #6f7886;
    cursor: pointer;
  }
  &__header button:hover {
    border-radius: 5px;
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
    margin-top: 36px;
    justify-content: flex-end;
    gap: 12px;
  }
  &__footer .el-button {
    min-width: 74px;
    height: 44px;
    font-size: 17px;
  }
}
</style>
