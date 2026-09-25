<script setup lang="ts">
import { RiDeleteBin6Line } from '@remixicon/vue';
import { computed } from 'vue';
import type {
  IntelligentActorOption,
  IntelligentFieldAssignment,
  IntelligentFieldOption,
  IntelligentFieldValueSource,
  IntelligentJsonValue,
  IntelligentNodeFieldValueSource,
  IntelligentSourceFieldGroup,
} from '../../../schema';
import CustomFieldValueEditor from '../createRecord/CustomFieldValueEditor.vue';
import NodeFieldPicker from '../createRecord/NodeFieldPicker.vue';
import ValueSourceTypeSelect from '../createRecord/ValueSourceTypeSelect.vue';
import RecordFieldSelect from './RecordFieldSelect.vue';

defineOptions({ name: 'UpdateRecordAssignmentRow' });

const props = defineProps<{
  assignment: IntelligentFieldAssignment;
  fields: readonly IntelligentFieldOption[];
  disabledFieldIds: readonly string[];
  sourceGroups: readonly IntelligentSourceFieldGroup[];
  members?: readonly IntelligentActorOption[];
  departments?: readonly IntelligentActorOption[];
}>();
const emit = defineEmits<{
  change: [assignment: IntelligentFieldAssignment];
  remove: [];
}>();

const field = computed(() =>
  props.fields.find((item) => item.fieldId === props.assignment.targetFieldId),
);
const sourceMode = computed<IntelligentFieldValueSource['type']>(
  () => props.assignment.source.type,
);
const nodeFieldSource = computed<IntelligentNodeFieldValueSource | null>(() =>
  props.assignment.source.type === 'node-field' ? props.assignment.source : null,
);
const customValue = computed<IntelligentJsonValue>(() =>
  props.assignment.source.type === 'custom'
    ? props.assignment.source.value
    : field.value
      ? defaultCustomValue(field.value)
      : null,
);

function defaultCustomValue(target: IntelligentFieldOption): IntelligentJsonValue {
  return ['multi-choice', 'members', 'departments'].includes(target.valueKind) ? [] : null;
}

function selectField(fieldId: string): void {
  const target = props.fields.find((item) => item.fieldId === fieldId);
  if (!target) return;
  emit('change', {
    targetFieldId: target.fieldId,
    targetWidgetName: target.widgetName,
    source: { type: 'node-field', nodeId: '', field: '' },
  });
}

function changeMode(mode: IntelligentFieldValueSource['type']): void {
  const target = field.value;
  if (!target) return;
  if (mode === 'empty') {
    emit('change', { ...props.assignment, source: { type: 'empty' } });
  } else if (mode === 'custom') {
    emit('change', {
      ...props.assignment,
      source: { type: 'custom', value: defaultCustomValue(target) },
    });
  } else {
    emit('change', {
      ...props.assignment,
      source: { type: 'node-field', nodeId: '', field: '' },
    });
  }
}

function changeNodeField(source: IntelligentNodeFieldValueSource): void {
  emit('change', { ...props.assignment, source });
}

function changeCustomValue(value: IntelligentJsonValue): void {
  emit('change', { ...props.assignment, source: { type: 'custom', value } });
}
</script>

<template>
  <div class="update-assignment-row">
    <RecordFieldSelect
      :model-value="assignment.targetFieldId"
      :options="fields"
      :disabled-field-ids="disabledFieldIds"
      @update:model-value="selectField"
    />
    <span class="update-assignment-row__equals">=</span>
    <div
      class="update-assignment-row__source"
      :class="{ 'is-invalid': assignment.source.type === 'node-field' && !assignment.source.field }"
    >
      <ValueSourceTypeSelect :model-value="sourceMode" @update:model-value="changeMode" />
      <NodeFieldPicker
        v-if="sourceMode === 'node-field' && field"
        :model-value="nodeFieldSource"
        :target-field="field"
        :groups="sourceGroups"
        @update:model-value="changeNodeField"
      />
      <CustomFieldValueEditor
        v-else-if="sourceMode === 'custom' && field"
        :model-value="customValue"
        :field="field"
        :members="members"
        :departments="departments"
        @update:model-value="changeCustomValue"
      />
      <span v-else class="update-assignment-row__empty">空值</span>
    </div>
    <button
      type="button"
      class="update-assignment-row__remove"
      aria-label="删除字段赋值"
      @click="emit('remove')"
    >
      <RiDeleteBin6Line />
    </button>
  </div>
</template>

<style scoped lang="scss">
.update-assignment-row {
  display: grid;
  align-items: center;
  grid-template-columns: minmax(220px, 32%) 24px minmax(360px, 1fr) 34px;
  gap: 10px;

  &__equals {
    color: #273247;
    text-align: center;
  }
  &__source {
    display: flex;
    min-width: 0;
    min-height: 46px;
    background: #fff;
    border: 1px solid #d8dee8;
    border-radius: 7px;

    &:focus-within {
      border-color: #11b8ad;
    }
    &.is-invalid {
      border-color: #ef5858;
    }
  }
  &__empty {
    display: flex;
    padding: 0 16px;
    align-items: center;
    color: #273247;
    background: #f9fafb;
  }
  &__remove {
    display: inline-flex;
    width: 34px;
    height: 42px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: #687386;
    background: transparent;
    border: 0;
    cursor: pointer;

    svg {
      width: 19px;
      height: 19px;
    }
  }
}

@media (max-width: 920px) {
  .update-assignment-row {
    grid-template-columns: minmax(170px, 34%) 18px minmax(260px, 1fr) 34px;
  }
}
</style>
