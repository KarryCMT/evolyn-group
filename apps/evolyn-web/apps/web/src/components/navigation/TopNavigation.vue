<script setup lang="ts">
import {
  RiApps2Fill,
  RiArrowLeftLine,
  RiContactsBook3Fill,
  RiHomeGearFill,
  RiLayoutGridFill,
  RiNotification3Fill,
  RiQuestionFill,
} from '@remixicon/vue';
import { shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import appLogo from '~/assets/logo/logo.png';
import MessageCenterDrawer from '~/components/dashboard/messageCenter/MessageCenterDrawer.vue';
import UserMenu from '~/components/navigation/UserMenu.vue';

defineOptions({ name: 'TopNavigation' });

const props = withDefaults(
  defineProps<{
    /** 顶栏默认标题；通过 title 插槽可替换为面包屑或其他自定义内容。 */
    title?: string;
    /** 传入后进入详情页形态，左侧默认展示返回入口。 */
    backTo?: string;
    /** 是否展示成员工作台的默认快捷导航。 */
    showDefaultNavigation?: boolean;
    /** 是否展示全局帮助入口；详情页可按页面需要隐藏。 */
    showHelp?: boolean;
    /** 顶栏承载页面内容时使用白色表面；工作台保持浅色画布。 */
    surface?: 'canvas' | 'surface';
  }>(),
  {
    title: '工作台',
    backTo: undefined,
    showDefaultNavigation: true,
    showHelp: true,
    surface: 'canvas',
  },
);

/**
 * 顶栏是各业务页的公共壳：
 * - leading/title 用于替换左侧品牌或标题；
 * - navigation 用于替换工作台的默认快捷导航；
 * - actions 在全局工具前追加页面级操作；
 * - trailing 用于完整接管右侧的全局工具区。
 *
 * 插槽只负责渲染，页面跳转仍由页面自身处理，避免公共组件耦合具体业务。
 */
defineSlots<{
  leading?: (props: { backTo?: string; goBack: () => void }) => unknown;
  title?: (props: { title: string }) => unknown;
  navigation?: () => unknown;
  actions?: () => unknown;
  trailing?: () => unknown;
}>();

const router = useRouter();
const messageCenterVisible = shallowRef(false);
const unreadMessageCount = shallowRef(0);

function goBack() {
  if (props.backTo) {
    void router.push(props.backTo);
  }
}

/** 成员端只提供入口，卡片编排始终在独立设计页完成。 */
function openWorkbenchEditor() {
  void router.push({ name: 'custom_workbench' });
}

/** 模板中心和通讯录页面尚未落地，先保留可复用的视觉入口。 */
function notifyUnavailable() {}
</script>

<template>
  <header class="top-navigation" :class="{ 'top-navigation--surface': surface === 'surface' }">
    <div class="top-navigation__brand">
      <slot name="leading" :back-to="backTo" :go-back="goBack">
        <button
          v-if="backTo"
          class="top-navigation__icon-button top-navigation__back-button"
          type="button"
          aria-label="返回工作台"
          @click="goBack"
        >
          <RiArrowLeftLine />
        </button>
        <template v-else>
          <button
            class="top-navigation__switcher"
            type="button"
            aria-label="切换产品"
            @click="notifyUnavailable"
          >
            <RiApps2Fill />
          </button>
          <img class="top-navigation__logo" :src="appLogo" alt="" aria-hidden="true" />
        </template>
      </slot>

      <slot name="title" :title="title">
        <strong class="top-navigation__title">{{ title }}</strong>
      </slot>
    </div>

    <div class="top-navigation__actions">
      <slot name="navigation">
        <nav
          v-if="!backTo && showDefaultNavigation"
          class="top-navigation__quick-nav"
          aria-label="工作台导航"
        >
          <button
            class="top-navigation__icon-button"
            type="button"
            aria-label="自定义工作台"
            @click="openWorkbenchEditor"
          >
            <RiHomeGearFill />
          </button>
          <span class="top-navigation__divider" aria-hidden="true" />
          <button class="top-navigation__nav-button" type="button" @click="notifyUnavailable">
            <RiLayoutGridFill />
            <span>模板中心</span>
          </button>
          <button class="top-navigation__nav-button" type="button" @click="notifyUnavailable">
            <RiContactsBook3Fill />
            <span>通讯录</span>
          </button>
          <span class="top-navigation__divider" aria-hidden="true" />
        </nav>
      </slot>

      <div v-if="$slots.actions" class="top-navigation__page-actions">
        <slot name="actions" />
      </div>

      <slot name="trailing">
        <div class="top-navigation__global-actions">
          <button
            class="top-navigation__icon-button top-navigation__notice"
            type="button"
            aria-label="通知"
            @click="messageCenterVisible = true"
          >
            <RiNotification3Fill />
            <span
              v-if="unreadMessageCount"
              class="top-navigation__notice-dot"
              aria-label="有未读通知"
            />
          </button>
          <button
            v-if="showHelp"
            class="top-navigation__icon-button"
            type="button"
            aria-label="帮助"
            @click="notifyUnavailable"
          >
            <RiQuestionFill />
          </button>
          <!-- 用户头像下拉：信息区 + 菜单面板。 -->
          <UserMenu />
        </div>
      </slot>
    </div>
  </header>
  <MessageCenterDrawer
    v-model="messageCenterVisible"
    @unread-change="unreadMessageCount = $event"
  />
</template>

<style scoped lang="scss">
.top-navigation {
  display: flex;
  height: 52px;
  min-height: 52px;
  padding: 0 16px;
  align-items: center;
  justify-content: space-between;
  color: #202938;
  background: var(--gp-color-bg-page);
  font-size: 16px;

  &--surface {
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__brand,
  &__actions,
  &__quick-nav,
  &__global-actions,
  &__page-actions {
    display: flex;
    min-width: 0;
    align-items: center;
  }

  &__brand {
    gap: 12px;
  }

  &__actions {
    height: 100%;
    gap: 10px;
  }

  &__quick-nav,
  &__page-actions {
    gap: 6px;
  }

  /* 全局图标与用户头像之间保持统一的 12px 留白。 */
  &__global-actions {
    gap: 12px;
  }

  &__switcher,
  &__icon-button,
  &__nav-button {
    display: inline-flex;
    border: 0;
    align-items: center;
    justify-content: center;
    color: #515968;
    background: transparent;
    cursor: pointer;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease,
      transform 0.18s ease;

    &:hover {
      color: #1e2938;
      background: rgb(54 65 82 / 8%);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__switcher {
    box-sizing: border-box;
    width: 32px;
    height: 32px;
    padding: 0 4px;
    border-radius: 10px;
    background: #ffffff;
    box-shadow: 0 2px 7px rgb(30 41 59 / 11%);

    svg {
      width: 24px;
      height: 24px;
    }

    &:hover {
      background: #ffffff;
      box-shadow: 0 4px 10px rgb(30 41 59 / 16%);
      transform: translateY(-1px);
    }
  }

  &__back-button,
  &__icon-button {
    box-sizing: border-box;
    width: 32px;
    height: 32px;
    padding: 0 4px;
    border-radius: 8px;

    svg {
      width: 24px;
      height: 24px;
    }
  }

  &__back-button {
    margin-right: 2px;
  }

  &__logo {
    display: block;
    width: 28px;
    height: 28px;
    flex: 0 0 28px;
    object-fit: contain;
  }

  &__title {
    overflow: hidden;
    font-size: 16px;
    font-weight: 700;
    letter-spacing: 0.02em;
    line-height: 1;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__nav-button {
    height: 32px;
    gap: 6px;
    padding: 0 8px;
    border-radius: 8px;
    font-size: 16px;
    font-weight: 500;
    white-space: nowrap;

    svg {
      width: 24px;
      height: 24px;
    }
  }

  &__divider {
    width: 1px;
    height: 20px;
    margin: 0 5px;
    background: #dfe3eb;
  }

  &__notice {
    position: relative;
  }

  &__notice-dot {
    position: absolute;
    top: 5px;
    right: 5px;
    width: 7px;
    height: 7px;
    border: 1.5px solid #f6f7fb;
    border-radius: 50%;
    background: #f15b5f;
  }
}

@media (max-width: 760px) {
  .top-navigation {
    padding: 0 10px;

    &__quick-nav .top-navigation__nav-button > span,
    &__quick-nav .top-navigation__divider {
      display: none;
    }

    &__quick-nav {
      gap: 2px;
    }

    &__title {
      font-size: 16px;
    }
  }
}
</style>
