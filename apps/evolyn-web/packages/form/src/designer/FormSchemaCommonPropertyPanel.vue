<script setup lang="ts">
import { EvolynRichTextEditor } from '@evolyn.do/ui';
import { ElCheckbox, ElFormItem, ElInput, ElSwitch } from 'element-plus';
import type { FormItem } from '../schema/types';
import { FORM_FIELD_WIDTH_OPTIONS } from '../schema/dictionary';

/**
 * 所有字段共享的属性区。字段类型差异只在外层面板的专属区出现，避免标题、说明、
 * 权限等基础属性随控件类型变换位置或使用不同控件。
 */
const model = defineModel<FormItem>({ required: true });
const props = withDefaults(
  defineProps<{
    isSeparator?: boolean;
    showWidth?: boolean;
  }>(),
  { isSeparator: false, showWidth: true },
);
const emit = defineEmits<{ renameKey: [key: string] }>();
</script>

<template>
  <section class="form-schema-common-property" aria-label="通用属性">
    <el-form-item label="标题">
      <el-input
        v-model="model.label"
        :maxlength="64"
        placeholder="请输入标题"
      />
    </el-form-item>
    <el-checkbox
      :model-value="!model.labelHidden"
      class="form-schema-common-property__show-title"
      @update:model-value="model.labelHidden = !$event"
    >
      显示标题
    </el-checkbox>
    <el-form-item label="字段键（widgetName）">
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
</template>

<style scoped lang="scss">
.form-schema-common-property {
  &__show-title {
    margin: calc(var(--el-space-sm) * -1) 0 var(--el-space-lg);
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
