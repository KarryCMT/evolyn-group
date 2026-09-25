<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, onBeforeUnmount, shallowRef, watch } from 'vue';
import {
  type IntelligentActionConfig,
  type IntelligentDesignerResources,
  type IntelligentFieldAssignment,
  type IntelligentFieldOption,
  type IntelligentNode,
  type IntelligentSourceFieldGroup,
  type IntelligentUpdateFilter,
  type IntelligentUpdateMatchMode,
  type IntelligentUpdateTargetMode,
  createIntelligentUpdateFilter,
} from '../../../schema';
import TriggerOptionSelect from '../TriggerOptionSelect.vue';
import TargetFormSelect from '../createRecord/TargetFormSelect.vue';
import TargetNodeSelect from './TargetNodeSelect.vue';
import UpdateAssignmentEditor from './UpdateAssignmentEditor.vue';
import UpdateFilterEditor from './UpdateFilterEditor.vue';

defineOptions({ name: 'UpdateRecordPropertyPanel' });

const props = defineProps<{
  node: IntelligentNode;
  nodes: readonly IntelligentNode[];
  resources?: IntelligentDesignerResources;
}>();
const emit = defineEmits<{
  update: [patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>];
}>();

const targetFields = shallowRef<readonly IntelligentFieldOption[]>([]);
const fieldsLoading = shallowRef(false);
const fieldsLoadFailed = shallowRef(false);
let fieldsController: AbortController | null = null;

const targetMode = computed<IntelligentUpdateTargetMode>(
  () => props.node.config?.updateTargetMode ?? 'form',
);
const previousDataNodes = computed(() => {
  const currentIndex = props.nodes.findIndex((node) => node.id === props.node.id);
  return props.nodes
    .slice(0, currentIndex < 0 ? 0 : currentIndex)
    .filter(
      (node) =>
        node.type === 'action' &&
        ['create-record', 'update-record'].includes(node.actionType ?? '') &&
        Boolean(node.config?.targetFormCode),
    );
});
const targetNodeOptions = computed(() =>
  previousDataNodes.value.map((node) => ({
    id: node.id,
    name: node.name,
    formName: node.config?.targetFormName ?? '目标表单',
  })),
);
const sourceGroups = computed<readonly IntelligentSourceFieldGroup[]>(() => {
  const triggerIds = new Set(
    props.nodes.filter((node) => node.type === 'trigger').map((node) => node.id),
  );
  const previousIds = new Set(previousDataNodes.value.map((node) => node.id));
  const groups = (props.resources?.sourceGroups ?? []).filter(
    (group) =>
      group.nodeId === 'trigger' || triggerIds.has(group.nodeId) || previousIds.has(group.nodeId),
  );
  const existingIds = new Set(groups.map((group) => group.nodeId));
  const currentFormCode = props.node.config?.targetFormCode;
  const derived = previousDataNodes.value.flatMap((node) => {
    if (existingIds.has(node.id) || node.config?.targetFormCode !== currentFormCode) return [];
    // 同表单数据节点的输出结构与目标表单一致，可直接作为后续值来源。
    return [{ nodeId: node.id, nodeName: node.name, fields: targetFields.value }];
  });
  return [...groups, ...derived];
});
const targetReady = computed(() =>
  targetMode.value === 'node'
    ? Boolean(props.node.config?.targetNodeId && props.node.config?.targetFormCode)
    : Boolean(props.node.config?.targetFormCode),
);
const modeOptions = [
  { value: 'form', label: '选择表单修改数据' },
  { value: 'node', label: '选择节点修改数据' },
] as const;

function inputValue(event: Event): string {
  return (event.target as HTMLInputElement).value;
}

function updateConfig(patch: Partial<IntelligentActionConfig>, description?: string): void {
  emit('update', {
    ...(description ? { description } : {}),
    config: { ...props.node.config, ...patch },
  });
}

function hasDependentConfiguration(): boolean {
  return Boolean(
    props.node.config?.targetFormCode ||
    props.node.config?.updateFilters?.length ||
    props.node.config?.fieldAssignments?.length,
  );
}

async function confirmReset(message: string): Promise<boolean> {
  if (!hasDependentConfiguration()) return true;
  try {
    await ElMessageBox.confirm(message, '修改对象', {
      confirmButtonText: '继续切换',
      cancelButtonText: '取消',
      type: 'warning',
    });
    return true;
  } catch {
    return false;
  }
}

async function selectMode(value: string): Promise<void> {
  const mode = value as IntelligentUpdateTargetMode;
  if (mode === targetMode.value) return;
  if (!(await confirmReset('切换修改对象会清空筛选条件和字段值，是否继续？'))) return;
  updateConfig(
    {
      updateTargetMode: mode,
      targetNodeId: undefined,
      targetFormCode: undefined,
      targetFormName: undefined,
      targetFormPublishedVersion: undefined,
      updateFilters: [],
      fieldAssignments: [],
      createWhenNoMatch: false,
    },
    '待配置修改对象',
  );
}

async function selectForm(formCode: string): Promise<void> {
  if (formCode === props.node.config?.targetFormCode) return;
  if (!(await confirmReset('切换目标表单会清空筛选条件和字段值，是否继续？'))) return;
  const form = props.resources?.forms.find((item) => item.code === formCode);
  updateConfig(
    {
      updateTargetMode: 'form',
      targetNodeId: undefined,
      targetFormCode: form?.code,
      targetFormName: form?.name,
      targetFormPublishedVersion: form?.publishedVersion,
      updateMatchMode: 'all',
      updateFilters: [createIntelligentUpdateFilter()],
      fieldAssignments: [],
      createWhenNoMatch: false,
    },
    form ? `修改${form.name}中符合条件的数据` : '待配置目标表单',
  );
}

async function selectNode(nodeId: string): Promise<void> {
  if (nodeId === props.node.config?.targetNodeId) return;
  if (!(await confirmReset('切换目标节点会清空已经设置的字段值，是否继续？'))) return;
  const target = previousDataNodes.value.find((node) => node.id === nodeId);
  if (!target) return;
  updateConfig(
    {
      updateTargetMode: 'node',
      targetNodeId: target.id,
      targetFormCode: target.config?.targetFormCode,
      targetFormName: target.config?.targetFormName,
      targetFormPublishedVersion: target.config?.targetFormPublishedVersion,
      updateFilters: [],
      fieldAssignments: [],
      createWhenNoMatch: false,
    },
    `修改${target.name}输出的数据`,
  );
}

function updateFilters(filters: IntelligentUpdateFilter[]): void {
  updateConfig({ updateFilters: filters });
}

function updateMatchMode(mode: IntelligentUpdateMatchMode): void {
  updateConfig({ updateMatchMode: mode });
}

function updateAssignments(assignments: IntelligentFieldAssignment[]): void {
  const target =
    targetMode.value === 'node'
      ? previousDataNodes.value.find((node) => node.id === props.node.config?.targetNodeId)?.name
      : props.node.config?.targetFormName;
  updateConfig(
    { fieldAssignments: assignments },
    `修改${target ?? '目标数据'} · 已设置 ${assignments.length} 个字段`,
  );
}

function notifyQuickFill(filledCount: number): void {
  if (filledCount > 0) ElMessage.success(`已自动填充 ${filledCount} 个字段`);
  else ElMessage.info('没有找到新的高置信度匹配字段');
}

watch(
  [() => props.node.config?.targetFormCode, () => props.resources?.loadFormFields],
  async ([formCode, loadFormFields], _previous, onCleanup) => {
    fieldsController?.abort();
    targetFields.value = [];
    fieldsLoadFailed.value = false;
    if (!formCode || !loadFormFields) return;
    const controller = new AbortController();
    fieldsController = controller;
    onCleanup(() => controller.abort());
    fieldsLoading.value = true;
    try {
      targetFields.value = await loadFormFields(formCode, controller.signal);
    } catch {
      if (!controller.signal.aborted) fieldsLoadFailed.value = true;
    } finally {
      if (fieldsController === controller) fieldsLoading.value = false;
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => fieldsController?.abort());
</script>

<template>
  <div class="update-record-panel">
    <section class="update-record-panel__basics">
      <label>
        <span><b>*</b>节点名称</span>
        <input
          :value="node.name"
          maxlength="40"
          placeholder="请输入节点名称"
          :class="{ 'is-invalid': !node.name.trim() }"
          @input="emit('update', { name: inputValue($event) })"
        />
      </label>
      <div class="update-record-panel__target">
        <span><b>*</b>修改对象</span>
        <div>
          <TriggerOptionSelect
            :model-value="targetMode"
            :options="modeOptions"
            control-label="选择修改对象类型"
            @update:model-value="selectMode"
          />
          <TargetFormSelect
            v-if="targetMode === 'form'"
            :model-value="node.config?.targetFormCode ?? ''"
            :options="resources?.forms ?? []"
            @update:model-value="selectForm"
          />
          <TargetNodeSelect
            v-else
            :model-value="node.config?.targetNodeId ?? ''"
            :options="targetNodeOptions"
            @update:model-value="selectNode"
          />
        </div>
        <p v-if="!targetReady">请选择修改对象</p>
      </div>
    </section>

    <template v-if="targetReady">
      <div v-if="fieldsLoading" class="update-record-panel__state">正在加载目标字段…</div>
      <div v-else-if="fieldsLoadFailed" class="update-record-panel__state is-error">
        目标字段加载失败，请重新选择或稍后重试
      </div>
      <template v-else>
        <UpdateFilterEditor
          v-if="targetMode === 'form'"
          :filters="node.config?.updateFilters ?? []"
          :mode="node.config?.updateMatchMode ?? 'all'"
          :fields="targetFields"
          :source-groups="sourceGroups"
          :members="resources?.members"
          :departments="resources?.departments"
          :create-when-no-match="node.config?.createWhenNoMatch ?? false"
          @update:filters="updateFilters"
          @update:mode="updateMatchMode"
          @update:create-when-no-match="updateConfig({ createWhenNoMatch: $event })"
        />
        <div class="update-record-panel__divider" />
        <UpdateAssignmentEditor
          :fields="targetFields"
          :assignments="node.config?.fieldAssignments ?? []"
          :source-groups="sourceGroups"
          :members="resources?.members"
          :departments="resources?.departments"
          @change="updateAssignments"
          @quick-fill="notifyQuickFill"
        />
      </template>
    </template>
  </div>
</template>

<style scoped lang="scss">
.update-record-panel {
  min-height: 100%;
  color: #172033;
  background: #fff;

  &__basics {
    display: grid;
    padding: 28px 32px 30px;
    gap: 24px;
  }
  &__basics > label {
    display: grid;
    gap: 11px;
  }
  &__basics > label > span,
  &__target > span {
    font-size: 17px;
    font-weight: 650;
  }
  b {
    margin-right: 2px;
    color: #ef5252;
  }
  &__basics > label > input {
    width: 100%;
    height: 58px;
    padding: 0 16px;
    color: #172033;
    background: #fff;
    border: 1px solid #d6dce6;
    border-radius: 8px;
    font: inherit;
    font-size: 16px;
    outline: none;

    &:focus {
      border-color: #12b8ad;
    }
    &.is-invalid {
      border-color: #ef5858;
    }
  }
  &__target {
    display: grid;
    gap: 11px;
  }
  &__target > div {
    display: grid;
    grid-template-columns: minmax(250px, 31%) minmax(0, 1fr);
    gap: 14px;

    :deep(.trigger-option-select__button) {
      height: 58px;
      font-size: 16px;
    }
  }
  &__target > p {
    margin: -3px 0 0;
    color: #ef5252;
    font-size: 13px;
  }
  &__divider {
    height: 16px;
    background: #f6f7f9;
    border-top: 1px solid #f1f3f6;
  }
  &__state {
    padding: 44px 32px;
    color: #8791a0;
    text-align: center;
  }
  &__state.is-error {
    color: #d65050;
  }
}

@media (max-width: 820px) {
  .update-record-panel__basics {
    padding: 24px 20px;
  }
  .update-record-panel__target > div {
    grid-template-columns: 1fr;
  }
}
</style>
