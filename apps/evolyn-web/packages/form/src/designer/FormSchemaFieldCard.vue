<script setup lang="ts">
import { CopyDocument, Delete } from '@element-plus/icons-vue';
import { ElIcon, ElPopconfirm } from 'element-plus';
import { computed } from 'vue';
import type { FormItem } from '../schema/types';
import { isLayoutWidgetType } from '../schema/codec';
import FormSchemaItemPreview from './FormSchemaItemPreview.vue';

/** 设计器字段卡片：纯展示组件，字段操作全部通过事件交还编辑状态层。 */
const props = defineProps<{ item: FormItem; selected: boolean }>();

defineEmits<{
  select: [key: string];
  copy: [item: FormItem];
  remove: [key: string];
}>();

const hasLabel = computed(() => Boolean(props.item.label.trim()));
const showLabel = computed(() => !props.item.labelHidden && hasLabel.value);
const isLayoutItem = computed(() => isLayoutWidgetType(props.item.widget.type));
// 整行数据字段只收窄设计器中的卡片预览，保留栅格占位，避免后续字段回填到同一行。
// 标签页、子表单、分割线和富文本均为宽内容，始终占满各自容器。
const isCompactFullRow = computed(
  () =>
    props.item.lineWidth === 12 &&
    !isLayoutItem.value &&
    props.item.widget.type !== 'subform' &&
    props.item.widget.type !== 'richtext',
);
</script>

<template>
  <article
    class="form-schema-field-card"
    :class="{
      'is-active': selected,
      'is-label-empty': !showLabel,
      'is-layout': isLayoutItem,
      'is-compact-full-row': isCompactFullRow,
    }"
    @click="$emit('select', item.widget.widgetName)"
  >
    <header class="form-schema-field-card__header">
      <span v-if="showLabel" class="form-schema-field-card__label">
        {{ item.label }}
        <span v-if="!item.widget.allowBlank" class="form-schema-field-card__required">*</span>
      </span>
      <div class="form-schema-field-card__actions">
        <button type="button" title="复制字段" @click.stop="$emit('copy', item)">
          <el-icon><CopyDocument /></el-icon>
        </button>
        <el-popconfirm
          placement="bottom-end"
          :width="220"
          hide-icon
          confirm-button-type="danger"
          title="确定删除此项？删除后不可恢复，请确保你的工作不受影响"
          cancel-button-text="取消"
          confirm-button-text="删除"
          @confirm="$emit('remove', item.widget.widgetName)"
        >
          <template #reference>
            <button type="button" title="删除字段" @click.stop>
              <el-icon><Delete /></el-icon>
            </button>
          </template>
        </el-popconfirm>
      </div>
    </header>
    <div class="form-schema-field-card__control">
      <FormSchemaItemPreview :item="item" />
    </div>
  </article>
</template>

<style scoped lang="scss">
.form-schema-field-card {
  position: relative;
  box-sizing: border-box;
  width: 100%;
  padding: var(--el-space-md) var(--el-space-xl);
  margin-bottom: 0;
  cursor: grab;
  border: 1px solid transparent;
  border-radius: var(--el-border-radius-base);
  transition:
    background-color 0.16s ease,
    border-color 0.16s ease;

  &:hover {
    background-color: var(--el-fill-color-light);
  }
  &.is-active {
    background-color: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-9);
  }
  &.is-compact-full-row {
    width: 354px;
    max-width: 100%;
  }
  &:hover &__actions,
  &.is-active &__actions {
    pointer-events: auto;
    opacity: 1;
  }

  &__header {
    display: flex;
    gap: var(--el-space-xs);
    align-items: center;
    min-height: 24px;
    padding-right: 64px;
    margin-bottom: var(--el-space-xs);
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
  }
  &__label {
    flex-shrink: 0;
  }
  &__required {
    margin-left: var(--el-space-xs);
    color: var(--el-color-error);
  }
  &__control {
    pointer-events: none;
  }
  &__actions {
    position: absolute;
    top: var(--el-space-xs);
    right: var(--el-space-xl);
    display: flex;
    overflow: hidden;
    pointer-events: none;
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-base);
    box-shadow: var(--el-box-shadow);
    opacity: 0;
  }
  &__actions button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    border: 0;
  }
}
</style>
