<script setup lang="ts">
import type { FormRuntimeActionDefinition, FormRuntimeAdapter } from '@evolyn.do/form/runtime';
import type { ApplicationWorkspaceAsset } from '../workspace/applicationWorkspace.types';
import type { FormRuntimeBootstrap } from '~/types';
import { FormRuntimeSurface } from '@evolyn.do/form/runtime';
import { ApiError } from '@evolyn.do/utils';
import { ElMessage } from 'element-plus';
import { shallowRef, watch } from 'vue';
import { getFormRuntime, submitFormRecord } from '~/api/form';
// 应用工作区按需加载最终运行时关键样式，不引入设计器样式图。
import '@evolyn.do/form/runtime/style.css';

defineOptions({ name: 'ApplicationWorkspaceFormRuntime' });

const props = defineProps<{
  appCode: string;
  asset: ApplicationWorkspaceAsset;
}>();

type RuntimeStatus = 'loading' | 'ready' | 'not-published' | 'error';

const status = shallowRef<RuntimeStatus>('loading');
const bootstrap = shallowRef<FormRuntimeBootstrap | null>(null);
const errorMessage = shallowRef('表单加载失败，请稍后重试');
const reloadRevision = shallowRef(0);
const unsupportedTypes = new Set<string>();

const actions: FormRuntimeActionDefinition[] = [
  {
    key: 'submit',
    label: '提交',
    behavior: 'submit',
    intent: 'primary',
    order: 100,
    mobilePresentation: 'button',
  },
];

const runtimeAdapter: FormRuntimeAdapter = {
  async submit(payload, signal) {
    try {
      await submitFormRecord(
        {
          formCode: payload.formId,
          publishedVersion: payload.publishedVersion,
          schemaRevision: payload.schemaRevision,
          values: payload.values,
        },
        signal,
      );
      return { accepted: true };
    } catch (error) {
      if (isAbortError(error)) throw error;
      if (error instanceof ApiError && error.errCode === 'FORM_RECORD_INVALID') {
        return {
          accepted: false,
          fieldErrors: (error.data as { fieldErrors?: Record<string, string[]> } | undefined)
            ?.fieldErrors,
          message: error.message,
        };
      }
      return {
        accepted: false,
        message:
          error instanceof ApiError && error.errCode === 'FORM_VERSION_CONFLICT'
            ? '表单已发布新版本，请刷新后重新填写'
            : '提交失败，请稍后重试',
      };
    }
  },
};

watch(
  [() => props.appCode, () => props.asset.targetCode, reloadRevision],
  async ([appCode, formCode], _previous, onCleanup) => {
    const controller = new AbortController();
    onCleanup(() => controller.abort());
    bootstrap.value = null;
    status.value = 'loading';

    if (!formCode) {
      status.value = 'error';
      errorMessage.value = '当前菜单未关联有效表单';
      return;
    }

    try {
      const nextBootstrap = await getFormRuntime(appCode, formCode, controller.signal);
      // 即使请求实现未遵守 AbortSignal，也禁止旧资产响应覆盖当前表单。
      if (controller.signal.aborted) return;
      bootstrap.value = nextBootstrap;
      status.value = 'ready';
    } catch (error) {
      if (controller.signal.aborted) return;
      if (error instanceof ApiError && error.errCode === 'FORM_NOT_PUBLISHED') {
        status.value = 'not-published';
        return;
      }
      errorMessage.value = '表单加载失败，请稍后重试';
      status.value = 'error';
    }
  },
  { immediate: true },
);

function reload(): void {
  reloadRevision.value += 1;
}

function onUnsupportedField(info: { fieldKey: string; type: string }): void {
  if (unsupportedTypes.has(info.type)) return;
  unsupportedTypes.add(info.type);
  ElMessage.info(`字段类型「${info.type}」的填写能力尚未上线，当前暂不可交互`);
}

function onSubmitSuccess(): void {
  ElMessage.success('提交成功');
}

function isAbortError(error: unknown): boolean {
  return (
    typeof error === 'object' &&
    error !== null &&
    ((error as { name?: unknown }).name === 'AbortError' ||
      (error as { code?: unknown }).code === 'ERR_CANCELED')
  );
}
</script>

<template>
  <main class="application-workspace-form-runtime" :aria-label="`${props.asset.label}填写区`">
    <section
      v-if="status === 'loading'"
      v-loading="true"
      class="application-workspace-form-runtime__state"
      aria-label="正在加载表单"
    />

    <el-result
      v-else-if="status === 'not-published'"
      class="application-workspace-form-runtime__state"
      icon="warning"
      title="表单尚未发布"
      sub-title="请先进入编辑页面发布表单，再返回应用填写数据。"
    />

    <el-result
      v-else-if="status === 'error'"
      class="application-workspace-form-runtime__state"
      icon="error"
      title="加载表单失败"
      :sub-title="errorMessage"
    >
      <template #extra>
        <!-- prettier-ignore -->
        <el-button type="primary" @click="reload">
          重新加载
        </el-button>
      </template>
    </el-result>

    <FormRuntimeSurface
      v-else-if="bootstrap"
      class="application-workspace-form-runtime__surface"
      :schema="bootstrap.content"
      :form-id="bootstrap.formCode"
      :published-version="bootstrap.publishedVersion"
      :schema-revision="bootstrap.schemaRevision"
      :adapter="runtimeAdapter"
      :actions="actions"
      layout="auto"
      content-width="1100px"
      @unsupported-field="onUnsupportedField"
      @submit-success="onSubmitSuccess"
    />
  </main>
</template>

<style scoped lang="scss">
.application-workspace-form-runtime {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--el-bg-color-page);

  &__state {
    flex: 1;
    min-height: 0;
  }

  &__surface {
    flex: 1;
    min-height: 0;
  }
}
</style>
