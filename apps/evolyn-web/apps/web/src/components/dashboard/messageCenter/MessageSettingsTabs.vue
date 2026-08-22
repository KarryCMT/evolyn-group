<script setup lang="ts">
import type { MessageCategoryId } from './messageCenter.types';
import { computed } from 'vue';
import { messageCategories } from './messageCenter.constants';

defineOptions({ name: 'MessageSettingsTabs' });

defineProps<{
  activeCategoryId: MessageCategoryId;
}>();

const emit = defineEmits<{
  select: [categoryId: MessageCategoryId];
}>();

/** 数据提醒与文档动态暂无专属通知策略，设置页仅显示可配置的企业消息类目。 */
const configurableCategories = computed(() =>
  messageCategories.filter(
    (item) => item.id !== 'data-reminder' && item.id !== 'document-activity',
  ),
);
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
      @click="emit('select', category.id)"
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
  gap: 34px;
  border-bottom: 1px solid #dfe4ea;

  &__tab {
    position: relative;
    display: inline-flex;
    height: 54px;
    border: 0;
    flex: 0 0 auto;
    align-items: center;
    padding: 0;
    color: #202938;
    background: transparent;
    cursor: pointer;
    font-size: 17px;
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
