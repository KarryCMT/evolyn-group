<script setup lang="ts">
import type { DataPushItem } from './dataPush.types';
import { RiAddFill, RiBookOpenFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import DataPushCreateDialog from './DataPushCreateDialog.vue';
import DataPushEditor from './DataPushEditor.vue';
import DataPushFilterBar from './DataPushFilterBar.vue';
import DataPushList from './DataPushList.vue';

defineOptions({ name: 'DataPushSettings' });

const createDialogVisible = shallowRef(false);
const editorVisible = shallowRef(false);
const selectedForm = shallowRef('order-management');
const serverAddressFilter = shallowRef('');
const formNameFilter = shallowRef('');
const pushItems = shallowRef<DataPushItem[]>([
  {
    id: 'data_push_demo_001',
    name: '未命名数据推送',
    serverAddress: 'https://www.lingyanyun.com/',
    formName: '订单管理',
    events: '有新数据提交时，有数据被修改时，有数据被删除时，有数据被恢复时',
    remark: '',
    enabled: true,
  },
]);

const formNames: Record<string, string> = {
  'order-management': '订单管理',
  'purchase-request': '采购申请',
  'employee-profile': '员工档案',
  'office-supplies': '办公用品申请',
};
const selectedFormName = computed(() => formNames[selectedForm.value] ?? '采购申请');
const serverAddresses = computed(() => [
  ...new Set(pushItems.value.map((item) => item.serverAddress)),
]);
const formNamesForFilter = computed(() => [
  ...new Set(pushItems.value.map((item) => item.formName)),
]);
const filteredPushItems = computed(() =>
  pushItems.value.filter(
    (item) =>
      (!serverAddressFilter.value || item.serverAddress === serverAddressFilter.value) &&
      (!formNameFilter.value || item.formName === formNameFilter.value),
  ),
);

function openCreateDialog() {
  createDialogVisible.value = true;
}

function openEditor() {
  createDialogVisible.value = false;
  editorVisible.value = true;
}

function savePush(name: string) {
  pushItems.value = [
    ...pushItems.value,
    {
      id: `data_push_${Date.now()}`,
      name,
      serverAddress: 'https://www.lingyanyun.com/',
      formName: selectedFormName.value,
      events: '有新数据提交时，有数据被修改时，有数据被删除时，有数据被恢复时',
      remark: '',
      enabled: true,
    },
  ];
  editorVisible.value = false;
  ElMessage.success('数据推送已保存');
}

function showHelp() {
  ElMessage.info('数据推送帮助文档将在后续版本开放');
}

function queryPushItems() {
  ElMessage.success(`已找到 ${filteredPushItems.value.length} 条数据推送`);
}
</script>

<template>
  <section class="data-push-settings" aria-label="数据推送">
    <header class="data-push-settings__header">
      <h1 class="data-push-settings__title">数据推送</h1>
      <div class="data-push-settings__actions">
        <button class="data-push-settings__help" type="button" @click="showHelp">
          <RiBookOpenFill aria-hidden="true" />
          <span>帮助文档</span>
        </button>
        <el-button type="primary" @click="openCreateDialog">
          <RiAddFill aria-hidden="true" />
          新建数据推送
        </el-button>
      </div>
    </header>

    <DataPushFilterBar
      v-model:server-address="serverAddressFilter"
      v-model:form-name="formNameFilter"
      :server-addresses="serverAddresses"
      :form-names="formNamesForFilter"
      @query="queryPushItems"
    />
    <DataPushList :items="filteredPushItems" />

    <DataPushCreateDialog
      v-model:visible="createDialogVisible"
      v-model:selected-form="selectedForm"
      @confirm="openEditor"
    />
    <DataPushEditor
      v-if="editorVisible"
      :form-name="selectedFormName"
      @close="editorVisible = false"
      @save="savePush"
    />
  </section>
</template>

<style scoped lang="scss">
.data-push-settings {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);

  &__header {
    display: flex;
    min-height: 76px;
    padding: 0 22px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    align-items: center;
    justify-content: space-between;
    gap: 20px;
  }

  &__title {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 20px;
    font-weight: 650;
    line-height: 28px;
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: 20px;

    :deep(.el-button) {
      height: 40px;
      padding: 0 16px;
      font-size: 15px;

      svg {
        width: 18px;
        height: 18px;
        margin-right: 5px;
      }
    }
  }

  &__help {
    display: inline-flex;
    padding: 7px 4px;
    border: 0;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    font: inherit;
    font-size: 15px;
    gap: 6px;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      background: var(--el-color-primary-light-9);
    }
  }
}

@media (max-width: 680px) {
  .data-push-settings {
    &__header {
      min-height: 64px;
      padding: 0 14px;
    }

    &__help span {
      display: none;
    }

    &__actions {
      gap: 8px;

      :deep(.el-button) {
        padding: 0 10px;
      }
    }
  }
}
</style>
