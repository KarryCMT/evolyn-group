<script setup lang="ts">
import {
  RiArrowLeftFill,
  RiEyeFill,
  RiLightbulbFlashFill,
  RiNotification3Fill,
  RiQuestionFill,
  RiSave3Fill,
  RiShareForwardFill,
} from '@remixicon/vue';
import { FormDesigner, type FormFieldPreset } from '@evolyn.do/form';
import { ElMessage } from 'element-plus';
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import UserMenu from '~/components/navigation/UserMenu.vue';

defineOptions({ name: 'FormDesignerPage' });

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
  { name: 'form-extensions', label: '扩展功能' },
  { name: 'form-publish', label: '表单发布' },
  { name: 'form-data', label: '数据管理' },
];

const appCode = computed(() => String(route.params.appCode ?? ''));
const formId = computed(() => String(route.params.formId ?? ''));
/** 新建流程表单暂以 query 表达类型；后续改为读取表单详情中的类型字段。 */
const isWorkflowForm = computed(
  () => route.query.type === 'workflow' || route.name === 'form-workflow-design',
);
const navigationItems = computed<FormNavigationItem[]>(() => {
  if (!isWorkflowForm.value) return standardNavigationItems;

  return [
    standardNavigationItems[0],
    { name: 'form-workflow-design', label: '流程设计' },
    ...standardNavigationItems.slice(1),
  ];
});
const activeNavigationName = computed<FormRouteName>(() => {
  const currentName = navigationItems.value.find((item) => item.name === route.name)?.name;
  return currentName ?? 'form-design';
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
    query: isWorkflowForm.value ? { ...route.query, type: 'workflow' } : route.query,
  });
}

/** 保存、预览和字段添加依赖后续设计器内核，本期保留明确的交互反馈。 */
function notifyUnavailable(action: string) {
  ElMessage.info(`${action}将在表单设计器接入后提供`);
}

function handleFieldSelect(field: FormFieldPreset) {
  notifyUnavailable(`添加${field.title}`);
}
</script>

<template>
  <div class="form-designer-page">
    <header class="form-designer-page__header">
      <div class="form-designer-page__identity">
        <button
          class="form-designer-page__icon-button form-designer-page__utility-button"
          type="button"
          aria-label="返回应用"
          @click="returnToApplication"
        >
          <RiArrowLeftFill />
        </button>
        <strong class="form-designer-page__title">{{ formName }}</strong>
      </div>

      <nav class="form-designer-page__navigation" aria-label="表单管理导航">
        <button
          v-for="item in navigationItems"
          :key="item.name"
          class="form-designer-page__navigation-item"
          :class="{
            'form-designer-page__navigation-item--active': activeNavigationName === item.name,
          }"
          type="button"
          :aria-current="activeNavigationName === item.name ? 'page' : undefined"
          @click="navigateTo(item.name)"
        >
          {{ item.label }}
        </button>
      </nav>

      <div class="form-designer-page__global-actions">
        <button
          class="form-designer-page__icon-button form-designer-page__utility-button"
          type="button"
          aria-label="通知"
          @click="notifyUnavailable('通知中心')"
        >
          <RiNotification3Fill />
        </button>
        <button
          class="form-designer-page__icon-button"
          type="button"
          aria-label="帮助"
          @click="notifyUnavailable('帮助中心')"
        >
          <RiQuestionFill />
        </button>
        <UserMenu />
      </div>
    </header>

    <section class="form-designer-page__surface" aria-label="表单设计工作台">
      <div class="form-designer-page__toolbar" aria-label="表单设计操作">
        <button
          class="form-designer-page__guide-button"
          type="button"
          @click="notifyUnavailable('新手引导')"
        >
          <RiLightbulbFlashFill />
          <span class="form-designer-page__guide-label">查看新手引导</span>
        </button>
        <div class="form-designer-page__toolbar-actions">
          <button
            class="form-designer-page__action-button form-designer-page__action-button--secondary"
            type="button"
            @click="notifyUnavailable('预览')"
          >
            <RiEyeFill />
            <span class="form-designer-page__action-label">预览</span>
          </button>
          <button
            class="form-designer-page__action-button form-designer-page__action-button--primary"
            type="button"
            @click="notifyUnavailable('保存')"
          >
            <RiSave3Fill />
            <span class="form-designer-page__action-label">保存</span>
          </button>
          <button
            class="form-designer-page__icon-button form-designer-page__share-button"
            type="button"
            aria-label="分享表单"
            @click="notifyUnavailable('分享')"
          >
            <RiShareForwardFill />
          </button>
        </div>
      </div>

      <FormDesigner
        class="form-designer-page__workspace"
        @select-field="handleFieldSelect"
        @open-recycle-bin="notifyUnavailable('字段回收站')"
      >
        <template #canvas>
          <RouterView />
        </template>
      </FormDesigner>
    </section>
  </div>
</template>

<style scoped lang="scss">
.form-designer-page {
  display: flex;
  width: 100%;
  min-width: 0;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;
  color: var(--el-text-color-primary);
  background: #f3f3f8;

  &__header,
  &__identity,
  &__global-actions,
  &__toolbar,
  &__toolbar-actions,
  &__guide-button,
  &__action-button,
  &__field-group-heading,
  &__field-item,
  &__recycle-button,
  &__inspector-tabs {
    display: flex;
    align-items: center;
  }

  &__header {
    position: relative;
    height: 52px;
    min-height: 52px;
    padding: 0 20px;
    justify-content: space-between;
    background: transparent;
  }

  &__identity,
  &__global-actions {
    z-index: 1;
    min-width: 260px;
    gap: 10px;
  }

  &__global-actions {
    justify-content: flex-end;
  }

  &__title {
    overflow: hidden;
    font-size: 18px;
    font-weight: 650;
    line-height: 28px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__icon-button,
  &__action-button,
  &__guide-button,
  &__navigation-item,
  &__field-item,
  &__recycle-button,
  &__inspector-tab {
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
    padding: 0 20px;
    color: var(--el-text-color-regular);
    background: transparent;
    font-size: 16px;
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

  &__toolbar {
    height: 50px;
    min-height: 50px;
    padding: 0 16px 0 24px;
    justify-content: space-between;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__surface {
    display: flex;
    min-height: 0;
    margin: 0 8px 8px;
    overflow: hidden;
    flex: 1;
    flex-direction: column;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 14px;
    box-shadow: var(--el-box-shadow-light);
  }

  &__guide-button,
  &__action-button,
  &__recycle-button {
    justify-content: center;
    gap: 7px;
    font-size: 14px;
    font-weight: 600;
  }

  &__guide-button {
    height: 32px;
    padding: 0 10px;
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 18px;
      height: 18px;
      color: var(--el-color-primary);
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__toolbar-actions {
    gap: 10px;
  }

  &__action-button {
    min-width: 76px;
    height: 32px;
    padding: 0 12px;
    border-radius: var(--el-border-radius-base);
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      border-color 0.18s ease;

    svg {
      width: 17px;
      height: 17px;
    }

    &--secondary {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      border: 1px solid var(--el-color-primary);

      &:hover {
        background: var(--el-color-primary-light-9);
      }
    }

    &--primary {
      color: var(--el-color-white);
      background: var(--el-color-primary);

      &:hover {
        background: var(--el-color-primary-light-3);
      }
    }
  }

  &__share-button {
    width: 32px;
    height: 32px;
    border: 1px solid var(--el-border-color);

    &:hover {
      border-color: var(--el-color-primary);
    }
  }

  &__workspace {
    min-height: 0;
    flex: 1;
  }
}

@media (max-width: 1120px) {
  .form-designer-page {
    &__identity,
    &__global-actions {
      min-width: 180px;
    }

    &__navigation-item {
      padding: 0 14px;

      &::after {
        right: 14px;
        left: 14px;
      }
    }
  }
}

@media (max-width: 900px) {
  .form-designer-page {
    &__header {
      padding: 0 12px;
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
      padding: 0 12px;
      flex: 0 0 auto;
      font-size: 14px;

      &::after {
        right: 12px;
        left: 12px;
      }
    }
  }
}

@media (max-width: 620px) {
  .form-designer-page {
    &__surface {
      margin: 0 4px 4px;
      border-radius: 10px;
    }

    &__identity {
      min-width: 118px;
    }

    &__title {
      font-size: 16px;
    }

    &__toolbar {
      padding: 0 10px 0 12px;
    }

    &__guide-button {
      padding: 0 4px;
    }

    &__guide-label {
      display: none;
    }

    &__toolbar-actions {
      gap: 6px;
    }

    &__action-button {
      min-width: 34px;
      padding: 0 8px;
    }

    &__action-label {
      display: none;
    }
  }
}
</style>
