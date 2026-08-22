<script setup lang="ts">
import { ArrowDown } from '@element-plus/icons-vue';

defineOptions({ name: 'MessageToolbar' });

defineProps<{
  showUnreadOnly: boolean;
}>();

const emit = defineEmits<{
  'update:showUnreadOnly': [value: boolean];
  markAllAsRead: [];
}>();

/** Element Plus 的复选框支持多种值类型；本开关只保留严格布尔状态。 */
function updateUnreadOnly(value: boolean | string | number) {
  emit('update:showUnreadOnly', value === true);
}
</script>

<template>
  <div class="message-toolbar">
    <el-dropdown trigger="click">
      <button class="message-toolbar__filter" type="button">
        <span>全部</span>
        <el-icon><ArrowDown /></el-icon>
      </button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item>全部</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <div class="message-toolbar__actions">
      <el-checkbox :model-value="showUnreadOnly" @update:model-value="updateUnreadOnly($event)">
        只看未读
      </el-checkbox>
      <button class="message-toolbar__mark-read" type="button" @click="emit('markAllAsRead')">
        全部转为已读
      </button>
    </div>
  </div>
</template>

<style scoped lang="scss">
.message-toolbar {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;

  &__filter,
  &__mark-read {
    display: inline-flex;
    border: 0;
    align-items: center;
    gap: 6px;
    color: #202938;
    background: transparent;
    cursor: pointer;
    font-size: 18px;
    line-height: 26px;

    &:hover {
      color: #00aaa7;
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 3px;
    }
  }

  &__filter .el-icon {
    color: #626d7d;
    font-size: 18px;
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: 28px;
    color: #202938;
    font-size: 16px;
  }

  &__mark-read {
    color: #00aaa7;
    font-size: 16px;
  }
}

@media (max-width: 600px) {
  .message-toolbar {
    &__actions {
      gap: 12px;
    }
  }
}
</style>
