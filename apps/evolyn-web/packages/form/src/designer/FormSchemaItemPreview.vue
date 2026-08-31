<template>
  <div class="form-schema-item-preview" :data-widget-type="widget.type">
    <div
      v-if="descriptionHtml"
      class="form-schema-item-preview__description"
      v-html="descriptionHtml"
    />
    <!-- 分割线：与运行时一致的整行样式预览 -->
    <el-divider
      v-if="widget.type === 'separator'"
      class="form-schema-item-preview__divider"
      :direction="separatorDirection"
      :border-style="separatorBorderStyle"
      :content-position="separatorContentPosition"
    >
      <span v-if="separatorContent">{{ separatorContent }}</span>
    </el-divider>
    <!-- 文本/多行文本 -->
    <el-input
      v-else-if="widget.type === 'textarea'"
      type="textarea"
      :rows="1"
      disabled
      :placeholder="placeholderText"
    />
    <el-input v-else-if="widget.type === 'text'" disabled :placeholder="placeholderText" />
    <!-- 数字 -->
    <el-input-number
      v-else-if="widget.type === 'number'"
      class="form-schema-item-preview__number"
      disabled
      :placeholder="placeholderText"
      :controls="false"
    />
    <!-- 日期时间：按 format 提示输入形态 -->
    <el-input v-else-if="widget.type === 'datetime'" disabled :placeholder="datePlaceholder" />
    <!-- 选项类：禁用态下拉预览（单选/多选） -->
    <el-select
      v-else-if="widget.type === 'combo' || widget.type === 'radiogroup'"
      disabled
      :placeholder="placeholderText"
    />
    <el-select
      v-else-if="widget.type === 'combocheck' || widget.type === 'checkboxgroup'"
      multiple
      collapse-tags
      disabled
      :placeholder="placeholderText"
    />
    <!-- 其余控件（P3+ 分组）：统一占位预览 -->
    <el-input v-else disabled :placeholder="`${widgetLabel}（随后续版本开放）`" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ElDivider, ElInput, ElInputNumber, ElSelect } from 'element-plus';
import type { FormItem } from '../schema/types';
import { widgetTypeLabel } from '../schema/dictionary';
import { sanitizeRichTextDescription } from '../schema/richTextDescription';

/** 画布字段预览控件：按 widget.type 渲染禁用态形态，仅供设计参考，不承载值。 */
const props = defineProps<{ item: FormItem }>();

const widget = computed(() => props.item.widget);
const widgetLabel = computed(() => widgetTypeLabel(widget.value.type));
const descriptionHtml = computed(() => sanitizeRichTextDescription(props.item.description));
const separatorDirection = computed(() =>
  widget.value.type === 'separator' ? (widget.value.direction ?? 'horizontal') : 'horizontal',
);
const separatorBorderStyle = computed(() =>
  widget.value.type === 'separator' ? (widget.value.borderStyle ?? widget.value.style ?? 'solid') : 'solid',
);
const separatorContentPosition = computed(() =>
  widget.value.type === 'separator' ? (widget.value.contentPosition ?? 'center') : 'center',
);
const separatorContent = computed(() =>
  widget.value.type === 'separator' ? (widget.value.content ?? '') : '',
);
const placeholderText = computed(() => {
  const placeholder = (widget.value as { placeholder?: string }).placeholder;
  return placeholder || '请输入';
});
const datePlaceholder = computed(() => {
  switch ((widget.value as { format?: string }).format) {
    case 'date':
      return '请选择日期';
    case 'month':
      return '请选择月份';
    case 'time':
      return '请选择时间';
    default:
      return '请选择日期时间';
  }
});
</script>

<style lang="scss">
.form-schema-item-preview {
  width: 100%;

  &__divider {
    margin: var(--el-space-xs) 0;
  }

  &__description {
    margin-bottom: var(--el-space-sm);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-extra-small);
    line-height: 1.5;

    :deep(p),
    :deep(ul),
    :deep(ol),
    :deep(blockquote) {
      margin: 0 0 var(--el-space-xs);
    }

    :deep(ul),
    :deep(ol) {
      padding-left: var(--el-space-3xl);
    }

    :deep(a) {
      color: var(--el-color-primary);
      text-decoration: underline;
    }

    :deep(img) {
      display: block;
      max-width: 100%;
      height: auto;
      margin: var(--el-space-xs) 0;
    }
  }

  &__number {
    width: 100%;
  }
}
</style>
