<script setup lang="ts">
import { EvolynRichTextEditor } from '@evolyn.do/ui';
import { RiQuestionFill } from '@remixicon/vue';
import { ElCheckbox, ElFormItem, ElIcon, ElInput, ElSwitch, ElTooltip } from 'element-plus';
import { computed } from 'vue';
import type { FormItem } from '../schema/types';
import { FORM_FIELD_WIDTH_OPTIONS, WIDGET_SPECS } from '../schema/dictionary';

/**
 * 所有字段共享的属性区。字段类型差异只在外层面板的专属区出现，避免标题、说明、
 * 权限等基础属性随控件类型变换位置或使用不同控件。
 */
const model = defineModel<FormItem>({ required: true });
const props = withDefaults(
  defineProps<{
    isSeparator?: boolean;
    showWidth?: boolean;
    /** reference 按字段属性标准分区重排；default 保持其他控件现有布局。 */
    arrangement?: 'default' | 'reference';
    showWidgetName?: boolean;
    titleRequired?: boolean;
  }>(),
  {
    isSeparator: false,
    showWidth: true,
    arrangement: 'default',
    showWidgetName: true,
    titleRequired: true,
  },
);
const emit = defineEmits<{ renameKey: [key: string] }>();

/** 有占位提示能力的字段统一由公共属性区维护，专属面板不重复声明该配置。 */
const supportsPromptText = computed(
  () => 'placeholder' in WIDGET_SPECS[model.value.widget.type].props,
);
/**
 * 画布预览与运行时都有默认占位文案；属性面板读取缺省值时也展示同一文案，
 * 避免画布显示「请输入」而“提示文字”输入框为空。
 */
const promptTextFallback = computed(() => {
  switch (model.value.widget.type) {
    case 'datetime':
      return '请选择日期时间';
    case 'combo':
    case 'combocheck':
      return '请选择';
    default:
      return '请输入';
  }
});
const promptText = computed({
  get: () => {
    const value = (model.value.widget as { placeholder?: unknown }).placeholder;
    return typeof value === 'string' ? value : promptTextFallback.value;
  },
  set: (value: string) => {
    const widget = model.value.widget as { placeholder?: string };
    if ('placeholder' in WIDGET_SPECS[model.value.widget.type].props) {
      widget.placeholder = value;
    }
  },
});
</script>

<template>
  <section
    v-if="props.arrangement === 'default'"
    class="form-schema-common-property"
    aria-label="通用属性"
  >
    <el-form-item label="标题">
      <el-input v-model="model.label" :maxlength="64" placeholder="请输入标题" />
    </el-form-item>
    <el-checkbox
      :model-value="!model.labelHidden"
      class="form-schema-common-property__show-title"
      @update:model-value="model.labelHidden = !$event"
    >
      显示标题
    </el-checkbox>
    <el-form-item v-if="props.showWidgetName" label="字段键（widgetName）">
      <el-input
        :model-value="model.widget.widgetName"
        placeholder="字段值与规则引用的稳定键"
        @update:model-value="emit('renameKey', String($event ?? ''))"
      />
    </el-form-item>
    <el-form-item label="描述信息">
      <EvolynRichTextEditor
        v-model="model.description"
        :min-height="96"
        aria-label="字段说明"
        toolbar-size="small"
      />
    </el-form-item>
    <div class="form-schema-common-property__switches">
      <div v-if="!props.isSeparator" class="form-schema-common-property__switch">
        <span>必填</span>
        <el-switch
          :model-value="!model.widget.allowBlank"
          @update:model-value="model.widget.allowBlank = !$event"
        />
      </div>
      <div class="form-schema-common-property__switch">
        <span>可填写</span>
        <el-switch v-model="model.widget.enable" />
      </div>
      <div class="form-schema-common-property__switch">
        <span>可见</span>
        <el-switch v-model="model.widget.visible" />
      </div>
    </div>
    <el-form-item v-if="props.showWidth && !props.isSeparator" label="字段宽度">
      <div class="form-schema-common-property__width-options" role="group" aria-label="字段宽度">
        <button
          v-for="(option, index) in FORM_FIELD_WIDTH_OPTIONS"
          :key="`${option.label}-${index}`"
          type="button"
          :class="{ 'is-active': model.lineWidth === option.value }"
          :aria-pressed="model.lineWidth === option.value"
          @click="model.lineWidth = option.value"
        >
          {{ option.label }}
        </button>
      </div>
    </el-form-item>
  </section>
  <section v-else class="form-schema-common-property form-schema-common-property--reference">
    <div class="form-schema-common-property__reference-title-row">
      <label
        class="form-schema-common-property__reference-label"
        :class="{ 'form-schema-common-property__reference-label--required': props.titleRequired }"
        for="reference-property-label"
      >
        标题
      </label>
      <slot name="title-suffix" />
    </div>
    <el-input
      id="reference-property-label"
      v-model="model.label"
      :maxlength="64"
      aria-label="标题"
      placeholder="请输入标题"
    />
    <el-checkbox
      :model-value="!model.labelHidden"
      class="form-schema-common-property__reference-show-title"
      @update:model-value="model.labelHidden = !$event"
    >
      显示标题
    </el-checkbox>

    <section class="form-schema-common-property__reference-section">
      <h3 class="form-schema-common-property__reference-heading">描述信息</h3>
      <EvolynRichTextEditor
        v-model="model.description"
        :min-height="96"
        aria-label="字段说明"
        toolbar-size="small"
      />
    </section>

    <section v-if="supportsPromptText" class="form-schema-common-property__reference-section">
      <h3 class="form-schema-common-property__reference-heading">
        提示文字
        <el-tooltip content="填写字段前展示的输入提示" placement="top">
          <el-icon class="form-schema-common-property__reference-help" aria-label="提示文字说明">
            <RiQuestionFill />
          </el-icon>
        </el-tooltip>
      </h3>
      <el-input v-model="promptText" :maxlength="100" aria-label="提示文字" />
    </section>

    <slot name="after-prompt" />

    <section v-if="!props.isSeparator" class="form-schema-common-property__reference-section">
      <h3 class="form-schema-common-property__reference-heading">校验</h3>
      <div class="form-schema-common-property__reference-checks">
        <el-checkbox
          :model-value="!model.widget.allowBlank"
          @update:model-value="model.widget.allowBlank = !$event"
        >
          必填
        </el-checkbox>
        <!-- 重复值校验需要服务端唯一索引，能力开放前公共栏位不可操作。 -->
        <el-checkbox disabled>不允许重复值</el-checkbox>
      </div>
    </section>

    <section class="form-schema-common-property__reference-section">
      <h3 class="form-schema-common-property__reference-heading">字段权限</h3>
      <div class="form-schema-common-property__reference-checks">
        <el-checkbox v-model="model.widget.visible">可见</el-checkbox>
        <el-checkbox v-model="model.widget.enable">可编辑</el-checkbox>
      </div>
    </section>

    <section
      v-if="props.showWidth && !props.isSeparator"
      class="form-schema-common-property__reference-section"
    >
      <h3 class="form-schema-common-property__reference-heading">字段宽度</h3>
      <div class="form-schema-common-property__width-options" role="group" aria-label="字段宽度">
        <button
          v-for="(option, index) in FORM_FIELD_WIDTH_OPTIONS"
          :key="`${option.label}-${index}`"
          type="button"
          :class="{ 'is-active': model.lineWidth === option.value }"
          :aria-pressed="model.lineWidth === option.value"
          @click="model.lineWidth = option.value"
        >
          {{ option.label }}
        </button>
      </div>
    </section>

    <slot name="after-width" />
  </section>
</template>

<style scoped lang="scss">
.form-schema-common-property {
  &--reference {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-lg);
    padding: var(--el-space-lg) var(--el-space-lg) var(--el-space-lg);
  }

  &__show-title {
    margin: calc(var(--el-space-sm) * -1) 0 var(--el-space-lg);
  }

  &__reference-title-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 104px;
    gap: var(--el-space-md);
    align-items: center;
    margin-bottom: calc(var(--el-space-md) * -1);
  }

  &__reference-label,
  &__reference-heading {
    margin: 0;
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 1.5;
    color: var(--el-text-color-primary);
  }

  &__reference-label--required::before {
    margin-right: 2px;
    color: var(--el-color-danger);
    content: '*';
  }

  &__reference-show-title {
    margin-top: calc(var(--el-space-md) * -1);
  }

  &__reference-section {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-md);
  }

  &__reference-checks {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-lg);
    align-items: flex-start;
  }

  &__reference-help {
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  &__switches {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
    margin-bottom: var(--el-space-lg);
  }

  &__switch {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 32px;
    padding: 0 var(--el-space-md);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__width-options {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    width: 100%;
    overflow: hidden;
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
  }

  &__width-options button {
    min-width: 0;
    height: 32px;
    padding: 0;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: var(--el-bg-color);
    border: 0;
    border-right: 1px solid var(--el-border-color);
  }

  &__width-options button:last-child {
    border-right: 0;
  }

  &__width-options button.is-active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}
</style>
