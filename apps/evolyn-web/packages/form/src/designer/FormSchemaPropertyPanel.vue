<template>
  <aside class="form-schema-property" aria-label="属性配置面板">
    <div class="form-schema-property__tabs">
      <el-segmented
        v-model="activeTab"
        :options="propertyTabs"
        block
        class="form-schema-property__tab-control"
        aria-label="属性类型"
      />
    </div>

    <template v-if="activeTab === 'field'">
      <header v-if="layout" class="form-schema-property__header">
        <span class="form-schema-property__type-tag">{{ layout ? '布局' : typeLabel }}</span>
        <span class="form-schema-property__title">
          {{ layout ? '标签页' : draftItem?.label || typeLabel }}
        </span>
      </header>

      <div v-if="!draftItem && !layout" class="form-schema-property__empty">
        请在画布中选择字段或布局
      </div>

      <EvolynScrollbar v-else class="form-schema-property__body">
        <MultitabPropertyPanel
          v-if="layout"
          :layout="layout"
          @set-tab-style="$emit('set-tab-style', $event)"
          @add-tab="$emit('add-tab')"
          @remove-tab="$emit('remove-tab', $event)"
          @duplicate-tab="$emit('duplicate-tab', $event)"
          @rename-tab="(tabName, title) => $emit('rename-tab', tabName, title)"
          @reorder-tabs="$emit('reorder-tabs', $event)"
        />
        <SubformPropertyPanel
          v-else-if="draftItem && isSubform"
          v-model="subformDraft"
          @rename-key="$emit('rename-key', $event)"
        />
        <el-form v-else-if="draftItem" label-position="top" size="default" @submit.prevent>
          <TextPropertyPanel v-if="widget.type === 'text'" v-model="draftItem" />

          <template v-else>
            <FormSchemaCommonPropertyPanel
              v-model="draftItem"
              arrangement="reference"
              :is-separator="isSeparator"
              :title-required="!isSeparator"
              :show-widget-name="false"
              @rename-key="$emit('rename-key', $event)"
            >
              <template #title-suffix>
                <!-- 当前协议不支持在线切换字段类型，保留只读类型栏位统一布局。 -->
                <el-select
                  class="form-schema-property__type-select"
                  :model-value="widget.type"
                  disabled
                  aria-label="字段类型"
                >
                  <el-option :label="typeLabel" :value="widget.type" />
                </el-select>
              </template>

              <template #after-prompt>
                <!-- 控件专属配置统一置于公共提示文字之后、校验之前。 -->
                <TextareaPropertyPanel v-if="widget.type === 'textarea'" :widget="widget" />
                <NumberPropertyPanel v-else-if="widget.type === 'number'" :widget="widget" />
                <DateTimePropertyPanel v-else-if="widget.type === 'datetime'" :widget="widget" />
                <SeparatorPropertyPanel v-else-if="widget.type === 'separator'" :widget="widget" />
                <OptionsPropertyPanel v-else-if="optionsWidget" :widget="optionsWidget" />
                <FormSchemaPropertySection v-else title="专属设置">
                  <p class="form-schema-property__deferred">
                    该控件的专属配置已按协议保存，运行能力随后续版本开放。
                  </p>
                </FormSchemaPropertySection>
              </template>
            </FormSchemaCommonPropertyPanel>
          </template>
        </el-form>
      </EvolynScrollbar>
    </template>

    <EvolynScrollbar v-else class="form-schema-property__body">
      <el-form class="form-schema-property__form-settings" label-position="top" size="default">
        <el-form-item label="表单名称">
          <el-input
            v-model="formNameDraft"
            :maxlength="128"
            :disabled="formNameSaving"
            placeholder="请输入表单名称"
            @blur="updateFormName"
            @keyup.enter="updateFormName"
          />
        </el-form-item>
        <el-form-item label="表单布局">
          <el-select
            :model-value="formLayout"
            @update:model-value="emit('update-form-layout', $event as FormLayoutMode)"
          >
            <el-option
              v-for="option in formLayoutOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item class="form-schema-property__show-rules-form-item">
          <template #label>
            <span class="form-schema-property__show-rules-label">
              字段显隐规则
              <el-tooltip
                content="满足设定条件时显示指定字段；条件不满足时字段会隐藏"
                placement="top"
              >
                <el-icon
                  class="form-schema-property__show-rules-help"
                  aria-label="字段显隐规则说明"
                >
                  <RiQuestionFill />
                </el-icon>
              </el-tooltip>
            </span>
          </template>
          <div class="form-schema-property__show-rules">
            <el-button
              class="form-schema-property__show-rules-entry"
              type="primary"
              plain
              @click="showRulesDrawer = true"
            >
              {{ fieldShowRules.length > 0 ? '管理显隐规则' : '添加显隐规则' }}
            </el-button>
            <p v-if="fieldShowRules.length > 0" class="form-schema-property__show-rules-meta">
              已配置 {{ fieldShowRules.length }} 条规则
            </p>
          </div>
        </el-form-item>
        <el-form-item class="form-schema-property__show-rules-form-item">
          <template #label>
            <span class="form-schema-property__show-rules-label">
              不可见字段赋值
              <el-tooltip content="字段对本次提交人不可见时，其记录值的处理策略" placement="top">
                <el-icon
                  class="form-schema-property__show-rules-help"
                  aria-label="不可见字段赋值说明"
                >
                  <RiQuestionFill />
                </el-icon>
              </el-tooltip>
            </span>
          </template>
          <div class="form-schema-property__submit-rule">
            <div class="form-schema-property__submit-rule-row">
              <el-select
                :model-value="submitRule"
                aria-label="不可见字段默认赋值策略"
                @update:model-value="emit('update-submit-rule', $event as SubmitRule)"
              >
                <el-option
                  v-for="option in submitRuleOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
              <el-tooltip content="配置特殊字段赋值规则" placement="top">
                <el-button
                  class="form-schema-property__submit-rule-gear"
                  aria-label="配置特殊字段赋值规则"
                  @click="submitRuleDialog = true"
                >
                  <el-icon><RiSettings3Fill /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
            <p class="form-schema-property__submit-rule-hint">{{ submitRuleHint }}</p>
            <div v-if="specialRuleCount > 0" class="form-schema-property__submit-rule-summary">
              <div class="form-schema-property__submit-rule-summary-head">
                <span class="form-schema-property__submit-rule-summary-title">
                  已设置特殊字段赋值规则
                </span>
                <button
                  type="button"
                  class="form-schema-property__submit-rule-summary-toggle"
                  :aria-expanded="summaryExpanded"
                  @click="summaryExpanded = !summaryExpanded"
                >
                  {{ summaryExpanded ? '收起详情' : '查看详情' }}
                  <el-icon
                    class="form-schema-property__submit-rule-summary-arrow"
                    :class="{
                      'form-schema-property__submit-rule-summary-arrow--open': summaryExpanded,
                    }"
                  >
                    <RiArrowDownSFill />
                  </el-icon>
                </button>
              </div>
              <div v-if="summaryExpanded" class="form-schema-property__submit-rule-summary-body">
                <div
                  v-for="group in summaryGroups"
                  :key="group.value"
                  class="form-schema-property__submit-rule-summary-group"
                >
                  <p class="form-schema-property__submit-rule-group-name">{{ group.label }}</p>
                  <p class="form-schema-property__submit-rule-group-fields">
                    {{ group.expanded ? group.labels.join('、') : group.collapsedText }}
                    <button
                      v-if="group.overflow > 0"
                      type="button"
                      class="form-schema-property__submit-rule-summary-more"
                      :aria-expanded="group.expanded"
                      @click="toggleSummaryGroup(group.value)"
                    >
                      {{
                        group.expanded
                          ? '收起'
                          : `展开全部（等 ${group.overflow + summaryCollapseAt} 个字段）`
                      }}
                    </button>
                  </p>
                </div>
              </div>
            </div>
          </div>
        </el-form-item>
      </el-form>
      <FormSchemaFieldShowRulesDrawer
        v-if="schemaDocument"
        v-model="showRulesDrawer"
        :rules="fieldShowRules"
        :document="schemaDocument"
        :items="schemaDocument.content.items"
        @save-rule="$emit('save-field-show-rule', $event)"
        @remove-rule="$emit('remove-field-show-rule', $event)"
        @duplicate-rule="$emit('duplicate-field-show-rule', $event)"
        @reorder-rules="$emit('reorder-field-show-rules', $event)"
      />
      <FormSchemaSubmitRuleDialog
        v-model="submitRuleDialog"
        :submit-rule="submitRule"
        :widget-submit-rules="widgetSubmitRules"
        :items="schemaDocument?.content.items ?? []"
        @save="$emit('update-widget-submit-rules', $event)"
      />
    </EvolynScrollbar>
  </aside>
</template>

<script setup lang="ts">
import { RiArrowDownSFill, RiQuestionFill, RiSettings3Fill } from '@remixicon/vue';
import { computed, ref, shallowRef, watch } from 'vue';
import { EvolynScrollbar } from '@evolyn.do/ui';
import {
  ElButton,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElOption,
  ElSegmented,
  ElSelect,
  ElTooltip,
} from 'element-plus';
import type {
  FieldShowRule,
  FormItem,
  FormLayoutMode,
  FormMultitabLayout,
  FormSchemaDocument,
  FormTabStyle,
  SubformWidget,
  CheckboxGroupWidget,
  ComboCheckWidget,
  ComboWidget,
  RadioGroupWidget,
  SubmitRule,
} from '../schema/types';
import { widgetTypeLabel } from '../schema/dictionary';
import { submitRuleLabel } from '../schema/invisible-value-policy';
import FormSchemaCommonPropertyPanel from './FormSchemaCommonPropertyPanel.vue';
import DateTimePropertyPanel from './properties/DateTimePropertyPanel.vue';
import MultitabPropertyPanel from './properties/MultitabPropertyPanel.vue';
import NumberPropertyPanel from './properties/NumberPropertyPanel.vue';
import OptionsPropertyPanel from './properties/OptionsPropertyPanel.vue';
import SeparatorPropertyPanel from './properties/SeparatorPropertyPanel.vue';
import FormSchemaPropertySection from './properties/FormSchemaPropertySection.vue';
import SubformPropertyPanel from './properties/SubformPropertyPanel.vue';
import TextareaPropertyPanel from './properties/TextareaPropertyPanel.vue';
import TextPropertyPanel from './properties/TextPropertyPanel.vue';
import FormSchemaFieldShowRulesDrawer from './FormSchemaFieldShowRulesDrawer.vue';
import FormSchemaSubmitRuleDialog from './FormSchemaSubmitRuleDialog.vue';

/**
 * 字段属性面板：编辑 item 公共属性与按 widget.type 分派的专属配置。
 * item 是画布数组中的响应式对象，除 widgetName（需同步页面选中键）外
 * 直接就地修改；全部取值范围与字段字典一致，保存时校验器兜底。
 */
const props = withDefaults(
  defineProps<{
    item?: FormItem;
    layout?: FormMultitabLayout;
    formLayout?: FormLayoutMode;
    formName?: string;
    formNameSaving?: boolean;
    /** 协议文档：显隐规则抽屉的候选校验与字段清单数据源。 */
    schemaDocument?: FormSchemaDocument;
    fieldShowRules?: FieldShowRule[];
    /** 不可见字段赋值默认策略（v6 content.submitRule）。 */
    submitRule?: SubmitRule;
    /** 特殊字段赋值规则映射（v6 content.widget_submit_rules）。 */
    widgetSubmitRules?: Record<string, SubmitRule>;
  }>(),
  {
    item: undefined,
    layout: undefined,
    formLayout: 'normal',
    formName: '',
    formNameSaving: false,
    schemaDocument: undefined,
    fieldShowRules: () => [],
    submitRule: 2,
    widgetSubmitRules: () => ({}),
  },
);

const emit = defineEmits<{
  'rename-key': [key: string];
  'update-item': [item: FormItem];
  'update-form-name': [name: string];
  'update-form-layout': [layout: FormLayoutMode];
  'set-tab-style': [style: FormTabStyle];
  'add-tab': [];
  'remove-tab': [tabName: string];
  'duplicate-tab': [tabName: string];
  'rename-tab': [tabName: string, title: string];
  'reorder-tabs': [tabNames: string[]];
  'save-field-show-rule': [rule: FieldShowRule];
  'remove-field-show-rule': [ruleId: string];
  'duplicate-field-show-rule': [ruleId: string];
  'reorder-field-show-rules': [ruleIds: string[]];
  'update-submit-rule': [rule: SubmitRule];
  'update-widget-submit-rules': [rules: Record<string, SubmitRule>];
}>();

const showRulesDrawer = shallowRef(false);
const submitRuleDialog = shallowRef(false);
// 特殊规则摘要卡初始收起，避免属性面板被长字段列表占满（§5.1）。
const summaryExpanded = shallowRef(false);
const expandedSummaryGroups = ref<Set<number>>(new Set());

const activeTab = shallowRef<'field' | 'form'>('field');
const propertyTabs: Array<{ label: string; value: 'field' | 'form' }> = [
  { label: '字段属性', value: 'field' },
  { label: '表单属性', value: 'form' },
];
const formNameDraft = shallowRef(props.formName);
const draftItem = ref<FormItem>();
const widget = computed(() => draftItem.value!.widget);
const isSubform = computed(() => widget.value.type === 'subform');
const subformDraft = computed<FormItem<SubformWidget>>({
  get: () => draftItem.value as FormItem<SubformWidget>,
  set: (item) => {
    draftItem.value = item;
  },
});
const typeLabel = computed(() => widgetTypeLabel(widget.value.type));
const isSeparator = computed(() => widget.value.type === 'separator');
type OptionsWidget = RadioGroupWidget | CheckboxGroupWidget | ComboWidget | ComboCheckWidget;
const optionsWidget = computed<OptionsWidget | null>(() => {
  switch (widget.value.type) {
    case 'radiogroup':
    case 'checkboxgroup':
    case 'combo':
    case 'combocheck':
      return widget.value;
    default:
      return null;
  }
});
const formLayoutOptions: ReadonlyArray<{ label: string; value: FormLayoutMode }> = [
  { label: '单列', value: 'normal' },
  { label: '双列', value: 'grid-2' },
  { label: '三列', value: 'grid-3' },
  { label: '四列', value: 'grid-4' },
];

// 外部加载或保存成功后，以资产详情的名称覆盖编辑中的临时值。
watch(
  () => props.formName,
  (name) => {
    formNameDraft.value = name;
  },
);

// 属性面板维护字段草稿并通过显式事件提交，避免子组件直接修改父级协议对象。
watch(
  () => props.item,
  (item) => {
    if (sameJSON(item, draftItem.value)) return;
    draftItem.value = item ? cloneItem(item) : undefined;
  },
  { immediate: true, deep: true },
);

watch(
  draftItem,
  (item) => {
    if (!item || sameJSON(item, props.item)) return;
    emit('update-item', cloneItem(item));
  },
  { deep: true },
);

function sameJSON(left: unknown, right: unknown): boolean {
  return JSON.stringify(left) === JSON.stringify(right);
}

/**
 * 属性面板只接受经 Schema 校验的 JSON 安全字段项；深拷贝可隔离外部响应式对象，
 * 避免草稿编辑直接改写画布源数据。
 */
function cloneItem(item: FormItem): FormItem {
  return JSON.parse(JSON.stringify(item)) as FormItem;
}

/** 仅在名称确有变化且非空时通知页面调用资产改名接口。 */
function updateFormName(): void {
  const name = formNameDraft.value.trim();
  if (!name || name === props.formName) {
    formNameDraft.value = props.formName;
    return;
  }
  emit('update-form-name', name);
}

// ---- 不可见字段赋值（v6 §5.1：紧邻显隐规则的独立区块，二者互不隐式修改） ----

/** 摘要卡每组折叠时展示的字段标签数（超出以「等 N 个字段」收起）。 */
const summaryCollapseAt = 4;

const submitRuleOptions = computed(() =>
  ([1, 2, 3] as const).map((value) => ({
    label: submitRuleLabel(value),
    value,
  })),
);

const submitRuleHint = computed(() => {
  switch (props.submitRule) {
    case 1:
      return '仅编辑记录或流程后续节点提交时保留原值；新建提交仍写入空值。';
    case 3:
      return '与可见字段执行同一计算链路，由服务端重算后写入。';
    default:
      return '字段隐藏后清除原有值，按字段类型写入空值。';
  }
});

/** 顶层字段标签索引（摘要卡按标签展示，不暴露 widgetName）。 */
const labelByName = computed(() => {
  const map = new Map<string, string>();
  for (const item of props.schemaDocument?.content.items ?? []) {
    map.set(item.widget.widgetName, item.label);
  }
  return map;
});

/** 字段展示序（field_layout 先于未入布局的字段，与画布顺序一致）。 */
const orderedNames = computed(() => {
  const layout = props.schemaDocument?.content.field_layout ?? [];
  const known = new Set(layout);
  const extras = [...labelByName.value.keys()].filter((name) => !known.has(name));
  return [...layout, ...extras];
});

const specialRuleCount = computed(() => Object.keys(props.widgetSubmitRules).length);

interface SummaryGroup {
  value: number;
  label: string;
  labels: string[];
  collapsedText: string;
  overflow: number;
  expanded: boolean;
}

/** 摘要卡分组：按「保持原值 / 空值 / 始终重新计算」固定顺序展示非空分组。 */
const summaryGroups = computed<SummaryGroup[]>(() => {
  const groups: Array<{ value: number; names: string[] }> = [1, 2, 3].map((value) => ({
    value,
    names: Object.entries(props.widgetSubmitRules)
      .filter(([, rule]) => rule === value)
      .map(([name]) => name),
  }));
  const orderIndex = new Map(orderedNames.value.map((name, index) => [name, index]));
  return groups
    .filter((group) => group.names.length > 0)
    .map((group) => {
      const labels = group.names
        .sort((left, right) => (orderIndex.get(left) ?? 0) - (orderIndex.get(right) ?? 0))
        .map((name) => labelByName.value.get(name) ?? name);
      const overflow = Math.max(0, labels.length - summaryCollapseAt);
      return {
        value: group.value,
        label: submitRuleLabel(group.value),
        labels,
        collapsedText: labels.slice(0, summaryCollapseAt).join('、'),
        overflow,
        expanded: expandedSummaryGroups.value.has(group.value),
      };
    });
});

function toggleSummaryGroup(value: number): void {
  const next = new Set(expandedSummaryGroups.value);
  if (next.has(value)) next.delete(value);
  else next.add(value);
  expandedSummaryGroups.value = next;
}

/** 选项编辑：label 与 value 同步维护（P2 简化，协议仍允许二者不同）。 */
</script>

<style lang="scss">
.form-schema-property {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 264px;
  background-color: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
  border-left: 1px solid var(--el-border-color);

  &__tabs {
    padding: var(--el-space-md) var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__tab-control {
    width: 100%;
    --el-segmented-bg-color: var(--el-fill-color);
    --el-segmented-item-selected-color: var(--el-color-primary);
    --el-segmented-item-selected-bg-color: var(--el-bg-color);
    --el-segmented-item-hover-bg-color: var(--el-fill-color-light);
    --el-segmented-item-active-bg-color: var(--el-fill-color);
  }

  &__empty {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    height: 100%;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }

  &__header {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    padding: var(--el-space-lg) var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__type-tag {
    padding: 0 var(--el-space-sm);
    font-size: var(--el-font-size-extra-small);
    line-height: 20px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-base);
  }

  &__title {
    overflow: hidden;
    font-size: var(--el-font-size-base);
    font-weight: 600;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__body {
    flex: 1;
    min-height: 0;
    padding: var(--el-space-sm);

    .el-form-item {
      margin-bottom: var(--el-space-lg);

      .el-select,
      .el-input-number {
        width: 100%;
      }
    }
  }

  &__pair {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
  }

  &__control-label {
    display: block;
    margin-bottom: var(--el-space-xs);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-regular);
  }

  &__type-select {
    width: 100%;
  }

  &__options {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
    width: 100%;
  }

  &__option {
    display: flex;
    gap: var(--el-space-xs);
    align-items: center;
  }

  &__option-add {
    align-self: flex-start;
  }

  &__deferred {
    margin: 0;
    padding: var(--el-space-md);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__hint {
    margin: 0;
    padding: var(--el-space-md);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-extra-small);
    line-height: 1.5;
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__show-rules-form-item {
    margin-top: var(--el-space-sm);

    :deep(.el-form-item__label) {
      padding-bottom: var(--el-space-sm);
    }

    :deep(.el-form-item__content) {
      line-height: normal;
    }
  }

  &__show-rules-label {
    display: inline-flex;
    gap: var(--el-space-xs);
    align-items: center;
    font-size: var(--el-font-size-base);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__show-rules-help {
    font-size: var(--el-font-size-base);
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  &__show-rules {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
    width: 100%;
  }

  // 使用 primary + plain 作为组件级兜底，确保主题样式未额外覆盖时也保留描边按钮。
  &__show-rules-entry.el-button {
    width: 100%;
    height: 32px;
    margin: 0;
    font-size: var(--el-font-size-base);
    font-weight: 500;
    color: var(--el-color-primary);
    background-color: var(--el-bg-color);
    border-color: var(--el-color-primary);

    &:hover,
    &:focus-visible {
      color: var(--el-color-primary);
      background-color: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary);
    }
  }

  &__show-rules-meta {
    margin: 0;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }

  // ---- 不可见字段赋值（v6 §5.1：选择框 + 齿轮 + 浅色摘要卡） ----

  &__submit-rule {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
    width: 100%;
  }

  &__submit-rule-row {
    display: flex;
    gap: var(--el-space-sm);
    width: 100%;

    .el-select {
      flex: 1;
    }
  }

  &__submit-rule-gear.el-button {
    width: 32px;
    height: 32px;
    padding: 0;
    margin: 0;
  }

  &__submit-rule-hint {
    margin: 0;
    font-size: var(--el-font-size-extra-small);
    line-height: 1.5;
    color: var(--el-text-color-secondary);
  }

  &__submit-rule-summary {
    padding: var(--el-space-sm) var(--el-space-md);
    background: var(--el-fill-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  &__submit-rule-summary-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__submit-rule-summary-title {
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-regular);
  }

  &__submit-rule-summary-toggle {
    display: inline-flex;
    gap: var(--el-space-xs);
    align-items: center;
    padding: 0;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-color-primary);
    background: none;
    border: none;
    cursor: pointer;

    &:hover {
      color: var(--el-color-primary-light-3);
    }
  }

  &__submit-rule-summary-arrow {
    transition: transform 0.2s;

    &--open {
      transform: rotate(180deg);
    }
  }

  &__submit-rule-summary-body {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
    margin-top: var(--el-space-sm);
  }

  &__submit-rule-group-name {
    margin: 0;
    font-size: var(--el-font-size-extra-small);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__submit-rule-group-fields {
    margin: var(--el-space-xs) 0 0;
    font-size: var(--el-font-size-extra-small);
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }

  &__submit-rule-summary-more {
    padding: 0;
    margin-left: var(--el-space-xs);
    font-size: var(--el-font-size-extra-small);
    color: var(--el-color-primary);
    background: none;
    border: none;
    cursor: pointer;

    &:hover {
      color: var(--el-color-primary-light-3);
    }
  }
}
</style>
