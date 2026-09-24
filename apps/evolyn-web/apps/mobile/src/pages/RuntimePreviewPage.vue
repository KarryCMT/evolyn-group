<script setup lang="ts">
import type {
  FormRuntimeActionDefinition,
  FormRuntimeAdapter,
  FormSubmitPayload,
} from '@evolyn.do/form/runtime-mobile';
import { FormMobileRuntimeSurface } from '@evolyn.do/form/runtime-mobile';
import { showSuccessToast } from 'vant';
import { runtimePreviewSchema } from './runtime-preview-schema';

const actions: readonly FormRuntimeActionDefinition[] = [
  {
    key: 'reset',
    label: '重置',
    behavior: 'reset',
    intent: 'plain',
    order: 10,
  },
  {
    key: 'submit',
    label: '提交申请',
    behavior: 'submit',
    intent: 'primary',
    order: 20,
  },
];

// 验收页只验证运行时闭环；正式页面在同一边界注入真实 API adapter。
const previewAdapter: FormRuntimeAdapter = {
  async submit() {
    return { accepted: true };
  },
};

function handleSubmitSuccess(_payload: FormSubmitPayload): void {
  showSuccessToast('运行时提交验证成功');
}
</script>

<template>
  <section class="runtime-preview-page">
    <FormMobileRuntimeSurface
      :schema="runtimePreviewSchema"
      form-id="mobile-runtime-preview"
      :published-version="1"
      schema-revision="preview-v1"
      :adapter="previewAdapter"
      :actions="actions"
      @submit-success="handleSubmitSuccess"
    />
  </section>
</template>

<style scoped>
.runtime-preview-page {
  height: calc(100dvh - var(--van-nav-bar-height) - env(safe-area-inset-top));
  min-height: 0;
  background: var(--van-background-2);
}
</style>
