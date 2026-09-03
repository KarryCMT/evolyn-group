<script setup lang="ts">
import { ElMessageBox, ElScrollbar } from 'element-plus';
import { computed, onMounted, shallowRef, useId, useTemplateRef } from 'vue';
import type { FormSchemaDocument } from '../../schema/types';
import type { FormRuntimeAdapter } from '../adapters/types';
import type { FormRuntimeActionDefinition, FormRuntimeLayout } from '../actions/types';
import type { FormRendererExpose } from '../renderer/types';
import type { FormRuntime } from '../store/createFormRuntime';
import type {
  FormDraftPayload,
  FormIssue,
  FormRuntimeFieldPermission,
  FormSubmitPayload,
  FormValue,
} from '../types';
import type { FormFieldRegistry } from '../widgets/registry';
import FormRenderer from '../renderer/FormRenderer.vue';
import FormRuntimeActionBar from './FormRuntimeActionBar.vue';
import FormMultitabRenderer from '../renderer/FormMultitabRenderer.vue';
import { createWebFieldRegistry } from '../../runtime-web/widgets/registry';

defineOptions({ name: 'FormWebRuntimeSurface' });

const props = withDefaults(
  defineProps<{
    schema: FormSchemaDocument;
    formId?: string;
    publishedVersion?: number;
    schemaRevision?: string;
    initialValues?: Record<string, FormValue>;
    contextDefaults?: Record<string, FormValue>;
    /** 当前登录成员 ID：显隐规则 includeCurrentMember 的注入源。 */
    currentMemberId?: string;
    /** 字段权限矩阵（bootstrap permissions 按模式投影）；未提供全量放行。 */
    fieldPermissions?: Record<string, FormRuntimeFieldPermission>;
    adapter?: FormRuntimeAdapter;
    registry?: FormFieldRegistry;
    actions?: readonly FormRuntimeActionDefinition[];
    layout?: FormRuntimeLayout;
    contentWidth?: string;
    formDomId?: string;
  }>(),
  {
    formId: '',
    publishedVersion: 0,
    schemaRevision: '',
    initialValues: undefined,
    contextDefaults: undefined,
    currentMemberId: undefined,
    fieldPermissions: undefined,
    adapter: undefined,
    registry: undefined,
    actions: () => [],
    layout: 'auto',
    contentWidth: '860px',
    formDomId: undefined,
  },
);

const emit = defineEmits<{
  submit: [payload: FormSubmitPayload];
  submitSuccess: [payload: FormSubmitPayload];
  submitError: [error: unknown];
  draft: [payload: FormDraftPayload];
  draftSuccess: [payload: FormDraftPayload];
  draftError: [error: unknown];
  draftUnavailable: [];
  reset: [];
  action: [action: FormRuntimeActionDefinition];
  unsupportedField: [info: { fieldKey: string; type: string }];
}>();

defineSlots<{
  actionLabel(props: { action: FormRuntimeActionDefinition }): unknown;
}>();

const rendererRef = useTemplateRef<FormRendererExpose>('renderer');
const runtime = shallowRef<FormRuntime | null>(null);
// Web 表面在组件生命周期内保持同一注册表，避免字段异步加载缓存被重复清空。
const webRegistry = createWebFieldRegistry();
const generatedId = useId().replace(/:/g, '');
const resolvedFormDomId = computed(() => props.formDomId || `evf-runtime-form-${generatedId}`);
const resolvedRegistry = computed(() => props.registry ?? webRegistry);
const surfaceClasses = computed(() => [
  'evf-runtime-surface',
  `evf-runtime-surface--${props.layout}`,
]);
const formIssues = computed<readonly FormIssue[]>(() =>
  (runtime.value?.state.issues ?? []).filter((issue) => !issue.fieldKey),
);
const resolvedActions = computed(() =>
  props.actions.map<FormRuntimeActionDefinition>((action) => {
    const operation = runtime.value?.state.activeOperation ?? null;
    const operationMatches =
      (action.behavior === 'submit' && operation === 'submit') ||
      (action.behavior === 'save-draft' && operation === 'save-draft');
    // custom 完全由宿主持有状态，不能被表单提交/草稿生命周期连带禁用。
    if (action.behavior === 'custom') {
      return {
        ...action,
        loading: Boolean(action.loading),
        disabled: Boolean(action.disabled),
      };
    }
    return {
      ...action,
      loading: Boolean(action.loading || operationMatches),
      disabled:
        Boolean(action.disabled) ||
        !runtime.value ||
        (action.behavior !== 'reset' && runtime.value.state.lifecycle === 'submitted') ||
        (operation !== null && !operationMatches),
    };
  }),
);

function setRuntime(nextRuntime: FormRuntime | null): void {
  runtime.value = nextRuntime;
}

// 子组件 setup 期间的 immediate 事件可能早于父监听，挂载后以公开只读 getter 补齐初始会话。
onMounted(() => setRuntime(rendererRef.value?.getRuntime() ?? null));

async function confirmAction(action: FormRuntimeActionDefinition): Promise<boolean> {
  if (!action.confirmText) return true;
  try {
    await ElMessageBox.confirm(action.confirmText, '请确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: action.intent === 'danger' ? 'warning' : 'info',
    });
    return true;
  } catch {
    return false;
  }
}

async function handleAction(action: FormRuntimeActionDefinition): Promise<void> {
  if (!(await confirmAction(action))) return;
  if (action.behavior === 'submit') {
    await rendererRef.value?.submit();
    return;
  }
  if (action.behavior === 'save-draft') {
    const outcome = await rendererRef.value?.saveDraft();
    if (outcome && !outcome.ok && outcome.reason === 'unavailable') emit('draftUnavailable');
    return;
  }
  if (action.behavior === 'reset') {
    rendererRef.value?.reset();
    emit('reset');
    return;
  }
  emit('action', props.actions.find((item) => item.key === action.key) ?? action);
}

defineExpose({
  runtime,
  getRuntime: () => runtime.value,
  submit: () => rendererRef.value?.submit(),
  saveDraft: () => rendererRef.value?.saveDraft(),
  reset: () => rendererRef.value?.reset(),
});
</script>

<template>
  <section :class="surfaceClasses">
    <ElScrollbar class="evf-runtime-surface__scrollbar">
      <div class="evf-runtime-surface__canvas" :style="{ maxWidth: props.contentWidth }">
        <FormRenderer
          ref="renderer"
          class="evf-runtime-surface__form"
          :schema="props.schema"
          :form-id="props.formId"
          :published-version="props.publishedVersion"
          :schema-revision="props.schemaRevision"
          :initial-values="props.initialValues"
          :context-defaults="props.contextDefaults"
          :current-member-id="props.currentMemberId"
          :field-permissions="props.fieldPermissions"
          :adapter="props.adapter"
          :registry="resolvedRegistry"
          :multitab-renderer="FormMultitabRenderer"
          :form-dom-id="resolvedFormDomId"
          @runtime-change="setRuntime"
          @submit="emit('submit', $event)"
          @submit-success="emit('submitSuccess', $event)"
          @submit-error="emit('submitError', $event)"
          @draft="emit('draft', $event)"
          @draft-success="emit('draftSuccess', $event)"
          @draft-error="emit('draftError', $event)"
          @unsupported-field="emit('unsupportedField', $event)"
        />
      </div>
    </ElScrollbar>

    <FormRuntimeActionBar
      v-if="resolvedActions.length || formIssues.length"
      :actions="resolvedActions"
      :issues="formIssues"
      :layout="props.layout"
      :form-dom-id="resolvedFormDomId"
      :content-width="props.contentWidth"
      @action="handleAction"
    >
      <template v-if="$slots.actionLabel" #action-label="slotProps">
        <slot name="actionLabel" v-bind="slotProps" />
      </template>
    </FormRuntimeActionBar>
  </section>
</template>
