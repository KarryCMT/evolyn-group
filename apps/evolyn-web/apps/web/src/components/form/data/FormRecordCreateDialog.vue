<script setup lang="ts">
import type { FormRuntimeActionDefinition, FormRuntimeAdapter } from '@evolyn.do/form/runtime-web';
import type { FormRecordSubmitResult } from '~/types';
import { FormWebRuntimeSurface } from '@evolyn.do/form/runtime-web';
import { ApiError } from '@evolyn.do/utils';
import { RiCloseFill, RiFullscreenExitLine, RiFullscreenLine } from '@remixicon/vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, shallowRef, useTemplateRef, watch } from 'vue';
import { createFormDataOperationId, submitFormRecord } from '~/api/form';
import { getMemberFieldRegistry } from '~/components/form/memberFieldRegistry';
import { useAuth } from '~/composables/auth';
import { useFormRecordCreate } from '~/composables/useFormRecordCreate';
// 数据管理弹窗按需加载填写运行时样式，不依赖设计器页面的样式副作用。
import '@evolyn.do/form/runtime-web/style.css';

defineOptions({ name: 'FormRecordCreateDialog' });

const props = defineProps<{
  appCode: string;
  formCode: string;
}>();

const emit = defineEmits<{
  submitted: [result: FormRecordSubmitResult];
}>();

const visible = defineModel<boolean>({ required: true });
const { userInfo } = useAuth();
const appCode = computed(() => props.appCode);
const formCode = computed(() => props.formCode);
const { bootstrap, status, errorMessage, load, reset } = useFormRecordCreate({ appCode, formCode });
const currentMemberId = computed(() => userInfo.value?.member?.memberCode);
const fieldPermissions = computed(() => bootstrap.value?.permissions?.addFields);
const dialogTitle = computed(() => bootstrap.value?.name || '添加数据');
const fullScreen = shallowRef(false);
const lastSubmitResult = shallowRef<FormRecordSubmitResult | null>(null);
const unsupportedTypes = new Set<string>();
const runtimeSurfaceRef = useTemplateRef<{
  getRuntime: () => { isDirty: () => boolean } | null;
}>('runtimeSurface');

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

/** 表单运行时只负责构造受控提交快照；应用 API、幂等键和成功后的列表协作留在宿主层。 */
const runtimeAdapter: FormRuntimeAdapter = {
  async submit(payload, signal) {
    try {
      const result = await submitFormRecord(
        {
          appCode: props.appCode,
          formCode: payload.formId,
          publishedVersion: payload.publishedVersion,
          schemaRevision: payload.schemaRevision,
          values: payload.values,
          hasResult: true,
          dataOpId: payload.dataOpId ?? createFormDataOperationId(),
        },
        signal,
      );
      lastSubmitResult.value = result;
      return { accepted: true };
    } catch (error) {
      if (isAbortError(error)) throw error;
      if (
        error instanceof ApiError &&
        (error.errCode === 'FORM_RECORD_INVALID' ||
          error.errCode === 'FORM_RECORD_VALIDATION_FAILED')
      ) {
        const data = error.data as
          | {
              fieldErrors?: Record<string, string[]>;
              validatorErrors?: Array<{ remind?: string; fields?: string[] }>;
            }
          | undefined;
        return {
          accepted: false,
          fieldErrors: data?.fieldErrors,
          validatorErrors: normalizeValidatorErrors(data?.validatorErrors),
          message: error.message,
        };
      }
      return {
        accepted: false,
        message:
          error instanceof ApiError && error.errCode === 'FORM_VERSION_CONFLICT'
            ? '表单已发布新版本，请关闭后重新填写'
            : '提交失败，请稍后重试',
      };
    }
  },
};

watch(visible, (open) => {
  if (!open) {
    reset();
    return;
  }
  fullScreen.value = false;
  lastSubmitResult.value = null;
  unsupportedTypes.clear();
  void load();
});

function normalizeValidatorErrors(
  errors: Array<{ index?: number; remind?: string; fields?: string[] }> | undefined,
): Array<{ index: number; remind: string; fields: string[] }> {
  return (errors ?? [])
    .flatMap((error, index) =>
      typeof error.remind === 'string'
        ? [
            {
              index: typeof error.index === 'number' ? error.index : index,
              remind: error.remind,
              fields: error.fields ?? [],
            },
          ]
        : [],
    )
    .sort((left, right) => left.index - right.index);
}

function isAbortError(error: unknown): boolean {
  return (
    typeof error === 'object' &&
    error !== null &&
    ((error as { name?: unknown }).name === 'AbortError' ||
      (error as { code?: unknown }).code === 'ERR_CANCELED')
  );
}

async function discardDraft(): Promise<boolean> {
  if (!runtimeSurfaceRef.value?.getRuntime()?.isDirty()) return true;
  try {
    await ElMessageBox.confirm('关闭后，当前已填写的内容将不会保存。', '放弃本次填写？', {
      confirmButtonText: '放弃填写',
      cancelButtonText: '继续填写',
      type: 'warning',
    });
    return true;
  } catch {
    return false;
  }
}

async function requestClose(): Promise<void> {
  if (await discardDraft()) visible.value = false;
}

async function beforeClose(done: () => void): Promise<void> {
  if (await discardDraft()) done();
}

function onUnsupportedField(info: { fieldKey: string; type: string }): void {
  if (unsupportedTypes.has(info.type)) return;
  unsupportedTypes.add(info.type);
  ElMessage.info(`字段类型「${info.type}」的填写能力尚未上线，当前暂不可交互`);
}

function onSubmitSuccess(): void {
  const result = lastSubmitResult.value;
  if (result) emit('submitted', result);
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="form-record-create-dialog"
    width="780px"
    top="5vh"
    :fullscreen="fullScreen"
    :show-close="false"
    :close-on-click-modal="false"
    :before-close="beforeClose"
    destroy-on-close
    append-to-body
  >
    <template #header>
      <header class="form-record-create-dialog__header">
        <h2 class="form-record-create-dialog__title">
          {{ dialogTitle }}
        </h2>
        <div class="form-record-create-dialog__header-actions">
          <button
            class="form-record-create-dialog__icon-button"
            type="button"
            :aria-label="fullScreen ? '退出全屏' : '全屏显示'"
            @click="fullScreen = !fullScreen"
          >
            <RiFullscreenExitLine v-if="fullScreen" />
            <RiFullscreenLine v-else />
          </button>
          <button
            class="form-record-create-dialog__icon-button"
            type="button"
            aria-label="关闭添加数据"
            @click="requestClose"
          >
            <RiCloseFill />
          </button>
        </div>
      </header>
    </template>

    <section
      v-if="status === 'loading'"
      class="form-record-create-dialog__state"
      aria-live="polite"
    >
      <span class="form-record-create-dialog__loading-mark" aria-hidden="true" />
      <p class="form-record-create-dialog__loading-text">正在加载表单…</p>
    </section>

    <el-result
      v-else-if="status === 'not-published' || status === 'forbidden'"
      class="form-record-create-dialog__state"
      icon="warning"
      :title="status === 'forbidden' ? '无法添加数据' : '表单尚未发布'"
      :sub-title="errorMessage"
    />

    <el-result
      v-else-if="status === 'error'"
      class="form-record-create-dialog__state"
      icon="error"
      title="加载表单失败"
      :sub-title="errorMessage"
    >
      <template #extra>
        <el-button type="primary" @click="load"> 重新加载 </el-button>
      </template>
    </el-result>

    <FormWebRuntimeSurface
      v-else-if="status === 'ready' && bootstrap"
      ref="runtimeSurface"
      class="form-record-create-dialog__runtime"
      :schema="bootstrap.content"
      :form-id="bootstrap.formCode"
      :published-version="bootstrap.publishedVersion"
      :schema-revision="bootstrap.schemaRevision"
      :current-member-id="currentMemberId"
      :field-permissions="fieldPermissions"
      :adapter="runtimeAdapter"
      :registry="getMemberFieldRegistry()"
      :actions="actions"
      layout="auto"
      content-width="100%"
      @unsupported-field="onUnsupportedField"
      @submit-success="onSubmitSuccess"
    />
  </el-dialog>
</template>

<style lang="scss">
// Dialog 会传送到 body；所有覆盖均由唯一块类收口，避免影响其他弹窗。
.form-record-create-dialog {
  display: flex;
  flex-direction: column;
  height: min(90vh, 980px);
  max-height: calc(100vh - 48px);
  padding: 0;
  margin-bottom: 0;
  overflow: hidden;
  border-radius: var(--el-border-radius-large);

  &.is-fullscreen {
    height: 100%;
    max-height: none;
    border-radius: 0;
  }

  .el-dialog__header {
    position: relative;
    z-index: 1;
    flex: 0 0 64px;
    padding: 0;
    margin: 0;
    background: var(--el-bg-color);
    box-shadow: 0 5px 14px rgb(15 23 42 / 9%);
  }

  .el-dialog__body {
    display: flex;
    flex: 1;
    min-height: 0;
    padding: 0;
    overflow: hidden;
    background: var(--el-fill-color-lighter);
  }

  &__header,
  &__header-actions,
  &__icon-button {
    display: flex;
    align-items: center;
  }

  &__header {
    justify-content: space-between;
    height: 100%;
    padding: 0 var(--el-space-3xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__title {
    margin: 0;
    font-size: 18px;
    font-weight: 650;
    color: var(--el-text-color-primary);
  }

  &__header-actions {
    gap: var(--el-space-sm);
  }

  &__icon-button {
    justify-content: center;
    width: 32px;
    height: 32px;
    padding: 0;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    svg {
      width: 20px;
      height: 20px;
    }
  }

  &__state {
    display: flex;
    gap: var(--el-space-md);
    align-items: center;
    justify-content: center;
    width: min(760px, calc(100% - 48px));
    margin: auto;
  }

  &__loading-mark {
    width: 18px;
    height: 18px;
    border: 2px solid var(--el-border-color);
    border-top-color: var(--el-color-primary);
    border-radius: 50%;
    animation: form-record-create-spin 0.8s linear infinite;
  }

  &__loading-text {
    margin: 0;
    color: var(--el-text-color-secondary);
  }

  &__runtime {
    flex: 1;
    min-width: 0;
    min-height: 0;
    background: var(--el-bg-color);
  }
}

@keyframes form-record-create-spin {
  to {
    transform: rotate(1turn);
  }
}

@media (width <= 620px) {
  .form-record-create-dialog {
    width: 100% !important;
    height: 100%;
    max-height: none;
    margin: 0;
    border-radius: 0;

    &__header {
      padding: 0 var(--el-space-lg);
    }

    &__title {
      font-size: var(--el-font-size-large);
    }
  }
}
</style>
