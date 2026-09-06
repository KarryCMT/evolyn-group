<script setup lang="ts">
import type { WorkflowCenterScope } from '~/composables/useWorkflowCenter';
import type { WorkflowPendingTaskSummaryDto } from '~/types';
import {
  RiArrowDownSLine,
  RiArrowRightSLine,
  RiLayoutGridFill,
  RiNotification3Fill,
  RiPlayCircleFill,
  RiPushpin2Fill,
  RiSendPlaneFill,
  RiTaskFill,
} from '@remixicon/vue';
import { computed, shallowRef, watch } from 'vue';

export interface WorkflowNavigationForm {
  code: string;
  label: string;
}

defineOptions({ name: 'WorkflowCenterNavigation' });

const props = defineProps<{
  scope: WorkflowCenterScope;
  activeFormCode: string;
  /** 由待办摘要筛选后的流程表单，空数组表示当前成员没有可按表单筛选的待办。 */
  pendingForms: readonly WorkflowNavigationForm[];
  pendingSummary: WorkflowPendingTaskSummaryDto | null;
  /** 应用工作区沿用蓝色侧栏时使用浅色前景。 */
  inverted?: boolean;
}>();

const emit = defineEmits<{
  updateScope: [scope: WorkflowCenterScope];
  updateFormCode: [formCode: string];
  openDashboard: [];
}>();

// 仅在进入「我的待办」时默认展开；从其他个人范围进入流程菜单不能无故露出子项。
const pendingExpanded = shallowRef(props.scope === 'pending');
const pinnedFormCodes = shallowRef<string[]>([]);

const formCountByCode = computed(
  () => new Map(props.pendingSummary?.formCounts.map(({ formCode, count }) => [formCode, count])),
);
const orderedForms = computed(() => {
  const pinnedCodes = new Set(pinnedFormCodes.value);
  return [...props.pendingForms].sort((left, right) => {
    const leftPinned = pinnedCodes.has(left.code);
    const rightPinned = pinnedCodes.has(right.code);
    if (leftPinned !== rightPinned) return leftPinned ? -1 : 1;
    return 0;
  });
});

function isScopeActive(scope: WorkflowCenterScope): boolean {
  return props.scope === scope && (scope !== 'pending' || !props.activeFormCode);
}

function selectPending(): void {
  pendingExpanded.value = true;
  emit('updateScope', 'pending');
  emit('updateFormCode', '');
}

function selectForm(formCode: string): void {
  pendingExpanded.value = true;
  emit('updateScope', 'pending');
  emit('updateFormCode', formCode);
}

function togglePinned(formCode: string): void {
  pinnedFormCodes.value = pinnedFormCodes.value.includes(formCode)
    ? pinnedFormCodes.value.filter((code) => code !== formCode)
    : [...pinnedFormCodes.value, formCode];
}

watch(
  () => props.scope,
  (scope) => {
    if (scope !== 'pending') pendingExpanded.value = false;
  },
);
</script>

<template>
  <nav
    class="workflow-center-navigation"
    :class="{ 'workflow-center-navigation--inverted': props.inverted }"
    aria-label="流程菜单"
  >
    <div class="workflow-center-navigation__pending-group">
      <div
        class="workflow-center-navigation__item workflow-center-navigation__item--pending"
        :class="{ 'workflow-center-navigation__item--active': isScopeActive('pending') }"
      >
        <button
          class="workflow-center-navigation__main-action"
          type="button"
          :aria-current="isScopeActive('pending') ? 'page' : undefined"
          @click="selectPending"
        >
          <RiNotification3Fill aria-hidden="true" />
          <span>我的待办</span>
          <span
            v-if="props.pendingSummary && props.pendingSummary.total > 0"
            class="workflow-center-navigation__total-count"
          >
            {{ props.pendingSummary.total }}
          </span>
        </button>
        <button
          v-if="props.pendingForms.length"
          class="workflow-center-navigation__expand"
          type="button"
          :aria-label="pendingExpanded ? '收起待办流程' : '展开待办流程'"
          :aria-expanded="pendingExpanded"
          @click="pendingExpanded = !pendingExpanded"
        >
          <RiArrowDownSLine v-if="pendingExpanded" aria-hidden="true" />
          <RiArrowRightSLine v-else aria-hidden="true" />
        </button>
      </div>

      <div v-show="pendingExpanded && props.pendingForms.length" class="workflow-center-navigation__forms">
        <div
          v-for="form in orderedForms"
          :key="form.code"
          class="workflow-center-navigation__form"
          :class="{ 'workflow-center-navigation__form--active': props.activeFormCode === form.code }"
        >
          <button
            class="workflow-center-navigation__form-action"
            type="button"
            :aria-current="props.activeFormCode === form.code ? 'page' : undefined"
            @click="selectForm(form.code)"
          >
            <span class="workflow-center-navigation__form-label">{{ form.label }}</span>
            <span
              v-if="formCountByCode.get(form.code)"
              class="workflow-center-navigation__form-count"
            >
              {{ formCountByCode.get(form.code) }}
            </span>
          </button>
          <el-tooltip :content="pinnedFormCodes.includes(form.code) ? '取消置顶' : '置顶'" placement="right">
            <button
              class="workflow-center-navigation__pin"
              :class="{ 'workflow-center-navigation__pin--active': pinnedFormCodes.includes(form.code) }"
              type="button"
              :aria-label="pinnedFormCodes.includes(form.code) ? `取消置顶 ${form.label}` : `置顶 ${form.label}`"
              @click="togglePinned(form.code)"
            >
              <RiPushpin2Fill aria-hidden="true" />
            </button>
          </el-tooltip>
        </div>
      </div>
    </div>

    <button
      v-for="item in [
        { scope: 'started' as const, label: '我发起的', icon: RiPlayCircleFill },
        { scope: 'completed' as const, label: '我处理的', icon: RiTaskFill },
        { scope: 'cc-to-me' as const, label: '抄送我的', icon: RiSendPlaneFill },
      ]"
      :key="item.scope"
      class="workflow-center-navigation__item workflow-center-navigation__top-level"
      :class="{ 'workflow-center-navigation__item--active': isScopeActive(item.scope) }"
      type="button"
      :aria-current="isScopeActive(item.scope) ? 'page' : undefined"
      @click="emit('updateScope', item.scope)"
    >
      <component :is="item.icon" aria-hidden="true" />
      <span>{{ item.label }}</span>
    </button>

    <button
      class="workflow-center-navigation__item workflow-center-navigation__top-level"
      type="button"
      @click="emit('openDashboard')"
    >
      <RiLayoutGridFill aria-hidden="true" />
      <span>我的仪表盘</span>
    </button>
  </nav>
</template>

<style scoped lang="scss">
.workflow-center-navigation {
  display: flex;
  flex-direction: column;
  /* 与应用资产树的行间距保持一致，避免范围切换后视觉节奏变化。 */
  gap: var(--el-space-xs);

  &__pending-group,
  &__forms {
    display: flex;
    flex-direction: column;
  }

  &__forms {
    gap: var(--el-space-xs);
    padding-top: var(--el-space-sm);
  }

  &__item,
  &__main-action,
  &__form,
  &__form-action {
    display: flex;
    min-width: 0;
    align-items: center;
  }

  &__item,
  &__main-action,
  &__form-action,
  &__expand,
  &__pin {
    border: 0;
    color: inherit;
    cursor: pointer;
    background: transparent;
  }

  &__item {
    box-sizing: border-box;
    width: 100%;
    min-height: var(--application-workspace-menu-item-height, 42px);
    padding: 0 var(--el-space-md);
    gap: var(--el-space-md);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-base);
    font-weight: 500;
    text-align: left;

    svg {
      width: 19px;
      height: 19px;
      flex: 0 0 auto;
      color: var(--el-text-color-secondary);
    }

    &:hover,
    &--active {
      background: var(--el-fill-color-light);
    }

    &:focus-visible,
    &:has(button:focus-visible) {
      outline: 2px solid var(--el-color-primary-light-5);
      outline-offset: -2px;
    }
  }

  &__item--pending {
    padding: 0;
    gap: 0;
  }

  &__main-action {
    min-width: 0;
    min-height: var(--application-workspace-menu-item-height, 42px);
    flex: 1;
    padding: 0 var(--el-space-md);
    gap: var(--el-space-md);
    font: inherit;
    text-align: left;

    svg {
      width: 19px;
      height: 19px;
      flex: 0 0 auto;
      color: var(--el-text-color-secondary);
    }
  }

  &__total-count {
    display: inline-flex;
    min-width: 24px;
    height: 24px;
    padding: 0 6px;
    margin-left: auto;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--el-color-white);
    border-radius: 999px;
    color: var(--el-color-white);
    background: var(--el-color-danger);
    font-size: var(--el-font-size-small);
    font-variant-numeric: tabular-nums;
    line-height: 1;
  }

  &__expand {
    display: inline-flex;
    width: 36px;
    min-height: var(--application-workspace-menu-item-height, 42px);
    padding: 0;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;

    svg {
      width: 18px;
      height: 18px;
      color: var(--el-text-color-secondary);
    }
  }

  &__form {
    min-height: var(--application-workspace-menu-item-height, 42px);
    padding: 0 var(--el-space-xs) 0 38px;
    border-radius: var(--el-border-radius-medium);

    &:hover,
    &--active {
      background: var(--el-fill-color-light);
    }

    &:hover .workflow-center-navigation__pin,
    &:has(.workflow-center-navigation__pin--active) .workflow-center-navigation__pin {
      opacity: 1;
    }
  }

  &__form-action {
    min-width: 0;
    min-height: var(--application-workspace-menu-item-height, 42px);
    flex: 1;
    gap: var(--el-space-sm);
    font: inherit;
    text-align: left;
  }

  &__form-label {
    overflow: hidden;
    min-width: 0;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__form-count {
    margin-left: auto;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    font-variant-numeric: tabular-nums;
  }

  &__pin {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    border-radius: var(--el-border-radius-small);
    color: var(--el-text-color-placeholder);
    opacity: 0;

    svg {
      width: 18px;
      height: 18px;
    }

    &:hover,
    &--active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      opacity: 1;
      outline: 2px solid var(--el-color-primary-light-5);
    }
  }

  &--inverted {
    .workflow-center-navigation__item,
    .workflow-center-navigation__main-action,
    .workflow-center-navigation__form-action,
    .workflow-center-navigation__expand,
    .workflow-center-navigation__pin {
      color: var(--el-color-white);
    }

    .workflow-center-navigation__item svg,
    .workflow-center-navigation__main-action svg,
    .workflow-center-navigation__expand svg {
      color: var(--el-color-white);
    }

    .workflow-center-navigation__item:hover,
    .workflow-center-navigation__item--active,
    .workflow-center-navigation__form:hover,
    .workflow-center-navigation__form--active {
      background: rgb(0 0 0 / 14%);
    }

    .workflow-center-navigation__form-count {
      color: rgb(255 255 255 / 76%);
    }

    .workflow-center-navigation__pin {
      color: rgb(255 255 255 / 72%);

      &:hover,
      &--active {
        color: var(--el-color-white);
        background: rgb(0 0 0 / 18%);
      }
    }
  }
}
</style>
