<script setup lang="ts">
import { showConfirmDialog } from 'vant';
import { computed, onMounted, shallowRef, useId, useTemplateRef } from 'vue';
import type { FormSchemaDocument } from '../schema/types';
import type { FormRuntimeAdapter } from '../runtime/adapters/types';
import type { FormRuntimeActionDefinition } from '../runtime/actions/types';
import type { FormRendererExpose } from '../runtime/renderer/types';
import type {
  FormRuntime,
  FormSubmitConfirmationContext,
} from '../runtime/store/createFormRuntime';
import type {
  FormDraftPayload,
  FormIssue,
  FormRuntimeFieldPermission,
  FormSubmitPayload,
  FormValue,
} from '../runtime/types';
import type { FormFieldRegistry } from '../runtime/widgets/registry';
import FormRenderer from '../runtime/renderer/FormRenderer.vue';
import { createMobileFieldRegistry } from './widgets/registry';
import FormMobileActionBar from './FormMobileActionBar.vue';
import FormMobileMultitabRenderer from './FormMobileMultitabRenderer.vue';

defineOptions({ name: 'FormMobileRuntimeSurface' });

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

const rendererRef = useTemplateRef<FormRendererExpose>('renderer');
const runtime = shallowRef<FormRuntime | null>(null);
const mobileRegistry = createMobileFieldRegistry();
const generatedId = useId().replace(/:/g, '');
const resolvedFormDomId = computed(() => props.formDomId || `evf-mobile-form-${generatedId}`);
const resolvedRegistry = computed(() => props.registry ?? mobileRegistry);
const formIssues = computed<readonly FormIssue[]>(() =>
  (runtime.value?.state.issues ?? []).filter((issue) => !issue.fieldKey),
);
const resolvedActions = computed(() =>
  props.actions.map<FormRuntimeActionDefinition>((action) => {
    const operation = runtime.value?.state.activeOperation ?? null;
    const operationMatches =
      (action.behavior === 'submit' && operation === 'submit') ||
      (action.behavior === 'save-draft' && operation === 'save-draft');
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

onMounted(() => setRuntime(rendererRef.value?.getRuntime() ?? null));

async function confirmAction(action: FormRuntimeActionDefinition): Promise<boolean> {
  if (!action.confirmText || typeof window === 'undefined') return true;
  try {
    await showConfirmDialog({
      title: '请确认',
      message: action.confirmText,
      confirmButtonText: '确定',
      cancelButtonText: '取消',
    });
    return true;
  } catch {
    return false;
  }
}

async function confirmSubmit(context: FormSubmitConfirmationContext): Promise<boolean> {
  if (typeof window === 'undefined') return true;
  try {
    if (context.warnings.length > 0) {
      await showConfirmDialog({
        title: '部分校验未通过',
        message: context.warnings.map((warning) => `• ${warning.remind}`).join('\n'),
        confirmButtonText: '忽略并继续',
        cancelButtonText: '返回修改',
      });
    }
    if (context.confirmation) {
      await showConfirmDialog({
        title: context.confirmation.title,
        message: context.confirmation.content,
        confirmButtonText: '确认提交',
        cancelButtonText: '取消',
      });
    }
    return true;
  } catch {
    return false;
  }
}

async function handleAction(action: FormRuntimeActionDefinition): Promise<void> {
  if (!(await confirmAction(action))) return;
  if (action.behavior === 'submit') {
    await rendererRef.value?.submit();
  } else if (action.behavior === 'save-draft') {
    const outcome = await rendererRef.value?.saveDraft();
    if (outcome && !outcome.ok && outcome.reason === 'unavailable') emit('draftUnavailable');
  } else if (action.behavior === 'reset') {
    rendererRef.value?.reset();
    emit('reset');
  } else {
    emit('action', props.actions.find((item) => item.key === action.key) ?? action);
  }
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
  <section class="evf-mobile-runtime-surface">
    <div class="evf-mobile-runtime-surface__scroll">
      <div class="evf-mobile-runtime-surface__canvas">
        <FormRenderer
          ref="renderer"
          :schema="props.schema"
          :form-id="props.formId"
          :published-version="props.publishedVersion"
          :schema-revision="props.schemaRevision"
          :initial-values="props.initialValues"
          :context-defaults="props.contextDefaults"
          :current-member-id="props.currentMemberId"
          :field-permissions="props.fieldPermissions"
          :adapter="props.adapter"
          :submit-confirmation="confirmSubmit"
          :registry="resolvedRegistry"
          :multitab-renderer="FormMobileMultitabRenderer"
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
    </div>
    <FormMobileActionBar
      v-if="resolvedActions.length || formIssues.length"
      :actions="resolvedActions"
      :issues="formIssues"
      @action="handleAction"
    />
  </section>
</template>

<style scoped lang="scss">
.evf-mobile-runtime-surface {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  height: 100%;
  min-height: 0;
  color: var(--van-text-color);
  background: var(--van-background-2);
}

.evf-mobile-runtime-surface__scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.evf-mobile-runtime-surface__canvas {
  min-height: 100%;
  padding: var(--van-padding-sm) 0 var(--van-padding-xl);
}

.evf-mobile-runtime-surface :deep(.evf-form) {
  --evf-columns: 1;
  --evf-control-height: 44px;
  --evf-color-text: var(--van-text-color);
  --evf-color-text-regular: var(--van-text-color-2);
  --evf-color-text-secondary: var(--van-text-color-3);
  --evf-color-text-placeholder: var(--van-text-color-3);
  --evf-color-text-disabled: var(--van-gray-5);
  --evf-color-border: var(--van-border-color);
  --evf-color-border-light: var(--van-gray-3);
  --evf-color-border-lighter: var(--van-gray-2);
  --evf-color-fill-light: var(--van-background);
  --evf-color-bg: var(--van-background-2);
  --evf-color-primary: var(--van-primary-color);
  --evf-color-danger: var(--van-danger-color);
  --evf-font-size-base: var(--van-font-size-md);
  --evf-font-size-small: var(--van-font-size-sm);
  --evf-font-size-extra-small: var(--van-font-size-xs);
  --evf-space-sm: var(--van-padding-base);
  --evf-space-md: var(--van-padding-xs);
  --evf-space-lg: var(--van-padding-sm);
  --evf-space-xl: var(--van-padding-md);
  --evf-space-3xl: var(--van-padding-xl);
  --evf-radius-base: var(--van-radius-md);
  --evf-radius-medium: var(--van-radius-lg);
}

.evf-mobile-runtime-surface :deep(.evf-form__body) {
  padding: var(--van-padding-md);
}

.evf-mobile-runtime-surface :deep(.evf-mobile-field) {
  padding: 0;
}

.evf-mobile-runtime-surface :deep(.evf-mobile-field::after) {
  display: none;
}

.evf-mobile-runtime-surface :deep(.evf-mobile-choice-group) {
  display: flex;
  flex-direction: column;
  gap: var(--van-padding-sm);
  min-height: 44px;
}

.evf-mobile-runtime-surface :deep(.evf-mobile-choice-group--horizontal) {
  flex-flow: row wrap;
  align-items: center;
}
</style>
