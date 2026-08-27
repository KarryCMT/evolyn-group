<script setup lang="ts">
import { RiCloseFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, watch } from 'vue';
import OrganizationInviteBatchImportTab from './OrganizationInviteBatchImportTab.vue';
import OrganizationInviteManualTab from './OrganizationInviteManualTab.vue';
import OrganizationInvitePublicLinkTab from './OrganizationInvitePublicLinkTab.vue';
import type { OrganizationDepartment } from './organization.types';
import { useMemberInvitation } from '~/composables/tenant/useMemberInvitation';

interface DepartmentOption {
  value: number;
  label: string;
  children?: DepartmentOption[];
}

const props = defineProps<{
  departments: OrganizationDepartment;
}>();

const visible = defineModel<boolean>({ required: true });
const emit = defineEmits<{
  completed: [];
}>();

const {
  activeTab,
  submitting,
  publicLinkLoading,
  importResult,
  publicLink,
  form,
  publicInvitationUrl,
  clearManualForm,
  submitManualInvitation,
  submitImport,
  loadPublicLink,
  setPublicLinkEnabled,
  reset,
} = useMemberInvitation();

const departmentOptions = computed(() => {
  function mapDepartment(department: OrganizationDepartment): DepartmentOption | null {
    if (department.isTenantRoot || !Number.isFinite(Number(department.id))) {
      return null;
    }
    return {
      value: Number(department.id),
      label: department.name,
      children: department.children
        ?.map(mapDepartment)
        .filter((item): item is DepartmentOption => item !== null),
    };
  }
  return (
    props.departments.children
      ?.map(mapDepartment)
      .filter((item): item is DepartmentOption => item !== null) ?? []
  );
});

function close() {
  visible.value = false;
}

async function invite() {
  if (!form.name.trim()) {
    ElMessage.warning('请填写姓名');
    return;
  }
  if (!form.phone.trim() && !form.email.trim()) {
    ElMessage.warning('手机号和邮箱至少填写一项');
    return;
  }
  try {
    await submitManualInvitation();
    ElMessage.success('成员邀请已创建');
    emit('completed');
    close();
  } catch {
    ElMessage.error('邀请创建失败，请检查成员信息');
  }
}

async function importFile(file: File) {
  try {
    const result = await submitImport(file);
    if (result.successCount > 0) emit('completed');
    ElMessage.success(`已创建 ${result.successCount} 条成员邀请`);
    return result;
  } catch (error) {
    throw error;
  }
}

async function updatePublicLink(enabled: boolean) {
  try {
    await setPublicLinkEnabled(enabled);
    ElMessage.success(enabled ? '公开邀请链接已开启' : '公开邀请链接已关闭');
  } catch {
    ElMessage.error('公开邀请链接设置失败');
  }
}

watch(
  () => visible.value,
  (open) => {
    if (!open) {
      reset();
      return;
    }
    void loadPublicLink();
  },
);

watch(activeTab, (tab) => {
  if (tab === 'public') void loadPublicLink();
});
</script>

<template>
  <el-dialog
    v-model="visible"
    class="organization-invite-dialog"
    width="640px"
    top="64px"
    :style="{ width: 'min(640px, calc(100vw - 24px))' }"
    :show-close="false"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <template #header>
      <header class="organization-invite-dialog__header">
        <h1>邀请成员</h1>
        <button type="button" aria-label="关闭邀请成员" @click="close"><RiCloseFill /></button>
      </header>
    </template>

    <el-scrollbar class="organization-invite-dialog__content">
      <el-tabs v-model="activeTab" class="organization-invite-dialog__tabs" stretch>
        <el-tab-pane label="手动添加成员" name="manual">
          <OrganizationInviteManualTab
            v-model:form="form"
            :departments="departmentOptions"
            :submitting="submitting"
            @submit="invite"
            @clear="clearManualForm"
          />
        </el-tab-pane>
        <el-tab-pane label="批量导入成员" name="batch">
          <OrganizationInviteBatchImportTab
            :submitting="submitting"
            :result="importResult"
            :upload="importFile"
          />
        </el-tab-pane>
        <el-tab-pane label="公开链接邀请" name="public">
          <OrganizationInvitePublicLinkTab
            :enabled="publicLink?.enabled ?? false"
            :loading="publicLinkLoading"
            :invitation-url="publicInvitationUrl"
            @update:enabled="updatePublicLink"
          />
        </el-tab-pane>
      </el-tabs>
    </el-scrollbar>
  </el-dialog>
</template>

<style scoped lang="scss">
.organization-invite-dialog {
  &__header {
    display: flex;
    height: 56px;
    padding: 0 var(--el-space-2xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: space-between;
  }

  &__header h1 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 650;
    line-height: 28px;
  }
  &__header button {
    display: grid;
    width: 32px;
    height: 32px;
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-medium);
    place-items: center;
    color: var(--el-text-color-regular);
    background: transparent;
    cursor: pointer;
  }
  &__header button:hover {
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
  }
  &__header button svg {
    width: 24px;
    height: 24px;
  }
  &__content {
    height: min(460px, calc(100vh - 128px));
  }
  &__content :deep(.el-scrollbar__view) {
    min-height: 100%;
    padding: var(--el-space-xl) var(--el-space-2xl);
  }
  &__tabs {
    display: flex;
    height: 100%;
    width: 100%;
    min-width: 0;
    flex-direction: column;
  }
  &__tabs :deep(.el-tabs__header) {
    margin: 0 0 var(--el-space-xl);
  }
  &__tabs :deep(.el-tabs__nav-wrap::after) {
    height: 1px;
  }
  &__tabs :deep(.el-tabs__item) {
    height: 42px;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    line-height: 42px;
  }
  &__tabs :deep(.el-tabs__item.is-active) {
    color: var(--el-color-primary);
    font-weight: 600;
  }
  &__tabs :deep(.el-tabs__active-bar) {
    height: 4px;
  }
  &__tabs :deep(.el-tabs__content) {
    min-height: 0;
    flex: 1;
  }
  &__tabs :deep(.el-tab-pane) {
    height: 100%;
  }
}

:global(.organization-invite-dialog) {
  --el-dialog-padding-primary: 0;
  width: min(640px, calc(100vw - 24px)) !important;
  margin-bottom: var(--el-space-3xl);
  border-radius: var(--el-border-radius-round);
  background: var(--el-bg-color-overlay);
}

:global(.organization-invite-dialog .el-dialog__header),
:global(.organization-invite-dialog .el-dialog__body) {
  margin: 0;
  padding: 0;
}

@media (max-width: 720px) {
  .organization-invite-dialog__header {
    height: 56px;
    padding: 0 var(--el-space-2xl);
  }
  .organization-invite-dialog__header h1 {
    font-size: var(--el-font-size-large);
    line-height: 26px;
  }
  .organization-invite-dialog__header button {
    width: 32px;
    height: 32px;
  }
  .organization-invite-dialog__header button svg {
    width: 22px;
    height: 22px;
  }
  .organization-invite-dialog__content {
    height: calc(100vh - 140px);
  }
  .organization-invite-dialog__content :deep(.el-scrollbar__view) {
    padding: var(--el-space-2xl) var(--el-space-2xl);
  }
  .organization-invite-dialog__tabs :deep(.el-tabs__item) {
    font-size: var(--el-font-size-base);
  }
}
</style>
