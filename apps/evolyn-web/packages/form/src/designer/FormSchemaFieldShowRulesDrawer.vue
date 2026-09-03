<template>
  <el-drawer
    :model-value="modelValue"
    title="字段显隐规则"
    size="90%"
    direction="btt"
    append-to-body
    destroy-on-close
    class="form-field-show-rules"
    @update:model-value="$emit('update:model-value', $event)"
  >
    <template #default>
      <section v-if="rules.length === 0" class="form-field-show-rules__empty-state">
        <img
          :src="fieldShowRulesEmptyStateImage"
          class="form-field-show-rules__empty-illustration"
          alt=""
        />
        <p class="form-field-show-rules__empty-copy">
          让字段在特定条件成立时显示，反之隐藏。
          <el-link type="primary" underline="never">了解更多</el-link>
        </p>
        <el-button
          class="form-field-show-rules__empty-action"
          type="primary"
          :disabled="rules.length >= 200"
          @click="startCreate"
        >
          <el-icon><RiAddFill /></el-icon>
          添加显隐规则
        </el-button>
      </section>

      <div v-else class="form-field-show-rules__toolbar">
        <p class="form-field-show-rules__hint">
          条件成立时显示所选字段，否则隐藏；隐藏字段保留已填内容，仅不进入提交载荷。
        </p>
        <el-button type="primary" :disabled="rules.length >= 200" @click="startCreate">
          <el-icon><RiAddFill /></el-icon>
          添加显隐规则
        </el-button>
      </div>

      <Draggable
        v-if="rules.length > 0"
        :list="localRules"
        item-key="id"
        handle=".form-field-show-rules__drag"
        :animation="150"
        @end="emitReorder"
      >
        <template #item="{ element }">
          <div class="form-field-show-rules__row">
            <el-icon class="form-field-show-rules__drag"><RiDragMoveFill /></el-icon>
            <div class="form-field-show-rules__summary">
              <p class="form-field-show-rules__summary-line">当{{ conditionSummary(element) }}时</p>
              <p class="form-field-show-rules__summary-line">显示{{ targetSummary(element) }}</p>
            </div>
            <div class="form-field-show-rules__actions">
              <el-tooltip content="编辑" placement="top">
                <el-button
                  text
                  :icon="RiEditFill"
                  aria-label="编辑规则"
                  @click="startEdit(element)"
                />
              </el-tooltip>
              <el-tooltip content="复制" placement="top">
                <el-button
                  text
                  :icon="RiFileCopyFill"
                  aria-label="复制规则"
                  @click="$emit('duplicate-rule', element.id)"
                />
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button
                  text
                  type="danger"
                  :icon="RiDeleteBin6Fill"
                  aria-label="删除规则"
                  @click="confirmRemove(element)"
                />
              </el-tooltip>
            </div>
          </div>
        </template>
      </Draggable>
    </template>

    <template #footer>
      <el-button @click="$emit('update:model-value', false)">关闭</el-button>
    </template>
  </el-drawer>

  <FormSchemaFieldShowRuleDialog
    :model-value="editorVisible"
    :rule="editingRule"
    :rules="rules"
    :document="document"
    :items="items"
    @update:model-value="onEditorVisibleChange"
    @save="saveRule"
  />
</template>

<script setup lang="ts">
import fieldShowRulesEmptyStateImageBase64 from './assets/field-show-rules-empty-state.png';
import FormSchemaFieldShowRuleDialog from './FormSchemaFieldShowRuleDialog.vue';
import { ref, shallowRef, watch } from 'vue';
import Draggable from 'vuedraggable';
import { ElButton, ElDrawer, ElIcon, ElLink, ElMessageBox, ElTooltip } from 'element-plus';
import {
  RiAddFill,
  RiDeleteBin6Fill,
  RiDragMoveFill,
  RiEditFill,
  RiFileCopyFill,
} from '@remixicon/vue';
import { FIELD_SHOW_EMPTY_METHODS, FIELD_SHOW_METHOD_LABELS } from '../schema/dictionary';
import type {
  FieldShowCondition,
  FieldShowRule,
  FormItem,
  FormSchemaDocument,
} from '../schema/types';

const fieldShowRulesEmptyStateImage = `data:image/png;base64,${fieldShowRulesEmptyStateImageBase64}`;

/**
 * 字段显隐规则列表抽屉：只管理规则列表和操作分发。新增、编辑草稿及校验完全由
 * FormSchemaFieldShowRuleDialog 独立承担，列表层不会持有弹窗编辑态。
 */
const props = defineProps<{
  modelValue: boolean;
  rules: FieldShowRule[];
  document: FormSchemaDocument;
  items: FormItem[];
}>();

const emit = defineEmits<{
  'update:model-value': [value: boolean];
  'save-rule': [rule: FieldShowRule];
  'remove-rule': [ruleId: string];
  'duplicate-rule': [ruleId: string];
  'reorder-rules': [ruleIds: string[]];
}>();

const editorVisible = shallowRef(false);
const editingRule = ref<FieldShowRule | null>(null);
const localRules = ref<FieldShowRule[]>([]);

watch(
  () => props.rules,
  (rules) => {
    localRules.value = JSON.parse(JSON.stringify(rules)) as FieldShowRule[];
  },
  { immediate: true, deep: true },
);

watch(
  () => props.modelValue,
  (open) => {
    if (!open) closeEditor();
  },
);

function startCreate(): void {
  editingRule.value = null;
  editorVisible.value = true;
}

function startEdit(rule: FieldShowRule): void {
  editingRule.value = JSON.parse(JSON.stringify(rule)) as FieldShowRule;
  editorVisible.value = true;
}

function onEditorVisibleChange(open: boolean): void {
  if (!open) closeEditor();
}

function saveRule(rule: FieldShowRule): void {
  emit('save-rule', rule);
  closeEditor();
}

function closeEditor(): void {
  editorVisible.value = false;
  editingRule.value = null;
}

async function confirmRemove(rule: FieldShowRule): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `删除后「${targetSummary(rule)}」将不再受该规则控制，确定删除？`,
      '删除显隐规则',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    );
  } catch {
    return;
  }
  emit('remove-rule', rule.id);
}

function emitReorder(): void {
  emit(
    'reorder-rules',
    localRules.value.map((rule) => rule.id),
  );
}

function labelOf(field: string): string {
  return props.items.find((entry) => entry.widget.widgetName === field)?.label ?? field;
}

function isEmptyMethod(method: string): boolean {
  return FIELD_SHOW_EMPTY_METHODS.has(method);
}

function valueText(condition: FieldShowCondition): string {
  const values = condition.value ?? [];
  if (values.length === 0) return '（未设置）';
  return values.map((entry) => (typeof entry === 'string' ? entry : String(entry))).join('、');
}

function conditionSummary(rule: FieldShowRule): string {
  const joiner = rule.filter.rel === 'and' ? '且' : '或';
  const parts = rule.filter.cond.map((condition) => {
    const base = `［${labelOf(condition.field)}］${FIELD_SHOW_METHOD_LABELS[condition.method] ?? condition.method}`;
    if (isEmptyMethod(condition.method)) return base;
    const suffix = condition.includeCurrentMember ? '（或当前成员）' : '';
    return `${base}［${valueText(condition)}］${suffix}`;
  });
  return parts.length > 0 ? parts.join(joiner) : '（未配置条件）';
}

function targetSummary(rule: FieldShowRule): string {
  if (rule.fields.length === 0) return '（未选择字段）';
  return rule.fields.map((field) => `［${labelOf(field)}］`).join('、');
}
</script>

<style lang="scss">
.form-field-show-rules {
  .el-drawer__header {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    height: 56px;
    margin-bottom: 0;
    padding: 0 var(--el-space-lg);
  }

  .el-drawer__title {
    font-size: 18px;
    font-weight: 600;
    line-height: 26px;
    text-align: center;
  }

  .el-drawer__close-btn {
    position: absolute;
    right: var(--el-space-lg);
    width: 32px;
    height: 32px;

    .el-icon {
      font-size: 22px;
    }
  }

  .el-drawer__body {
    display: flex;
    min-height: 0;
    flex-direction: column;
  }

  &__toolbar {
    display: flex;
    gap: var(--el-space-md);
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--el-space-lg);
  }

  &__hint {
    flex: 1;
    margin: 0;
    font-size: var(--el-font-size-extra-small);
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }

  &__empty-state {
    display: flex;
    flex: 1;
    min-height: 360px;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 0 var(--el-space-xl) 9%;
  }

  &__empty-illustration {
    display: block;
    width: min(240px, 48vw);
    height: auto;
    margin-bottom: var(--el-space-lg);
    object-fit: contain;
  }

  &__empty-copy {
    display: flex;
    flex-wrap: wrap;
    gap: var(--el-space-xs);
    align-items: baseline;
    justify-content: center;
    margin: 0 0 var(--el-space-xl);
    font-size: var(--el-font-size-base);
    line-height: 1.6;
    color: var(--el-text-color-secondary);
    text-align: center;
  }

  &__empty-action.el-button {
    height: 44px;
    padding: 0 var(--el-space-xl);
    font-size: var(--el-font-size-base);
    font-weight: 500;
  }

  &__row {
    display: flex;
    gap: var(--el-space-sm);
    align-items: center;
    padding: var(--el-space-md);
    margin-bottom: var(--el-space-sm);
    background-color: var(--el-fill-color-extra-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);

    &:hover {
      border-color: var(--el-color-primary-light-5);
    }
  }

  &__drag {
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
    cursor: grab;
  }

  &__summary {
    flex: 1;
    min-width: 0;
  }

  &__summary-line {
    margin: 0;
    font-size: var(--el-font-size-small);
    line-height: 22px;
    color: var(--el-text-color-regular);
    overflow-wrap: anywhere;
  }

  &__actions {
    display: flex;
    flex-shrink: 0;
    gap: var(--el-space-xs);
  }
}
</style>
