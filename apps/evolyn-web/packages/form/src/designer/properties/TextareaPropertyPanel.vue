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
  FieldFormulaDefinition,
  FormItem,
  FormSchemaDocument,
  TextAreaWidget,
} from '../../schema/types';
import DataLinkageSettingDialog from '../linkage/DataLinkageSettingDialog.vue';
import FieldFormulaSettingDialog from '../formula/FieldFormulaSettingDialog.vue';
import { fieldFormulaForTarget } from '../../schema/field-formula';
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
const emit = defineEmits<{
  'update-linkages': [rules: DataLinkageDefinition[]];
  'update-field-formulas': [rules: FieldFormulaDefinition[]];
}>();

const widget = computed(() => model.value.widget as TextAreaWidget);
const linkageDialogVisible = shallowRef(false);
const formulaDialogVisible = shallowRef(false);
const linkageRule = computed(() =>
  linkageRuleForTarget(props.schemaDocument?.content.linkages ?? [], model.value.widget.widgetName),
);
const fieldFormula = computed(() =>
  fieldFormulaForTarget(
    props.schemaDocument?.content.fieldFormulas ?? [],
    model.value.widget.widgetName,
  ),
);
const defaultValueMode = shallowRef<'custom' | 'data-linkage' | 'formula'>('custom');

watch(
  [() => model.value.widget.widgetName, () => linkageRule.value?.id, () => fieldFormula.value?.id],
  ([, ruleId, formulaId]) => {
    defaultValueMode.value = formulaId ? 'formula' : ruleId ? 'data-linkage' : 'custom';
  },
  { immediate: true },
);

/** 默认值模式切换时只移除当前字段对应的映射，不影响同一规则的其他回填字段。 */
function changeDefaultValueMode(mode: 'custom' | 'data-linkage' | 'formula'): void {
  defaultValueMode.value = mode;
  if (mode === 'data-linkage') {
    removeFieldFormula();
    linkageDialogVisible.value = true;
    return;
  }
  if (mode === 'formula') {
    removeLinkageMapping();
    formulaDialogVisible.value = true;
    return;
  }
  removeFieldFormula();
  removeLinkageMapping();
}

function removeLinkageMapping(): void {
  if (!linkageRule.value || !props.schemaDocument) return;
  const rules = cloneDataLinkageDefinitions(props.schemaDocument.content.linkages);
  const index = rules.findIndex((rule) => rule.id === linkageRule.value?.id);
  if (index < 0) return;
  rules[index]!.mappings = rules[index]!.mappings.filter(
    (mapping) => mapping.targetFieldId !== model.value.widget.widgetName,
  );
  if (rules[index]!.mappings.length === 0) rules.splice(index, 1);
  emit('update-linkages', rules);
}

function removeFieldFormula(): void {
  if (!fieldFormula.value || !props.schemaDocument) return;
  emit(
    'update-field-formulas',
    props.schemaDocument.content.fieldFormulas.filter((rule) => rule.id !== fieldFormula.value?.id),
  );
}

function saveFieldFormula(formula: FieldFormulaDefinition): void {
  const formulas = (props.schemaDocument?.content.fieldFormulas ?? []).map((entry) => ({ ...entry }));
  const index = formulas.findIndex((entry) => entry.targetFieldId === formula.targetFieldId);
  if (index >= 0) formulas[index] = formula;
  else formulas.push(formula);
  widget.value.defaultValue = null;
  emit('update-field-formulas', formulas);
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
    <el-button
      v-else
      class="textarea-property__formula-button"
      @click="formulaDialogVisible = true"
    >
      <span>{{ fieldFormula ? '已设置公式' : 'ƒx 编辑公式' }}</span>
      <span aria-hidden="true">↗</span>
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
  <FieldFormulaSettingDialog
    v-if="schemaDocument"
    v-model="formulaDialogVisible"
    :target="model"
    :items="schemaDocument.content.items"
    :formula="fieldFormula"
    @confirm="saveFieldFormula"
  />
</template>

<style scoped lang="scss">
.textarea-property__linkage-button,
.textarea-property__formula-button {
  width: 100%;
  color: var(--el-color-primary);
  background: var(--el-bg-color);
  border-color: var(--el-color-primary);
}

.textarea-property__formula-button {
  display: flex;
  justify-content: space-between;
  color: var(--el-text-color-primary);
  border-color: var(--el-border-color);
}
</style>
