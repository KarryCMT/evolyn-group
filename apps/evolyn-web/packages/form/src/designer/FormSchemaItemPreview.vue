<template>
  <div class="form-schema-item-preview" :data-widget-type="widget.type">
    <div
      v-if="descriptionHtml && widget.type !== 'separator'"
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
      <span v-if="separatorDescriptionHtml" v-html="separatorDescriptionHtml" />
      <span v-else-if="separatorContent">{{ separatorContent }}</span>
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
    <!-- 流水号由服务端在成功提交时写入，设计态只展示不可填写的占位。 -->
    <el-input v-else-if="widget.type === 'sn'" disabled placeholder="自动生成无需填写" />
    <!-- 单选组/复选组直接呈现对应的选择控件，而不是复用下拉框外观。 -->
    <el-radio-group
      v-else-if="widget.type === 'radiogroup'"
      class="form-schema-item-preview__choices"
      :class="{ 'form-schema-item-preview__choices--vertical': choiceLayout === 'vertical' }"
      disabled
    >
      <el-radio v-for="option in choiceOptions" :key="option.value" :value="option.value">
        {{ option.label }}
      </el-radio>
    </el-radio-group>
    <el-checkbox-group
      v-else-if="widget.type === 'checkboxgroup'"
      class="form-schema-item-preview__choices"
      :class="{ 'form-schema-item-preview__choices--vertical': choiceLayout === 'vertical' }"
      disabled
    >
      <el-checkbox v-for="option in choiceOptions" :key="option.value" :value="option.value">
        {{ option.label }}
      </el-checkbox>
    </el-checkbox-group>
    <!-- 下拉框/下拉多选框保留下拉式预览。 -->
    <el-select v-else-if="widget.type === 'combo'" disabled :placeholder="placeholderText" />
    <el-select
      v-else-if="widget.type === 'combocheck'"
      multiple
      collapse-tags
      disabled
      :placeholder="placeholderText"
    />
    <!-- 成员字段的画布预览与最终填写形态一致，但不在设计画布内触发成员目录弹窗。 -->
    <button
      v-else-if="widget.type === 'user' || widget.type === 'usergroup'"
      class="form-schema-item-preview__member"
      :class="{ 'form-schema-item-preview__member--multiple': widget.type === 'usergroup' }"
      type="button"
      disabled
    >
      <span class="form-schema-item-preview__member-placeholder">＋ 选择成员</span>
    </button>
    <!-- 部门字段与运行时使用同一「选择部门」视觉语言；设计画布不触发组织树弹窗。 -->
    <button
      v-else-if="widget.type === 'dept' || widget.type === 'deptgroup'"
      class="form-schema-item-preview__member"
      :class="{
        'form-schema-item-preview__member--department-multiple': widget.type === 'deptgroup',
      }"
      type="button"
      disabled
    >
      <span class="form-schema-item-preview__member-placeholder">＋ 选择部门</span>
    </button>
    <!-- 其余控件（P3+ 分组）：统一占位预览 -->
    <el-input v-else disabled :placeholder="`${widgetLabel}（随后续版本开放）`" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  ElCheckbox,
  ElCheckboxGroup,
  ElDivider,
  ElInput,
  ElInputNumber,
  ElRadio,
  ElRadioGroup,
  ElSelect,
} from 'element-plus';
import type { FormItem } from '../schema/types';
import { readWidgetOptions } from '../schema/codec';
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
  widget.value.type === 'separator'
    ? (widget.value.borderStyle ?? widget.value.style ?? 'solid')
    : 'solid',
);
const separatorContentPosition = computed(() =>
  widget.value.type === 'separator' ? (widget.value.contentPosition ?? 'center') : 'center',
);
const separatorContent = computed(() =>
  widget.value.type === 'separator' ? (widget.value.content ?? '') : '',
);
const separatorDescriptionHtml = computed(() =>
  widget.value.type === 'separator' ? descriptionHtml.value : '',
);
const choiceOptions = computed(() => readWidgetOptions(widget.value));
const choiceLayout = computed(() => {
  if (widget.value.type === 'radiogroup' || widget.value.type === 'checkboxgroup') {
    return widget.value.layout ?? 'horizontal';
  }
  return 'horizontal';
});
const placeholderText = computed(() => {
  const placeholder = (widget.value as { placeholder?: string }).placeholder;
  if (placeholder !== undefined) return placeholder;
  return widget.value.type === 'combo' || widget.value.type === 'combocheck' ? '请选择' : '请输入';
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
    // Element Plus 的 .el-input-number 默认宽度为 150px；画布预览须与文本输入
    // 使用同一字段卡片宽度，避免设计态和填写态的视觉尺度不一致。
    width: 100% !important;
  }

  &__choices {
    display: flex;
    flex-flow: row wrap;
    gap: var(--el-space-sm) var(--el-space-lg);
  }

  &__choices--vertical {
    flex-direction: column;
    align-items: flex-start;

    :deep(.el-radio),
    :deep(.el-checkbox) {
      margin-right: 0;
    }
  }

  &__member {
    display: flex;
    width: 100%;
    min-height: 32px;
    padding: var(--el-space-xs) var(--el-space-sm);
    align-items: center;
    justify-content: center;
    border: 1px dashed var(--el-border-color);
    border-radius: var(--el-border-radius-small);
    color: var(--el-text-color-regular);
    background: var(--el-bg-color);
    font: inherit;
    font-size: var(--el-font-size-small);
    opacity: 1;

    &--multiple {
      min-height: 62px;
    }

    // 部门多选与运行时统一限制为 68px；标签超出时在字段内部滚动，避免设计画布
    // 因单个字段被撑高而影响相邻字段的布局。
    &--department-multiple {
      box-sizing: border-box;
      height: 68px;
      min-height: 68px;
      max-height: 68px;
      overflow-x: hidden;
      overflow-y: auto;
    }
  }
}
</style>
