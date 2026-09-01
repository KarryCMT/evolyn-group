<script setup lang="ts">
import { ElCheckbox, ElInput, ElOption, ElSelect } from 'element-plus';
import { computed } from 'vue';
import type { FormItem, TextWidget } from '../../schema/types';
import FormSchemaCommonPropertyPanel from '../FormSchemaCommonPropertyPanel.vue';

/**
 * 单行文本专属属性面板只承载格式与默认值；标题、描述、提示文字、校验、权限和
 * 宽度仍由通用属性面板负责，以保证公共属性只有一个实现入口。
 */
const model = defineModel<FormItem>({ required: true });

const widget = computed(() => model.value.widget as TextWidget);
</script>

<template>
  <FormSchemaCommonPropertyPanel v-model="model" arrangement="reference" :show-widget-name="false">
    <template #title-suffix>
      <!-- 字段切换尚未进入协议，保留类型栏位以对齐属性面板布局。 -->
      <el-select class="text-property__select" model-value="text" disabled aria-label="字段类型">
        <el-option label="单行文本" value="text" />
      </el-select>
    </template>

    <template #after-prompt>
      <section class="text-property__section">
        <h3 class="text-property__heading">格式</h3>
        <el-select v-model="widget.format" aria-label="格式">
          <el-option label="无" value="" />
          <el-option label="邮箱" value="email" />
        </el-select>
      </section>

      <section class="text-property__section">
        <h3 class="text-property__heading">默认值</h3>
        <!-- 当前仅保存固定默认值；联动与公式默认值将在规则运行时开放后接入。 -->
        <el-select
          class="text-property__select"
          model-value="custom"
          disabled
          aria-label="默认值类型"
        >
          <el-option label="自定义" value="custom" />
        </el-select>
        <el-input
          v-model="widget.defaultValue"
          class="text-property__default-value"
          :maxlength="1000"
          aria-label="默认值"
        />
      </section>
    </template>

    <template #after-width>
      <section class="text-property__section">
        <h3 class="text-property__heading">字段安全</h3>
        <!-- 脱敏展示仅适用于单行文本，能力开放前不写入 Schema。 -->
        <el-checkbox disabled>脱敏显示</el-checkbox>
      </section>
    </template>
  </FormSchemaCommonPropertyPanel>
</template>

<style scoped lang="scss">
.text-property {
  &__section {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-md);
  }

  &__heading {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    margin: 0;
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    line-height: 1.5;
    color: var(--el-text-color-primary);
  }

  &__select {
    width: 100%;
  }

  &__default-value {
    margin-top: var(--el-space-xs);
  }
}
</style>
