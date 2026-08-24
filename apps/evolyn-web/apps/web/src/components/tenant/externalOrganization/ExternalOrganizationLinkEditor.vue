<script setup lang="ts">
import { reactive, watch } from 'vue';
import type { ExternalOrganizationLink } from './externalOrganization.types';

const props = defineProps<{
  link: ExternalOrganizationLink | null;
}>();

const visible = defineModel<boolean>('visible', { required: true });
const emit = defineEmits<{
  save: [link: ExternalOrganizationLink];
}>();

const form = reactive<ExternalOrganizationLink>({
  id: '',
  url: '',
  label: '',
  role: '',
  directoryPermission: '代管通讯录',
  enabled: true,
});

const roleOptions = ['外部协作成员', '外部管理员'];
const permissionOptions = ['代管通讯录', '只读通讯录'];

watch(
  () => props.link,
  (link) => {
    if (link) Object.assign(form, link);
  },
  { immediate: true },
);

function confirm() {
  emit('save', { ...form, label: form.label.trim(), role: form.role.trim() });
  visible.value = false;
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="external-organization-link-editor"
    title="编辑链接"
    width="min(688px, calc(100vw - 32px))"
    align-center
    append-to-body
    destroy-on-close
  >
    <el-form class="external-organization-link-editor__form" label-position="top">
      <el-form-item label="互联组织标签">
        <el-select v-model="form.label" placeholder="设置互联组织标签" clearable>
          <el-option label="供应商" value="供应商" />
          <el-option label="经销商" value="经销商" />
          <el-option label="合作伙伴" value="合作伙伴" />
        </el-select>
      </el-form-item>
      <el-form-item label="互联对接人角色">
        <el-select v-model="form.role" placeholder="请选择角色" clearable>
          <el-option v-for="role in roleOptions" :key="role" :label="role" :value="role" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <template #label>
          <span>通讯录权限</span>
          <small>可设置代管对方通讯录，代管状态可以自主给对方通讯录添加成员</small>
        </template>
        <el-select v-model="form.directoryPermission">
          <el-option
            v-for="permission in permissionOptions"
            :key="permission"
            :label="permission"
            :value="permission"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="链接状态">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="confirm">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.external-organization-link-editor {
  &__form :deep(.el-form-item) {
    margin-bottom: 24px;
  }
  &__form :deep(.el-form-item__label) {
    color: var(--el-text-color-primary);
    font-size: 16px;
  }
  &__form :deep(.el-select) {
    width: 100%;
  }
  &__form small {
    display: block;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    font-weight: 400;
    line-height: 20px;
  }
}

:global(.external-organization-link-editor) {
  /* 弹窗传送至 body 后不继承管理后台的页面变量，显式固定为浅色管理端表面。 */
  --el-bg-color: #ffffff;
  --el-bg-color-overlay: #ffffff;
  --el-fill-color: #f4f6fa;
  --el-fill-color-light: #f7f8fc;
  --el-fill-color-lighter: #fafbfc;
  --el-fill-color-blank: #ffffff;
  --el-text-color-primary: #202938;
  --el-text-color-regular: #515968;
  --el-text-color-secondary: #8a94a6;
  --el-border-color: #dbe1eb;
  --el-border-color-light: #e7eaf0;
  border-radius: 14px;
  background: #ffffff;
  color-scheme: light;
}
:global(.external-organization-link-editor .el-dialog__header) {
  height: 56px;
  box-sizing: border-box;
  margin-right: 0;
  padding: 15px 28px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
:global(.external-organization-link-editor .el-dialog__title) {
  font-size: 18px;
  font-weight: 650;
  line-height: 26px;
}
:global(.external-organization-link-editor .el-dialog__headerbtn) {
  top: 12px;
  right: 20px;
  width: 32px;
  height: 32px;
}
:global(.external-organization-link-editor .el-dialog__body) {
  padding: 28px 28px 8px;
}
:global(.external-organization-link-editor .el-dialog__footer) {
  padding: 16px 28px;
  border-top: 1px solid var(--el-border-color-lighter);
}
:global(.external-organization-link-editor .el-select__wrapper) {
  background: #ffffff;
}
:global(.external-organization-link-editor .el-button:not(.el-button--primary)) {
  --el-button-bg-color: #ffffff;
  --el-button-text-color: var(--el-text-color-regular);
  --el-button-border-color: var(--el-border-color);
}
</style>
