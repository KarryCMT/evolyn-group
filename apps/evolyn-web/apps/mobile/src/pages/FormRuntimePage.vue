<script setup lang="ts">
import type {
  FormDraftPayload,
  FormRuntimeActionDefinition,
  FormRuntimeAdapter,
  FormSubmitPayload,
  FormValue,
} from '@evolyn.do/form/runtime-mobile';
import type { MobileFormRuntimeBootstrap } from '~/types/app';
import { FormMobileRuntimeSurface } from '@evolyn.do/form/runtime-mobile';
import { migrateFormSchema } from '@evolyn.do/form/schema';
import { showFailToast, showSuccessToast, showToast, Empty as VanEmpty, Icon as VanIcon, Loading as VanLoading } from 'vant';
import { computed, shallowRef, useTemplateRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  executeFormLinkage,
  getFormRuntime,
  queryRelatedOptions,
  submitFormRecord,
} from '~/api/forms';
import { useAuthStore } from '~/stores/auth';
import { demoOrderSchema } from './demo-order-schema';

type RuntimeStatus = 'loading' | 'ready' | 'not-published' | 'error';

interface StoredDraft {
  publishedVersion: number;
  schemaRevision: string;
  values: Record<string, FormValue>;
  savedAt: string;
}

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const runtimeSurfaceRef = useTemplateRef<{ reset: () => void }>('runtimeSurface');
const status = shallowRef<RuntimeStatus>('loading');
const errorMessage = shallowRef('表单加载失败，请稍后重试');
const bootstrap = shallowRef<MobileFormRuntimeBootstrap | null>(null);
const initialValues = shallowRef<Record<string, FormValue> | undefined>();
const reloadRevision = shallowRef(0);

const appCode = computed(() => routeParam('appCode'));
const formCode = computed(() => routeParam('formCode'));
const menuCode = computed(() => (typeof route.query.menuCode === 'string' ? route.query.menuCode : ''));
const formType = computed(() => (route.query.formType === 'workflow' ? 'workflow' : 'standard'));
const formName = computed(
  () => bootstrap.value?.name || (typeof route.query.name === 'string' ? route.query.name : '填写表单'),
);
const tenantName = computed(() => auth.userInfo?.tenant.name || '灵衍云');
const currentMemberId = computed(() => auth.userInfo?.member.memberCode);
const fieldPermissions = computed(() => bootstrap.value?.permissions?.addFields);
const runtimeKey = computed(
  () => `${formCode.value}:${bootstrap.value?.schemaRevision ?? reloadRevision.value}`,
);
const isDemo = computed(() => appCode.value.startsWith('demo-'));

const actions: readonly FormRuntimeActionDefinition[] = [
  {
    key: 'draft',
    label: '保存草稿',
    behavior: 'save-draft',
    intent: 'plain',
    order: 10,
    mobilePresentation: 'compact',
  },
  {
    key: 'submit',
    label: '提交',
    behavior: 'submit',
    intent: 'primary',
    order: 20,
    mobilePresentation: 'button',
  },
];

const runtimeAdapter: FormRuntimeAdapter = {
  queryRelatedOptions(input, signal) {
    return queryRelatedOptions(
      input.formId,
      input.fieldId,
      {
        schemaVersion: input.schemaVersion,
        values: input.values,
        keyword: input.keyword,
        pageSize: input.pageSize,
      },
      signal,
    );
  },
  executeLinkage(input, signal) {
    return executeFormLinkage(
      input.formId,
      input.ruleId,
      {
        schemaVersion: input.schemaVersion,
        values: input.values,
        requestVersion: input.requestVersion,
      },
      signal,
    );
  },
  submit(payload, signal) {
    if (isDemo.value) return Promise.resolve({ accepted: true });
    return submitFormRecord(appCode.value, menuCode.value, payload, signal);
  },
  async saveDraft(payload) {
    const draft: StoredDraft = {
      publishedVersion: payload.publishedVersion,
      schemaRevision: payload.schemaRevision,
      values: payload.values,
      savedAt: new Date().toISOString(),
    };
    localStorage.setItem(draftStorageKey(), JSON.stringify(draft));
  },
};

function routeParam(key: 'appCode' | 'formCode'): string {
  const value = route.params[key];
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '');
}

function draftStorageKey(): string {
  const accountId = auth.userInfo?.account.id ?? 'anonymous';
  const tenantId = auth.userInfo?.tenant.id ?? 'tenant';
  return `evolyn.mobile.form-draft.${tenantId}.${accountId}.${appCode.value}.${formCode.value}`;
}

function restoreDraft(nextBootstrap: MobileFormRuntimeBootstrap): Record<string, FormValue> | undefined {
  try {
    const raw = localStorage.getItem(draftStorageKey());
    if (!raw) return undefined;
    const stored = JSON.parse(raw) as StoredDraft;
    if (
      stored.publishedVersion !== nextBootstrap.publishedVersion ||
      stored.schemaRevision !== nextBootstrap.schemaRevision
    ) {
      localStorage.removeItem(draftStorageKey());
      return undefined;
    }
    showToast('已恢复上次保存的草稿');
    return stored.values;
  } catch {
    localStorage.removeItem(draftStorageKey());
    return undefined;
  }
}

function demoBootstrap(): MobileFormRuntimeBootstrap {
  return {
    formCode: formCode.value,
    name: typeof route.query.name === 'string' ? route.query.name : '订单管理',
    publishedVersion: 1,
    schemaRevision: 'demo-mobile-v1',
    protocolVersion: 13,
    content: demoOrderSchema,
  };
}

watch(
  [appCode, formCode, reloadRevision],
  async (_values, _previous, onCleanup) => {
    const controller = new AbortController();
    onCleanup(() => controller.abort());
    status.value = 'loading';
    bootstrap.value = null;
    initialValues.value = undefined;
    try {
      const response = isDemo.value
        ? demoBootstrap()
        : await getFormRuntime(appCode.value, formCode.value, controller.signal);
      if (controller.signal.aborted) return;
      const migrated = migrateFormSchema(response.content, response.protocolVersion);
      if (!migrated.document) {
        errorMessage.value = `表单配置无效：${migrated.issues[0]?.message ?? '未知错误'}`;
        status.value = 'error';
        return;
      }
      const nextBootstrap = {
        ...response,
        protocolVersion: migrated.protocolVersion,
        content: migrated.document,
      };
      initialValues.value = restoreDraft(nextBootstrap);
      bootstrap.value = nextBootstrap;
      status.value = 'ready';
    } catch (error) {
      if (controller.signal.aborted) return;
      const code = typeof error === 'object' && error !== null && 'errCode' in error
        ? (error as { errCode?: string }).errCode
        : undefined;
      if (code === 'FORM_NOT_PUBLISHED') {
        status.value = 'not-published';
        return;
      }
      errorMessage.value = error instanceof Error ? error.message : '表单加载失败，请稍后重试';
      status.value = 'error';
    }
  },
  { immediate: true },
);

function reload(): void {
  reloadRevision.value += 1;
}

function handleDraftSuccess(_payload: FormDraftPayload): void {
  showSuccessToast('草稿已保存');
}

function handleSubmitSuccess(_payload: FormSubmitPayload): void {
  localStorage.removeItem(draftStorageKey());
  runtimeSurfaceRef.value?.reset();
  showSuccessToast(isDemo.value ? '演示提交成功' : '提交成功');
}
</script>

<template>
  <main class="form-page">
    <header class="form-brand-header">
      <button type="button" aria-label="返回应用目录" @click="router.back()">
        <VanIcon name="arrow-left" />
      </button>
      <strong><i aria-hidden="true" />{{ tenantName }}</strong>
      <span />
    </header>

    <section class="form-page__title">
      <h1>{{ formType === 'workflow' ? '发起流程' : formName }}</h1>
    </section>

    <p v-if="formType === 'workflow' && status === 'ready'" class="form-page__intro">
      请填写以下流程表单，提交后将进入审批流程。
    </p>

    <section v-if="status === 'loading'" class="form-page__state">
      <VanLoading color="var(--van-primary-color)" />
    </section>
    <section v-else-if="status === 'not-published'" class="form-page__state">
      <VanEmpty image="error" description="表单尚未发布，请联系管理员" />
    </section>
    <section v-else-if="status === 'error'" class="form-page__state form-page__state--error">
      <VanEmpty image="network" :description="errorMessage" />
      <button type="button" @click="reload">
        重新加载
      </button>
    </section>
    <FormMobileRuntimeSurface
      v-else-if="bootstrap"
      :key="runtimeKey"
      ref="runtimeSurface"
      class="form-page__runtime"
      :schema="bootstrap.content"
      :form-id="bootstrap.formCode"
      :published-version="bootstrap.publishedVersion"
      :schema-revision="bootstrap.schemaRevision"
      :initial-values="initialValues"
      :current-member-id="currentMemberId"
      :field-permissions="fieldPermissions"
      :adapter="runtimeAdapter"
      :actions="actions"
      @draft-success="handleDraftSuccess"
      @draft-error="showFailToast('草稿保存失败，请重试')"
      @draft-unavailable="showFailToast('当前表单暂不支持保存草稿')"
      @submit-success="handleSubmitSuccess"
      @unsupported-field="({ type }) => showToast(`字段类型“${type}”暂不支持移动端填写`)"
    />
  </main>
</template>

<style scoped>
.form-page {
  display: flex;
  width: min(100%, 604px);
  height: 100dvh;
  min-height: 0;
  margin: 0 auto;
  color: #1c2739;
  background: #fff;
  flex-direction: column;
}

.form-brand-header {
  display: grid;
  min-height: calc(88px + env(safe-area-inset-top));
  padding: env(safe-area-inset-top) 16px 0;
  background: #f5f6f8;
  grid-template-columns: 44px 1fr 44px;
  align-items: end;
}

.form-brand-header button {
  display: grid;
  width: 40px;
  height: 54px;
  padding: 0;
  font-size: 20px;
  color: #566171;
  background: none;
  border: 0;
  place-items: center;
}

.form-brand-header strong {
  display: flex;
  height: 54px;
  gap: 10px;
  font-size: 20px;
  font-weight: 500;
  align-items: center;
  justify-content: center;
}

.form-brand-header strong i {
  width: 18px;
  height: 18px;
  background: color-mix(in srgb, var(--van-primary-color) 72%, white);
  border-radius: 50%;
}

.form-page__title {
  min-height: 76px;
  padding: 0 24px;
  background: #f5f6f8;
}

.form-page__title h1 {
  margin: 0;
  font-size: 23px;
  line-height: 76px;
}

.form-page__intro {
  padding: 24px 25px 25px;
  margin: 0;
  font-size: 19px;
  line-height: 1.7;
  color: var(--van-primary-color);
  background: #fff;
  border-bottom: 1px solid #d4d8de;
}

.form-page__state {
  display: grid;
  min-height: 0;
  flex: 1;
  place-items: center;
}

.form-page__state--error {
  align-content: center;
}

.form-page__state--error > button {
  padding: 8px 20px;
  color: var(--van-primary-color);
  background: none;
  border: 1px solid var(--van-primary-color);
  border-radius: 20px;
}

.form-page__runtime {
  height: 0;
  min-height: 0;
  overflow: hidden;
  flex: 1;
}

.form-page__runtime :deep(.evf-mobile-runtime-surface__canvas) {
  padding-top: 12px;
}

.form-page__runtime :deep(.evf-form__body) {
  padding: 0 25px 24px;
}

.form-page__runtime :deep(.evf-mobile-action-bar) {
  padding-right: 22px;
  padding-left: 22px;
  background: rgb(255 255 255 / 96%);
}

.form-page__runtime :deep([data-action-key='draft']) {
  max-width: 145px;
  border-color: transparent;
}

@media (max-width: 420px) {
  .form-brand-header {
    min-height: calc(68px + env(safe-area-inset-top));
  }

  .form-page__title {
    min-height: 60px;
    padding: 0 18px;
  }

  .form-page__title h1 {
    font-size: 20px;
    line-height: 60px;
  }

  .form-page__intro {
    padding: 18px;
    font-size: 16px;
  }
}
</style>
