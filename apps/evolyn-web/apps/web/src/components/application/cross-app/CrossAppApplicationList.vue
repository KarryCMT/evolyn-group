<script setup lang="ts">
import type { CrossAppSourceApplication } from './crossApp.types';
import { RiApps2Fill } from '@remixicon/vue';
import { computed } from 'vue';

defineOptions({ name: 'CrossAppApplicationList' });

const props = defineProps<{
  applications: CrossAppSourceApplication[];
  activeApplicationId: string;
  selectedFormIds: string[];
}>();

const emit = defineEmits<{
  select: [id: string];
}>();

const selectedIdSet = computed(() => new Set(props.selectedFormIds));

function selectedCount(application: CrossAppSourceApplication) {
  return application.groups.reduce(
    (total, group) => total + group.forms.filter((form) => selectedIdSet.value.has(form.id)).length,
    0,
  );
}
</script>

<template>
  <!-- Element Plus 负责该栏滚动，避免原生滚动条与外层高度竞争。 -->
  <el-scrollbar class="cross-app-application-list" always>
    <nav class="cross-app-application-list__content" aria-label="可调用应用">
      <button
        v-for="application in props.applications"
        :key="application.id"
        class="cross-app-application-list__item"
        :class="{
          'cross-app-application-list__item--active': application.id === props.activeApplicationId,
          [`cross-app-application-list__item--${application.tone}`]: true,
        }"
        type="button"
        @click="emit('select', application.id)"
      >
        <span class="cross-app-application-list__icon" aria-hidden="true"><RiApps2Fill /></span>
        <span class="cross-app-application-list__name">{{ application.name }}</span>
        <span v-if="selectedCount(application)" class="cross-app-application-list__count">
          {{ selectedCount(application) }}
        </span>
      </button>
    </nav>
  </el-scrollbar>
</template>

<style scoped lang="scss">
.cross-app-application-list {
  // 左栏位于 Grid 单元格中，使用 100% 保持自身高度，避免 flex 基准值为 0 时内容被压缩。
  width: 100%;
  height: 100%;
  min-height: 0;
  min-width: 0;
  flex: none;

  &__content {
    display: flex;
    box-sizing: border-box;
    min-height: 100%;
    padding: var(--el-space-md) var(--el-space-lg);
    flex-direction: column;
    gap: var(--el-space-xs);
  }

  &__item {
    display: flex;
    min-width: 0;
    height: 38px;
    padding: 0 var(--el-space-md);
    border: 0;
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    font: inherit;
    text-align: left;
    transition:
      background-color 0.16s ease,
      color 0.16s ease;

    &:hover {
      background: var(--el-fill-color-light);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }

    &--active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);

      &:hover {
        background: var(--el-color-primary-light-8);
      }
    }
  }

  &__icon {
    display: inline-flex;
    width: 22px;
    height: 22px;
    border-radius: var(--el-border-radius-base);
    align-items: center;
    justify-content: center;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-8);

    svg {
      width: 14px;
      height: 14px;
    }
  }

  &__item--success &__icon {
    color: var(--el-color-success);
    background: var(--el-color-success-light-8);
  }

  &__item--warning &__icon {
    color: var(--el-color-warning);
    background: var(--el-color-warning-light-8);
  }

  &__item--danger &__icon {
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-8);
  }

  &__item--info &__icon {
    color: var(--el-color-info);
    background: var(--el-color-info-light-8);
  }

  &__name {
    overflow: hidden;
    flex: 1;
    font-size: var(--el-font-size-base);
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__count {
    min-width: 18px;
    padding: 0 var(--el-space-xs);
    border-radius: var(--el-border-radius-large);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-8);
    font-size: var(--el-font-size-extra-small);
    line-height: 18px;
    text-align: center;
  }
}
</style>
