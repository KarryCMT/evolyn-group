<script setup lang="ts">
import { ElButton, ElInput, ElSwitch } from 'element-plus';
import { computed, shallowRef, watch } from 'vue';
import {
  type LinkageDesignerAdapter,
  cloneDataLinkageDefinitions,
  linkageRuleForTarget,
} from '../../schema/linkage';
import type {
  DataLinkageDefinition,
  FormItem,
  FormSchemaDocument,
  TextAreaWidget,
} from '../../schema/types';
import DataLinkageSettingDialog from '../linkage/DataLinkageSettingDialog.vue';
import DefaultValueModeSelect from './DefaultValueModeSelect.vue';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';

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

const widget = computed(() => model.value.widget as TextAreaWidget);
const linkageDialogVisible = shallowRef(false);
const linkageRule = computed(() =>
  linkageRuleForTarget(props.schemaDocument?.content.linkages ?? [], model.value.widget.widgetName),
);
const defaultValueMode = shallowRef<'custom' | 'data-linkage' | 'formula'>('custom');

watch(
  [() => model.value.widget.widgetName, () => linkageRule.value?.id],
  ([, ruleId]) => {
    defaultValueMode.value = ruleId ? 'data-linkage' : 'custom';
  },
  { immediate: true },
);

/** 默认值模式切换时只移除当前字段对应的映射，不影响同一规则的其他回填字段。 */
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
  <FormSchemaPropertySection title="显示设置">
    <el-switch v-model="widget.autoHeight" inline-prompt active-text="自动增高" />
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="默认值">
    <DefaultValueModeSelect
      :model-value="defaultValueMode"
      @update:model-value="changeDefaultValueMode"
    />
    <el-input
      v-if="defaultValueMode === 'custom'"
      v-model="widget.defaultValue"
      :maxlength="2000"
      aria-label="默认值"
    />
    <el-button
      v-else-if="defaultValueMode === 'data-linkage'"
      class="textarea-property__linkage-button"
      @click="linkageDialogVisible = true"
    >
      {{ linkageRule ? '已设置数据联动' : '数据联动设置' }}
    </el-button>
  </FormSchemaPropertySection>

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
.textarea-property__linkage-button {
  width: 100%;
  color: var(--el-color-primary);
  background: var(--el-bg-color);
  border-color: var(--el-color-primary);
}
</style>
