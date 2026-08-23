<script setup lang="ts">
import {
  RiArrowLeftFill,
  RiEyeFill,
  RiLightbulbFlashFill,
  RiNotification3Fill,
  RiQuestionFill,
  RiRecycleFill,
  RiSave3Fill,
  RiShareForwardFill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import UserMenu from '~/components/navigation/UserMenu.vue';

defineOptions({ name: 'FormDesignerPage' });

type FormRouteName =
  | 'form-design'
  | 'form-workflow-design'
  | 'form-extensions'
  | 'form-publish'
  | 'form-data';
type InspectorTab = 'field' | 'form';

interface FormNavigationItem {
  name: FormRouteName;
  label: string;
}

interface FieldGroup {
  title: string;
  fields: string[];
}

const route = useRoute();
const router = useRouter();
const inspectorTab = shallowRef<InspectorTab>('field');

/** 本期仅提供页面骨架；真实表单名称将在表单详情接口接入后替换。 */
const formName = '未命名表单';
const standardNavigationItems: FormNavigationItem[] = [
  { name: 'form-design', label: '表单设计' },
  { name: 'form-extensions', label: '扩展功能' },
  { name: 'form-publish', label: '表单发布' },
  { name: 'form-data', label: '数据管理' },
];

/** 字段配置仅用于还原左侧面板层级，不承载字段创建、拖拽或排序逻辑。 */
const fieldGroups: FieldGroup[] = [
  {
    title: '常用',
    fields: [
      '单行文本',
      '多行文本',
      '数字',
      '日期时间',
      '单选按钮组',
      '复选框组',
      '下拉框',
      '下拉复选框',
      '成员单选',
      '成员多选',
      '部门单选',
      '部门多选',
      '分割线',
      '多标签页',
    ],
  },
  {
    title: '高级',
    fields: [
      '图片',
      '附件',
      '地址',
      '定位',
      '子表单',
      '查询',
      '选择数据',
      '手写签名',
      '流水号',
      '手机',
      '文字识别',
      '按钮',
      '计算',
      '富文本',
    ],
  },
  {
    title: '关联',
    fields: ['关联数据', '关联查询', '关联表单'],
  },
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
  void router.push({ name: 'App', params: { appCode: appCode.value } });
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
</script>

<template>
  <div class="form-designer-page">
    <header class="form-designer-page__header">
      <div class="form-designer-page__identity">
        <button
          class="form-designer-page__icon-button"
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
          class="form-designer-page__icon-button"
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

    <section class="form-designer-page__toolbar" aria-label="表单设计操作">
      <button
        class="form-designer-page__guide-button"
        type="button"
        @click="notifyUnavailable('新手引导')"
      >
        <RiLightbulbFlashFill />
        <span>查看新手引导</span>
      </button>
      <div class="form-designer-page__toolbar-actions">
        <button
          class="form-designer-page__action-button form-designer-page__action-button--secondary"
          type="button"
          @click="notifyUnavailable('预览')"
        >
          <RiEyeFill />
          <span>预览</span>
        </button>
        <button
          class="form-designer-page__action-button form-designer-page__action-button--primary"
          type="button"
          @click="notifyUnavailable('保存')"
        >
          <RiSave3Fill />
          <span>保存</span>
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
    </section>

    <section class="form-designer-page__workspace">
      <aside class="form-designer-page__palette" aria-label="字段组件">
        <div class="form-designer-page__palette-scroll">
          <section
            v-for="group in fieldGroups"
            :key="group.title"
            class="form-designer-page__field-group"
          >
            <div class="form-designer-page__field-group-heading">
              <h2 class="form-designer-page__field-group-title">{{ group.title }}</h2>
              <span v-if="group.title === '常用'" class="form-designer-page__ai-tag"
                >AI 推荐字段</span
              >
            </div>
            <div class="form-designer-page__field-grid">
              <button
                v-for="field in group.fields"
                :key="field"
                class="form-designer-page__field-item"
                type="button"
                @click="notifyUnavailable('字段添加')"
              >
                <span class="form-designer-page__field-symbol" aria-hidden="true">{{
                  field.slice(0, 1)
                }}</span>
                <span class="form-designer-page__field-name">{{ field }}</span>
              </button>
            </div>
          </section>
        </div>
        <button
          class="form-designer-page__recycle-button"
          type="button"
          @click="notifyUnavailable('字段回收站')"
        >
          <RiRecycleFill />
          <span>字段回收站</span>
        </button>
      </aside>

      <!-- 中间画布由未来的 @evolyn.do/form FormDesigner 接管；本期仅保留其空间边界。 -->
      <main class="form-designer-page__canvas" aria-label="表单设计画布">
        <RouterView />
      </main>

      <aside class="form-designer-page__inspector" aria-label="属性设置">
        <div class="form-designer-page__inspector-tabs" role="tablist" aria-label="属性类型">
          <button
            class="form-designer-page__inspector-tab"
            :class="{ 'form-designer-page__inspector-tab--active': inspectorTab === 'field' }"
            type="button"
            role="tab"
            :aria-selected="inspectorTab === 'field'"
            @click="inspectorTab = 'field'"
          >
            字段属性
          </button>
          <button
            class="form-designer-page__inspector-tab"
            :class="{ 'form-designer-page__inspector-tab--active': inspectorTab === 'form' }"
            type="button"
            role="tab"
            :aria-selected="inspectorTab === 'form'"
            @click="inspectorTab = 'form'"
          >
            表单属性
          </button>
        </div>
        <div class="form-designer-page__inspector-empty" role="tabpanel">
          <template v-if="inspectorTab === 'field'">
            <p class="form-designer-page__empty-copy">点击选择字段来设置属性</p>
            <p class="form-designer-page__empty-copy">按住 Ctrl 或 Command 可选择多个字段</p>
            <p class="form-designer-page__empty-copy">按住 Shift 单击字段可连续选择</p>
          </template>
          <template v-else>
            <p class="form-designer-page__empty-copy">表单属性将在设计器接入后提供</p>
          </template>
        </div>
      </aside>
    </section>
  </div>
</template>

<style scoped lang="scss">
.form-designer-page {
  display: flex;
  min-width: 1120px;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-lighter);

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
    min-height: 58px;
    padding: 0 20px;
    justify-content: space-between;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
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
    min-height: 64px;
    padding: 0 16px 0 24px;
    justify-content: space-between;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
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
    padding: 8px 10px;
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
    height: 34px;
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
    width: 34px;
    height: 34px;
    border: 1px solid var(--el-border-color);

    &:hover {
      border-color: var(--el-color-primary);
    }
  }

  &__workspace {
    display: grid;
    min-height: 0;
    flex: 1;
    grid-template-columns: 340px minmax(440px, 1fr) 360px;
  }

  &__palette,
  &__inspector {
    display: flex;
    min-width: 0;
    overflow: hidden;
    flex-direction: column;
    background: var(--el-bg-color);
  }

  &__palette {
    border-right: 1px solid var(--el-border-color-lighter);
  }

  &__palette-scroll {
    min-height: 0;
    padding: 20px 20px 12px;
    flex: 1;
    overflow-y: auto;
  }

  &__field-group {
    margin-bottom: 26px;
  }

  &__field-group-heading {
    min-height: 28px;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 10px;
  }

  &__field-group-title {
    margin: 0;
    font-size: 16px;
    font-weight: 650;
    line-height: 24px;
  }

  &__ai-tag {
    padding: 3px 8px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-small);
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
  }

  &__field-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  &__field-item {
    min-width: 0;
    height: 42px;
    padding: 0 10px;
    gap: 8px;
    color: var(--el-text-color-regular);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
    text-align: left;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      border-color 0.18s ease,
      transform 0.18s ease;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary-light-5);
      transform: translateY(-1px);
    }
  }

  &__field-symbol {
    display: inline-grid;
    width: 18px;
    height: 18px;
    flex: 0 0 auto;
    place-items: center;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    border-radius: 4px;
    font-size: 11px;
    font-weight: 700;
  }

  &__field-name {
    overflow: hidden;
    font-size: 14px;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__recycle-button {
    min-height: 50px;
    color: var(--el-text-color-regular);
    background: var(--el-bg-color);
    border-top: 1px solid var(--el-border-color-lighter);
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__canvas {
    min-width: 0;
    overflow: auto;
    background: var(--el-fill-color-lighter);
  }

  &__inspector {
    border-left: 1px solid var(--el-border-color-lighter);
  }

  &__inspector-tabs {
    min-height: 60px;
    justify-content: space-around;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__inspector-tab {
    position: relative;
    height: 60px;
    padding: 0 18px;
    color: var(--el-text-color-regular);
    background: transparent;
    font-size: 16px;
    font-weight: 600;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &::after {
      position: absolute;
      right: 18px;
      bottom: 0;
      left: 18px;
      height: 2px;
      content: '';
      background: transparent;
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

  &__inspector-empty {
    display: flex;
    min-height: 0;
    padding: 48px 36px;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
  }

  &__empty-copy {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 14px;
    line-height: 24px;
  }
}

@media (max-width: 1120px) {
  .form-designer-page {
    overflow: auto;
  }
}
</style>
