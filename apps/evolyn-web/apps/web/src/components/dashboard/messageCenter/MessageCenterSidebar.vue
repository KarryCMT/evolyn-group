<script setup lang="ts">
import type { MessageCategoryId, MessageCenterView } from './messageCenter.types';
import {
  RiApps2Fill,
  RiBarChartBoxFill,
  RiContactsBookFill,
  RiFileTextFill,
  RiLineChartFill,
  RiLinksFill,
  RiMailFill,
  RiSettings3Fill,
} from '@remixicon/vue';
import { computed } from 'vue';
import { messageCategories } from './messageCenter.constants';

defineOptions({ name: 'MessageCenterSidebar' });

defineProps<{
  activeCategoryId: MessageCategoryId;
  activeView: MessageCenterView;
  unreadCountByCategory: Partial<Record<MessageCategoryId, number>>;
}>();

const emit = defineEmits<{
  selectCategory: [categoryId: MessageCategoryId];
  openSettings: [];
}>();

const productCategories = computed(() =>
  messageCategories.filter((item) => item.group === 'product'),
);
const enterpriseCategories = computed(() =>
  messageCategories.filter((item) => item.group === 'enterprise'),
);

const categoryIcons = {
  'data-reminder': RiBarChartBoxFill,
  'app-log': RiApps2Fill,
  'document-activity': RiFileTextFill,
  'usage-reminder': RiLineChartFill,
  'contacts-management': RiContactsBookFill,
  'open-platform': RiLinksFill,
  'system-management': RiSettings3Fill,
  'operation-notice': RiMailFill,
} as const;
</script>

<template>
  <aside class="message-center-sidebar" aria-label="消息分类">
    <div class="message-center-sidebar__group">
      <p class="message-center-sidebar__group-label">灵衍云</p>
      <button
        v-for="category in productCategories"
        :key="category.id"
        class="message-center-sidebar__category"
        :class="{
          'message-center-sidebar__category--active':
            activeView === 'inbox' && activeCategoryId === category.id,
        }"
        type="button"
        @click="emit('selectCategory', category.id)"
      >
        <el-icon class="message-center-sidebar__category-icon">
          <component :is="categoryIcons[category.id]" />
        </el-icon>
        <span>{{ category.label }}</span>
        <span
          v-if="unreadCountByCategory[category.id]"
          class="message-center-sidebar__unread-count"
          :aria-label="`${unreadCountByCategory[category.id]} 条未读`"
        >
          {{ unreadCountByCategory[category.id] }}
        </span>
      </button>
    </div>

    <div class="message-center-sidebar__group">
      <p class="message-center-sidebar__group-label">企业消息</p>
      <button
        v-for="category in enterpriseCategories"
        :key="category.id"
        class="message-center-sidebar__category"
        :class="{
          'message-center-sidebar__category--active':
            activeView === 'inbox' && activeCategoryId === category.id,
        }"
        type="button"
        @click="emit('selectCategory', category.id)"
      >
        <el-icon class="message-center-sidebar__category-icon">
          <component :is="categoryIcons[category.id]" />
        </el-icon>
        <span>{{ category.label }}</span>
        <span
          v-if="unreadCountByCategory[category.id]"
          class="message-center-sidebar__unread-count"
          :aria-label="`${unreadCountByCategory[category.id]} 条未读`"
        >
          {{ unreadCountByCategory[category.id] }}
        </span>
      </button>
    </div>

    <div class="message-center-sidebar__footer">
      <button
        class="message-center-sidebar__settings"
        :class="{
          'message-center-sidebar__settings--active':
            activeView === 'settings' || activeView === 'recipient-management',
        }"
        type="button"
        @click="emit('openSettings')"
      >
        <el-icon><RiSettings3Fill /></el-icon>
        <span>通知设置</span>
      </button>
    </div>
  </aside>
</template>

<style scoped lang="scss">
.message-center-sidebar {
  display: flex;
  width: 244px;
  min-width: 244px;
  height: 100%;
  box-sizing: border-box;
  flex-direction: column;
  padding: var(--el-space-4xl) var(--el-space-2xl) var(--el-space-2xl);
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);

  &__group + &__group {
    margin-top: var(--el-space-xl);
  }

  &__group-label {
    margin: 0 var(--el-space-md) var(--el-space-lg);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 22px;
  }

  &__category,
  &__settings {
    display: flex;
    width: 100%;
    min-height: 48px;
    border: 0;
    align-items: center;
    gap: var(--el-space-lg);
    padding: 0 var(--el-space-lg);
    color: var(--el-text-color-primary);
    background: transparent;
    border-radius: var(--el-border-radius-large);
    cursor: pointer;
    font-size: var(--el-font-size-medium);
    line-height: 24px;
    text-align: left;
    transition:
      color 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      background: var(--el-color-primary-light-9);
      color: var(--el-color-primary);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }
  }

  &__category--active,
  &__settings--active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  // 分类按钮保持独立卡片感，避免相邻选中态视觉粘连。
  &__category + &__category {
    margin-top: var(--el-space-xs);
  }

  &__category-icon {
    color: var(--el-text-color-placeholder);
    font-size: var(--el-font-size-extra-large);
  }

  &__category--active &__category-icon,
  &__settings--active .el-icon {
    color: var(--el-color-primary);
  }

  &__unread-count {
    display: inline-flex;
    min-width: 18px;
    height: 18px;
    margin-left: auto;
    align-items: center;
    justify-content: center;
    padding: 0 var(--el-space-xs);
    color: var(--el-color-white);
    background: var(--el-color-danger);
    border-radius: var(--el-border-radius-large);
    font-size: var(--el-font-size-extra-small);
    line-height: 18px;
  }

  &__footer {
    margin-top: auto;
    padding-top: var(--el-space-2xl);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  &__settings {
    gap: var(--el-space-lg);

    .el-icon {
      color: var(--el-text-color-secondary);
      font-size: var(--el-font-size-extra-large);
    }
  }
}

@media (max-width: 760px) {
  .message-center-sidebar {
    width: 64px;
    min-width: 64px;
    padding: var(--el-space-xl) var(--el-space-md) var(--el-space-lg);

    &__group-label,
    &__category > span:not(.message-center-sidebar__unread-count),
    &__settings > span {
      display: none;
    }

    &__category,
    &__settings {
      justify-content: center;
      padding: 0;
    }

    &__unread-count {
      position: absolute;
      right: 3px;
      top: 5px;
      min-width: 14px;
      height: 14px;
      padding: 0 var(--el-space-xs);
      font-size: var(--el-font-size-extra-small);
      line-height: 14px;
    }

    &__category {
      position: relative;
    }
  }
}
</style>
