<script setup lang="ts">
import { RiDeleteBin6Line } from '@remixicon/vue';
import { computed } from 'vue';
import {
  type IntelligentActorOption,
  type IntelligentCustomValueSource,
  type IntelligentFieldOption,
  type IntelligentJsonValue,
  type IntelligentNodeFieldValueSource,
  type IntelligentSourceFieldGroup,
  type IntelligentUpdateFilter,
  type IntelligentUpdateFilterOperator,
  intelligentUpdateOperatorOptions,
  isIntelligentUnaryUpdateOperator,
} from '../../../schema';
import TriggerOptionSelect from '../TriggerOptionSelect.vue';
import CustomFieldValueEditor from '../createRecord/CustomFieldValueEditor.vue';
import NodeFieldPicker from '../createRecord/NodeFieldPicker.vue';
import ValueSourceTypeSelect from '../createRecord/ValueSourceTypeSelect.vue';
import RecordFieldSelect from './RecordFieldSelect.vue';

defineOptions({ name: 'UpdateRecordFilterRow' });

const props = defineProps<{
  filter: IntelligentUpdateFilter;
  fields: readonly IntelligentFieldOption[];
  sourceGroups: readonly IntelligentSourceFieldGroup[];
  members?: readonly IntelligentActorOption[];
  departments?: readonly IntelligentActorOption[];
}>();
const emit = defineEmits<{
  change: [filter: IntelligentUpdateFilter];
  remove: [];
}>();

const selectedField = computed(() =>
  props.fields.find((field) => field.fieldId === props.filter.targetFieldId),
);
const unary = computed(() => isIntelligentUnaryUpdateOperator(props.filter.operator));
const sourceMode = computed<'node-field' | 'custom'>(() =>
  props.filter.source?.type === 'custom' ? 'custom' : 'node-field',
);
const nodeFieldSource = computed<IntelligentNodeFieldValueSource | null>(() =>
  props.filter.source?.type === 'node-field' ? props.filter.source : null,
);
const customSource = computed<IntelligentCustomValueSource | null>(() =>
  props.filter.source?.type === 'custom' ? props.filter.source : null,
);

function defaultCustomValue(field: IntelligentFieldOption): IntelligentJsonValue {
  return ['multi-choice', 'members', 'departments'].includes(field.valueKind) ? [] : null;
}

function selectField(fieldId: string): void {
  const field = props.fields.find((item) => item.fieldId === fieldId);
  if (!field) return;
  emit('change', {
    ...props.filter,
    targetFieldId: field.fieldId,
    targetWidgetName: field.widgetName,
    source: undefined,
  });
}

function selectOperator(value: string): void {
  const operator = value as IntelligentUpdateFilterOperator;
  emit('change', {
    ...props.filter,
    operator,
    source: isIntelligentUnaryUpdateOperator(operator) ? undefined : props.filter.source,
  });
}

function selectSourceMode(mode: 'node-field' | 'custom' | 'empty'): void {
  const field = selectedField.value;
  if (!field || mode === 'node-field') {
    emit('change', { ...props.filter, source: undefined });
    return;
  }
  emit('change', {
    ...props.filter,
    source: { type: 'custom', value: defaultCustomValue(field) },
  });
}

function selectNodeField(source: IntelligentNodeFieldValueSource): void {
  emit('change', { ...props.filter, source });
}

function updateCustomValue(value: IntelligentJsonValue): void {
  emit('change', { ...props.filter, source: { type: 'custom', value } });
}
</script>

<template>
  <div class="update-filter-row" :class="{ 'is-unary': unary }">
    <RecordFieldSelect
      :model-value="filter.targetFieldId"
      :options="fields"
      @update:model-value="selectField"
    />
    <TriggerOptionSelect
      :model-value="filter.operator"
      :options="intelligentUpdateOperatorOptions"
      control-label="选择筛选运算符"
      @update:model-value="selectOperator"
    />
    <div v-if="!unary" class="update-filter-row__source" :class="{ 'is-invalid': !filter.source }">
      <ValueSourceTypeSelect
        :model-value="sourceMode"
        :allow-empty="false"
        @update:model-value="selectSourceMode"
      />
      <NodeFieldPicker
        v-if="sourceMode === 'node-field' && selectedField"
        :model-value="nodeFieldSource"
        :target-field="selectedField"
        :groups="sourceGroups"
        placement="bottom"
        @update:model-value="selectNodeField"
      />
      <CustomFieldValueEditor
        v-else-if="selectedField"
        :model-value="customSource?.value ?? defaultCustomValue(selectedField)"
        :field="selectedField"
        :members="members"
        :departments="departments"
        @update:model-value="updateCustomValue"
      />
      <span v-else class="update-filter-row__placeholder">请先选择字段</span>
    </div>
    <button
      type="button"
      class="update-filter-row__remove"
      aria-label="删除筛选条件"
      @click="emit('remove')"
    >
      <RiDeleteBin6Line />
    </button>
  </div>
</template>

<style scoped lang="scss">
.update-filter-row {
  display: grid;
  align-items: center;
  grid-template-columns: minmax(210px, 1fr) minmax(155px, 0.62fr) minmax(350px, 1.5fr) 34px;
  gap: 12px;

  &.is-unary {
    grid-template-columns: minmax(210px, 1fr) minmax(155px, 0.62fr) 34px;
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
  &__placeholder {
    display: flex;
    padding: 0 15px;
    align-items: center;
    color: #a0a8b5;
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

@media (max-width: 980px) {
  .update-filter-row,
  .update-filter-row.is-unary {
    grid-template-columns: minmax(0, 1fr) 34px;
  }
  .update-filter-row > :not(.update-filter-row__remove) {
    grid-column: 1;
  }
  .update-filter-row__remove {
    grid-column: 2;
    grid-row: 1;
  }
}
</style>
