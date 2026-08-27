<script setup lang="ts">
import { RiAddFill } from '@remixicon/vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { shallowRef } from 'vue';
import ManagementGroupDialog from '~/components/application/management-groups/ManagementGroupDialog.vue';
import ManagementGroupList from '~/components/application/management-groups/ManagementGroupList.vue';
import ManagementGroupsEmptyState from '~/components/application/management-groups/ManagementGroupsEmptyState.vue';
import type {
  ManagementGroup,
  ManagementGroupDraft,
} from '~/components/application/management-groups/managementGroup.types';

defineOptions({ name: 'ApplicationSettingManagementGroupsPage' });

/** 管理组接口尚未落地，页面先以本地状态呈现完整操作流，后续可替换为接口读写。 */
const groups = shallowRef<ManagementGroup[]>([]);
const dialogVisible = shallowRef(false);

function openCreateDialog() {
  dialogVisible.value = true;
}

function createGroup(draft: ManagementGroupDraft) {
  groups.value = [
    ...groups.value,
    {
      id: `management_group_${Date.now()}`,
      ...draft,
    },
  ];
  ElMessage.success('管理组已添加');
}

async function removeGroup(id: string) {
  try {
    await ElMessageBox.confirm('删除后，该组成员将不再拥有对应的应用管理权限。', '删除管理组', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    });
    groups.value = groups.value.filter((group) => group.id !== id);
    ElMessage.success('管理组已删除');
  } catch {
    // 用户取消时保持页面状态不变；接口接入后在此补充请求失败处理。
  }
}
</script>

<template>
  <section class="management-groups-page" aria-label="应用管理组">
    <header class="management-groups-page__header">
      <h1>应用管理组</h1>
      <button class="management-groups-page__create" type="button" @click="openCreateDialog">
        <RiAddFill aria-hidden="true" />
        添加管理组
      </button>
    </header>

    <ManagementGroupsEmptyState v-if="!groups.length" @create="openCreateDialog" />
    <ManagementGroupList v-else :groups="groups" @remove="removeGroup" />

    <ManagementGroupDialog v-model="dialogVisible" @confirm="createGroup" />
  </section>
</template>

<style scoped lang="scss">
.management-groups-page {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);

  &__header {
    display: flex;
    min-height: 76px;
    padding: 0 var(--el-space-2xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-xl);

    h1 {
      margin: 0;
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-extra-large);
      font-weight: 650;
      line-height: 28px;
    }
  }

  &__create {
    display: inline-flex;
    min-height: 40px;
    padding: 0 var(--el-space-xl);
    border: 0;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    color: var(--el-color-white);
    cursor: pointer;
    background: var(--el-color-primary);
    font: inherit;
    font-size: var(--el-font-size-base);
    gap: var(--el-space-xs);

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      background: var(--el-color-primary-light-3);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }
}

@media (max-width: 680px) {
  .management-groups-page {
    &__header {
      min-height: 64px;
      padding: 0 var(--el-space-lg);
    }

    &__create {
      padding: 0 var(--el-space-md);
    }
  }
}
</style>
