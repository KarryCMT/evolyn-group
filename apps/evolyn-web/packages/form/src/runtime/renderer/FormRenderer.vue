<script setup lang="ts">
import { onBeforeUnmount, onMounted, shallowRef, watch, type Component } from 'vue';
import { type FormSchemaIssue, validateFormSchema } from '../../schema/validate';
import type { FormSchemaDocument } from '../../schema/types';
import type { FormRuntimeAdapter } from '../adapters/types';
import {
  type FormDraftOutcome,
  type FormRuntime,
  type FormSubmitOutcome,
  createFormRuntime,
} from '../store/createFormRuntime';
import { createFocusRegistry, provideFormRendererContext } from '../store/injection';
import type { FormDraftPayload, FormSubmitPayload, FormValue } from '../types';
import { type FormFieldRegistry, createMobileFieldRegistry } from '../widgets/registry';
import FormSectionRenderer from './FormSectionRenderer.vue';
import type { FormRendererExpose } from './types';
import FormPlainMultitabRenderer from '../../runtime-core/FormPlainMultitabRenderer.vue';

/**
 * 最终渲染器总装：初始化会话、提供上下文、管理提交与整体状态。
 * 只消费「已校验的已发布 Schema」（目标保存协议）；无效 Schema 不创建半初始化会话，
 * 直接渲染受控错误态。业务提交经 adapter 注入，事件面向宿主页面。
 */
const props = withDefaults(
  defineProps<{
    schema: FormSchemaDocument;
    formId?: string;
    publishedVersion?: number;
    schemaRevision?: string;
    /** 已保存记录值，初始化优先级最高。 */
    initialValues?: Record<string, FormValue>;
    /** 记录上下文默认值（当前成员、当前日期等）。 */
    contextDefaults?: Record<string, FormValue>;
    adapter?: FormRuntimeAdapter;
    /** 自定义字段注册表；缺省使用基础字段默认注册表。 */
    registry?: FormFieldRegistry;
    /** 终端专属布局呈现器；Web 和移动端分别注入自己的标签页实现。 */
    multitabRenderer?: Component;
    /** 原生 form 的稳定 DOM ID，供外部操作栏通过 form 属性建立语义关联。 */
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
    multitabRenderer: FormPlainMultitabRenderer,
    formDomId: undefined,
  },
);

const emit = defineEmits<{
  submit: [payload: FormSubmitPayload];
  'submit-success': [payload: FormSubmitPayload];
  'submit-error': [error: unknown];
  draft: [payload: FormDraftPayload];
  'draft-success': [payload: FormDraftPayload];
  'draft-error': [error: unknown];
  'runtime-change': [runtime: FormRuntime | null];
  'unsupported-field': [info: { fieldKey: string; type: string }];
}>();

const runtimeRef = shallowRef<FormRuntime | null>(null);
const schemaIssues = shallowRef<FormSchemaIssue[]>([]);
const operationController = shallowRef<AbortController | null>(null);
// 注册表在会话间复用：字段组件解析与异步包装缓存不应随 Schema 替换丢失。
const fieldRegistry = props.registry ?? createMobileFieldRegistry();
const focusRegistry = createFocusRegistry();

provideFormRendererContext({
  runtime: runtimeRef,
  registry: fieldRegistry,
  registerFieldFocus: focusRegistry.registerFieldFocus,
  unregisterFieldFocus: focusRegistry.unregisterFieldFocus,
  registerFieldReveal: focusRegistry.registerFieldReveal,
  unregisterFieldReveal: focusRegistry.unregisterFieldReveal,
  focusField: focusRegistry.focusField,
  reportUnsupportedField: (info) => emit('unsupported-field', info),
});

// Schema 引用变更即替换整个会话（发布不可变，运行中不做字段级热更新）。
watch(
  [
    () => props.schema,
    () => props.formId,
    () => props.publishedVersion,
    () => props.schemaRevision,
    () => props.initialValues,
    () => props.contextDefaults,
    () => props.adapter,
  ],
  ([schema]) => {
    // Schema/资产切换时终止旧会话请求，避免旧响应回写新表单。
    operationController.value?.abort();
    operationController.value = null;
    const result = validateFormSchema(schema);
    if (result.valid) {
      schemaIssues.value = [];
      runtimeRef.value = createFormRuntime({
        schema: result.document!,
        formId: props.formId,
        publishedVersion: props.publishedVersion,
        schemaRevision: props.schemaRevision,
        initialValues: props.initialValues,
        contextDefaults: props.contextDefaults,
        adapter: props.adapter,
      });
    } else {
      schemaIssues.value = result.issues;
      runtimeRef.value = null;
    }
    emit('runtime-change', runtimeRef.value);
  },
  { immediate: true },
);

onBeforeUnmount(() => operationController.value?.abort());
// immediate watch 可能发生在父组件事件监听完成前，挂载后补发当前会话供 Surface 建立只读投影。
onMounted(() => emit('runtime-change', runtimeRef.value));

/** 错误定位：优先聚焦第一个出错字段；聚焦不到时由 Surface 展示操作区错误摘要。 */
function focusFirstError(): boolean {
  const runtime = runtimeRef.value;
  if (!runtime) return false;
  for (const item of runtime.schema.content.items) {
    const key = item.widget.widgetName;
    if ((runtime.state.fieldStates[key]?.errors.length ?? 0) > 0) {
      return focusRegistry.focusField(key);
    }
  }
  return false;
}

async function handleSubmit(): Promise<FormSubmitOutcome | undefined> {
  const runtime = runtimeRef.value;
  if (!runtime) return undefined;
  // 保留正在执行请求的控制器，避免重复命令覆盖后导致卸载时无法取消旧请求。
  if (operationController.value) return { ok: false, reason: 'busy' };
  const controller = new AbortController();
  operationController.value = controller;
  const outcome: FormSubmitOutcome = await runtime.submit(controller.signal);
  if (operationController.value === controller) operationController.value = null;
  if (outcome.ok) {
    emit('submit', outcome.payload);
    if (outcome.submitted) emit('submit-success', outcome.payload);
    return outcome;
  }
  if (outcome.reason === 'invalid' || outcome.reason === 'server') focusFirstError();
  if (outcome.reason === 'error') emit('submit-error', outcome.error);
  return outcome;
}

async function handleSaveDraft(): Promise<FormDraftOutcome | undefined> {
  const runtime = runtimeRef.value;
  if (!runtime) return undefined;
  if (operationController.value) return { ok: false, reason: 'busy' };
  const controller = new AbortController();
  operationController.value = controller;
  const outcome = await runtime.saveDraft(controller.signal);
  if (operationController.value === controller) operationController.value = null;
  if (outcome.ok) {
    emit('draft', outcome.payload);
    emit('draft-success', outcome.payload);
  } else if (outcome.reason === 'error') {
    emit('draft-error', outcome.error);
  }
  return outcome;
}

const exposed = {
  /** 只读会话引用；宿主可读取状态但不得绕过 action 直接改写。 */
  runtime: runtimeRef,
  getRuntime: () => runtimeRef.value,
  submit: handleSubmit,
  saveDraft: handleSaveDraft,
  buildSubmitPayload: () => runtimeRef.value?.buildSubmitPayload() ?? null,
  buildDraftPayload: () => runtimeRef.value?.buildDraftPayload() ?? null,
  focusFirstError,
  /** 手动重置会话（继续填写场景）。 */
  reset: () => runtimeRef.value?.reset(),
} satisfies FormRendererExpose;

defineExpose(exposed);
</script>

<template>
  <!-- novalidate：校验统一由运行时接管，避免原生气泡与错误态双轨展示。 -->
  <form :id="formDomId" class="evf-form" novalidate @submit.prevent="handleSubmit">
    <div v-if="schemaIssues.length > 0" class="evf-form__invalid" role="alert">
      <p class="evf-form__invalid-title">表单配置无法加载</p>
      <ul class="evf-form__invalid-list">
        <li v-for="(issue, index) in schemaIssues" :key="index">
          {{ issue.path }}：{{ issue.message }}
        </li>
      </ul>
    </div>
    <FormSectionRenderer v-else-if="runtimeRef" :multitab-renderer="props.multitabRenderer" />
  </form>
</template>
