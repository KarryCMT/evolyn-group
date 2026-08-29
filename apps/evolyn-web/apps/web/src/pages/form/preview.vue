<script setup lang="ts">
import type { FormRuntimeActionDefinition, FormRuntimeAdapter } from '@evolyn.do/form/runtime';
import type { FormSchemaIssue } from '@evolyn.do/form/schema';
import type { FormSchemaDocument } from '~/types';
import { FormRuntimeSurface } from '@evolyn.do/form/runtime';
import { migrateFormSchema } from '@evolyn.do/form/schema';
import { ApiError } from '@evolyn.do/utils';
import { RiArrowGoBackFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getFormRuntime, submitFormRecord } from '~/api/form';
import { loadFormPreviewDocument } from './preview-storage';
// 运行时样式独立于设计器 style.css，最终用户填写页只加载关键 CSS。
import '@evolyn.do/form/runtime/style.css';

defineOptions({ name: 'FormPreviewPage' });

const route = useRoute();
const router = useRouter();

/**
 * 表单填写/预览页（P2 闭环）：
 * - 已发布：GET bootstrap（快照 + 双口令）→ 真实提交（POST /form-records，
 *   字段错误按 widgetName 回填）；
 * - 未发布（设计器预览入口）：回退 sessionStorage 草稿本地回放，提交不落库。
 */
const formCode = computed(() => String(route.params.formCode ?? ''));
const appCode = computed(() => String(route.params.appCode ?? ''));

// 协议文档含递归 JSON 类型，shallowRef 避免深层 UnwrapRef 实例化（TS2589）。
const documentRef = shallowRef<FormSchemaDocument | null>(null);
const runtimeInfo = shallowRef<{
  name: string;
  publishedVersion: number;
  schemaRevision: string;
} | null>(null);
const emptyText = shallowRef('未找到可预览的表单，请先在表单设计器中编辑并点击预览');
const initializing = shallowRef(true);

void (async () => {
  // 读取侧统一走迁移器：迁移 + 校验，无效配置展示受控错误态。
  const adopt = (raw: unknown, issues: FormSchemaIssue[]) => {
    if (issues.length > 0) {
      emptyText.value = `表单配置无效：${issues[0].path} ${issues[0].message}`;
      return;
    }
    documentRef.value = raw as FormSchemaDocument;
  };

  // 路由参数防御：非 form_ 编码（如新建态残留）不打 bootstrap，直接走草稿回放。
  const canBootstrap = formCode.value.startsWith('form_');

  try {
    if (!canBootstrap) {
      throw new ApiError('未发布', 0, 'FORM_NOT_PUBLISHED');
    }
    const bootstrap = await getFormRuntime(appCode.value, formCode.value);
    runtimeInfo.value = {
      name: bootstrap.name,
      publishedVersion: bootstrap.publishedVersion,
      schemaRevision: bootstrap.schemaRevision,
    };
    adopt(bootstrap.content, []);
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'FORM_NOT_PUBLISHED') {
      // 未发布：回退设计器传递的草稿本地回放
      const draft = loadFormPreviewDocument(formCode.value);
      if (draft) {
        const migrated = migrateFormSchema(draft);
        adopt(migrated.document, migrated.issues);
      } else {
        emptyText.value = '表单尚未发布；如需预览，请先在表单设计器中编辑并点击预览';
      }
    } else {
      emptyText.value = '表单加载失败，请稍后重试';
    }
  } finally {
    initializing.value = false;
  }
})();

const pageTitle = computed(() => runtimeInfo.value?.name ?? '表单预览');

const unsupportedTypes = new Set<string>();

/** 已发布走真实提交（服务端终审 + 字段错误回填）；草稿回放本地通过。 */
const runtimeAdapter: FormRuntimeAdapter = {
  async submit(payload) {
    if (!runtimeInfo.value) {
      return { accepted: true };
    }
    try {
      await submitFormRecord({
        formCode: payload.formId,
        publishedVersion: payload.publishedVersion,
        schemaRevision: payload.schemaRevision,
        values: payload.values,
      });
      return { accepted: true };
    } catch (error) {
      if (error instanceof ApiError && error.errCode === 'FORM_RECORD_INVALID') {
        const fieldErrors = (error.data as { fieldErrors?: Record<string, string[]> } | undefined)
          ?.fieldErrors;
        return {
          accepted: false,
          fieldErrors,
          message: error.message,
        };
      }
      return { accepted: false, message: '提交失败，请稍后重试' };
    }
  },
};

const runtimeActions: FormRuntimeActionDefinition[] = [
  {
    key: 'submit',
    label: '提交',
    behavior: 'submit',
    intent: 'primary',
    order: 100,
    mobilePresentation: 'button',
  },
];

function onUnsupportedField(info: { fieldKey: string; type: string }): void {
  if (unsupportedTypes.has(info.type)) return;
  unsupportedTypes.add(info.type);
  ElMessage.info(`字段类型「${info.type}」的填写能力尚未上线，预览中暂不可交互`);
}

function onSubmitSuccess(): void {
  ElMessage.success(runtimeInfo.value ? '提交成功' : '预览提交成功');
}

function goBack(): void {
  router.push({
    name: 'form-design',
    params: { appCode: route.params.appCode, formCode: formCode.value },
  });
}
</script>

<template>
  <section class="form-preview-page" aria-label="表单预览">
    <header class="form-preview-page__header">
      <button
        class="form-preview-page__back"
        type="button"
        aria-label="返回表单设计"
        @click="goBack"
      >
        <RiArrowGoBackFill />
        <span class="form-preview-page__back-label">返回设计</span>
      </button>
      <h1 class="form-preview-page__title">
        {{ pageTitle }}
        <span class="form-preview-page__title-tag">
          {{ runtimeInfo ? `已发布 v${runtimeInfo.publishedVersion}` : '预览' }}
        </span>
      </h1>
    </header>

    <div class="form-preview-page__body">
      <el-empty v-if="initializing" description="正在加载表单…" />
      <el-empty v-else-if="!documentRef" :description="emptyText" />
      <FormRuntimeSurface
        v-else
        class="form-preview-page__runtime"
        :schema="documentRef"
        :form-id="formCode"
        :published-version="runtimeInfo?.publishedVersion ?? 0"
        :schema-revision="runtimeInfo?.schemaRevision ?? ''"
        :adapter="runtimeAdapter"
        :actions="runtimeActions"
        layout="auto"
        content-width="860px"
        @unsupported-field="onUnsupportedField"
        @submit-success="onSubmitSuccess"
      />
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-preview-page {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  background: var(--el-bg-color-page);

  &__header {
    display: flex;
    gap: var(--el-space-lg);
    align-items: center;
    height: 50px;
    min-height: 50px;
    padding: 0 var(--el-space-xl);
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__back {
    display: inline-flex;
    gap: var(--el-space-sm);
    align-items: center;
    height: 32px;
    padding: 0 var(--el-space-md);
    font-size: var(--el-font-size-base);
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__title {
    display: flex;
    gap: var(--el-space-md);
    align-items: center;
    margin: 0;
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__title-tag {
    padding: 0 var(--el-space-md);
    font-size: var(--el-font-size-extra-small);
    font-weight: 500;
    line-height: 20px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-base);
  }

  &__body {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  &__runtime {
    flex: 1;
    min-height: 0;
  }
}

@media (width <= 620px) {
  .form-preview-page {
    &__back-label {
      display: none;
    }
  }
}
</style>
