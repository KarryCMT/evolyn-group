<script setup lang="ts">
import {
  ElAlert,
  ElButton,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
} from 'element-plus';
import { computed } from 'vue';
import {
  WORKFLOW_SERVICE_MAX_RETRIES,
  WORKFLOW_SERVICE_MAX_TIMEOUT_SECONDS,
  type WorkflowNode,
  type WorkflowServiceConfig,
} from '../../schema';

/**
 * 服务节点面板（Phase 7，V1 仅出站 HTTP）：URL/请求体支持 {{expr}} 插值；
 * 鉴权凭据由平台侧注入——authorization/cookie 等敏感头禁止明文入 DSL
 *（后端校验器同样拒绝，此处前端先行提示）。
 */
defineOptions({ name: 'WorkflowServicePanel' });

const props = defineProps<{
  node: WorkflowNode;
}>();

const emit = defineEmits<{
  updateConfig: [config: WorkflowNode['config']];
}>();

const METHOD_OPTIONS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

const service = computed<WorkflowServiceConfig>(
  () => props.node.config.service ?? { action: 'http', url: '' },
);

/** 请求头键值对编辑态（Record ↔ 行数组互转，空行过滤） */
const headerRows = computed(() =>
  Object.entries(service.value.headers ?? {}).map(([name, value]) => ({ name, value })),
);

function patchService(patch: Partial<WorkflowServiceConfig>) {
  emit('updateConfig', {
    ...props.node.config,
    service: { ...service.value, ...patch },
  });
}

function updateHeader(index: number, patch: { name?: string; value?: string }) {
  const rows = headerRows.value.map((row) => ({ ...row }));
  const target = rows[index];
  if (!target) return;
  Object.assign(target, patch);
  commitHeaders(rows);
}

function removeHeader(index: number) {
  commitHeaders(headerRows.value.filter((_, i) => i !== index));
}

function addHeader() {
  commitHeaders([...headerRows.value, { name: '', value: '' }]);
}

function commitHeaders(rows: Array<{ name: string; value: string }>) {
  const headers: Record<string, string> = {};
  for (const row of rows) {
    const name = row.name.trim();
    if (!name) continue;
    headers[name] = row.value;
  }
  patchService(Object.keys(headers).length > 0 ? { headers } : { headers: undefined });
}

/** 响应映射行编辑：variable（必填）/ path / required */
const mappingRows = computed(() => service.value.responseMapping ?? []);

function updateMapping(
  index: number,
  patch: Partial<{ variable: string; path: string; required: boolean }>,
) {
  const rows = mappingRows.value.map((row) => ({ ...row }));
  const target = rows[index];
  if (!target) return;
  Object.assign(target, patch);
  patchService({ responseMapping: rows });
}

function removeMapping(index: number) {
  patchService({ responseMapping: mappingRows.value.filter((_, i) => i !== index) });
}

function addMapping() {
  patchService({
    responseMapping: [...mappingRows.value, { variable: '', path: '' }],
  });
}
</script>

<template>
  <div class="workflow-service-panel">
    <ElAlert
      class="workflow-service-panel__tip"
      type="info"
      :closable="false"
      show-icon
      title="节点异步调用外部 HTTP 服务：URL/请求体支持 {{form.x}} / {{starter.x}} / {{variables.x}} 插值；鉴权头由平台注入，请勿填写 authorization/cookie"
    />

    <ElForm label-position="top" size="default" @submit.prevent>
      <ElFormItem label="请求方法">
        <ElSelect
          :model-value="service.method ?? 'POST'"
          @update:model-value="(value) => patchService({ method: value })"
        >
          <ElOption
            v-for="method in METHOD_OPTIONS"
            :key="method"
            :value="method"
            :label="method"
          />
        </ElSelect>
      </ElFormItem>
      <ElFormItem label="请求地址（http/https）">
        <ElInput
          :model-value="service.url"
          placeholder="https://api.example.com/orders/{{form.orderId}}"
          @update:model-value="(value) => patchService({ url: value })"
        />
      </ElFormItem>

      <div class="workflow-service-panel__section">
        <div class="workflow-service-panel__section-head">
          <span>请求头</span>
          <ElButton text type="primary" size="small" @click="addHeader">添加</ElButton>
        </div>
        <div v-for="(row, index) in headerRows" :key="index" class="workflow-service-panel__row">
          <ElInput
            :model-value="row.name"
            placeholder="Header"
            @update:model-value="(value) => updateHeader(index, { name: value })"
          />
          <ElInput
            :model-value="row.value"
            placeholder="值（支持 {{expr}}）"
            @update:model-value="(value) => updateHeader(index, { value })"
          />
          <ElButton text type="danger" size="small" @click="removeHeader(index)">删除</ElButton>
        </div>
      </div>

      <ElFormItem
        v-if="(service.method ?? 'POST') !== 'GET' && (service.method ?? 'POST') !== 'DELETE'"
        label="请求体（JSON 模板，支持插值）"
      >
        <ElInput
          :model-value="service.body"
          type="textarea"
          :rows="3"
          placeholder='{"orderId": {{form.orderId}}}'
          @update:model-value="(value) => patchService({ body: value || undefined })"
        />
      </ElFormItem>

      <div class="workflow-service-panel__inline">
        <ElFormItem label="超时（秒，1~120）">
          <ElInputNumber
            :model-value="service.timeoutSeconds ?? 10"
            :min="1"
            :max="WORKFLOW_SERVICE_MAX_TIMEOUT_SECONDS"
            controls-position="right"
            @update:model-value="(value) => patchService({ timeoutSeconds: value ?? 10 })"
          />
        </ElFormItem>
        <ElFormItem label="失败重试（0~8）">
          <ElInputNumber
            :model-value="service.maxRetries ?? 3"
            :min="0"
            :max="WORKFLOW_SERVICE_MAX_RETRIES"
            controls-position="right"
            @update:model-value="(value) => patchService({ maxRetries: value ?? 3 })"
          />
        </ElFormItem>
      </div>

      <div class="workflow-service-panel__section">
        <div class="workflow-service-panel__section-head">
          <span>响应映射（写入流程变量，供后续节点引用）</span>
          <ElButton text type="primary" size="small" @click="addMapping">添加</ElButton>
        </div>
        <div
          v-for="(row, index) in mappingRows"
          :key="index"
          class="workflow-service-panel__mapping"
        >
          <ElInput
            :model-value="row.variable"
            placeholder="变量名"
            @update:model-value="(value) => updateMapping(index, { variable: value })"
          />
          <ElInput
            :model-value="row.path"
            placeholder="JSON 路径，如 data.orderId"
            @update:model-value="(value) => updateMapping(index, { path: value })"
          />
          <ElButton text type="danger" size="small" @click="removeMapping(index)">删除</ElButton>
        </div>
      </div>
    </ElForm>
  </div>
</template>

<style scoped lang="scss">
.workflow-service-panel {
  display: flex;
  padding: 0 var(--el-space-md);
  flex-direction: column;
  gap: var(--el-space-sm);

  &__tip {
    border-radius: var(--el-border-radius-base);
  }

  &__section {
    padding: var(--el-space-sm);
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__section-head {
    display: flex;
    margin-bottom: var(--el-space-xs);
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-regular);
    font-size: 13px;
    font-weight: 600;
  }

  &__row,
  &__mapping {
    display: flex;
    align-items: center;
    gap: var(--el-space-xs);

    & + & {
      margin-top: var(--el-space-xs);
    }
  }

  &__inline {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--el-space-sm);
  }
}
</style>
