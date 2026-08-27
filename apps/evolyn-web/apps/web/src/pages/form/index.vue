<script setup lang="ts">
import { RiArrowLeftFill, RiNotification3Fill, RiQuestionFill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import UserMenu from '~/components/navigation/UserMenu.vue';

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

/** 本期仅提供页面骨架；真实表单名称将在表单详情接口接入后替换。 */
const formName = '未命名表单';
const standardNavigationItems: FormNavigationItem[] = [
  { name: 'form-design', label: '表单设计' },
  { name: 'form-workflow-design', label: '流程设计' },
  { name: 'form-extensions', label: '扩展功能' },
  { name: 'form-publish', label: '表单发布' },
  { name: 'form-data', label: '数据管理' },
];

const appCode = computed(() => String(route.params.appCode ?? ''));
const formId = computed(() => String(route.params.formId ?? ''));
// 流程设计作为统一的一级工作区显示；后续由表单详情和权限控制其可访问性。
const navigationItems = computed<FormNavigationItem[]>(() => standardNavigationItems);
const activeNavigationName = computed<FormRouteName>(() => {
  const active = navigationItems.value.find((item) =>
    route.matched.some((record) => record.name === item.name),
  );
  return active?.name ?? 'form-design';
});

function returnToApplication() {
  // 新建态暂未持久化表单资产；用查询参数承接「创建完成后进入应用工作区」的过渡体验。
  // 表单运行时接口落地后，由返回的默认资产替代这段临时标识。
  void router.push({
    name: 'App',
    params: { appCode: appCode.value },
    query: formId.value === 'new' ? { workspace: 'form' } : undefined,
  });
}

function navigateTo(name: FormRouteName) {
  if (name === activeNavigationName.value) return;

  void router.push({
    name,
    params: { appCode: appCode.value, formId: formId.value },
    query: route.query,
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
        <strong class="form-workspace-shell__title">{{ formName }}</strong>
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

  &__title {
    overflow: hidden;
    font-size: var(--el-font-size-large);
    font-weight: 650;
    line-height: 28px;
    text-overflow: ellipsis;
    white-space: nowrap;
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
