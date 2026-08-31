<script setup lang="ts">
import { computed, onMounted, shallowRef, useId, useTemplateRef } from 'vue';
import type { FormSchemaDocument } from '../schema/types';
import type { FormRuntimeAdapter } from '../runtime/adapters/types';
import type { FormRuntimeActionDefinition } from '../runtime/actions/types';
import type { FormRendererExpose } from '../runtime/renderer/types';
import type { FormRuntime } from '../runtime/store/createFormRuntime';
import type { FormDraftPayload, FormIssue, FormSubmitPayload, FormValue } from '../runtime/types';
import type { FormFieldRegistry } from '../runtime/widgets/registry';
import FormRenderer from '../runtime/renderer/FormRenderer.vue';
import { createMobileFieldRegistry } from '../runtime/widgets/registry';
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

function confirmAction(action: FormRuntimeActionDefinition): boolean {
  if (!action.confirmText || typeof window === 'undefined') return true;
  return window.confirm(action.confirmText);
}

async function handleAction(action: FormRuntimeActionDefinition): Promise<void> {
  if (!confirmAction(action)) return;
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
          :adapter="props.adapter"
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
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
}

.evf-mobile-runtime-surface__scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.evf-mobile-runtime-surface__canvas {
  min-height: 100%;
  padding: var(--el-space-lg) 0 var(--el-space-3xl);
}

.evf-mobile-runtime-surface :deep(.evf-form) {
  --evf-columns: 1;
  --evf-control-height: 44px;
  --evf-color-text: var(--el-text-color-primary);
  --evf-color-text-regular: var(--el-text-color-regular);
  --evf-color-text-secondary: var(--el-text-color-secondary);
  --evf-color-text-placeholder: var(--el-text-color-placeholder);
  --evf-color-text-disabled: var(--el-text-color-disabled);
  --evf-color-border: var(--el-border-color);
  --evf-color-border-light: var(--el-border-color-light);
  --evf-color-border-lighter: var(--el-border-color-lighter);
  --evf-color-fill-light: var(--el-fill-color-light);
  --evf-color-bg: var(--el-bg-color);
  --evf-color-primary: var(--el-color-primary);
  --evf-color-danger: var(--el-color-danger);
  --evf-font-size-base: var(--el-font-size-base);
  --evf-font-size-small: var(--el-font-size-small);
  --evf-font-size-extra-small: var(--el-font-size-extra-small);
  --evf-space-sm: var(--el-space-sm);
  --evf-space-md: var(--el-space-md);
  --evf-space-lg: var(--el-space-lg);
  --evf-space-xl: var(--el-space-xl);
  --evf-space-3xl: var(--el-space-3xl);
  --evf-radius-base: var(--el-border-radius-base);
  --evf-radius-medium: var(--el-border-radius-medium);
}

.evf-mobile-runtime-surface :deep(.evf-form__body) {
  padding: var(--el-space-xl);
}
</style>
