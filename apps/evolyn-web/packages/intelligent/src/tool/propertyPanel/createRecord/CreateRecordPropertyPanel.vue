<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus';
import { onBeforeUnmount, shallowRef, watch } from 'vue';
import type {
  IntelligentActionConfig,
  IntelligentDesignerResources,
  IntelligentFieldAssignment,
  IntelligentFieldOption,
  IntelligentNode,
} from '../../../schema';
import FieldAssignmentList from './FieldAssignmentList.vue';
import TargetFormSelect from './TargetFormSelect.vue';

defineOptions({ name: 'CreateRecordPropertyPanel' });

const props = defineProps<{
  node: IntelligentNode;
  resources?: IntelligentDesignerResources;
}>();

const emit = defineEmits<{
  update: [patch: Partial<Omit<IntelligentNode, 'id' | 'type'>>];
}>();

const targetFields = shallowRef<readonly IntelligentFieldOption[]>([]);
const fieldsLoading = shallowRef(false);
const fieldsLoadFailed = shallowRef(false);
let fieldsController: AbortController | null = null;

function inputValue(event: Event): string {
  return (event.target as HTMLInputElement).value;
}

function updateConfig(patch: Partial<IntelligentActionConfig>, description?: string): void {
  emit('update', {
    ...(description ? { description } : {}),
    config: { ...props.node.config, ...patch },
  });
}

async function selectForm(formCode: string): Promise<void> {
  if (formCode === props.node.config?.targetFormCode) return;
  const assignments = props.node.config?.fieldAssignments ?? [];
  if (assignments.length > 0) {
    try {
      await ElMessageBox.confirm(
        '切换目标表单会清空已经设置的字段值，是否继续？',
        '切换目标表单',
        { confirmButtonText: '继续切换', cancelButtonText: '取消', type: 'warning' },
      );
    } catch {
      return;
    }
  }
  const form = props.resources?.forms.find((item) => item.code === formCode);
  updateConfig(
    {
      targetFormCode: form?.code,
      targetFormName: form?.name,
      targetFormPublishedVersion: form?.publishedVersion,
      fieldAssignments: [],
    },
    form ? `向${form.name}新增一条数据` : '待配置目标表单',
  );
}

function updateAssignments(assignments: IntelligentFieldAssignment[]): void {
  const formName = props.node.config?.targetFormName ?? '目标表单';
  updateConfig(
    { fieldAssignments: assignments },
    `向${formName}新增数据 · 已设置 ${assignments.length}/${targetFields.value.length} 个字段`,
  );
}

function notifyQuickFill(filledCount: number): void {
  if (filledCount > 0) {
    ElMessage.success(`已自动填充 ${filledCount} 个字段`);
  } else {
    ElMessage.info('没有找到新的高置信度匹配字段');
  }
}

watch(
  [
    () => props.node.config?.targetFormCode,
    () => props.resources?.loadFormFields,
  ],
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
  <div class="create-record-panel">
    <section class="create-record-panel__basics">
      <label>
        <span><b>*</b>节点名称</span>
        <input
          :value="node.name"
          maxlength="40"
          placeholder="请输入节点名称"
          @input="emit('update', { name: inputValue($event) })"
        >
      </label>
      <label>
        <span><b>*</b>目标表单</span>
        <TargetFormSelect
          :model-value="node.config?.targetFormCode ?? ''"
          :options="resources?.forms ?? []"
          @update:model-value="selectForm"
        />
      </label>
    </section>

    <div v-if="node.config?.targetFormCode" class="create-record-panel__divider" />
    <div v-if="fieldsLoading" class="create-record-panel__state">正在加载目标表单字段…</div>
    <div v-else-if="fieldsLoadFailed" class="create-record-panel__state is-error">
      目标表单字段加载失败，请重新选择或稍后重试
    </div>
    <FieldAssignmentList
      v-else-if="node.config?.targetFormCode"
      :fields="targetFields"
      :assignments="node.config?.fieldAssignments ?? []"
      :source-groups="resources?.sourceGroups ?? []"
      :members="resources?.members"
      :departments="resources?.departments"
      @change="updateAssignments"
      @quick-fill="notifyQuickFill"
    />
  </div>
</template>

<style scoped lang="scss">
.create-record-panel {
  min-height: 100%;
  color: #172033;
  background: #fff;

  &__basics {
    display: grid;
    padding: 28px 32px 30px;
    gap: 24px;

    label { display: grid; gap: 11px; }
    label > span { font-size: 17px; font-weight: 650; }
    b { margin-right: 2px; color: #ef5252; }
    input {
      width: 100%;
      height: 58px;
      padding: 0 16px;
      color: #172033;
      background: #fff;
      border: 1px solid #d6dce6;
      border-radius: 8px;
      box-sizing: border-box;
      font: inherit;
      font-size: 16px;
      outline: none;

      &:focus { border-color: #12b8ad; }
    }
  }

  &__divider { height: 16px; background: #f6f7f9; border-top: 1px solid #f1f3f6; }
  &__state { padding: 40px 32px; color: #8791a0; text-align: center; }
  &__state.is-error { color: #d65050; }
}
</style>
