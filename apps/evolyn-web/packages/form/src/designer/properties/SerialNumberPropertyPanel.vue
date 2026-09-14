<script setup lang="ts">
import { RiAddLine, RiDeleteBin6Line, RiDraggable, RiEditLine } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import Draggable from 'vuedraggable';
import {
  ElButton,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElIcon,
  ElInput,
  ElOption,
  ElSelect,
} from 'element-plus';
import type { FormItem, SnCounterRulePart, SnRulePart, SnSubmittedAtRulePart, SnWidget } from '../../schema/types';
import FormSchemaPropertySection from './FormSchemaPropertySection.vue';
import SerialNumberCounterDialog from './SerialNumberCounterDialog.vue';
import SerialNumberDateFormatDialog from './SerialNumberDateFormatDialog.vue';

/** 流水号专属规则编辑器：只维护片段序列，通用标题/描述/可见性仍由父面板负责。 */
const props = defineProps<{ widget: SnWidget; items: FormItem[] }>();

const counterDialogVisible = shallowRef(false);
const counter = computed(() => props.widget.rules.find((part): part is SnCounterRulePart => part.type === 'counter'));
const dateDialogVisible = shallowRef(false);
const editingDateRule = shallowRef<SnSubmittedAtRulePart | null>(null);
const rules = computed<SnRulePart[]>({
  get: () => props.widget.rules,
  set: (value) => {
    props.widget.rules = value;
  },
});
const eligibleFields = computed(() =>
  props.items.filter(
    (item) =>
      item.widget.fieldId !== props.widget.fieldId &&
      ['text', 'textarea', 'number', 'decimal', 'money', 'percent', 'datetime', 'radiogroup', 'combo'].includes(item.widget.type),
  ),
);
const fieldLabelById = computed(() => new Map(props.items.map((item) => [item.widget.fieldId, item.label])));
const ruleKeys = new WeakMap<object, string>();
let ruleKeySequence = 0;

function ruleKey(rule: SnRulePart): string {
  const existing = ruleKeys.get(rule);
  if (existing) return existing;
  ruleKeySequence += 1;
  const key = `sn-rule-${ruleKeySequence}`;
  ruleKeys.set(rule, key);
  return key;
}

function addPart(type: string): void {
  if (type === 'submittedAt') rules.value.push({ type: 'submittedAt', format: 'yyyyMMdd', formatType: 'preset' });
  if (type === 'literal') rules.value.push({ type: 'literal', value: '' });
  if (type === 'field' && eligibleFields.value[0]) {
    rules.value.push({ type: 'field', fieldId: eligibleFields.value[0].widget.fieldId! });
  }
}

function removePart(index: number): void {
  if (rules.value[index]?.type === 'counter') return;
  rules.value.splice(index, 1);
}

function openCounterSettings(): void {
  if (!counter.value) return;
  counterDialogVisible.value = true;
}

function updateCounterSettings(value: SnCounterRulePart): void {
  if (counter.value) Object.assign(counter.value, value);
}

function openDateSettings(rule: SnSubmittedAtRulePart): void {
  editingDateRule.value = rule;
  dateDialogVisible.value = true;
}

function updateDateSettings(value: Pick<SnSubmittedAtRulePart, 'format' | 'formatType'>): void {
  if (editingDateRule.value) Object.assign(editingDateRule.value, value);
}

function ruleSummary(part: SnRulePart): string {
  if (part.type === 'counter') {
    const resetLabel = {
      none: '不自动重置',
      daily: '每天重置',
      monthly: '每月重置',
      yearly: '每年重置',
    }[part.resetCycle];
    return `自动计数 ${part.digits} 位数字，${resetLabel}`;
  }
  if (part.type === 'submittedAt') return `格式：${dateFormatPreview(part.format)}`;
  if (part.type === 'literal') return `固定字符 ${part.value || '未设置'}`;
  return `表单字段 ${fieldLabelById.value.get(part.fieldId) ?? '字段已删除'}`;
}

function dateFormatPreview(format: string): string {
  return format.replace(/yyyy|MM|dd/g, (token) => ({ yyyy: '2015', MM: '01', dd: '01' })[token] ?? token);
}
</script>

<template>
  <FormSchemaPropertySection title="流水号规则">
    <Draggable
      v-model="rules"
      :item-key="ruleKey"
      handle=".serial-number-property__drag"
      class="serial-number-property__rules"
      :animation="160"
    >
      <template #item="{ element, index }">
        <div class="serial-number-property__rule">
          <el-icon class="serial-number-property__drag" aria-label="拖动排序"><RiDraggable /></el-icon>
          <button
            v-if="element.type === 'counter'"
            class="serial-number-property__summary"
            type="button"
            @click="openCounterSettings"
          >
            {{ ruleSummary(element) }}
          </button>
          <template v-else-if="element.type === 'submittedAt'">
            <span class="serial-number-property__type">提交日期</span>
            <span class="serial-number-property__date-summary" :title="`格式：${dateFormatPreview(element.format)}`">{{ ruleSummary(element) }}</span>
            <el-button class="serial-number-property__edit" text aria-label="编辑日期格式" @click="openDateSettings(element)">
              <el-icon><RiEditLine /></el-icon>
            </el-button>
          </template>
          <template v-else-if="element.type === 'literal'">
            <span class="serial-number-property__type">固定字符</span>
            <el-input v-model="element.value" class="serial-number-property__literal" :maxlength="32" aria-label="固定字符" />
          </template>
          <template v-else>
            <span class="serial-number-property__type">表单字段</span>
            <el-select v-model="element.fieldId" aria-label="流水号引用字段">
              <el-option v-for="field in eligibleFields" :key="field.widget.fieldId" :label="field.label" :value="field.widget.fieldId ?? ''" />
            </el-select>
          </template>
          <el-button v-if="element.type !== 'counter'" text aria-label="删除规则" @click="removePart(index)">
            <el-icon><RiDeleteBin6Line /></el-icon>
          </el-button>
        </div>
      </template>
    </Draggable>
    <el-dropdown trigger="click" @command="addPart">
      <el-button class="serial-number-property__add" plain>
        <el-icon><RiAddLine /></el-icon>添加
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item command="submittedAt">提交日期</el-dropdown-item>
          <el-dropdown-item command="literal">固定字符</el-dropdown-item>
          <el-dropdown-item :disabled="eligibleFields.length === 0" command="field">表单字段</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </FormSchemaPropertySection>

  <SerialNumberCounterDialog v-model="counterDialogVisible" :rule="counter ?? null" @confirm="updateCounterSettings" />
  <SerialNumberDateFormatDialog v-model="dateDialogVisible" :rule="editingDateRule" @confirm="updateDateSettings" />
</template>

<style scoped lang="scss">
.serial-number-property {
  &__rules { display: flex; flex-direction: column; gap: var(--el-space-sm); }
  &__rule { display: flex; gap: var(--el-space-sm); align-items: center; min-height: 40px; padding: 0 var(--el-space-sm); border: 1px solid var(--el-border-color); border-radius: var(--el-border-radius-base); }
  &__rule:hover { border-color: var(--el-color-primary-light-5); }
  &__drag { flex: none; color: var(--el-text-color-secondary); cursor: grab; }
  &__summary { flex: 1; padding: 0; overflow: hidden; color: var(--el-color-primary); text-align: left; text-overflow: ellipsis; white-space: nowrap; cursor: pointer; background: transparent; border: 0; }
  &__type { flex: none; color: var(--el-color-primary); white-space: nowrap; }
  &__date-summary { flex: 1; overflow: hidden; font-size: 14px; text-overflow: ellipsis; white-space: nowrap; }
  &__edit { flex: none; opacity: 0; transition: opacity .16s ease; }
  &__rule:hover &__edit, &__edit:focus-visible { opacity: 1; }
  &__rule :deep(.el-select), &__rule :deep(.el-input) { flex: 1; min-width: 0; }
  &__literal :deep(.el-input__wrapper) { padding: 0; background: transparent; box-shadow: none; }
  &__literal :deep(.el-input__inner) { height: 28px; font-size: 14px; }
  &__add { width: 100%; }
}
</style>
