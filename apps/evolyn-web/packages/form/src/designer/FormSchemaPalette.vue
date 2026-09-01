<template>
  <EvolynScrollbar class="form-schema-palette" aria-label="字段素材面板">
    <section v-for="group in groups" :key="group.key" class="form-schema-palette__group">
      <p class="form-schema-palette__group-title">{{ group.title }}</p>
      <Draggable
        :list="group.entries"
        :group="{ name: FORM_SCHEMA_DRAG_GROUP, pull: 'clone', put: false }"
        :sort="false"
        :clone="clonePaletteItem"
        item-key="type"
        class="form-schema-palette__list"
        chosen-class="form-schema-palette__chosen"
        fallback-class="form-schema-palette__fallback"
        :data-enabled="group.enabled"
        :move="createMoveGuard(group)"
      >
        <template #item="{ element }">
          <button
            class="form-schema-palette__item"
            type="button"
            :disabled="!entryEnabled(group, element)"
            :title="entryEnabled(group, element) ? element.label : '该字段随后续版本开放'"
            @click="$emit('add-field', element)"
          >
            <el-icon><component :is="element.icon" /></el-icon>
            <span>{{ element.label }}</span>
          </button>
        </template>
      </Draggable>
    </section>
  </EvolynScrollbar>
</template>

<script setup lang="ts">
import { EvolynScrollbar } from '@evolyn.do/ui';
import { ElIcon } from 'element-plus';
import Draggable from 'vuedraggable';
import { FORM_SCHEMA_DRAG_GROUP, type FormSchemaPaletteDrag } from './palette';

/**
 * 字段素材面板：按分组展示控件入口，支持点击添加与拖拽克隆到画布。
 * 拖拽时 clone 出仅含 paletteType 标记的临时对象，由画布 add 事件换成本页
 * 通过 createWidgetItem 生成的真实字段项（协议文档不落任何临时结构）。
 */
export interface FormSchemaPaletteGroup {
  key: string;
  title: string;
  /** 未开放的分组置灰只展示（后续阶段开放），本期仅基础字段可添加。 */
  enabled: boolean;
  entries: Array<{ type: string; label: string; icon: unknown; enabled?: boolean }>;
}

defineProps<{ groups: FormSchemaPaletteGroup[] }>();

defineEmits<{
  (event: 'add-field', value: { type: string; label: string; icon: unknown }): void;
}>();

// 临时对象以 paletteType 标记控件类型；画布 add 后立即被真实字段项替换。
const clonePaletteItem = (item: { type: string }): FormSchemaPaletteDrag => ({
  paletteType: item.type,
});

function entryEnabled(
  group: FormSchemaPaletteGroup,
  entry: FormSchemaPaletteGroup['entries'][number],
): boolean {
  return entry.enabled ?? group.enabled;
}

function canDrag(
  group: FormSchemaPaletteGroup,
  event: { draggedContext?: { element?: FormSchemaPaletteGroup['entries'][number] } },
): boolean {
  const entry = event.draggedContext?.element;
  return Boolean(entry && entryEnabled(group, entry));
}

/**
 * vuedraggable 将 move 声明为宽泛的 Function，模板内联回调会丢失事件参数类型。
 * 在脚本侧封装守卫，既保留分组上下文，也让拖拽事件具备明确的最小类型契约。
 */
function createMoveGuard(group: FormSchemaPaletteGroup) {
  return (event: Parameters<typeof canDrag>[1]): boolean => canDrag(group, event);
}
</script>

<style lang="scss">
.form-schema-palette {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: var(--el-space-lg);
  width: 264px;
  padding: var(--el-space-xl) var(--el-space-lg);
  background-color: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
  border-right: 1px solid var(--el-border-color);

  &__group-title {
    margin: 0 0 var(--el-space-md);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    color: var(--el-text-color-secondary);
  }

  &__list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
  }

  &__item {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    justify-content: flex-start;
    width: 100%;
    min-height: 32px;
    padding: 0 var(--el-space-lg);
    margin-bottom: 0;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-primary);
    cursor: pointer;
    background-color: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-medium);
    transition:
      background-color 0.16s ease,
      border-color 0.16s ease;

    &:hover:not(:disabled) {
      color: var(--el-color-primary);
      cursor: move;
      border-color: var(--el-color-primary);
      border-style: dashed;
      .el-icon {
        color: var(--el-color-primary);
      }
    }

    &:disabled {
      color: var(--el-text-color-disabled);
      cursor: not-allowed;
      background-color: var(--el-fill-color-lighter);
    }

    .el-icon {
      flex-shrink: 0;
      font-size: var(--el-font-size-medium);
      color: var(--el-text-color-regular);
    }
  }

  // 拖动源保持轻量选中态，跟随鼠标的拖动卡片固定为横向尺寸，避免窄列中标题折行。
  &__chosen:not(:disabled) {
    color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
    border-style: dashed;
  }

  &__fallback {
    box-sizing: border-box;
    display: flex !important;
    width: 200px !important;
    min-width: 200px;
    max-width: 200px;
    min-height: 40px;
    padding: 0 var(--el-space-lg) !important;
    overflow: hidden;
    color: var(--el-color-primary) !important;
    white-space: nowrap;
    background-color: var(--el-bg-color) !important;
    border-color: var(--el-color-primary) !important;
    border-style: dashed !important;
    box-shadow: var(--el-box-shadow-light);
    opacity: 0.92 !important;
  }

  &__fallback span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__fallback .el-icon {
    color: var(--el-color-primary) !important;
  }
}
</style>
