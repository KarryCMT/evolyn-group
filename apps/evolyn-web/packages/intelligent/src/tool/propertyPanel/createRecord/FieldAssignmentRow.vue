<script setup lang="ts">
import {
  RiBuildingLine,
  RiCalendarLine,
  RiMapPinLine,
  RiPhoneLine,
  RiText,
  RiUserLine,
} from '@remixicon/vue';
import { type Component, computed } from 'vue';
import type {
  IntelligentActorOption,
  IntelligentFieldAssignment,
  IntelligentFieldOption,
  IntelligentFieldValueSource,
  IntelligentJsonValue,
  IntelligentNodeFieldValueSource,
  IntelligentSourceFieldGroup,
} from '../../../schema';
import CustomFieldValueEditor from './CustomFieldValueEditor.vue';
import NodeFieldPicker from './NodeFieldPicker.vue';
import ValueSourceTypeSelect from './ValueSourceTypeSelect.vue';

defineOptions({ name: 'IntelligentFieldAssignmentRow' });

const props = defineProps<{
  field: IntelligentFieldOption;
  assignment?: IntelligentFieldAssignment;
  sourceGroups: readonly IntelligentSourceFieldGroup[];
  members?: readonly IntelligentActorOption[];
  departments?: readonly IntelligentActorOption[];
}>();

const emit = defineEmits<{
  change: [assignment: IntelligentFieldAssignment | null];
}>();

type SourceMode = IntelligentFieldValueSource['type'];

const sourceMode = computed<SourceMode>(() => props.assignment?.source.type ?? 'node-field');
const nodeFieldSource = computed<IntelligentNodeFieldValueSource | null>(() =>
  props.assignment?.source.type === 'node-field' ? props.assignment.source : null,
);
const customValue = computed<IntelligentJsonValue>(() =>
  props.assignment?.source.type === 'custom'
    ? props.assignment.source.value
    : defaultCustomValue(props.field),
);

const fieldIcons: Partial<Record<IntelligentFieldOption['valueKind'], Component>> = {
  member: RiUserLine,
  members: RiUserLine,
  department: RiBuildingLine,
  departments: RiBuildingLine,
  date: RiCalendarLine,
  address: RiMapPinLine,
};
const fieldIcon = computed(() =>
  props.field.widgetType === 'phone' ? RiPhoneLine : (fieldIcons[props.field.valueKind] ?? RiText),
);

function defaultCustomValue(field: IntelligentFieldOption): IntelligentJsonValue {
  if (['multi-choice', 'members', 'departments'].includes(field.valueKind)) return [];
  return null;
}

function assignment(source: IntelligentFieldValueSource): IntelligentFieldAssignment {
  return {
    targetFieldId: props.field.fieldId,
    targetWidgetName: props.field.widgetName,
    source,
  };
}

function changeMode(mode: SourceMode): void {
  if (mode === 'node-field') {
    // 尚未选择字段即保持 unset，不保存半成品字段引用。
    emit('change', null);
    return;
  }
  if (mode === 'empty') {
    emit('change', assignment({ type: 'empty' }));
    return;
  }
  emit('change', assignment({ type: 'custom', value: defaultCustomValue(props.field) }));
}

function changeNodeField(source: IntelligentNodeFieldValueSource): void {
  emit('change', assignment(source));
}

function changeCustomValue(value: IntelligentJsonValue): void {
  emit('change', assignment({ type: 'custom', value }));
}
</script>

<template>
  <div class="field-assignment-row">
    <div class="field-assignment-row__target" :title="field.label">
      <component :is="fieldIcon" aria-hidden="true" />
      <span>{{ field.label }}</span>
      <small v-if="field.required">必填</small>
    </div>
    <span class="field-assignment-row__equals">=</span>
    <div class="field-assignment-row__source" :class="{ 'is-empty': sourceMode === 'empty' }">
      <ValueSourceTypeSelect :model-value="sourceMode" @update:model-value="changeMode" />
      <NodeFieldPicker
        v-if="sourceMode === 'node-field'"
        :model-value="nodeFieldSource"
        :target-field="field"
        :groups="sourceGroups"
        @update:model-value="changeNodeField"
      />
      <CustomFieldValueEditor
        v-else-if="sourceMode === 'custom'"
        :model-value="customValue"
        :field="field"
        :members="members"
        :departments="departments"
        @update:model-value="changeCustomValue"
      />
      <span v-else class="field-assignment-row__empty-label">空值</span>
    </div>
  </div>
</template>

<style scoped lang="scss">
.field-assignment-row {
  display: grid;
  align-items: center;
  grid-template-columns: minmax(220px, 32%) 24px minmax(360px, 1fr);
  gap: 10px;

  &__target,
  &__source {
    min-width: 0;
    min-height: 46px;
    color: #273247;
    background: #fff;
    border: 1px solid #d8dee8;
    border-radius: 7px;
  }

  &__target {
    display: flex;
    padding: 0 14px;
    align-items: center;
    gap: 10px;

    svg { width: 20px; height: 20px; flex: 0 0 auto; color: #536075; }
    span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    small { margin-left: auto; color: #e25252; font-size: 11px; }
  }

  &__equals { color: #273247; text-align: center; }
  &__source { display: flex; align-items: stretch; }
  &__source:focus-within { border-color: #13b8ad; }
  &__source.is-empty { background: #f9fafb; }
  &__empty-label { display: flex; padding: 0 16px; align-items: center; color: #273247; }
}

@media (max-width: 920px) {
  .field-assignment-row {
    grid-template-columns: minmax(170px, 34%) 18px minmax(260px, 1fr);
  }
}
</style>
