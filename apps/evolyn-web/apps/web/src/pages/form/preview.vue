<script setup lang="ts">
import { FormRenderer, type FormRuntimeAdapter } from '@evolyn.do/form/runtime';
import { migrateFormSchema, type FormSchemaIssue } from '@evolyn.do/form/schema';
import { ApiError } from '@evolyn.do/utils';
import { RiArrowGoBackFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
// 运行时样式独立于设计器 style.css，最终用户填写页只加载关键 CSS。
import '@evolyn.do/form/runtime/style.css';
import { getFormRuntime, submitFormRecord } from '~/api/form';
import type { FormSchemaDocument } from '~/types';
import { loadFormPreviewDocument } from './preview-storage';

defineOptions({ name: 'FormPreviewPage' });

const route = useRoute();
const router = useRouter();

/**
 * 表单填写/预览页（P2 闭环）：
 * - 已发布：GET bootstrap（快照 + 双口令）→ 真实提交（POST /form-records，
 *   字段错误按 widgetName 回填）；
 * - 未发布（设计器预览入口）：回退 sessionStorage 草稿本地回放，提交不落库。
 */
const formId = computed(() => String(route.params.formId ?? ''));
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

  // 路由参数防御：非正整数（如新建态残留）不打 bootstrap，直接走草稿回放。
  const numericFormId = Number(formId.value);
  const canBootstrap = Number.isInteger(numericFormId) && numericFormId > 0;

  try {
    if (!canBootstrap) {
      throw new ApiError('未发布', 0, 'FORM_NOT_PUBLISHED');
    }
    const bootstrap = await getFormRuntime(appCode.value, numericFormId);
    runtimeInfo.value = {
      name: bootstrap.name,
      publishedVersion: bootstrap.publishedVersion,
      schemaRevision: bootstrap.schemaRevision,
    };
    adopt(bootstrap.content, []);
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'FORM_NOT_PUBLISHED') {
      // 未发布：回退设计器传递的草稿本地回放
      const draft = loadFormPreviewDocument(formId.value);
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
        formId: Number(payload.formId),
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
    params: { appCode: route.params.appCode, formId: formId.value },
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
      <el-scrollbar>
        <div class="form-preview-page__canvas">
          <el-empty v-if="initializing" description="正在加载表单…" />
          <el-empty v-else-if="!documentRef" :description="emptyText" />
          <!-- Web 宿主接入：--evf-* 映射到 --el-*，运行时随宿主主题与暗色模式联动。 -->
          <FormRenderer
            v-else
            class="form-preview-page__runtime"
            :schema="documentRef"
            :form-id="formId"
            :published-version="runtimeInfo?.publishedVersion ?? 0"
            :schema-revision="runtimeInfo?.schemaRevision ?? ''"
            :adapter="runtimeAdapter"
            @unsupported-field="onUnsupportedField"
            @submit-success="onSubmitSuccess"
          />
        </div>
      </el-scrollbar>
    </div>
  </section>
</template>

<style scoped lang="scss">
.form-preview-page {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color-page);

  &__header {
    display: flex;
    height: 50px;
    min-height: 50px;
    align-items: center;
    gap: var(--el-space-lg);
    padding: 0 var(--el-space-xl);
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__back {
    display: inline-flex;
    height: 32px;
    align-items: center;
    gap: var(--el-space-sm);
    padding: 0 var(--el-space-md);
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-base);
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    cursor: pointer;

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
    align-items: center;
    gap: var(--el-space-md);
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 600;
  }

  &__title-tag {
    padding: 0 var(--el-space-md);
    color: var(--el-color-primary);
    font-size: var(--el-font-size-extra-small);
    font-weight: 500;
    line-height: 20px;
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-base);
  }

  &__body {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
  }

  &__canvas {
    display: flex;
    max-width: 860px;
    min-height: 100%;
    margin: 0 auto;
    padding: var(--el-space-3xl) var(--el-space-md) var(--el-space-4xl);
    flex-direction: column;
  }

  // 运行时主题映射：中性色/状态色全部取宿主 Element Plus 变量，暗色模式自动跟随。
  &__runtime {
    --evf-color-text: var(--el-text-color-primary);
    --evf-color-text-regular: var(--el-text-color-regular);
    --evf-color-text-secondary: var(--el-text-color-secondary);
    --evf-color-text-placeholder: var(--el-text-color-placeholder);
    --evf-color-text-disabled: var(--el-text-color-disabled);
    --evf-color-border: var(--el-border-color);
    --evf-color-border-light: var(--el-border-color-light);
    --evf-color-border-lighter: var(--el-border-color-lighter);
    --evf-color-fill-light: var(--el-fill-color-light);
    --evf-color-bg: var(--el-bg-color);
    --evf-color-primary: var(--el-color-primary);
    --evf-color-danger: var(--el-color-danger);
    --evf-font-size-base: var(--el-font-size-base);
    --evf-font-size-small: var(--el-font-size-small);
    --evf-font-size-extra-small: var(--el-font-size-extra-small);
    --evf-space-sm: var(--el-space-sm);
    --evf-space-md: var(--el-space-md);
    --evf-space-lg: var(--el-space-lg);
    --evf-space-xl: var(--el-space-xl);
    --evf-space-3xl: var(--el-space-3xl);
    --evf-radius-base: var(--el-border-radius-base);
    --evf-radius-medium: var(--el-border-radius-medium);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-large);
    box-shadow: var(--el-box-shadow-light);
  }
}

@media (max-width: 620px) {
  .form-preview-page {
    &__back-label {
      display: none;
    }

    &__canvas {
      padding: var(--el-space-lg) var(--el-space-xs) var(--el-space-3xl);
    }
  }
}
</style>
