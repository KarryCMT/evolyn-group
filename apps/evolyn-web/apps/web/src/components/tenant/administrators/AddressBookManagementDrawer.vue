<script setup lang="ts">
import { RiCloseFill, RiQuestionFill } from '@remixicon/vue';
import type { AdministratorGroup } from './administrator.types';

defineOptions({ name: 'AddressBookManagementDrawer' });

defineProps<{ group: AdministratorGroup }>();
const visible = defineModel<boolean>({ default: false });
</script>

<template>
  <el-drawer
    v-model="visible"
    class="address-book-management-drawer"
    direction="rtl"
    size="58%"
    :show-close="false"
    append-to-body
  >
    <template #header
      ><header class="address-book-management-drawer__header">
        <h2>通讯录管理</h2>
        <button type="button" aria-label="关闭" @click="visible = false">
          <RiCloseFill />
        </button></header
    ></template>
    <div class="address-book-management-drawer__body">
      <section class="address-book-management-drawer__section">
        <h3>内部部门</h3>
        <el-checkbox v-model="group.departmentEnabled">可见/可管理</el-checkbox>
      </section>
      <section class="address-book-management-drawer__section">
        <h3>内部角色</h3>
        <div>
          <el-checkbox v-model="group.roleVisible">可见</el-checkbox
          ><el-checkbox v-model="group.roleManage">可管理</el-checkbox>
        </div>
      </section>
      <section class="address-book-management-drawer__section">
        <h3>互联组织</h3>
        <el-checkbox v-model="group.externalEnabled">可见/可管理 <RiQuestionFill /></el-checkbox>
      </section>
    </div>
    <template #footer
      ><el-button type="primary" @click="visible = false">保存</el-button
      ><el-button @click="visible = false">取消</el-button></template
    >
  </el-drawer>
</template>

<style scoped lang="scss">
:global(.address-book-management-drawer .el-drawer__header) {
  height: 56px;
  margin-bottom: 0;
  padding: 0 28px;
  border-bottom: 1px solid #dde2ea;
}
:global(.address-book-management-drawer .el-drawer__body) {
  padding: 0;
}
:global(.address-book-management-drawer .el-drawer__footer) {
  height: 70px;
  padding: 0 28px;
  border-top: 1px solid #dde2ea;
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
    color: #273142;
    font-size: 22px;
  }
  &__header button {
    display: inline-flex;
    border: 0;
    padding: 4px;
    background: transparent;
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
  &__body {
    padding: 30px 28px;
  }
  &__section {
    margin-bottom: 38px;
  }
  &__section h3 {
    margin: 0 0 22px;
    padding-left: 14px;
    border-left: 5px solid var(--el-color-primary);
    color: #273142;
    font-size: 20px;
  }
  &__section :deep(.el-checkbox) {
    margin-right: 32px;
    color: #4e5868;
    font-size: 17px;
  }
  &__section :deep(.el-checkbox__label) {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    font-size: 17px;
  }
}
</style>
