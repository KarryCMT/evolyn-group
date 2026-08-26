<script setup lang="ts">
import { usePermissionQuery } from '~/composables/tenant/usePermissionQuery';
import PermissionQueryExplorer from './PermissionQueryExplorer.vue';
import PermissionQueryManagementGroups from './PermissionQueryManagementGroups.vue';

defineOptions({ name: 'PermissionQueryPanel' });

const {
  drawerVisible,
  managementGroups,
  memberPickerVisible,
  members,
  permissionTree,
  saveMembers,
  selectedGroup,
  selectedSubjectId,
  selectGroup,
  selectSubject,
  subjectTrees,
  subjectType,
  view,
  workspace,
} = usePermissionQuery();
</script>

<template>
  <section class="permission-query-panel">
    <nav class="permission-query-panel__workspace-tabs" aria-label="权限查询范围">
      <button
        :class="{ 'permission-query-panel__workspace-tab--active': workspace === 'system' }"
        type="button"
        @click="workspace = 'system'"
      >
        系统</button
      ><button
        :class="{
          'permission-query-panel__workspace-tab--active': workspace === 'product',
        }"
        type="button"
        @click="workspace = 'product'"
      >
        灵衍云
      </button>
    </nav>
    <nav class="permission-query-panel__view-tabs" aria-label="查询类型">
      <button
        :class="{ 'permission-query-panel__view-tab--active': view === 'management-groups' }"
        type="button"
        @click="view = 'management-groups'"
      >
        管理组查询</button
      ><button
        :class="{ 'permission-query-panel__view-tab--active': view === 'permission-groups' }"
        type="button"
        @click="view = 'permission-groups'"
      >
        权限组查询
      </button>
    </nav>
    <PermissionQueryManagementGroups
      v-if="view === 'management-groups'"
      v-model:drawer-visible="drawerVisible"
      v-model:member-picker-visible="memberPickerVisible"
      :groups="managementGroups"
      :members="members"
      :selected-group="selectedGroup"
      :workspace="workspace"
      @edit="selectGroup"
      @save-members="saveMembers"
    />
    <PermissionQueryExplorer
      v-else
      :selected-id="selectedSubjectId"
      :subject-trees="subjectTrees"
      :subject-type="subjectType"
      :tree="permissionTree"
      @select="selectSubject"
    />
  </section>
</template>

<style scoped lang="scss">
.permission-query-panel {
  display: flex;
  height: 100%;
  min-height: 680px;
  flex-direction: column;
  overflow: hidden;
}
.permission-query-panel__workspace-tabs {
  display: flex;
  height: 66px;
  padding-left: 0;
  align-items: stretch;
  background: var(--el-fill-color-light);
}
.permission-query-panel__workspace-tabs button {
  min-width: 120px;
  padding: 0 26px;
  border: 0;
  border-radius: 0 0 20px 0;
  color: var(--el-text-color-primary);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: 16px;
}
.permission-query-panel__workspace-tabs button:hover {
  color: var(--el-color-primary);
  background: var(--el-fill-color);
}
.permission-query-panel__workspace-tab--active {
  color: var(--el-color-primary) !important;
  background: var(--el-bg-color) !important;
  font-weight: 600 !important;
}
.permission-query-panel__view-tabs {
  display: flex;
  min-height: 52px;
  padding: 0 28px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  align-items: flex-end;
  gap: 16px;
}
.permission-query-panel__view-tabs button {
  min-height: 52px;
  padding: 0 2px;
  border: 0;
  border-bottom: 3px solid transparent;
  color: var(--el-text-color-regular);
  background: transparent;
  cursor: pointer;
  font: inherit;
  font-size: 16px;
}
.permission-query-panel__view-tabs button:hover {
  color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}
.permission-query-panel__view-tab--active {
  border-bottom-color: var(--el-color-primary) !important;
  color: var(--el-color-primary) !important;
  font-weight: 600 !important;
}
</style>
