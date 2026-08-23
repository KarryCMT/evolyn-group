<script setup lang="ts">
import { RiSearch2Fill } from '@remixicon/vue';
import { computed, ref, watch } from 'vue';
import type { WorkflowField, WorkflowFieldPermission, WorkflowNode } from '../schema';

defineOptions({ name: 'WorkflowInspector' });

const props = defineProps<{
  fields: readonly WorkflowField[];
  node: WorkflowNode | null;
}>();

const emit = defineEmits<{
  updateNode: [nodeId: string, patch: Partial<WorkflowNode>];
}>();

type InspectorTab = 'node' | 'workflow';
type NodeTab = 'permissions' | 'actions' | 'rules';

const activeTab = ref<InspectorTab>('node');
const activeNodeTab = ref<NodeTab>('permissions');
const keyword = ref('');
const filteredFields = computed(() => {
  const normalized = keyword.value.trim().toLocaleLowerCase();
  if (!normalized) return props.fields;
  return props.fields.filter((field) => field.label.toLocaleLowerCase().includes(normalized));
});

watch(
  () => props.node?.id,
  () => {
    activeTab.value = 'node';
    activeNodeTab.value = 'permissions';
    keyword.value = '';
  },
);

function updateName(name: string) {
  if (!props.node || !name.trim()) return;
  emit('updateNode', props.node.id, { name: name.trim() });
}

function updatePermission(
  fieldId: string,
  key: keyof WorkflowFieldPermission,
  checked: boolean,
) {
  if (!props.node) return;
  const current = props.node.fieldPermissions[fieldId] ?? defaultPermission();
  emit('updateNode', props.node.id, {
    fieldPermissions: {
      ...props.node.fieldPermissions,
      [fieldId]: { ...current, [key]: checked },
    },
  });
}

function defaultPermission(): WorkflowFieldPermission {
  return { visible: true, editable: true, confidential: false };
}
</script>

<template>
  <aside class="workflow-inspector" aria-label="流程属性">
    <nav class="workflow-inspector__tabs" aria-label="流程属性标签">
      <button
        class="workflow-inspector__tab"
        :class="{ 'workflow-inspector__tab--active': activeTab === 'node' }"
        type="button"
        @click="activeTab = 'node'"
      >
        节点属性
      </button>
      <button
        class="workflow-inspector__tab"
        :class="{ 'workflow-inspector__tab--active': activeTab === 'workflow' }"
        type="button"
        @click="activeTab = 'workflow'"
      >
        流程属性
      </button>
    </nav>

    <div v-if="activeTab === 'workflow'" class="workflow-inspector__empty-state">
      流程全局属性将在版本、通知和限时规则接入后提供。
    </div>

    <template v-else-if="node">
      <div class="workflow-inspector__node-header">
        <label class="workflow-inspector__name-label" for="workflow-node-name">节点名称</label>
        <span class="workflow-inspector__node-id">节点 ID：{{ node.id }}</span>
        <input
          id="workflow-node-name"
          class="workflow-inspector__name-input"
          :value="node.name"
          @change="updateName(($event.target as HTMLInputElement).value)"
        />
      </div>

      <nav class="workflow-inspector__node-tabs" aria-label="节点配置标签">
        <button
          v-for="tab in [
            { key: 'permissions', label: '字段权限' },
            { key: 'actions', label: '节点操作' },
            { key: 'rules', label: '流转规则' },
          ]"
          :key="tab.key"
          class="workflow-inspector__node-tab"
          :class="{ 'workflow-inspector__node-tab--active': activeNodeTab === tab.key }"
          type="button"
          @click="activeNodeTab = tab.key as NodeTab"
        >
          {{ tab.label }}
        </button>
      </nav>

      <template v-if="activeNodeTab === 'permissions'">
        <label class="workflow-inspector__search">
          <RiSearch2Fill />
          <input v-model="keyword" type="search" placeholder="搜索字段" />
        </label>
        <div class="workflow-inspector__permission-header">
          <span>字段</span>
          <span>可见</span>
          <span>可编辑</span>
          <span>简报</span>
        </div>
        <div class="workflow-inspector__permission-list">
          <div v-for="field in filteredFields" :key="field.id" class="workflow-inspector__permission-row">
            <span class="workflow-inspector__field-label">{{ field.label }}</span>
            <input
              type="checkbox"
              :checked="node.fieldPermissions[field.id]?.visible ?? true"
              aria-label="字段可见"
              @change="updatePermission(field.id, 'visible', ($event.target as HTMLInputElement).checked)"
            />
            <input
              type="checkbox"
              :checked="node.fieldPermissions[field.id]?.editable ?? true"
              aria-label="字段可编辑"
              @change="updatePermission(field.id, 'editable', ($event.target as HTMLInputElement).checked)"
            />
            <input
              type="checkbox"
              :checked="node.fieldPermissions[field.id]?.confidential ?? false"
              aria-label="字段简报"
              @change="updatePermission(field.id, 'confidential', ($event.target as HTMLInputElement).checked)"
            />
          </div>
          <p v-if="!filteredFields.length" class="workflow-inspector__empty-fields">未找到匹配字段</p>
        </div>
      </template>
      <div v-else class="workflow-inspector__empty-state">
        {{ activeNodeTab === 'actions' ? '节点操作' : '流转规则' }}将在下一阶段接入。
      </div>
    </template>

    <div v-else class="workflow-inspector__empty-state">请选择流程节点以设置属性。</div>
  </aside>
</template>

<style scoped lang="scss">
.workflow-inspector {
  display: flex;
  min-width: 0;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-lighter);

  &__tabs,
  &__node-tabs {
    display: flex;
    align-items: stretch;
  }

  &__tabs {
    height: 64px;
    min-height: 64px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__tab,
  &__node-tab {
    border: 0;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }
  }

  &__tab {
    position: relative;
    flex: 1;
    color: var(--el-text-color-regular);
    font-size: 16px;
    font-weight: 650;
    background: transparent;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &--active {
      color: var(--el-color-primary);

      &::after {
        position: absolute;
        right: 20px;
        bottom: 0;
        left: 20px;
        height: 3px;
        content: '';
        background: var(--el-color-primary);
      }
    }
  }

  &__node-header {
    display: grid;
    padding: 16px;
    grid-template-columns: 1fr auto;
    gap: 10px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__name-label {
    color: var(--el-text-color-primary);
    font-size: 14px;
    font-weight: 650;
  }

  &__node-id {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  &__name-input {
    width: 100%;
    height: 36px;
    padding: 0 10px;
    grid-column: 1 / -1;
    color: var(--el-text-color-primary);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
    outline: none;

    &:focus {
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
    }
  }

  &__node-tabs {
    margin: 14px 12px 10px;
    padding: 3px;
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);
  }

  &__node-tab {
    min-width: 0;
    height: 32px;
    padding: 0 7px;
    flex: 1;
    color: var(--el-text-color-regular);
    font-size: 13px;
    white-space: nowrap;
    background: transparent;
    border-radius: calc(var(--el-border-radius-base) - 2px);

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &--active {
      color: var(--el-color-primary);
      font-weight: 650;
      background: var(--el-bg-color);
      box-shadow: var(--el-box-shadow-lighter);
    }
  }

  &__search {
    display: flex;
    height: 36px;
    margin: 0 12px 10px;
    padding: 0 10px;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);

    svg {
      width: 16px;
      height: 16px;
    }

    input {
      width: 100%;
      min-width: 0;
      color: var(--el-text-color-primary);
      background: transparent;
      border: 0;
      outline: 0;
    }
  }

  &__permission-header,
  &__permission-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 38px 54px 38px;
    align-items: center;
    column-gap: 4px;
  }

  &__permission-header {
    min-height: 32px;
    padding: 0 12px;
    color: var(--el-text-color-regular);
    font-size: 13px;
    font-weight: 650;
    text-align: center;

    span:first-child {
      text-align: left;
    }
  }

  &__permission-list {
    min-height: 0;
    overflow: auto;
  }

  &__permission-row {
    min-height: 42px;
    padding: 0 12px;
    color: var(--el-text-color-primary);
    font-size: 14px;

    &:hover {
      background: var(--el-fill-color-light);
    }

    input {
      width: 16px;
      height: 16px;
      margin: 0 auto;
      accent-color: var(--el-color-primary);
      cursor: pointer;
    }
  }

  &__field-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__empty-state,
  &__empty-fields {
    padding: 28px 20px;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 1.8;
    text-align: center;
  }

  &__empty-fields {
    margin: 0;
  }
}
</style>
