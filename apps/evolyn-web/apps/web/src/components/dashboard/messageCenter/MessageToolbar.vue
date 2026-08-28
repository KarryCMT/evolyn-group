<script setup lang="ts">
import { RiArrowDownSFill } from '@remixicon/vue';

defineOptions({ name: 'MessageToolbar' });

const props = defineProps<{
  showUnreadOnly: boolean;
  /** 事件筛选选项（服务端目录下发；为空时隐藏下拉，不显示占位项） */
  eventCode: string;
  eventOptions: { code: string; label: string }[];
}>();

const emit = defineEmits<{
  'update:showUnreadOnly': [value: boolean];
  'update:eventCode': [value: string];
  markAllAsRead: [];
}>();

/** Element Plus 的复选框支持多种值类型；本开关只保留严格布尔状态。 */
function updateUnreadOnly(value: boolean | string | number) {
  emit('update:showUnreadOnly', value === true);
}

function selectEvent(code: string) {
  emit('update:eventCode', code === props.eventCode ? '' : code);
}
</script>

<template>
  <div class="message-toolbar">
    <el-dropdown v-if="eventOptions.length" trigger="click" @command="selectEvent">
      <button class="message-toolbar__filter" type="button">
        <span>{{ eventOptions.find((option) => option.code === eventCode)?.label ?? '全部' }}</span>
        <el-icon><RiArrowDownSFill /></el-icon>
      </button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item :class="{ 'is-active': !eventCode }" command="">全部</el-dropdown-item>
          <el-dropdown-item
            v-for="option in eventOptions"
            :key="option.code"
            :class="{ 'is-active': option.code === eventCode }"
            :command="option.code"
          >
            {{ option.label }}
          </el-dropdown-item>
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
  gap: var(--el-space-xl);

  &__filter,
  &__mark-read {
    display: inline-flex;
    border: 0;
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
    font-size: var(--el-font-size-large);
    line-height: 26px;

    &:hover {
      color: var(--el-color-primary);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 3px;
    }
  }

  &__filter .el-icon {
    color: var(--el-text-color-regular);
    font-size: var(--el-font-size-large);
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: var(--el-space-3xl);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
  }

  &__mark-read {
    color: var(--el-color-primary);
    font-size: var(--el-font-size-medium);
  }
}

@media (max-width: 600px) {
  .message-toolbar {
    &__actions {
      gap: var(--el-space-lg);
    }
  }
}
</style>
