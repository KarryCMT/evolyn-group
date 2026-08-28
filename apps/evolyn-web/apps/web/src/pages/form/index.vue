<script setup lang="ts">
import type { FormDetail } from '~/types';
import { ApiError } from '@evolyn.do/utils';
import { RiArrowLeftFill, RiNotification3Fill, RiQuestionFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, provide, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getForm, updateFormName } from '~/api/form';
import FormWorkspaceTitleEditor from '~/components/form/FormWorkspaceTitleEditor.vue';
import UserMenu from '~/components/navigation/UserMenu.vue';
import { formWorkspaceContextKey } from './workspace-context';

defineOptions({ name: 'FormWorkspaceShell' });

type FormRouteName =
  | 'form-design'
  | 'form-workflow-design'
  | 'form-extensions'
  | 'form-publish'
  | 'form-data';

interface FormNavigationItem {
  name: FormRouteName;
  label: string;
}

const route = useRoute();
const router = useRouter();

/** 工作区外壳是详情请求的唯一所有者，所有子路由通过上下文复用同一响应。 */
const formDetail = shallowRef<FormDetail | null>(null);
const detailLoading = shallowRef(false);
const detailLoadFailed = shallowRef(false);
const renaming = shallowRef(false);
const formTitle = computed(() => formDetail.value?.name ?? '未命名表单');
const formType = computed(() => formDetail.value?.formType ?? null);
const allNavigationItems: FormNavigationItem[] = [
  { name: 'form-design', label: '表单设计' },
  { name: 'form-workflow-design', label: '流程设计' },
  { name: 'form-extensions', label: '扩展功能' },
  { name: 'form-publish', label: '表单发布' },
  { name: 'form-data', label: '数据管理' },
];

const appCode = computed(() => String(route.params.appCode ?? ''));
const formCode = computed(() => String(route.params.formCode ?? ''));
// 流程设计只属于流程表单；详情未加载时也不提前展示，避免标准表单入口闪现。
const navigationItems = computed<FormNavigationItem[]>(() =>
  allNavigationItems.filter(
    (item) => item.name !== 'form-workflow-design' || formType.value === 'workflow',
  ),
);
const activeNavigationName = computed<FormRouteName>(() => {
  const active = navigationItems.value.find((item) =>
    route.matched.some((record) => record.name === item.name),
  );
  return active?.name ?? 'form-design';
});

function setDetail(detail: FormDetail): void {
  formDetail.value = detail;
  detailLoadFailed.value = false;
}

function patchDetail(patch: Partial<FormDetail>): void {
  if (!formDetail.value) return;
  formDetail.value = { ...formDetail.value, ...patch };
}

let detailRequestVersion = 0;

async function loadDetail(code: string): Promise<FormDetail | null> {
  const requestVersion = ++detailRequestVersion;
  detailLoading.value = true;
  detailLoadFailed.value = false;
  try {
    const detail = await getForm(code);
    if (requestVersion !== detailRequestVersion || formCode.value !== code) return null;
    setDetail(detail);
    return detail;
  } catch {
    if (requestVersion !== detailRequestVersion) return null;
    detailLoadFailed.value = true;
    return null;
  } finally {
    if (requestVersion === detailRequestVersion) detailLoading.value = false;
  }
}

function reloadDetail(): Promise<FormDetail | null> {
  return formCode.value.startsWith('form_') ? loadDetail(formCode.value) : Promise.resolve(null);
}

/**
 * 名称属于表单资产，由工作区外壳统一提交并刷新共享详情；设计页只消费结果，
 * 不重新装载草稿，避免改名时覆盖画布中尚未保存的字段修改。
 */
async function renameForm(name: string): Promise<boolean> {
  const current = formDetail.value;
  const normalizedName = name.trim();
  if (!current || renaming.value || !normalizedName || normalizedName === current.name)
    return false;

  renaming.value = true;
  try {
    const detail = await updateFormName(current.code, normalizedName);
    // 路由可能在请求期间切换到另一张表单，旧响应不能覆盖新工作区详情。
    if (formCode.value === current.code) setDetail(detail);
    ElMessage.success('表单名称已更新');
    return true;
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'FORM_NAME_INVALID') {
      ElMessage.error('表单名称不能为空，且不能超过128个字符');
    } else if (error instanceof ApiError && error.errCode === 'FORBIDDEN') {
      ElMessage.error('没有修改表单名称的权限');
    } else {
      ElMessage.error('表单名称更新失败，请稍后重试');
    }
    return false;
  } finally {
    renaming.value = false;
  }
}

async function submitTitleRename(name: string, onSuccess: () => void): Promise<void> {
  if (await renameForm(name)) onSuccess();
}

provide(formWorkspaceContextKey, {
  detail: formDetail,
  loading: detailLoading,
  loadFailed: detailLoadFailed,
  renaming,
  setDetail,
  patchDetail,
  rename: renameForm,
  reload: reloadDetail,
});

// 同一路由切换 formCode 时组件会复用；watch 同时覆盖首次进入和参数变化。
watch(
  formCode,
  (value) => {
    detailRequestVersion += 1;
    if (value === 'new') {
      formDetail.value = null;
      detailLoading.value = false;
      detailLoadFailed.value = false;
      return;
    }
    if (!value.startsWith('form_')) {
      formDetail.value = null;
      detailLoading.value = false;
      detailLoadFailed.value = true;
      return;
    }
    // 兜底新建页已持有创建响应时，路由替换不再重复请求详情。
    if (formDetail.value?.code === value) return;

    formDetail.value = null;
    void loadDetail(value);
  },
  { immediate: true },
);

// 标准表单即使通过旧书签直达流程路由，也按详情事实重定向到表单设计。
watch(
  [formType, () => route.name],
  ([currentFormType, routeName]) => {
    if (currentFormType !== 'standard' || routeName !== 'form-workflow-design') return;
    void router.replace({
      name: 'form-design',
      params: { appCode: appCode.value, formCode: formCode.value },
    });
  },
  { immediate: true },
);

function returnToApplication() {
  void router.push({
    name: 'App',
    params: { appCode: appCode.value },
    query: formCode.value === 'new' ? { workspace: 'form' } : undefined,
  });
}

function navigateTo(name: FormRouteName) {
  if (name === activeNavigationName.value) return;

  void router.push({
    name,
    params: { appCode: appCode.value, formCode: formCode.value },
  });
}

function notifyUnavailable(action: string) {
  ElMessage.info(`${action}将在后续版本提供`);
}
</script>

<template>
  <div class="form-workspace-shell">
    <header class="form-workspace-shell__header">
      <div class="form-workspace-shell__identity">
        <button
          class="form-workspace-shell__icon-button form-workspace-shell__utility-button"
          type="button"
          aria-label="返回应用"
          @click="returnToApplication"
        >
          <RiArrowLeftFill />
        </button>
        <FormWorkspaceTitleEditor
          :key="formCode"
          :name="formTitle"
          :disabled="!formDetail || detailLoading || detailLoadFailed"
          :saving="renaming"
          @submit="submitTitleRename"
        />
      </div>

      <nav class="form-workspace-shell__navigation" aria-label="表单管理导航">
        <button
          v-for="item in navigationItems"
          :key="item.name"
          class="form-workspace-shell__navigation-item"
          :class="{
            'form-workspace-shell__navigation-item--active': activeNavigationName === item.name,
          }"
          type="button"
          :aria-current="activeNavigationName === item.name ? 'page' : undefined"
          @click="navigateTo(item.name)"
        >
          {{ item.label }}
        </button>
      </nav>

      <div class="form-workspace-shell__global-actions">
        <button
          class="form-workspace-shell__icon-button form-workspace-shell__utility-button"
          type="button"
          aria-label="通知"
          @click="notifyUnavailable('通知中心')"
        >
          <RiNotification3Fill />
        </button>
        <button
          class="form-workspace-shell__icon-button"
          type="button"
          aria-label="帮助"
          @click="notifyUnavailable('帮助中心')"
        >
          <RiQuestionFill />
        </button>
        <UserMenu />
      </div>
    </header>

    <!-- 一级路由负责切换完整工作区，不能固定嵌入表单设计器的画布。 -->
    <RouterView />
  </div>
</template>

<style scoped lang="scss">
.form-workspace-shell {
  display: flex;
  width: 100%;
  min-width: 0;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);

  &__header,
  &__identity,
  &__global-actions {
    display: flex;
    align-items: center;
  }

  &__header {
    position: relative;
    height: 52px;
    min-height: 52px;
    padding: 0 var(--el-space-2xl);
    justify-content: space-between;
  }

  &__identity,
  &__global-actions {
    z-index: 1;
    min-width: 260px;
    gap: var(--el-space-md);
  }

  &__global-actions {
    justify-content: flex-end;
  }

  &__icon-button,
  &__navigation-item {
    border: 0;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__icon-button {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 20px;
      height: 20px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__navigation {
    position: absolute;
    top: 0;
    left: 50%;
    display: flex;
    height: 100%;
    align-items: stretch;
    transform: translateX(-50%);
  }

  &__navigation-item {
    position: relative;
    padding: 0 var(--el-space-2xl);
    color: var(--el-text-color-regular);
    background: transparent;
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &::after {
      position: absolute;
      right: 20px;
      bottom: 0;
      left: 20px;
      height: 2px;
      content: '';
      background: transparent;
      transition: background-color 0.18s ease;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &--active {
      color: var(--el-color-primary);

      &::after {
        background: var(--el-color-primary);
      }
    }
  }
}

@media (max-width: 1120px) {
  .form-workspace-shell {
    &__identity,
    &__global-actions {
      min-width: 180px;
    }

    &__navigation-item {
      padding: 0 var(--el-space-lg);

      &::after {
        right: 14px;
        left: 14px;
      }
    }
  }
}

@media (max-width: 900px) {
  .form-workspace-shell {
    &__header {
      padding: 0 var(--el-space-lg);
    }

    &__identity {
      min-width: 150px;
    }

    &__global-actions {
      min-width: 32px;
    }

    &__utility-button {
      display: none;
    }

    &__navigation {
      position: static;
      min-width: 0;
      flex: 1;
      justify-content: center;
      transform: none;
    }

    &__navigation-item {
      padding: 0 var(--el-space-lg);
      flex: 0 0 auto;
      font-size: var(--el-font-size-base);

      &::after {
        right: 12px;
        left: 12px;
      }
    }
  }
}
</style>
