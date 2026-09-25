<script setup lang="ts">
import { RiDeleteBinLine } from '@remixicon/vue';
import { ElDrawer } from 'element-plus';
import { computed } from 'vue';
import type {
  IntelligentActionConfig,
  IntelligentActionType,
  IntelligentDesignerResources,
  IntelligentNode,
  IntelligentTrigger,
  IntelligentValueSource,
} from '../../schema';
import { intelligentNodeTemplates } from '../../mock/nodeTemplates';
import IntelligentPageTip from '../pageTip/index.vue';
import IntelligentValueCollector from '../valueCollector/index.vue';
import FormTriggerPropertyPanel from './FormTriggerPropertyPanel.vue';
import CreateRecordPropertyPanel from './createRecord/CreateRecordPropertyPanel.vue';
import UpdateRecordPropertyPanel from './updateRecord/UpdateRecordPropertyPanel.vue';

defineOptions({ name: 'IntelligentPropertyPanel' });

const props = defineProps<{
  node: IntelligentNode | null;
  nodes: readonly IntelligentNode[];
  trigger: IntelligentTrigger;
  resources?: IntelligentDesignerResources;
}>();

const emit = defineEmits<{
  close: [];
  remove: [nodeId: string];
  update: [nodeId: string, patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>];
  updateTrigger: [patch: Partial<IntelligentTrigger>];
}>();

const isAction = computed(() => props.node?.type === 'action');
const isTrigger = computed(() => props.node?.type === 'trigger');
const isCreateRecord = computed(
  () => props.node?.type === 'action' && props.node.actionType === 'create-record',
);
const isUpdateRecord = computed(
  () => props.node?.type === 'action' && props.node.actionType === 'update-record',
);
const isDeleteRecord = computed(
  () => props.node?.type === 'action' && props.node.actionType === 'delete-record',
);
const isDataAction = computed(
  () => isCreateRecord.value || isUpdateRecord.value || isDeleteRecord.value,
);
const hasDedicatedPanel = computed(() => isCreateRecord.value || isUpdateRecord.value);
const drawerVisible = computed({
  get: () => props.node !== null,
  set: (visible) => {
    if (!visible) emit('close');
  },
});
const drawerTitle = computed(() => {
  if (isTrigger.value) return '表单触发';
  if (isCreateRecord.value) return '新增数据';
  if (isUpdateRecord.value) return '修改数据';
  if (isDeleteRecord.value) return '删除数据';
  return props.node?.name ?? '节点属性';
});
// 数据操作需要容纳字段映射等复杂配置，其余动作沿用紧凑宽度。
const drawerSize = computed(() => {
  if (isTrigger.value) return 'min(1120px, 62vw)';
  if (isDataAction.value) return 'min(1260px, 72vw)';
  return 'min(380px, calc(100% - 32px))';
});
const valueSource = computed<IntelligentValueSource>({
  get: () => props.node?.config?.valueSource ?? { type: 'constant' },
  set: (source) => updateConfig({ valueSource: source }),
});

function inputValue(event: Event): string {
  return (event.target as HTMLInputElement).value;
}

function updateNode(patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>): void {
  if (props.node) emit('update', props.node.id, patch);
}

function updateConfig(patch: Partial<IntelligentActionConfig>): void {
  updateNode({ config: { ...props.node?.config, ...patch } });
}

function updateActionType(event: Event): void {
  const actionType = inputValue(event) as IntelligentActionType;
  const template = intelligentNodeTemplates.find((item) => item.type === actionType);
  updateNode({
    actionType,
    name: template?.name ?? props.node?.name ?? '执行节点',
    description: template?.description ?? props.node?.description ?? '',
  });
}
</script>

<template>
  <ElDrawer
    v-if="node"
    v-model="drawerVisible"
    append-to-body
    class="intelligent-property-drawer"
    :class="{
      'intelligent-property-drawer--trigger': isTrigger,
      'intelligent-property-drawer--data-action': isDataAction,
    }"
    direction="rtl"
    :lock-scroll="false"
    :modal="false"
    modal-penetrable
    :size="drawerSize"
    :title="drawerTitle"
    header-class="intelligent-property-drawer__header"
    body-class="intelligent-property-drawer__body"
    footer-class="intelligent-property-drawer__footer"
  >
    <template #header="{ titleId, titleClass }">
      <div class="intelligent-property-panel__heading">
        <small v-if="!isTrigger && !isDataAction">执行节点</small>
        <h2 :id="titleId" :class="titleClass">{{ drawerTitle }}</h2>
      </div>
    </template>

    <FormTriggerPropertyPanel
      v-if="isTrigger"
      :trigger="trigger"
      @update="emit('updateTrigger', $event)"
    />

    <CreateRecordPropertyPanel
      v-else-if="isCreateRecord"
      :node="node"
      :resources="resources"
      @update="emit('update', node.id, $event)"
    />

    <UpdateRecordPropertyPanel
      v-else-if="isUpdateRecord"
      :node="node"
      :nodes="nodes"
      :resources="resources"
      @update="emit('update', node.id, $event)"
    />

    <div v-else class="intelligent-property-panel__body">
      <label class="intelligent-property-panel__field">
        <span>节点名称</span>
        <input
          :value="node.name"
          maxlength="40"
          @input="updateNode({ name: inputValue($event) })"
        />
      </label>

      <template v-if="isAction">
        <label class="intelligent-property-panel__field">
          <span>执行动作</span>
          <select :value="node.actionType" @change="updateActionType">
            <option v-for="item in intelligentNodeTemplates" :key="item.type" :value="item.type">
              {{ item.name }}
            </option>
          </select>
        </label>

        <label v-if="node.actionType === 'delete-record'" class="intelligent-property-panel__field">
          <span>目标表单</span>
          <input
            :value="node.config?.targetFormName ?? ''"
            placeholder="请选择或输入目标表单"
            @input="updateConfig({ targetFormName: inputValue($event) })"
          />
        </label>

        <template v-if="node.actionType === 'http-request'">
          <label class="intelligent-property-panel__field">
            <span>请求方式</span>
            <select
              :value="node.config?.requestMethod ?? 'POST'"
              @change="updateConfig({ requestMethod: inputValue($event) as 'GET' | 'POST' })"
            >
              <option value="GET">GET</option>
              <option value="POST">POST</option>
            </select>
          </label>
          <label class="intelligent-property-panel__field">
            <span>请求地址</span>
            <input
              :value="node.config?.requestUrl ?? ''"
              placeholder="https://api.lingyanyun.com"
              @input="updateConfig({ requestUrl: inputValue($event) })"
            />
          </label>
        </template>

        <label
          v-if="node.actionType === 'send-notification'"
          class="intelligent-property-panel__field"
        >
          <span>通知内容</span>
          <textarea
            :value="node.config?.message ?? ''"
            rows="4"
            placeholder="请输入通知内容"
            @input="updateConfig({ message: inputValue($event) })"
          />
        </label>

        <label
          v-if="node.actionType === 'condition' || node.actionType === 'data-transform'"
          class="intelligent-property-panel__field"
        >
          <span>{{ node.actionType === 'condition' ? '条件表达式' : '转换表达式' }}</span>
          <textarea
            :value="node.config?.expression ?? ''"
            rows="5"
            placeholder="例如：status == 'active'"
            @input="updateConfig({ expression: inputValue($event) })"
          />
        </label>

        <div class="intelligent-property-panel__field">
          <span>参数值来源</span>
          <IntelligentValueCollector v-model="valueSource" />
        </div>

        <IntelligentPageTip text="节点配置会实时写入画布文档，保存后由运行时按节点顺序执行。" />
      </template>
    </div>

    <template v-if="isAction && !hasDedicatedPanel" #footer>
      <button
        class="intelligent-property-panel__remove"
        type="button"
        @click="emit('remove', node.id)"
      >
        <RiDeleteBinLine />删除节点
      </button>
    </template>
  </ElDrawer>
</template>

<style scoped lang="scss">
.intelligent-property-panel {
  &__heading {
    small {
      color: #8a94a3;
    }
    h2 {
      margin: 3px 0 0;
      color: #172033;
      font-size: 17px;
      line-height: 24px;
    }
  }

  &__body {
    display: grid;
    padding: 18px 16px;
    align-content: start;
    overflow-y: auto;
    gap: 18px;
  }
  &__field {
    display: grid;
    gap: 8px;
    color: #303b4d;
    font-size: 13px;

    > span {
      font-weight: 600;
    }
    input,
    select,
    textarea {
      width: 100%;
      padding: 9px 10px;
      color: #172033;
      background: #fff;
      border: 1px solid #d8dee8;
      border-radius: 6px;
      box-sizing: border-box;
      font: inherit;
      outline: none;
      resize: vertical;

      &:focus {
        border-color: #00afa2;
      }
    }
  }

  &__remove {
    display: inline-flex;
    padding: 8px 12px;
    align-items: center;
    gap: 6px;
    color: #e24b4b;
    background: #fff3f3;
    border: 0;
    border-radius: 6px;
    cursor: pointer;

    svg {
      width: 16px;
      height: 16px;
    }
  }
}
</style>

<style lang="scss">
.intelligent-property-drawer {
  --el-drawer-padding-primary: 0;

  border-left: 1px solid #e0e5ec;

  .intelligent-property-drawer__header {
    min-height: 68px;
    padding: 14px 16px;
    margin: 0;
    border-bottom: 1px solid #e8ebf0;
  }

  .intelligent-property-drawer__body {
    padding: 0;
  }

  .intelligent-property-drawer__footer {
    padding: 12px 16px;
    border-top: 1px solid #e8ebf0;
  }

  &--trigger,
  &--data-action {
    .intelligent-property-drawer__header {
      min-height: 76px;
      padding: 0 32px;
    }
  }
}

@media (max-width: 980px) {
  .intelligent-property-drawer--trigger,
  .intelligent-property-drawer--data-action {
    width: min(820px, 84vw) !important;
  }
}

@media (max-width: 720px) {
  .intelligent-property-drawer--trigger,
  .intelligent-property-drawer--data-action {
    width: 100% !important;
  }
}
</style>
