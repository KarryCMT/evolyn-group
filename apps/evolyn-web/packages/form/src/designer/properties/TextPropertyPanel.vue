<script setup lang="ts">
import { ElButton, ElCheckbox, ElInput, ElOption, ElSelect } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import type {
  DataLinkageDefinition,
  FormItem,
  FormSchemaDocument,
  TextWidget,
} from '../../schema/types';
import {
  type LinkageDesignerAdapter,
  cloneDataLinkageDefinitions,
  linkageRuleForTarget,
} from '../../schema/linkage';
import FormSchemaCommonPropertyPanel from '../FormSchemaCommonPropertyPanel.vue';
import DefaultValueModeSelect from './DefaultValueModeSelect.vue';
import DataLinkageSettingDialog from '../linkage/DataLinkageSettingDialog.vue';

/**
 * 单行文本专属属性面板只承载格式与默认值；标题、描述、提示文字、校验、权限和
 * 宽度仍由通用属性面板负责，以保证公共属性只有一个实现入口。
 */
const model = defineModel<FormItem>({ required: true });

const props = withDefaults(
  defineProps<{
    schemaDocument?: FormSchemaDocument;
    appId?: number;
    linkageAdapter?: LinkageDesignerAdapter;
  }>(),
  { schemaDocument: undefined, appId: 0, linkageAdapter: undefined },
);

const emit = defineEmits<{ 'update-linkages': [rules: DataLinkageDefinition[]] }>();

const widget = computed(() => model.value.widget as TextWidget);
const linkageDialogVisible = shallowRef(false);
const linkageRule = computed(() =>
  linkageRuleForTarget(props.schemaDocument?.content.linkages ?? [], model.value.widget.widgetName),
);
const defaultValueMode = shallowRef<'custom' | 'data-linkage' | 'formula'>('custom');

// 已保存规则决定初始模式；未保存时仍保留用户刚选择的“数据联动”状态，
// 使“数据联动设置”入口在关闭空白配置弹窗后继续可见。
watch(
  [() => model.value.widget.widgetName, () => linkageRule.value?.id],
  ([, ruleId]) => {
    defaultValueMode.value = ruleId ? 'data-linkage' : 'custom';
  },
  { immediate: true },
);

function changeDefaultValueMode(mode: 'custom' | 'data-linkage' | 'formula'): void {
  defaultValueMode.value = mode;
  if (mode === 'data-linkage') {
    linkageDialogVisible.value = true;
    return;
  }
  if (mode !== 'custom' || !linkageRule.value || !props.schemaDocument) return;
  const rules = cloneDataLinkageDefinitions(props.schemaDocument.content.linkages);
  const index = rules.findIndex((rule) => rule.id === linkageRule.value?.id);
  if (index < 0) return;
  rules[index]!.mappings = rules[index]!.mappings.filter(
    (mapping) => mapping.targetFieldId !== model.value.widget.widgetName,
  );
  if (rules[index]!.mappings.length === 0) rules.splice(index, 1);
  emit('update-linkages', rules);
}

function saveLinkage(rule: DataLinkageDefinition): void {
  const rules = cloneDataLinkageDefinitions(props.schemaDocument?.content.linkages ?? []);
  const index = rules.findIndex((entry) => entry.id === rule.id);
  if (index >= 0) rules[index] = rule;
  else rules.push(rule);
  emit('update-linkages', rules);
}
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
        <DefaultValueModeSelect
          :model-value="defaultValueMode"
          @update:model-value="changeDefaultValueMode"
        />
        <el-input
          v-if="defaultValueMode === 'custom'"
          v-model="widget.defaultValue"
          class="text-property__default-value"
          :maxlength="1000"
          aria-label="默认值"
        />
        <el-button
          v-else-if="defaultValueMode === 'data-linkage'"
          class="text-property__linkage-button"
          @click="linkageDialogVisible = true"
        >
          {{ linkageRule ? '已设置数据联动' : '数据联动设置' }}
        </el-button>
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

  <DataLinkageSettingDialog
    v-if="schemaDocument"
    v-model="linkageDialogVisible"
    :rule="linkageRule"
    :app-id="appId"
    :target-field-id="model.widget.widgetName"
    :items="schemaDocument.content.items"
    :adapter="linkageAdapter"
    @confirm="saveLinkage"
  />
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

  &__linkage-button {
    width: 100%;
    color: var(--el-color-primary);
    background: var(--el-bg-color);
    border-color: var(--el-color-primary);
  }
}
</style>
