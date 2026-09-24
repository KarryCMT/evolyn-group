<script setup lang="ts">
import type {
  IntelligentActorOption,
  IntelligentDesignerResources,
  IntelligentDocument,
  IntelligentFieldOption,
  IntelligentFormOption,
} from '@evolyn.do/intelligent';
import { createIntelligentDocument, IntelligentDesigner } from '@evolyn.do/intelligent';
import { ElMessage } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getDepartmentTree } from '~/api/department';
import { getForm, getFormRuntime, listForms } from '~/api/form';
import { listMembers } from '~/api/member';
import {
  intelligentSystemFields,
  projectIntelligentFormFields,
} from '~/components/form/assistant/intelligent-resources';

defineOptions({ name: 'FormAssistantDesignerPage' });

const route = useRoute();
const router = useRouter();
const forms = shallowRef<IntelligentFormOption[]>([]);
const triggerFields = shallowRef<IntelligentFieldOption[]>([]);
const members = shallowRef<IntelligentActorOption[]>([]);
const departments = shallowRef<IntelligentActorOption[]>([]);
let resourceLoadVersion = 0;

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
const resources = computed<IntelligentDesignerResources>(() => ({
  forms: forms.value,
  sourceGroups: [
    {
      nodeId: document.value.nodes.find((node) => node.type === 'trigger')?.id ?? 'trigger',
      nodeName: '触发数据',
      fields: [...triggerFields.value, ...intelligentSystemFields()],
    },
  ],
  members: members.value,
  departments: departments.value,
  loadFormFields,
}));

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

watch(
  [() => route.params.formCode, () => route.query.triggerFormCode],
  () => void loadResources(),
  { immediate: true },
);

function normalizeQueryArray(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String);
  return value ? [String(value)] : [];
}

function normalizeTriggerType(value: unknown): 'form' | 'schedule' | 'http' {
  return value === 'schedule' || value === 'http' ? value : 'form';
}

async function loadFormFields(
  formCode: string,
  signal: AbortSignal,
): Promise<readonly IntelligentFieldOption[]> {
  try {
    const runtime = await getFormRuntime(String(route.params.appCode ?? ''), formCode, signal);
    return projectIntelligentFormFields(runtime.content);
  } catch (error) {
    if (signal.aborted) throw error;
    // 未发布表单没有运行时快照；设计阶段仍投影草稿，表单目录会标注未发布状态。
    const detail = await getForm(formCode);
    if (signal.aborted) throw new DOMException('Aborted', 'AbortError');
    return projectIntelligentFormFields(detail.draft);
  }
}

async function loadResources(): Promise<void> {
  const requestVersion = ++resourceLoadVersion;
  const currentFormCode = String(route.params.formCode ?? '');
  const triggerFormCode = String(route.query.triggerFormCode || currentFormCode);
  if (!currentFormCode.startsWith('form_')) return;

  try {
    const current = await getForm(currentFormCode);
    if (requestVersion !== resourceLoadVersion) return;
    const [formPage, memberPage, departmentTree, sourceFieldList] = await Promise.all([
      listForms({ appId: current.appId, limit: 100 }),
      listMembers({ page: 1, pageSize: 200 }).catch(() => ({ items: [], total: 0 })),
      getDepartmentTree().catch(() => []),
      loadFormFields(triggerFormCode, new AbortController().signal).catch(() => []),
    ]);
    if (requestVersion !== resourceLoadVersion) return;
    forms.value = formPage.items.map((item) => ({
      code: item.code,
      name: item.name,
      formType: item.formType,
      publishedVersion: item.publishedVersion,
      disabled: item.publishedVersion === 0,
    }));
    triggerFields.value = [...sourceFieldList];
    members.value = memberPage.items.map((member) => ({
      value: member.memberCode,
      label: member.name,
    }));
    departments.value = flattenDepartments(departmentTree);
  } catch {
    if (requestVersion !== resourceLoadVersion) return;
    forms.value = [];
    triggerFields.value = [];
    ElMessage.warning('智能助手资源加载失败，请稍后重试');
  }
}

function flattenDepartments(
  nodes: Array<{ id: number; name: string; children?: unknown[] }>,
): IntelligentActorOption[] {
  return nodes.flatMap((node) => [
    { value: String(node.id), label: node.name },
    ...(Array.isArray(node.children)
      ? flattenDepartments(
          node.children as Array<{ id: number; name: string; children?: unknown[] }>,
        )
      : []),
  ]);
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
    :resources="resources"
    @update-document="updateDocument"
    @back="goBack"
    @save="save"
    @enable="enable"
    @help="showHelp"
  />
</template>
