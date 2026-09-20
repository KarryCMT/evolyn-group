<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue';
import { ElAlert, ElButton, ElDrawer, ElEmpty, ElInput, ElMessage } from 'element-plus';
import {
  referencedWidgetNames,
  type FormEvent,
  type FormEventDebugExecutor,
  type FormEventDebugResult,
  type FormEventFieldOption,
} from './frontend-events';

const props = defineProps<{
  event?: FormEvent;
  fields: readonly FormEventFieldOption[];
  execute?: FormEventDebugExecutor;
}>();
const open = defineModel<boolean>({ required: true });
const values = ref<Record<string, string>>({});
const sending = shallowRef(false);
const latestSequence = shallowRef(0);
const result = shallowRef<FormEventDebugResult>();
const requestError = shallowRef('');

/** 调试只要求填写请求模板实际引用到的字段，避免把整张表单搬进面板。 */
const inputFields = computed(() => {
  if (!props.event) return [];
  const templates = [
    props.event.request.url,
    ...props.event.request.header.flatMap((entry) => [entry.key, entry.value]),
    ...props.event.request.body.flatMap((entry) => [entry.key, entry.value]),
  ];
  const keys = new Set(templates.flatMap(referencedWidgetNames));
  return props.fields.filter((field) => keys.has(field.key));
});

function resolveTemplate(template: string): string {
  return template.replace(
    /\$\{([A-Za-z0-9_]+)\}/g,
    (_match, key: string) => values.value[key] ?? '',
  );
}

const requestPreview = computed(() => {
  if (!props.event) return null;
  return {
    method: props.event.request.method.toUpperCase(),
    url: resolveTemplate(props.event.request.url),
    headers: props.event.request.header
      .filter((entry) => entry.key.trim())
      .map((entry) => ({ key: resolveTemplate(entry.key), value: resolveTemplate(entry.value) })),
    body: props.event.request.body
      .filter((entry) => entry.key.trim())
      .map((entry) => ({ key: resolveTemplate(entry.key), value: resolveTemplate(entry.value) })),
  };
});

const responseWrites = computed(() => {
  if (!result.value) return '';
  return JSON.stringify(result.value.writes, null, 2);
});

async function sendRequest(): Promise<void> {
  if (!props.event || !props.execute || sending.value) return;
  const sequence = latestSequence.value + 1;
  latestSequence.value = sequence;
  sending.value = true;
  requestError.value = '';
  try {
    const response = await props.execute(props.event.id, {
      values: { ...values.value },
      sequence,
    });
    // 只呈现最后一次请求结果，避免慢响应覆盖新结果。
    if (response.sequence !== latestSequence.value) return;
    result.value = response;
    if (response.errorCode) ElMessage.warning(`调试完成：${response.errorCode}`);
    else ElMessage.success('调试请求已完成');
  } catch (error) {
    requestError.value = error instanceof Error ? error.message : '请求失败，请稍后重试';
  } finally {
    if (sequence === latestSequence.value) sending.value = false;
  }
}

watch(
  [open, () => props.event],
  ([visible, event]) => {
    if (!visible || !event) return;
    values.value = Object.fromEntries(inputFields.value.map((field) => [field.key, '']));
    result.value = undefined;
    requestError.value = '';
  },
  { immediate: true },
);
</script>

<template>
  <ElDrawer
    v-model="open"
    class="form-event-debug-drawer"
    direction="btt"
    size="90%"
    :with-header="false"
    append-to-body
  >
    <div class="form-event-debug-drawer__shell">
      <header class="form-event-debug-drawer__header">
        <h2>前端事件调试</h2>
        <button type="button" aria-label="关闭调试抽屉" @click="open = false">×</button>
      </header>

      <div v-if="props.event && requestPreview" class="form-event-debug-drawer__content">
        <section class="form-event-debug-drawer__column">
          <h3>请填写字段并发送请求</h3>
          <div class="form-event-debug-drawer__input-card">
            <div v-if="inputFields.length" class="form-event-debug-drawer__field-list">
              <label
                v-for="field in inputFields"
                :key="field.key"
                class="form-event-debug-drawer__field"
              >
                <span>{{ field.group ? `${field.group} · ${field.label}` : field.label }}</span>
                <ElInput
                  v-model="values[field.key]"
                  :placeholder="`输入${field.label || field.key}`"
                />
              </label>
            </div>
            <ElEmpty v-else description="该请求未引用表单字段，可直接发送请求" :image-size="88" />
            <ElAlert
              title="请求由后端安全代理发送"
              description="服务端会校验 HTTPS、域名白名单、私网地址、受限请求头及响应大小。"
              type="info"
              :closable="false"
              show-icon
            />
            <div class="form-event-debug-drawer__send-bar">
              <ElButton
                type="primary"
                :loading="sending"
                :disabled="!props.execute"
                @click="sendRequest"
                >发送请求</ElButton
              >
            </div>
          </div>
        </section>

        <aside class="form-event-debug-drawer__column form-event-debug-drawer__debug-column">
          <h3>调试信息</h3>
          <div class="form-event-debug-drawer__debug-card">
            <section class="form-event-debug-drawer__request-preview">
              <h4>请求内容 <span>（触发前端事件后自动刷新）</span></h4>
              <dl>
                <div>
                  <dt>请求类型</dt>
                  <dd>{{ result?.requestSummary.method ?? requestPreview.method }}</dd>
                </div>
                <div>
                  <dt>URL</dt>
                  <dd class="is-code">
                    {{ (result?.requestSummary.url ?? requestPreview.url) || '—' }}
                  </dd>
                </div>
                <div v-if="result?.requestSummary.headerNames.length">
                  <dt>Header</dt>
                  <dd class="is-code">{{ result.requestSummary.headerNames.join('\n') }}</dd>
                </div>
                <div v-else-if="requestPreview.headers.length">
                  <dt>Header</dt>
                  <dd class="is-code">
                    {{ requestPreview.headers.map((entry) => entry.key).join('\n') }}
                  </dd>
                </div>
                <div v-if="result?.requestSummary.body">
                  <dt>Body</dt>
                  <dd class="is-code">{{ result.requestSummary.body }}</dd>
                </div>
                <div v-else-if="requestPreview.body.length">
                  <dt>Body</dt>
                  <dd class="is-code">
                    {{
                      requestPreview.body.map((entry) => `${entry.key}: ${entry.value}`).join('\n')
                    }}
                  </dd>
                </div>
              </dl>
            </section>
            <section class="form-event-debug-drawer__response-preview">
              <h4>返回内容 <span>（触发前端事件后自动刷新）</span></h4>
              <dl v-if="result">
                <div>
                  <dt>状态 / 耗时</dt>
                  <dd>
                    {{ result.responseSummary.statusCode }} ·
                    {{ result.responseSummary.durationMs }} ms
                  </dd>
                </div>
                <div v-if="result.errorCode">
                  <dt>错误码</dt>
                  <dd>{{ result.errorCode }}</dd>
                </div>
                <div>
                  <dt>响应</dt>
                  <dd class="is-code">{{ result.responseSummary.body || '—' }}</dd>
                </div>
                <div>
                  <dt>字段回填</dt>
                  <dd class="is-code">{{ responseWrites }}</dd>
                </div>
              </dl>
              <p v-else-if="requestError" class="is-error">{{ requestError }}</p>
              <p v-else>发送请求后将在此展示状态、耗时、脱敏响应以及字段回填结果。</p>
            </section>
          </div>
        </aside>
      </div>

      <ElEmpty v-else description="请选择一个前端事件后再调试" :image-size="120" />
    </div>
  </ElDrawer>
</template>

<style scoped lang="scss">
.form-event-debug-drawer__shell {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--el-fill-color-extra-light);
}
.form-event-debug-drawer__header {
  position: relative;
  display: grid;
  min-height: 60px;
  flex: 0 0 auto;
  place-items: center;
  padding: 0 24px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.form-event-debug-drawer__header h2 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 600;
  line-height: 1.35;
}
.form-event-debug-drawer__header button {
  position: absolute;
  top: 12px;
  right: 14px;
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: var(--el-text-color-regular);
  font-size: 28px;
  font-weight: 300;
  line-height: 1;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--el-border-radius-base);
}
.form-event-debug-drawer__header button:hover {
  color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}
.form-event-debug-drawer__content {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: minmax(0, 1.35fr) minmax(340px, 0.95fr);
  gap: 16px;
  padding: 18px 20px 20px;
  overflow: hidden;
}
.form-event-debug-drawer__column {
  display: flex;
  min-height: 0;
  flex-direction: column;
}
.form-event-debug-drawer__column > h3 {
  flex: 0 0 auto;
  margin: 0 0 8px;
  color: var(--el-text-color-primary);
  font-size: 16px;
  font-weight: 600;
}
.form-event-debug-drawer__input-card,
.form-event-debug-drawer__debug-card {
  min-height: 0;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  box-shadow: 0 2px 6px rgb(0 0 0 / 4%);
}
.form-event-debug-drawer__input-card {
  display: flex;
  height: 100%;
  flex-direction: column;
}
.form-event-debug-drawer__field-list {
  display: grid;
  gap: 12px;
  padding: 24px 28px 16px;
  overflow-y: auto;
}
.form-event-debug-drawer__field {
  display: grid;
  max-width: 480px;
  gap: 6px;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}
.form-event-debug-drawer__input-card > :deep(.el-empty) {
  flex: 1;
}
.form-event-debug-drawer__input-card > :deep(.el-alert) {
  margin: 0 20px 12px;
}
.form-event-debug-drawer__send-bar {
  display: flex;
  min-height: 58px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-top: 1px solid var(--el-border-color-lighter);
}
.form-event-debug-drawer__debug-card {
  flex: 1;
  padding: 14px 16px;
  overflow-y: auto;
}
.form-event-debug-drawer h4 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.5;
}
.form-event-debug-drawer h4 span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
}
.form-event-debug-drawer__request-preview {
  padding: 2px 0 14px;
}
.form-event-debug-drawer dl {
  display: grid;
  gap: 10px;
  margin: 12px 0 0;
  padding: 12px 14px;
  background: var(--el-fill-color-extra-light);
}
.form-event-debug-drawer dl div {
  display: grid;
  gap: 4px;
}
.form-event-debug-drawer dt {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}
.form-event-debug-drawer dd {
  margin: 0;
  color: var(--el-text-color-regular);
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.form-event-debug-drawer .is-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}
.form-event-debug-drawer__response-preview {
  min-height: 0;
  padding: 14px 0 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.form-event-debug-drawer__response-preview p {
  margin: 12px 0 0;
  padding: 12px 14px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
  background: var(--el-fill-color-extra-light);
}
.form-event-debug-drawer__response-preview .is-error {
  color: var(--el-color-danger);
}
:global(.form-event-debug-drawer .el-drawer__body) {
  min-height: 0;
  padding: 0;
  overflow: hidden;
}
:global(.form-event-debug-drawer.el-drawer) {
  width: 100% !important;
  border-radius: 12px 12px 0 0;
}
@media (max-width: 760px) {
  .form-event-debug-drawer__header {
    padding: 0 18px;
  }
  .form-event-debug-drawer__header button {
    right: 10px;
  }
  .form-event-debug-drawer__content {
    grid-template-columns: 1fr;
    padding: 14px;
    overflow-y: auto;
  }
  .form-event-debug-drawer__input-card {
    min-height: 250px;
  }
  .form-event-debug-drawer__debug-card {
    min-height: 280px;
  }
}
</style>
