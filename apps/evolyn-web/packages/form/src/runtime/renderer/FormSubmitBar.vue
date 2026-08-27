<script setup lang="ts">
import { computed } from 'vue';
import { useFormRendererContext } from '../store/injection';

/**
 * 提交栏：提交按钮、提交中状态与非字段错误摘要。
 * 按钮为 type=submit，经 FormRenderer 的 form 元素统一接管；移动端固定在安全区内（样式层）。
 */
const props = withDefaults(defineProps<{ submitText?: string }>(), { submitText: '提交' });

const { runtime } = useFormRendererContext();

const phase = computed(() => runtime.value?.state.formState ?? 'initializing');
/** 非字段错误（提交失败、版本冲突等）展示在提交栏；字段错误由各字段回显。 */
const formIssues = computed(() =>
  (runtime.value?.state.issues ?? []).filter((issue) => !issue.fieldKey),
);
const busy = computed(() => phase.value === 'submitting' || phase.value === 'submitted');
const buttonText = computed(() => {
  if (phase.value === 'submitting') return '提交中…';
  if (phase.value === 'submitted') return '已提交';
  return props.submitText;
});
</script>

<template>
  <div class="evf-submit-bar">
    <ul v-if="formIssues.length > 0" class="evf-submit-bar__issues" aria-live="polite">
      <li v-for="(issue, index) in formIssues" :key="index" class="evf-submit-bar__issue">
        {{ issue.message }}
      </li>
    </ul>
    <button
      class="evf-submit-bar__submit"
      type="submit"
      :disabled="busy || phase === 'initializing'"
    >
      {{ buttonText }}
    </button>
  </div>
</template>
