<script setup lang="ts">
import { RiCloseLine, RiDeleteBinLine } from '@remixicon/vue';
import { computed } from 'vue';
import type {
  IntelligentActionConfig,
  IntelligentActionType,
  IntelligentNode,
  IntelligentTrigger,
  IntelligentValueSource,
} from '../../schema';
import { intelligentNodeTemplates } from '../../mock/nodeTemplates';
import IntelligentPageTip from '../pageTip/index.vue';
import IntelligentValueCollector from '../valueCollector/index.vue';
import FormTriggerPropertyPanel from './FormTriggerPropertyPanel.vue';

defineOptions({ name: 'IntelligentPropertyPanel' });

const props = defineProps<{
  node: IntelligentNode | null;
  trigger: IntelligentTrigger;
}>();

const emit = defineEmits<{
  close: [];
  remove: [nodeId: string];
  update: [nodeId: string, patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>];
  updateTrigger: [patch: Partial<IntelligentTrigger>];
}>();

const isAction = computed(() => props.node?.type === 'action');
const isTrigger = computed(() => props.node?.type === 'trigger');
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
  <aside
    v-if="node"
    class="intelligent-property-panel"
    :class="{ 'intelligent-property-panel--trigger': isTrigger }"
    aria-label="节点属性"
  >
    <header class="intelligent-property-panel__header" :class="{ 'is-trigger': isTrigger }">
      <div>
        <small v-if="!isTrigger">执行节点</small>
        <h2>{{ isTrigger ? '表单触发' : node.name }}</h2>
      </div>
      <button v-if="!isTrigger" type="button" aria-label="关闭属性面板" @click="emit('close')"><RiCloseLine /></button>
    </header>

    <FormTriggerPropertyPanel
      v-if="isTrigger"
      :trigger="trigger"
      @update="emit('updateTrigger', $event)"
    />

    <div v-else class="intelligent-property-panel__body">
      <label class="intelligent-property-panel__field">
        <span>节点名称</span>
        <input :value="node.name" maxlength="40" @input="updateNode({ name: inputValue($event) })">
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

        <label
          v-if="['create-record', 'update-record', 'delete-record'].includes(node.actionType ?? '')"
          class="intelligent-property-panel__field"
        >
          <span>目标表单</span>
          <input
            :value="node.config?.targetFormName ?? ''"
            placeholder="请选择或输入目标表单"
            @input="updateConfig({ targetFormName: inputValue($event) })"
          >
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
            >
          </label>
        </template>

        <label v-if="node.actionType === 'send-notification'" class="intelligent-property-panel__field">
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

    <footer v-if="isAction" class="intelligent-property-panel__footer">
      <button type="button" @click="emit('remove', node.id)"><RiDeleteBinLine />删除节点</button>
    </footer>
  </aside>
</template>

<style scoped lang="scss">
.intelligent-property-panel {
  position: absolute;
  z-index: 10;
  top: 16px;
  right: 16px;
  bottom: 16px;
  display: grid;
  width: min(380px, calc(100% - 32px));
  grid-template-rows: auto minmax(0, 1fr) auto;
  background: #fff;
  border: 1px solid #e0e5ec;
  border-radius: 10px;
  box-shadow: 0 12px 32px rgb(31 43 61 / 15%);

  &--trigger {
    position: fixed;
    z-index: 60;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(1120px, 62vw);
    border-width: 0 0 0 1px;
    border-radius: 0;
    box-shadow: -8px 0 24px rgb(31 43 61 / 10%);
    grid-template-rows: 76px minmax(0, 1fr);
    overflow-y: auto;
  }

  &__header {
    display: flex;
    min-height: 68px;
    padding: 14px 16px;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid #e8ebf0;

    &.is-trigger { min-height: 76px; padding: 0 32px; }

    small { color: #8a94a3; }
    h2 { margin: 3px 0 0; color: #172033; font-size: 17px; }
    button { display: inline-flex; padding: 6px; color: #586477; background: transparent; border: 0; cursor: pointer; }
    svg { width: 20px; height: 20px; }
  }

  &__body { display: grid; padding: 18px 16px; align-content: start; overflow-y: auto; gap: 18px; }
  &__field {
    display: grid;
    gap: 8px;
    color: #303b4d;
    font-size: 13px;

    > span { font-weight: 600; }
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

      &:focus { border-color: #00afa2; }
    }
  }

  &__footer {
    display: flex;
    padding: 12px 16px;
    justify-content: flex-end;
    border-top: 1px solid #e8ebf0;

    button {
      display: inline-flex;
      padding: 8px 12px;
      align-items: center;
      gap: 6px;
      color: #e24b4b;
      background: #fff3f3;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
    }
    svg { width: 16px; height: 16px; }
  }
}

@media (max-width: 980px) {
  .intelligent-property-panel--trigger { width: min(720px, 76vw); }
}

@media (max-width: 720px) {
  .intelligent-property-panel--trigger { width: 100%; }
}
</style>
