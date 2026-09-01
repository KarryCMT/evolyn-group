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
      </el-form>
    </EvolynScrollbar>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue';
import { EvolynScrollbar } from '@evolyn.do/ui';
import { ElForm, ElFormItem, ElInput, ElOption, ElSegmented, ElSelect } from 'element-plus';
import type {
  FormItem,
  FormLayoutMode,
  FormMultitabLayout,
  FormTabStyle,
  SubformWidget,
  CheckboxGroupWidget,
  ComboCheckWidget,
  ComboWidget,
  RadioGroupWidget,
} from '../schema/types';
import { widgetTypeLabel } from '../schema/dictionary';
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
  }>(),
  {
    item: undefined,
    layout: undefined,
    formLayout: 'normal',
    formName: '',
    formNameSaving: false,
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
}>();

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
}
</style>
