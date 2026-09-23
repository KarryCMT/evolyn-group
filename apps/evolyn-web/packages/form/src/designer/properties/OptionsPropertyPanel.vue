<script setup lang="ts">
/* eslint-disable vue/no-mutating-props -- 属性子面板编辑的是父组件专用反应式草稿，由父层统一深拷贝后上抛。 */
import { RiAddFill, RiDeleteBin6Fill, RiFilter3Line } from '@remixicon/vue';
import { ElButton, ElInput, ElMessage, ElOption, ElRadioButton, ElRadioGroup, ElSelect, ElSwitch } from 'element-plus';
import type { CheckboxGroupWidget, ComboCheckWidget, ComboWidget, FormItem, RadioGroupWidget } from '../../schema/types';
import { type LinkageDesignerAdapter, type LinkageSourceField, linkageCurrentFields } from '../../schema/linkage';
import { computed, onBeforeUnmount, ref, shallowRef, watch } from 'vue';
import DefaultValueModeSelect from './DefaultValueModeSelect.vue';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';
import RelatedOptionFilterDialog from './RelatedOptionFilterDialog.vue';

type OptionsWidget = RadioGroupWidget | CheckboxGroupWidget | ComboWidget | ComboCheckWidget;
type RelatedWidget = ComboWidget | ComboCheckWidget;
const props = withDefaults(defineProps<{ widget: OptionsWidget; appId?: number; items?: FormItem[]; adapter?: LinkageDesignerAdapter }>(), { appId: 0, items: () => [], adapter: undefined });
const layoutWidget = computed<RadioGroupWidget | CheckboxGroupWidget | null>(() => props.widget.type === 'radiogroup' || props.widget.type === 'checkboxgroup' ? props.widget : null);
const relatedWidget = computed<RelatedWidget | null>(() => props.widget.type === 'combo' || props.widget.type === 'combocheck' ? props.widget : null);
const choiceLayout = computed<'vertical' | 'horizontal'>({ get: () => layoutWidget.value?.layout ?? 'horizontal', set: (layout) => { if (layoutWidget.value) layoutWidget.value.layout = layout; } });
const isMultiple = computed(() => ['checkboxgroup', 'combocheck'].includes(props.widget.type));
const sourceMode = computed<'custom' | 'related'>({ get: () => relatedWidget.value?.optionSource?.mode ?? 'custom', set: setSourceMode });
const sources = ref<Array<{ sourceId: string; appId: number; name: string }>>([]);
const sourceFields = ref<LinkageSourceField[]>([]);
const loadingSources = shallowRef(false);
const loadingFields = shallowRef(false);
const filterDialogVisible = shallowRef(false);
let sourceRequestController: AbortController | undefined;
let fieldRequestController: AbortController | undefined;
const related = computed(() => relatedWidget.value?.optionSource?.related);
const currentFields = computed(() => linkageCurrentFields(props.items));
const sourceLabel = computed(() => {
  const source = sources.value.find((entry) => entry.sourceId === related.value?.source.sourceId);
  const field = sourceFields.value.find((entry) => entry.id === related.value?.source.fieldId);
  return source && field ? `${source.name}--${field.name}` : '';
});
const sortFields = computed<LinkageSourceField[]>(() => [
  { id: '__value__', name: '当前选项的值', type: 'text', operators: [] },
  ...sourceFields.value,
  { id: 'sys.submittedAt', name: '提交时间', type: 'datetime', operators: [] },
  { id: 'sys.updatedAt', name: '更新时间', type: 'datetime', operators: [] },
]);

watch(() => [sourceMode.value, related.value?.source.sourceId] as const, async ([mode, sourceId]) => {
  if (mode !== 'related') return;
  // 修复由旧版属性面板写入的 null/空数组：面板一旦识别为关联模式，就把
  // defaultValue 归一为“键不存在”，使当前画布无需重新切换模式即可预览。
  const widget = relatedWidget.value;
  if (widget && 'defaultValue' in widget) delete widget.defaultValue;
  await loadSources();
  if (sourceId) await loadSourceFields(sourceId);
}, { immediate: true });
onBeforeUnmount(() => {
  sourceRequestController?.abort();
  fieldRequestController?.abort();
});

function setSourceMode(mode: 'custom' | 'related'): void {
  const widget = relatedWidget.value;
  if (!widget) return;
  if (mode === 'custom') { widget.optionSource = { mode: 'custom' }; return; }
  // 关联选项由运行时按当前成员权限动态查询，不能把静态默认值冻结进 Schema。
  // defaultValue 是可选配置属性；关闭能力时必须移除该键，不能写入 null（字符串
  // 属性的显式 null 在目标协议中非法，会导致预览渲染器拒绝加载整份表单）。
  delete widget.defaultValue;
  widget.optionSource = { mode: 'related', related: {
    source: { type: 'form', appId: props.appId, sourceId: '', fieldId: '' },
    sort: { fieldId: '__value__', direction: 'asc' },
    filter: { logic: 'and', conditions: [] },
  } };
}
async function loadSources(): Promise<void> {
  if (!props.adapter || sources.value.length > 0) return;
  sourceRequestController?.abort();
  const controller = new AbortController();
  sourceRequestController = controller;
  loadingSources.value = true;
  try {
    sources.value = await props.adapter.listSources(controller.signal);
  } catch {
    if (!controller.signal.aborted) ElMessage.error('关联表单加载失败，请稍后重试');
  } finally {
    if (sourceRequestController === controller) loadingSources.value = false;
  }
}
async function loadSourceFields(sourceId: string): Promise<void> {
  if (!props.adapter) return;
  fieldRequestController?.abort();
  const controller = new AbortController();
  fieldRequestController = controller;
  loadingFields.value = true;
  try {
    sourceFields.value = await props.adapter.listSourceFields(sourceId, controller.signal);
  } catch {
    if (!controller.signal.aborted) ElMessage.error('关联字段加载失败，请稍后重试');
  } finally {
    if (fieldRequestController === controller) loadingFields.value = false;
  }
}
function selectSource(sourceId: string): void {
  const config = related.value; if (!config) return;
  const source = sources.value.find((entry) => entry.sourceId === sourceId);
  config.source.sourceId = sourceId; config.source.appId = source?.appId ?? props.appId; config.source.fieldId = '';
  config.sort = { fieldId: '__value__', direction: 'asc' }; config.filter = { logic: 'and', conditions: [] };
  sourceFields.value = [];
}
function updateOption(index: number, label: string) { props.widget.options[index] = { label, value: label }; }
function addOption() { const next = `选项${props.widget.options.length + 1}`; props.widget.options.push({ label: next, value: next }); }
function removeOption(index: number) { props.widget.options.splice(index, 1); }
</script>

<template>
  <FormSchemaPropertySection v-if="widget.type === 'combo'" title="显示设置"><el-switch v-model="widget.filterable" inline-prompt active-text="可搜索" /></FormSchemaPropertySection>
  <FormSchemaPropertySection v-if="layoutWidget" title="布局"><el-radio-group v-model="choiceLayout"><el-radio-button value="vertical">纵向</el-radio-button><el-radio-button value="horizontal">横向</el-radio-button></el-radio-group></FormSchemaPropertySection>
  <FormSchemaPropertySection v-if="sourceMode === 'custom'" title="默认值">
    <DefaultValueModeSelect />
    <el-select v-if="isMultiple" v-model="widget.defaultValue" multiple clearable aria-label="默认值"><el-option v-for="option in widget.options" :key="option.value" :label="option.label" :value="option.value" /></el-select>
    <el-select v-else v-model="widget.defaultValue" clearable aria-label="默认值"><el-option v-for="option in widget.options" :key="option.value" :label="option.label" :value="option.value" /></el-select>
  </FormSchemaPropertySection>
  <FormSchemaPropertySection title="选项">
    <div v-if="relatedWidget" class="option-source">
      <el-select v-model="sourceMode" aria-label="选项来源"><el-option label="自定义" value="custom" /><el-option label="关联其他表单数据" value="related" /><el-option label="数据联动（后续开放）" value="linkage" disabled /></el-select>
      <template v-if="sourceMode === 'related' && related">
        <el-select :model-value="related.source.sourceId" filterable :loading="loadingSources" placeholder="请选择关联表单" aria-label="关联表单" @update:model-value="selectSource(String($event))"><el-option v-for="source in sources" :key="source.sourceId" :label="source.name" :value="source.sourceId" /></el-select>
        <el-select v-model="related.source.fieldId" filterable :loading="loadingFields" :disabled="!related.source.sourceId" placeholder="请选择字段" aria-label="关联表单字段"><el-option v-for="field in sourceFields" :key="field.id" :label="field.name" :value="field.id" /></el-select>
        <p v-if="sourceLabel" class="option-source__summary">已关联 {{ sourceLabel }}</p>
      </template>
    </div>
    <div v-if="sourceMode === 'custom'" class="form-schema-property__options">
      <div v-for="(option, index) in widget.options" :key="index" class="form-schema-property__option"><el-input :model-value="option.label" :maxlength="100" :placeholder="`选项${index + 1}`" @update:model-value="updateOption(index, String($event ?? ''))" /><el-button text type="danger" :icon="RiDeleteBin6Fill" :disabled="widget.options.length <= 1" @click="removeOption(index)" /></div>
      <el-button class="form-schema-property__option-add" text type="primary" :icon="RiAddFill" :disabled="widget.options.length >= 200" @click="addOption">添加选项</el-button>
    </div>
  </FormSchemaPropertySection>
  <template v-if="sourceMode === 'related' && related">
    <FormSchemaPropertySection title="选项排序"><div class="option-source__sort"><el-select v-model="related.sort.fieldId" aria-label="选项排序字段"><el-option v-for="field in sortFields" :key="field.id" :label="field.name" :value="field.id" /></el-select><el-select v-model="related.sort.direction" aria-label="选项排序方向"><el-option label="升序" value="asc" /><el-option label="降序" value="desc" /></el-select></div></FormSchemaPropertySection>
    <FormSchemaPropertySection title="选项过滤"><el-button class="option-source__filter" :icon="RiFilter3Line" :disabled="!related.source.fieldId" @click="filterDialogVisible = true">{{ related.filter.conditions.length ? `已设置 ${related.filter.conditions.length} 个条件` : '添加过滤条件' }}</el-button></FormSchemaPropertySection>
    <RelatedOptionFilterDialog v-model="filterDialogVisible" :filter="related.filter" :source-fields="sourceFields" :current-fields="currentFields" @confirm="related.filter = $event" />
  </template>
</template>

<style scoped lang="scss">
.option-source { display: flex; flex-direction: column; gap: var(--el-space-sm); width: 100%; }
.option-source__summary { margin: 0; font-size: var(--el-font-size-extra-small); color: var(--el-text-color-secondary); }
.option-source__sort { display: grid; grid-template-columns: minmax(0, 1fr) 82px; gap: var(--el-space-xs); width: 100%; }
.option-source__filter { width: 100%; }
</style>
