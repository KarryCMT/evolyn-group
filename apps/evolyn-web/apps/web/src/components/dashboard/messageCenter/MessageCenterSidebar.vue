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
  padding: 34px 20px 20px;
  background: #fff;
  border-right: 1px solid #edf0f3;

  &__group + &__group {
    margin-top: 18px;
  }

  &__group-label {
    margin: 0 10px 12px;
    color: #9299a5;
    font-size: 15px;
    line-height: 22px;
  }

  &__category,
  &__settings {
    display: flex;
    width: 100%;
    min-height: 48px;
    border: 0;
    align-items: center;
    gap: 13px;
    padding: 0 12px;
    color: #202938;
    background: transparent;
    border-radius: 10px;
    cursor: pointer;
    font-size: 17px;
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
    margin-top: 4px;
  }

  &__category-icon {
    color: #9ba4b3;
    font-size: 22px;
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
    padding: 0 5px;
    color: #fff;
    background: #f15b5f;
    border-radius: 9px;
    font-size: 11px;
    line-height: 18px;
  }

  &__footer {
    margin-top: auto;
    padding-top: 20px;
    border-top: 1px solid #edf0f3;
  }

  &__settings {
    gap: 12px;

    .el-icon {
      color: #8f99a8;
      font-size: 21px;
    }
  }
}

@media (max-width: 760px) {
  .message-center-sidebar {
    width: 64px;
    min-width: 64px;
    padding: 18px 8px 12px;

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
      padding: 0 2px;
      font-size: 9px;
      line-height: 14px;
    }

    &__category {
      position: relative;
    }
  }
}
</style>
