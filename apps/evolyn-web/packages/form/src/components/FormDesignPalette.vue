<template>
  <aside class="form-design-palette">
    <Draggable
      :list="fields"
      :group="{ name: 'plugin-design-fields', pull: 'clone', put: false }"
      :sort="false"
      :clone="clonePaletteItem"
      item-key="widgetName"
      class="form-design-palette__list"
    >
      <template #item="{ element }">
        <button
          class="form-design-palette__item"
          type="button"
          :data-widget-name="element.widgetName"
          @click="$emit('add-field', element)"
        >
          <el-icon><component :is="element.icon" /></el-icon>
          <span>{{ element.label }}</span>
        </button>
      </template>
    </Draggable>
  </aside>
</template>

<script setup lang="ts">
import { ElIcon } from 'element-plus';
import Draggable from 'vuedraggable';
import type { FormDesignPaletteItem } from '../types';

defineProps<{
  fields: FormDesignPaletteItem[];
}>();

defineEmits<{
  (event: 'add-field', value: FormDesignPaletteItem): void;
}>();

// 拖入画布时先放入带临时 fieldKey 的克隆对象，父组件随后替换成插件中心自己的字段模型。
const clonePaletteItem = (item: FormDesignPaletteItem) => ({
  ...item,
  fieldKey: `palette_${item.widgetName}_${Date.now()}`,
});
</script>

<style lang="scss">
.form-design-palette {
  box-sizing: border-box;
  flex-shrink: 0;
  // 收窄控件面板及横向留白，为中间画布保留更多设计空间。
  width: 152px;
  padding: var(--el-space-xl) var(--gp-space-lg);
  background-color: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
  border-bottom: 1px solid var(--el-border-color);
  border-left: 1px solid var(--el-border-color);
  border-top-left-radius: var(--gp-radius-md);
  border-bottom-left-radius: var(--gp-radius-md);

  &__list {
    min-height: 100%;
  }

  &__item {
    display: flex;
    gap: var(--gp-space-sm);
    align-items: center;
    justify-content: flex-start;
    width: 100%;
    min-height: 34px;
    padding: 0 var(--gp-space-xl);
    margin-bottom: var(--gp-space-lg);
    font-size: var(--el-font-size-base);
    color: var(--el-text-color-primary);
    cursor: pointer;
    background-color: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--gp-radius-md);
    transition:
      background-color 0.16s ease,
      border-color 0.16s ease;

    &:hover {
      color: var(--el-color-primary);
      cursor: move;
      background-color: var(--el-color-primary-light-1);
      border-color: var(--el-color-primary-light-3);

      .el-icon {
        color: var(--el-color-primary);
      }
    }

    .el-icon {
      flex-shrink: 0;
      font-size: var(--el-font-size-medium);
      color: var(--el-text-color-regular);
    }
  }
}
</style>
