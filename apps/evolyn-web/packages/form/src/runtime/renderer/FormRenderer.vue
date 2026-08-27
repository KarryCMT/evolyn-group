<script setup lang="ts">
import { ref, shallowRef, watch } from 'vue';
import { validateFormSchema, type FormSchemaIssue } from '../../schema/validate';
import type { FormSchemaDocument } from '../../schema/types';
import type { FormRuntimeAdapter } from '../adapters/types';
import {
  createFormRuntime,
  type FormRuntime,
  type FormSubmitOutcome,
} from '../store/createFormRuntime';
import { createFocusRegistry, provideFormRendererContext } from '../store/injection';
import type { FormSubmitPayload, FormValue } from '../types';
import { createDefaultFieldRegistry, type FormFieldRegistry } from '../widgets/registry';
import FormSectionRenderer from './FormSectionRenderer.vue';
import FormSubmitBar from './FormSubmitBar.vue';

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
    /** 提交按钮文案；缺省「提交」（协议不承载表单级设置，由宿主注入）。 */
    submitText?: string;
  }>(),
  { formId: '', publishedVersion: 0, schemaRevision: '', submitText: '提交' },
);

const emit = defineEmits<{
  submit: [payload: FormSubmitPayload];
  'submit-success': [payload: FormSubmitPayload];
  'submit-error': [error: unknown];
  'unsupported-field': [info: { fieldKey: string; type: string }];
}>();

const runtimeRef = shallowRef<FormRuntime | null>(null);
const schemaIssues = ref<FormSchemaIssue[]>([]);
// 注册表在会话间复用：字段组件解析与异步包装缓存不应随 Schema 替换丢失。
const fieldRegistry = props.registry ?? createDefaultFieldRegistry();
const focusRegistry = createFocusRegistry();

provideFormRendererContext({
  runtime: runtimeRef,
  registry: fieldRegistry,
  registerFieldFocus: focusRegistry.registerFieldFocus,
  unregisterFieldFocus: focusRegistry.unregisterFieldFocus,
  focusField: focusRegistry.focusField,
  reportUnsupportedField: (info) => emit('unsupported-field', info),
});

// Schema 引用变更即替换整个会话（发布不可变，运行中不做字段级热更新）。
watch(
  () => props.schema,
  (schema) => {
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
  },
  { immediate: true },
);

/** 错误定位：优先聚焦第一个出错字段；聚焦不到时回退提交栏错误摘要。 */
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

async function handleSubmit(): Promise<void> {
  const runtime = runtimeRef.value;
  if (!runtime) return;
  const outcome: FormSubmitOutcome = await runtime.submit();
  if (outcome.ok) {
    emit('submit', outcome.payload);
    if (outcome.submitted) emit('submit-success', outcome.payload);
    return;
  }
  if (outcome.reason === 'invalid' || outcome.reason === 'server') focusFirstError();
  if (outcome.reason === 'error') emit('submit-error', outcome.error);
}

defineExpose({
  /** 只读会话引用；宿主可读取状态但不得绕过 action 直接改写。 */
  runtime: runtimeRef,
  submit: handleSubmit,
  focusFirstError,
  /** 手动重置会话（继续填写场景）。 */
  reset: () => runtimeRef.value?.reset(),
});
</script>

<template>
  <!-- novalidate：校验统一由运行时接管，避免原生气泡与错误态双轨展示。 -->
  <form class="evf-form" novalidate @submit.prevent="handleSubmit">
    <div v-if="schemaIssues.length > 0" class="evf-form__invalid" role="alert">
      <p class="evf-form__invalid-title">表单配置无法加载</p>
      <ul class="evf-form__invalid-list">
        <li v-for="(issue, index) in schemaIssues" :key="index">
          {{ issue.path }}：{{ issue.message }}
        </li>
      </ul>
    </div>
    <template v-else-if="runtimeRef">
      <FormSectionRenderer />
      <FormSubmitBar :submit-text="submitText" />
    </template>
  </form>
</template>
