<script setup lang="ts">
import { RiAiGenerate2 } from '@remixicon/vue';
import type {
  IntelligentActorOption,
  IntelligentFieldAssignment,
  IntelligentFieldOption,
  IntelligentSourceFieldGroup,
} from '../../../schema';
import { quickFillIntelligentAssignments } from '../../../schema';
import FieldAssignmentRow from './FieldAssignmentRow.vue';

defineOptions({ name: 'IntelligentFieldAssignmentList' });

const props = defineProps<{
  fields: readonly IntelligentFieldOption[];
  assignments: readonly IntelligentFieldAssignment[];
  sourceGroups: readonly IntelligentSourceFieldGroup[];
  members?: readonly IntelligentActorOption[];
  departments?: readonly IntelligentActorOption[];
}>();

const emit = defineEmits<{
  change: [assignments: IntelligentFieldAssignment[]];
  quickFill: [filledCount: number];
}>();

function assignmentFor(fieldId: string): IntelligentFieldAssignment | undefined {
  return props.assignments.find((item) => item.targetFieldId === fieldId);
}

function updateAssignment(
  field: IntelligentFieldOption,
  nextAssignment: IntelligentFieldAssignment | null,
): void {
  const next = props.assignments.filter((item) => item.targetFieldId !== field.fieldId);
  if (nextAssignment) next.push(nextAssignment);
  emit('change', next);
}

function quickFill(): void {
  const result = quickFillIntelligentAssignments(
    props.fields,
    props.sourceGroups,
    props.assignments,
  );
  if (result.filledCount > 0) emit('change', result.assignments);
  emit('quickFill', result.filledCount);
}
</script>

<template>
  <section class="field-assignment-list">
    <header class="field-assignment-list__header">
      <div>
        <h3>设置字段值</h3>
        <p>已配置 {{ assignments.length }}/{{ fields.length }} 个字段</p>
      </div>
      <button type="button" @click="quickFill">
        <RiAiGenerate2 aria-hidden="true" />快捷填充
      </button>
    </header>

    <div class="field-assignment-list__rows">
      <FieldAssignmentRow
        v-for="field in fields"
        :key="field.fieldId"
        :field="field"
        :assignment="assignmentFor(field.fieldId)"
        :source-groups="sourceGroups"
        :members="members"
        :departments="departments"
        @change="updateAssignment(field, $event)"
      />
      <p v-if="fields.length === 0" class="field-assignment-list__empty">
        目标表单没有可赋值字段
      </p>
    </div>
  </section>
</template>

<style scoped lang="scss">
.field-assignment-list {
  padding: 28px 32px 36px;

  &__header {
    display: flex;
    margin-bottom: 14px;
    align-items: center;
    justify-content: space-between;
    gap: 16px;

    h3 { margin: 0; color: #172033; font-size: 17px; }
    p { margin: 5px 0 0; color: #8b95a4; font-size: 12px; }
    button {
      display: inline-flex;
      height: 42px;
      padding: 0 14px;
      align-items: center;
      gap: 8px;
      color: #273247;
      background: #fff;
      border: 1px solid #d6dce6;
      border-radius: 7px;
      cursor: pointer;
      font: inherit;

      &:hover { color: #148ed8; border-color: #8bcdf0; }
      svg { width: 21px; height: 21px; color: #28a8ed; }
    }
  }

  &__rows { display: grid; gap: 12px; }
  &__empty { margin: 30px 0; color: #98a1ae; text-align: center; }
}
</style>
