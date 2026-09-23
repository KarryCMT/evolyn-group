<script setup lang="ts">
import { computed, ref } from 'vue';
import type { WorkflowField, WorkflowNode } from '../../schema';
import WorkflowFieldPermissions from './WorkflowFieldPermissions.vue';
import WorkflowNodeOperations from './WorkflowNodeOperations.vue';
import WorkflowTransitionRules from './WorkflowTransitionRules.vue';

defineOptions({ name: 'WorkflowNodeSettingsTabs' });

const props = defineProps<{ node: WorkflowNode; fields: readonly WorkflowField[] }>();
const emit = defineEmits<{ updateConfig: [config: WorkflowNode['config']] }>();

type TabKey = 'permissions' | 'operations' | 'rules';
const activeTab = ref<TabKey>('permissions');
const tabs: Array<{ key: TabKey; label: string }> = [
  { key: 'permissions', label: '字段权限' },
  { key: 'operations', label: '节点操作' },
  { key: 'rules', label: '流转规则' },
];
const config = computed(() => props.node.config);

function patch(patchValue: Partial<WorkflowNode['config']>) {
  emit('updateConfig', { ...config.value, ...patchValue });
}
</script>

<template>
  <section class="workflow-node-settings-tabs">
    <div class="workflow-node-settings-tabs__nav" role="tablist" aria-label="节点配置分类">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.key"
        :class="{ 'is-active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="workflow-node-settings-tabs__body">
      <WorkflowFieldPermissions
        v-if="activeTab === 'permissions'"
        :fields="fields"
        :form-permissions="config.formPermissions"
        :summary-fields="config.summaryFields"
        @update-permissions="(value) => patch({ formPermissions: value })"
        @update-summary-fields="(value) => patch({ summaryFields: value })"
      />
      <WorkflowNodeOperations
        v-else-if="activeTab === 'operations'"
        :operations="config.operations"
        @update="(value) => patch({ operations: value })"
      />
      <WorkflowTransitionRules
        v-else
        :submit-condition="config.submitCondition"
        @update="(value) => patch({ submitCondition: value })"
      />
    </div>
  </section>
</template>

<style scoped lang="scss">
.workflow-node-settings-tabs {
  min-height: 0;
  padding: 0 16px 18px;

  &__nav {
    display: grid;
    padding: 3px;
    margin-bottom: 14px;
    background: var(--el-fill-color-light);
    border-radius: 8px;
    grid-template-columns: repeat(3, 1fr);

    button {
      min-width: 0;
      height: 36px;
      padding: 0 6px;
      color: var(--el-text-color-primary);
      background: transparent;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;

      &.is-active {
        color: var(--el-color-primary);
        background: var(--el-bg-color);
        box-shadow: var(--el-box-shadow-lighter);
        font-weight: 600;
      }
    }
  }

  &__body { min-height: 240px; }
}
</style>
