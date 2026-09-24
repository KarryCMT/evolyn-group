<script setup lang="ts">
import type { AssistantListItem } from './assistant.types';
import {
  RiDeleteBinLine,
  RiFileCopyLine,
  RiHistoryLine,
  RiPencilLine,
  RiPlayCircleLine,
  RiStopCircleLine,
} from '@remixicon/vue';

defineOptions({ name: 'AssistantList' });

defineProps<{
  items: AssistantListItem[];
}>();

const emit = defineEmits<{
  edit: [item: AssistantListItem];
  copy: [item: AssistantListItem];
  remove: [item: AssistantListItem];
  toggle: [item: AssistantListItem];
}>();

function triggerLabel(item: AssistantListItem): string {
  if (item.triggerType === 'schedule') return '定时触发';
  if (item.triggerType === 'http') return 'HTTP 触发';
  return `表单触发 · ${item.triggerFormName ?? '未选择表单'}`;
}
</script>

<template>
  <div class="assistant-list">
    <article v-for="item in items" :key="item.id" class="assistant-list__card">
      <div class="assistant-list__content">
        <div class="assistant-list__heading">
          <h3>{{ item.name }}</h3>
          <span :class="{ 'is-enabled': item.enabled }">{{ item.enabled ? '已启用' : '未启用' }}</span>
        </div>
        <p>{{ triggerLabel(item) }}</p>
        <div class="assistant-list__tags">
          <span v-for="tag in item.tags" :key="tag">{{ tag }}</span>
          <small>更新于 {{ item.updatedAt }}</small>
        </div>
      </div>
      <div class="assistant-list__actions">
        <button type="button" @click="emit('edit', item)">
          <RiPencilLine />编辑
        </button>
        <button type="button" @click="emit('toggle', item)">
          <RiStopCircleLine v-if="item.enabled" />
          <RiPlayCircleLine v-else />
          {{ item.enabled ? '停用' : '启用' }}
        </button>
        <button type="button" disabled>
          <RiHistoryLine />执行日志
        </button>
        <button type="button" @click="emit('copy', item)">
          <RiFileCopyLine />复制
        </button>
        <button type="button" class="is-danger" @click="emit('remove', item)">
          <RiDeleteBinLine />删除
        </button>
      </div>
    </article>
  </div>
</template>

<style scoped lang="scss">
.assistant-list {
  display: grid;
  gap: 14px;

  &__card {
    display: flex;
    min-height: 116px;
    padding: 22px 24px;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
    background: #fff;
    border: 1px solid #e0e5ec;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgb(20 35 55 / 4%);
  }

  &__content {
    min-width: 0;

    p {
      margin: 8px 0 10px;
      color: #5e697a;
      font-size: 14px;
    }
  }

  &__heading {
    display: flex;
    align-items: center;
    gap: 10px;

    h3 {
      margin: 0;
      color: #172033;
      font-size: 17px;
    }

    span {
      padding: 2px 8px;
      color: #8b6a1d;
      background: #fff4d7;
      border-radius: 4px;
      font-size: 12px;

      &.is-enabled {
        color: #16824b;
        background: #e8f8ef;
      }
    }
  }

  &__tags {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;

    span {
      padding: 2px 8px;
      color: #536074;
      background: #eef1f5;
      border-radius: 4px;
      font-size: 12px;
    }

    small {
      color: #9aa3af;
    }
  }

  &__actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 6px;

    button {
      display: inline-flex;
      height: 34px;
      padding: 0 10px;
      align-items: center;
      gap: 5px;
      color: #455166;
      background: transparent;
      border: 1px solid transparent;
      border-radius: 6px;
      cursor: pointer;

      &:hover:not(:disabled) {
        color: #00a99d;
        background: #ecf9f7;
      }

      &:disabled {
        cursor: not-allowed;
        opacity: 0.45;
      }

      &.is-danger:hover {
        color: #e5484d;
        background: #fff0f0;
      }

      svg {
        width: 17px;
        height: 17px;
      }
    }
  }
}

@media (max-width: 900px) {
  .assistant-list {
    &__card {
      align-items: flex-start;
      flex-direction: column;
    }

    &__actions {
      justify-content: flex-start;
    }
  }
}
</style>
