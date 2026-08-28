<script setup lang="ts">
import type { MessageCategoryId } from './messageCenter.types';
import type { NotificationCategorySetting } from '~/api/notificationSettings';
import { computed } from 'vue';

defineOptions({ name: 'MessageSettingsTabs' });

const props = defineProps<{
  activeCategoryId: MessageCategoryId;
  /** 服务端设置聚合的分类目录（configurable=false 的分类不出现在设置页） */
  categories: NotificationCategorySetting[];
}>();

const emit = defineEmits<{
  select: [categoryId: MessageCategoryId];
}>();

/** 只显示服务端标记为可配置的分类（数据提醒/文档动态等无事件注册的分类自动排除）。 */
const configurableCategories = computed(() => props.categories.filter((item) => item.configurable));

function select(categoryId: string) {
  // 目录 id 即稳定分类码（八个分类之一），收敛到视图层强类型
  emit('select', categoryId as MessageCategoryId);
}
</script>

<template>
  <div class="message-settings-tabs" role="tablist" aria-label="通知类型">
    <button
      v-for="category in configurableCategories"
      :key="category.id"
      class="message-settings-tabs__tab"
      :class="{ 'message-settings-tabs__tab--active': activeCategoryId === category.id }"
      type="button"
      role="tab"
      :aria-selected="activeCategoryId === category.id"
      @click="select(category.id)"
    >
      {{ category.label }}
    </button>
  </div>
</template>

<style scoped lang="scss">
.message-settings-tabs {
  display: flex;
  overflow-x: auto;
  overflow-y: hidden;
  min-height: 54px;
  flex: 0 0 54px;
  align-items: flex-start;
  gap: var(--el-space-4xl);
  border-bottom: 1px solid var(--el-border-color-light);

  &__tab {
    position: relative;
    display: inline-flex;
    height: 54px;
    border: 0;
    flex: 0 0 auto;
    align-items: center;
    padding: 0;
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
    font-size: var(--el-font-size-medium);
    line-height: 26px;
    white-space: nowrap;

    &:hover {
      color: var(--el-color-primary);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -3px;
    }
  }

  &__tab--active {
    color: var(--el-color-primary);

    &::after {
      position: absolute;
      right: 0;
      bottom: -1px;
      left: 0;
      height: 3px;
      background: var(--el-color-primary);
      content: '';
    }
  }
}
</style>
