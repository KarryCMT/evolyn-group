<script setup lang="ts">
import { RiAddFill, RiCloseFill, RiUserSettingsFill } from '@remixicon/vue';
import type {
  ManagementGroup,
  PermissionMember,
  PermissionWorkspace,
} from './permissionQuery.types';
import PermissionQueryMembersDialog from './PermissionQueryMembersDialog.vue';

defineOptions({ name: 'PermissionQueryManagementGroups' });

const props = defineProps<{
  groups: ManagementGroup[];
  members: PermissionMember[];
  selectedGroup: ManagementGroup;
  workspace: PermissionWorkspace;
}>();
const drawerVisible = defineModel<boolean>('drawerVisible', { default: false });
const memberPickerVisible = defineModel<boolean>('memberPickerVisible', { default: false });
const emit = defineEmits<{
  edit: [id: string];
  saveMembers: [members: PermissionMember[]];
}>();
</script>

<template>
  <section class="permission-query-management-groups">
    <div class="permission-query-management-groups__filter">
      <el-select model-value="管理员" aria-label="管理组类型"
        ><el-option label="管理员" value="管理员"
      /></el-select>
      <el-select v-if="props.workspace === 'product'" model-value="" aria-label="应用范围"
        ><el-option label="全部应用" value=""
      /></el-select>
    </div>
    <el-table :data="props.groups" class="permission-query-management-groups__table" height="100%">
      <el-table-column prop="name" label="管理组名称" min-width="230" />
      <el-table-column prop="type" label="管理组类型" min-width="230" />
      <el-table-column label="管理员" min-width="180"
        ><template #default="{ row }">{{
          row.members.map((member: PermissionMember) => member.name).join('、')
        }}</template></el-table-column
      >
      <el-table-column
        v-if="props.workspace === 'product'"
        prop="applicationScope"
        label="应用权限范围"
        min-width="230"
      />
      <el-table-column label="操作" width="128"
        ><template #default="{ row }"
          ><button
            class="permission-query-management-groups__link"
            type="button"
            @click="emit('edit', row.id)"
          >
            编辑
          </button></template
        ></el-table-column
      >
    </el-table>
    <footer class="permission-query-management-groups__footer">
      <el-select model-value="20" aria-label="每页条数"
        ><el-option label="20 条/页" value="20" /></el-select
      ><span>共 {{ props.groups.length }} 条</span
      ><el-pagination layout="prev, pager, next" :total="props.groups.length" :page-size="20" />
    </footer>

    <el-drawer
      v-model="drawerVisible"
      class="permission-query-management-groups__drawer"
      direction="rtl"
      size="58%"
      :show-close="false"
      append-to-body
    >
      <template #header
        ><header class="permission-query-management-groups__drawer-header">
          <h2>{{ props.selectedGroup.name }}</h2>
          <button type="button" aria-label="关闭" @click="drawerVisible = false">
            <RiCloseFill />
          </button></header
      ></template>
      <section class="permission-query-management-groups__drawer-content">
        <h3><i />管理员</h3>
        <button
          class="permission-query-management-groups__choose"
          type="button"
          @click="memberPickerVisible = true"
        >
          <RiAddFill />选择成员
        </button>
        <div
          v-if="props.selectedGroup.members.length"
          class="permission-query-management-groups__manager-list"
        >
          <span v-for="member in props.selectedGroup.members" :key="member.id"
            ><RiUserSettingsFill />{{ member.name }}</span
          >
        </div>
      </section>
      <template #footer
        ><el-button type="primary" @click="drawerVisible = false">保存</el-button
        ><el-button @click="drawerVisible = false">取消</el-button></template
      >
    </el-drawer>
    <PermissionQueryMembersDialog
      v-model="memberPickerVisible"
      :members="props.members"
      :selected-members="props.selectedGroup.members"
      @confirm="emit('saveMembers', $event)"
    />
  </section>
</template>

<style scoped lang="scss">
.permission-query-management-groups {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  padding: var(--el-space-3xl);
  box-sizing: border-box;
  &__filter {
    display: flex;
    margin-bottom: var(--el-space-3xl);
    gap: 0;
  }
  &__filter :deep(.el-select) {
    width: 190px;
  }
  &__filter :deep(.el-select + .el-select) {
    margin-left: -1px;
  }
  &__filter :deep(.el-select__wrapper) {
    min-height: 42px;
  }
  &__table {
    min-height: 0;
    flex: 1;
  }
  &__table :deep(.el-table__header-wrapper th.el-table__cell) {
    height: 62px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-medium);
  }
  &__table :deep(.el-table__cell) {
    height: 78px;
    font-size: var(--el-font-size-medium);
  }
  &__table :deep(.el-table__inner-wrapper::before) {
    background: var(--el-border-color-lighter);
  }
  &__link,
  &__choose {
    padding: var(--el-space-xs);
    border: 0;
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
  }
  &__link:hover,
  &__choose:hover {
    border-radius: var(--el-border-radius-base);
    background: var(--el-color-primary-light-9);
  }
  &__footer {
    display: flex;
    min-height: 54px;
    align-items: flex-end;
    gap: var(--el-space-lg);
  }
  &__footer :deep(.el-select) {
    width: 154px;
  }
  &__footer :deep(.el-select__wrapper) {
    min-height: 42px;
  }
  &__footer .el-pagination {
    margin-left: auto;
  }
  &__drawer-header {
    display: flex;
    height: 100%;
    align-items: center;
    justify-content: space-between;
  }
  &__drawer-header h2 {
    margin: 0;
    font-size: var(--el-font-size-medium);
  }
  &__drawer-header button {
    display: inline-flex;
    padding: var(--el-space-xs);
    border: 0;
    background: transparent;
    cursor: pointer;
  }
  &__drawer-header button:hover {
    border-radius: var(--el-border-radius-base);
    background: var(--el-fill-color-light);
  }
  &__drawer-content {
    padding: var(--el-space-3xl);
  }
  &__drawer-content h3 {
    display: flex;
    margin: 0 0 var(--el-space-3xl);
    align-items: center;
    gap: var(--el-space-md);
    font-size: var(--el-font-size-medium);
  }
  &__drawer-content h3 i {
    width: 5px;
    height: 20px;
    border-radius: var(--el-border-radius-base);
    background: var(--el-color-primary);
  }
  &__choose {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-sm);
    font-size: var(--el-font-size-medium);
  }
  &__choose svg {
    width: 20px;
    height: 20px;
  }
  &__manager-list {
    display: flex;
    margin-top: var(--el-space-2xl);
    flex-wrap: wrap;
    gap: var(--el-space-md);
  }
  &__manager-list span {
    display: inline-flex;
    padding: var(--el-space-sm) var(--el-space-md);
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-sm);
    background: var(--el-fill-color);
  }
}
:global(.permission-query-management-groups__drawer .el-drawer__header) {
  height: 56px;
  margin-bottom: 0;
  padding: 0 var(--el-space-3xl);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
:global(.permission-query-management-groups__drawer .el-drawer__body) {
  padding: 0;
}
:global(.permission-query-management-groups__drawer .el-drawer__footer) {
  height: 70px;
  padding: 0 var(--el-space-3xl);
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>
