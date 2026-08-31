<script setup lang="ts">
import { CopyDocument, Delete, Rank } from '@element-plus/icons-vue';
import {
  ElButton,
  ElCheckbox,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElInputNumber,
  ElOption,
  ElRadio,
  ElRadioButton,
  ElRadioGroup,
  ElSelect,
  ElSwitch,
} from 'element-plus';
import { computed, ref } from 'vue';
import Draggable from 'vuedraggable';
import type { FormItem, FormWidgetType, SubformWidget } from '../schema/types';
import { SUBFORM_ALLOWED_WIDGET_TYPES } from '../schema/types';
import { copyWidgetItem, createWidgetItem, widgetTypeLabel } from '../schema/dictionary';
import FormSchemaCommonPropertyPanel from './FormSchemaCommonPropertyPanel.vue';

/** 子表单专属属性区：基础信息、子字段、行权限与端侧展示策略集中配置。 */
const model = defineModel<FormItem<SubformWidget>>({ required: true });
const emit = defineEmits<{ renameKey: [key: string] }>();

const device = ref<'pc' | 'mobile'>('pc');
const addType = ref<FormWidgetType | ''>('');
const widget = computed(() => model.value.widget);
const allowedOptions = SUBFORM_ALLOWED_WIDGET_TYPES.map((type) => ({
  value: type,
  label: widgetTypeLabel(type),
}));
const maxStickyColumns = computed(() => Math.max(1, Math.min(widget.value.items.length, 5)));

function addChild(type: FormWidgetType): void {
  if (widget.value.items.length >= 200) return;
  const item = createWidgetItem(type);
  item.lineWidth = 12;
  widget.value.items.push(item);
  addType.value = '';
}

function copyChild(index: number): void {
  if (widget.value.items.length >= 200) return;
  const copied = copyWidgetItem(widget.value.items[index]!);
  copied.lineWidth = 12;
  widget.value.items.splice(index + 1, 0, copied);
}

function removeChild(index: number): void {
  widget.value.items.splice(index, 1);
  normalizeStickyLimits();
}

function normalizeStickyLimits(): void {
  const max = maxStickyColumns.value;
  if (widget.value.pcStickyColumn)
    widget.value.pcStickyColumn.limit = Math.min(max, widget.value.pcStickyColumn.limit);
  if (widget.value.mobileStickyColumn) {
    widget.value.mobileStickyColumn.limit = Math.min(max, widget.value.mobileStickyColumn.limit);
  }
}
</script>

<template>
  <el-form class="form-schema-subform-property" label-position="top" @submit.prevent>
    <FormSchemaCommonPropertyPanel v-model="model" @rename-key="emit('renameKey', $event)" />

    <section class="form-schema-subform-property__section">
      <div class="form-schema-subform-property__section-heading">
        <h3 class="form-schema-subform-property__section-title">子字段</h3>
        <span>{{ widget.items.length }}/200</span>
      </div>
      <Draggable
        v-model="widget.items"
        item-key="widget.widgetName"
        handle=".form-schema-subform-property__drag"
        class="form-schema-subform-property__children"
        :animation="180"
      >
        <template #item="{ element, index }">
          <div class="form-schema-subform-property__child">
            <button
              class="form-schema-subform-property__drag"
              type="button"
              title="拖拽排序"
              aria-label="拖拽排序"
            >
              <el-icon><Rank /></el-icon>
            </button>
            <span class="form-schema-subform-property__child-type">{{
              widgetTypeLabel(element.widget.type)
            }}</span>
            <span class="form-schema-subform-property__child-label">{{ element.label }}</span>
            <button type="button" title="复制子字段" @click="copyChild(index)">
              <el-icon><CopyDocument /></el-icon>
            </button>
            <button type="button" title="删除子字段" @click="removeChild(index)">
              <el-icon><Delete /></el-icon>
            </button>
          </div>
        </template>
      </Draggable>
      <el-select
        v-model="addType"
        class="form-schema-subform-property__add"
        placeholder="＋ 添加子字段"
        :disabled="widget.items.length >= 200"
        @change="addChild"
      >
        <el-option
          v-for="option in allowedOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        />
      </el-select>
      <p class="form-schema-subform-property__hint">
        可从左侧直接拖入；标签页、分割线、富文本和子表单不能嵌套。
      </p>
    </section>

    <section class="form-schema-subform-property__section">
      <h3 class="form-schema-subform-property__section-title">校验</h3>
      <el-checkbox
        :model-value="!widget.allowBlank"
        @update:model-value="widget.allowBlank = !$event"
      >
        必填
      </el-checkbox>
      <div class="form-schema-subform-property__row-limits">
        <el-form-item label="最少行数">
          <el-input-number
            :model-value="widget.minRowCount ?? undefined"
            :min="0"
            :max="200"
            placeholder="不限"
            @update:model-value="widget.minRowCount = $event ?? null"
          />
        </el-form-item>
        <el-form-item label="最多行数">
          <el-input-number
            :model-value="widget.maxRowCount ?? undefined"
            :min="1"
            :max="200"
            placeholder="不限"
            @update:model-value="widget.maxRowCount = $event ?? null"
          />
        </el-form-item>
      </div>
    </section>

    <section class="form-schema-subform-property__section">
      <div class="form-schema-subform-property__switch-row">
        <div>
          <h3 class="form-schema-subform-property__section-title">快速填报</h3>
          <p>连续录入多行时保留上一行可复用内容</p>
        </div>
        <el-switch
          v-model="widget.quickFill"
          :disabled="!widget.subformCreate || !widget.subformEdit"
        />
      </div>
      <div class="form-schema-subform-property__notice">
        需同时开启“可新增记录”和“可编辑已有记录”权限后才会生效。
      </div>
    </section>

    <section class="form-schema-subform-property__section">
      <h3 class="form-schema-subform-property__section-title">字段权限</h3>
      <div class="form-schema-subform-property__permission-list">
        <el-checkbox v-model="widget.visible">可见</el-checkbox>
        <el-checkbox v-model="widget.enable">可编辑</el-checkbox>
        <div class="form-schema-subform-property__permission-children">
          <el-checkbox v-model="widget.subformCreate" :disabled="!widget.enable"
            >可新增记录</el-checkbox
          >
          <el-checkbox v-model="widget.subformInsert" :disabled="!widget.enable"
            >可插入记录</el-checkbox
          >
          <el-checkbox v-model="widget.subformEdit" :disabled="!widget.enable"
            >可编辑已有记录</el-checkbox
          >
          <el-checkbox v-model="widget.subformDelete" :disabled="!widget.enable"
            >可删除已有记录</el-checkbox
          >
        </div>
      </div>
    </section>

    <section class="form-schema-subform-property__section">
      <h3 class="form-schema-subform-property__section-title">子表单展示样式</h3>
      <el-radio-group v-model="device" class="form-schema-subform-property__device" size="default">
        <el-radio-button value="pc">电脑端</el-radio-button>
        <el-radio-button value="mobile">移动端</el-radio-button>
      </el-radio-group>

      <template v-if="device === 'pc'">
        <div class="form-schema-subform-property__inline-control">
          <el-checkbox v-model="widget.pcStickyColumn.enable">固定前</el-checkbox>
          <el-select
            v-model="widget.pcStickyColumn.limit"
            :disabled="!widget.pcStickyColumn.enable"
          >
            <el-option
              v-for="count in maxStickyColumns"
              :key="count"
              :label="`${count}列`"
              :value="count"
            />
          </el-select>
        </div>
      </template>
      <template v-else>
        <el-radio-group
          v-model="widget.mobileViewStyle"
          class="form-schema-subform-property__mobile-style"
        >
          <el-radio value="vertical">纵向平铺</el-radio>
          <el-radio value="horizontal">横向表格</el-radio>
        </el-radio-group>
        <div class="form-schema-subform-property__inline-control">
          <el-checkbox v-model="widget.mobileStickyColumn.enable">固定前</el-checkbox>
          <el-select
            v-model="widget.mobileStickyColumn.limit"
            :disabled="!widget.mobileStickyColumn.enable || widget.mobileViewStyle !== 'horizontal'"
          >
            <el-option
              v-for="count in maxStickyColumns"
              :key="count"
              :label="`${count}列`"
              :value="count"
            />
          </el-select>
        </div>
        <el-form-item v-if="widget.mobileViewStyle === 'vertical'" label="数据收起时显示的简报">
          <el-select v-model="widget.mobileSummaryFieldCount">
            <el-option
              v-for="count in 5"
              :key="count"
              :label="`前${count}个字段的值`"
              :value="count"
            />
          </el-select>
        </el-form-item>
      </template>
    </section>
  </el-form>
</template>

<style scoped lang="scss">
.form-schema-subform-property {
  &__section {
    padding-bottom: var(--el-space-xl);
    margin-bottom: var(--el-space-xl);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__section:last-child {
    padding-bottom: 0;
    margin-bottom: 0;
    border-bottom: 0;
  }

  &__section-title {
    margin: 0 0 var(--el-space-md);
    font-size: var(--el-font-size-base);
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__section-heading,
  &__switch-row,
  &__inline-control {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__section-heading > span,
  &__switch-row p,
  &__hint {
    margin: 0;
    font-size: var(--el-font-size-extra-small);
    color: var(--el-text-color-secondary);
  }

  &__children {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
  }

  &__child {
    display: grid;
    grid-template-columns: 24px 1fr 1fr 28px 28px;
    gap: var(--el-space-xs);
    align-items: center;
    min-height: 38px;
    padding: 0 var(--el-space-xs);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
  }

  &__child > button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    background: transparent;
    border: 0;
  }

  &__drag {
    cursor: grab !important;
  }

  &__child-type,
  &__child-label {
    overflow: hidden;
    font-size: var(--el-font-size-extra-small);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__child-type {
    color: var(--el-text-color-primary);
  }

  &__child-label {
    color: var(--el-text-color-secondary);
  }

  &__add {
    width: 100%;
    margin-top: var(--el-space-sm);
  }

  &__hint {
    margin-top: var(--el-space-sm);
    line-height: 1.5;
  }

  &__row-limits {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-md);
    margin-top: var(--el-space-md);
  }

  &__switch-row .form-schema-subform-property__section-title {
    margin-bottom: var(--el-space-xs);
  }

  &__notice {
    padding: var(--el-space-md);
    margin-top: var(--el-space-md);
    font-size: var(--el-font-size-extra-small);
    line-height: 1.6;
    color: var(--el-color-success-dark-2);
    background: var(--el-color-success-light-9);
    border-radius: var(--el-border-radius-base);
  }

  &__permission-list,
  &__permission-children {
    display: flex;
    flex-direction: column;
    gap: var(--el-space-sm);
  }

  &__permission-children {
    padding-left: var(--el-space-xl);
  }

  &__device {
    display: flex;
    width: 100%;
    margin-bottom: var(--el-space-lg);
  }

  &__device :deep(.el-radio-button) {
    flex: 1;
  }

  &__device :deep(.el-radio-button__inner) {
    width: 100%;
  }

  &__inline-control {
    gap: var(--el-space-md);
  }

  &__inline-control .el-select {
    flex: 1;
  }

  &__mobile-style {
    display: flex;
    justify-content: space-between;
    width: 100%;
    margin-bottom: var(--el-space-lg);
  }
}
</style>
