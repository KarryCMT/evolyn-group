<script setup lang="ts">
import { RiCloseFill, RiQuestionFill } from '@remixicon/vue';
import { shallowRef, watch } from 'vue';
import type { AddressBookScope } from './administrator.types';

defineOptions({ name: 'AddressBookManagementDrawer' });

const props = defineProps<{
  /** 打开时的当前配置（null 为未配置，按空配置起步）。 */
  initial: AddressBookScope | null;
  /** 保存处理器：内部完成区块 PATCH 与失败回滚，返回 false 保持抽屉开启。 */
  save: (scope: AddressBookScope) => Promise<boolean>;
}>();
const visible = defineModel<boolean>({ default: false });
// 草稿副本：抽屉是显式保存语义，取消不落库；默认值对齐空配置
const emptyScope = (): AddressBookScope => ({
  departmentEnabled: false,
  roleVisible: false,
  roleManage: false,
  externalEnabled: false,
});
const draft = shallowRef<AddressBookScope>(emptyScope());
const saving = shallowRef(false);

watch(visible, (isVisible) => {
  if (isVisible) draft.value = props.initial ? { ...props.initial } : emptyScope();
});

async function submit() {
  saving.value = true;
  try {
    // 失败提示由保存方给出（业务码文案统一在组合层维护）
    if (await props.save(draft.value)) {
      visible.value = false;
    }
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <el-drawer
    v-model="visible"
    class="address-book-management-drawer"
    direction="rtl"
    size="58%"
    :show-close="false"
    :close-on-click-modal="!saving"
    append-to-body
  >
    <template #header
      ><header class="address-book-management-drawer__header">
        <h2>通讯录管理</h2>
        <button type="button" aria-label="关闭" :disabled="saving" @click="visible = false">
          <RiCloseFill />
        </button></header
    ></template>
    <div class="address-book-management-drawer__body">
      <section class="address-book-management-drawer__section">
        <h3>内部部门</h3>
        <el-checkbox v-model="draft.departmentEnabled">可见/可管理</el-checkbox>
      </section>
      <section class="address-book-management-drawer__section">
        <h3>内部角色</h3>
        <div>
          <!-- 可管理必然隐含可见：服务端联动校验，前端勾选联动 -->
          <el-checkbox
            :model-value="draft.roleVisible || draft.roleManage"
            @update:model-value="draft.roleVisible = Boolean($event)"
            >可见</el-checkbox
          ><el-checkbox
            v-model="draft.roleManage"
            @update:model-value="draft.roleManage && (draft.roleVisible = true)"
            >可管理</el-checkbox
          >
        </div>
      </section>
      <section class="address-book-management-drawer__section">
        <h3>互联组织</h3>
        <el-checkbox v-model="draft.externalEnabled">可见/可管理 <RiQuestionFill /></el-checkbox>
      </section>
    </div>
    <template #footer
      ><el-button :loading="saving" type="primary" @click="submit">保存</el-button
      ><el-button :disabled="saving" @click="visible = false">取消</el-button></template
    >
  </el-drawer>
</template>

<style scoped lang="scss">
:global(.address-book-management-drawer .el-drawer__header) {
  height: 56px;
  margin-bottom: 0;
  padding: 0 var(--el-space-3xl);
  border-bottom: 1px solid var(--el-border-color);
}
:global(.address-book-management-drawer .el-drawer__body) {
  padding: 0;
}
:global(.address-book-management-drawer .el-drawer__footer) {
  height: 70px;
  padding: 0 var(--el-space-3xl);
  border-top: 1px solid var(--el-border-color);
}
.address-book-management-drawer {
  &__header {
    display: flex;
    height: 100%;
    align-items: center;
    justify-content: space-between;
  }
  &__header h2 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);
    line-height: 26px;
  }
  &__header button {
    display: inline-flex;
    border: 0;
    padding: var(--el-space-xs);
    background: transparent;
    color: var(--el-text-color-secondary);
    cursor: pointer;
  }
  &__header button:hover {
    border-radius: var(--el-border-radius-base);
    background: var(--el-fill-color-light);
  }
  &__header svg {
    width: 22px;
    height: 22px;
  }
  &__body {
    padding: var(--el-space-4xl) var(--el-space-3xl);
  }
  &__section {
    margin-bottom: var(--el-space-5xl);
  }
  &__section h3 {
    margin: 0 0 var(--el-space-2xl);
    padding-left: var(--el-space-lg);
    border-left: 5px solid var(--el-color-primary);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
  }
  &__section :deep(.el-checkbox) {
    margin-right: var(--el-space-4xl);
    color: #4e5868;
    font-size: var(--el-font-size-medium);
  }
  &__section :deep(.el-checkbox__label) {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-sm);
    font-size: var(--el-font-size-medium);
  }
}
</style>
