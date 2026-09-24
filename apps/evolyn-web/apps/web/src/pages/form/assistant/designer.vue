<script setup lang="ts">
import type { IntelligentDocument } from '@evolyn.do/intelligent';
import { createIntelligentDocument, IntelligentDesigner } from '@evolyn.do/intelligent';
import { ElMessage } from 'element-plus';
import { shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

defineOptions({ name: 'FormAssistantDesignerPage' });

const route = useRoute();
const router = useRouter();

function createDocumentFromRoute(): IntelligentDocument {
  return createIntelligentDocument({
    id: String(route.params.assistantId ?? globalThis.crypto.randomUUID()),
    name: String(route.query.name || '未命名智能助手 Pro'),
    tags: normalizeQueryArray(route.query.tags),
    triggerType: normalizeTriggerType(route.query.triggerType),
    triggerFormCode: String(route.query.triggerFormCode || ''),
    triggerFormName: String(route.query.triggerFormName || '当前表单'),
  });
}

// 创建接口落地前，路由草稿只用于建立包内文档；后续可直接替换为服务端文档。
const document = shallowRef<IntelligentDocument>(createDocumentFromRoute());

// Vue Router 会复用同一路由组件，助手参数变化时必须显式刷新画布事实源。
watch(
  [
    () => route.params.assistantId,
    () => route.query.name,
    () => route.query.tags,
    () => route.query.triggerType,
    () => route.query.triggerFormCode,
    () => route.query.triggerFormName,
  ],
  () => {
    document.value = createDocumentFromRoute();
  },
);

function normalizeQueryArray(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String);
  return value ? [String(value)] : [];
}

function normalizeTriggerType(value: unknown): 'form' | 'schedule' | 'http' {
  return value === 'schedule' || value === 'http' ? value : 'form';
}

function goBack(): void {
  void router.push({
    name: 'form-extension-ai',
    params: {
      appCode: String(route.params.appCode ?? ''),
      formCode: String(route.params.formCode ?? ''),
    },
  });
}

function updateDocument(next: IntelligentDocument): void {
  document.value = next;
}

function save(): void {
  ElMessage.success('智能助手草稿已保存');
}

function enable(): void {
  ElMessage.success('智能助手已保存并启用');
}

function showHelp(): void {
  ElMessage.info('智能助手 Pro 帮助中心正在建设中');
}
</script>

<template>
  <IntelligentDesigner
    :document="document"
    @update-document="updateDocument"
    @back="goBack"
    @save="save"
    @enable="enable"
    @help="showHelp"
  />
</template>
