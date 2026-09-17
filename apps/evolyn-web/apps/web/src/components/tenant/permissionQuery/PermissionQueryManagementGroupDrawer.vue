<script setup lang="ts">
import { RiAddFill, RiCloseFill, RiUserSettingsFill } from '@remixicon/vue';
import type { ManagementGroup } from './permissionQuery.types';

const props = defineProps<{
  group: ManagementGroup;
}>();
const emit = defineEmits<{
  chooseMembers: [];
}>();
const visible = defineModel<boolean>({ required: true });
</script>

<template>
  <el-drawer
    v-model="visible"
    class="permission-query-management-group-drawer"
    direction="rtl"
    size="58%"
    :show-close="false"
    append-to-body
  >
    <template #header>
      <header class="permission-query-management-group-drawer__header">
        <h2>{{ props.group.name }}</h2>
        <button type="button" aria-label="关闭" @click="visible = false">
          <RiCloseFill />
        </button>
      </header>
    </template>
    <section class="permission-query-management-group-drawer__content">
      <h3><i />管理员</h3>
      <button
        class="permission-query-management-group-drawer__choose"
        type="button"
        @click="emit('chooseMembers')"
      >
        <RiAddFill />选择成员
      </button>
      <div
        v-if="props.group.members.length"
        class="permission-query-management-group-drawer__manager-list"
      >
        <span v-for="member in props.group.members" :key="member.id">
          <RiUserSettingsFill />{{ member.name }}
        </span>
      </div>
    </section>
    <template #footer>
      <el-button type="primary" @click="visible = false">保存</el-button>
      <el-button @click="visible = false">取消</el-button>
    </template>
  </el-drawer>
</template>

<style scoped lang="scss">
.permission-query-management-group-drawer__header {
  display: flex;
  height: 100%;
  align-items: center;
  justify-content: space-between;
}

.permission-query-management-group-drawer__header h2 {
  margin: 0;
  font-size: var(--el-font-size-medium);
}

.permission-query-management-group-drawer__header button {
  display: inline-flex;
  padding: var(--el-space-xs);
  border: 0;
  background: transparent;
  cursor: pointer;
}

.permission-query-management-group-drawer__header button:hover {
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-light);
}

.permission-query-management-group-drawer__content {
  padding: var(--el-space-3xl);
}

.permission-query-management-group-drawer__content h3 {
  display: flex;
  margin: 0 0 var(--el-space-3xl);
  align-items: center;
  gap: var(--el-space-md);
  font-size: var(--el-font-size-medium);
}

.permission-query-management-group-drawer__content h3 i {
  width: 5px;
  height: 20px;
  border-radius: var(--el-border-radius-base);
  background: var(--el-color-primary);
}

.permission-query-management-group-drawer__choose {
  display: inline-flex;
  padding: var(--el-space-xs);
  border: 0;
  align-items: center;
  gap: var(--el-space-sm);
  color: var(--el-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: var(--el-font-size-medium);
}

.permission-query-management-group-drawer__choose:hover {
  border-radius: var(--el-border-radius-base);
  background: var(--el-color-primary-light-9);
}

.permission-query-management-group-drawer__choose svg {
  width: 20px;
  height: 20px;
}

.permission-query-management-group-drawer__manager-list {
  display: flex;
  margin-top: var(--el-space-2xl);
  flex-wrap: wrap;
  gap: var(--el-space-md);
}

.permission-query-management-group-drawer__manager-list span {
  display: inline-flex;
  padding: var(--el-space-sm) var(--el-space-md);
  border-radius: var(--el-border-radius-medium);
  align-items: center;
  gap: var(--el-space-sm);
  background: var(--el-fill-color);
}

:global(.permission-query-management-group-drawer .el-drawer__header) {
  height: 56px;
  margin-bottom: 0;
  padding: 0 var(--el-space-3xl);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

:global(.permission-query-management-group-drawer .el-drawer__body) {
  padding: 0;
}

:global(.permission-query-management-group-drawer .el-drawer__footer) {
  height: 70px;
  padding: 0 var(--el-space-3xl);
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>
