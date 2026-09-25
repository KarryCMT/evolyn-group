<script setup lang="ts">
import { RiAddLine, RiAiGenerate2, RiFileList3Line, RiSearchLine } from '@remixicon/vue';
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';
import {
  type IntelligentActorOption,
  type IntelligentFieldAssignment,
  type IntelligentFieldOption,
  type IntelligentSourceFieldGroup,
  quickFillIntelligentAssignments,
} from '../../../schema';
import UpdateAssignmentRow from './UpdateAssignmentRow.vue';

defineOptions({ name: 'UpdateRecordAssignmentEditor' });

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

const addRootRef = useTemplateRef<HTMLElement>('addRootRef');
const addOpen = shallowRef(false);
const keyword = shallowRef('');
const configuredIds = computed(() =>
  props.assignments.map((assignment) => assignment.targetFieldId),
);
const availableFields = computed(() => {
  const configured = new Set(configuredIds.value);
  const normalized = keyword.value.trim().toLocaleLowerCase();
  return props.fields.filter(
    (field) =>
      !configured.has(field.fieldId) &&
      (!normalized || `${field.label}${field.widgetName}`.toLocaleLowerCase().includes(normalized)),
  );
});

function addField(field: IntelligentFieldOption): void {
  emit('change', [
    ...props.assignments,
    {
      targetFieldId: field.fieldId,
      targetWidgetName: field.widgetName,
      source: { type: 'node-field', nodeId: '', field: '' },
    },
  ]);
  addOpen.value = false;
  keyword.value = '';
}

function updateAssignment(index: number, assignment: IntelligentFieldAssignment): void {
  emit(
    'change',
    props.assignments.map((item, itemIndex) => (itemIndex === index ? assignment : item)),
  );
}

function removeAssignment(index: number): void {
  emit(
    'change',
    props.assignments.filter((_item, itemIndex) => itemIndex !== index),
  );
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

function closeOnOutside(event: PointerEvent): void {
  if (!addRootRef.value?.contains(event.target as Node)) addOpen.value = false;
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', closeOnOutside));
</script>

<template>
  <section class="update-assignment-editor">
    <header>
      <div>
        <h3><b>*</b>设置字段值</h3>
        <div ref="addRootRef" class="update-assignment-editor__add-wrap">
          <button type="button" class="update-assignment-editor__add" @click="addOpen = !addOpen">
            <RiAddLine />添加字段
          </button>
          <div v-if="addOpen" class="update-assignment-editor__menu">
            <label><RiSearchLine /><input v-model="keyword" placeholder="搜索" autocomplete="off" /></label>
            <div>
              <button
                v-for="field in availableFields"
                :key="field.fieldId"
                type="button"
                @click="addField(field)"
              >
                <RiFileList3Line /><span>{{ field.label }}</span>
              </button>
              <p v-if="availableFields.length === 0">没有可添加字段</p>
            </div>
          </div>
        </div>
      </div>
      <button
        type="button"
        class="update-assignment-editor__quick"
        :disabled="fields.length === 0"
        @click="quickFill"
      >
        <RiAiGenerate2 />快捷填充
      </button>
    </header>

    <div class="update-assignment-editor__rows">
      <UpdateAssignmentRow
        v-for="(assignment, index) in assignments"
        :key="assignment.targetFieldId"
        :assignment="assignment"
        :fields="fields"
        :disabled-field-ids="configuredIds"
        :source-groups="sourceGroups"
        :members="members"
        :departments="departments"
        @change="updateAssignment(index, $event)"
        @remove="removeAssignment(index)"
      />
    </div>
    <p v-if="assignments.length === 0" class="update-assignment-editor__error">请添加字段</p>
  </section>
</template>

<style scoped lang="scss">
.update-assignment-editor {
  padding: 28px 32px 38px;

  > header {
    display: flex;
    margin-bottom: 16px;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }
  h3 {
    margin: 0 0 16px;
    font-size: 17px;
  }
  h3 b {
    margin-right: 2px;
    color: #ef5252;
  }
  &__add-wrap {
    position: relative;
    width: max-content;
  }
  &__add {
    display: inline-flex;
    padding: 3px 0;
    align-items: center;
    gap: 5px;
    color: #00a99d;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;
    font-size: 16px;

    svg {
      width: 22px;
      height: 22px;
    }
  }
  &__quick {
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

    &:disabled {
      color: #9fa7b3;
      background: #fafbfc;
      cursor: not-allowed;
    }
    svg {
      width: 21px;
      height: 21px;
      color: #28a8ed;
    }
  }
  &__menu {
    position: absolute;
    z-index: 96;
    top: calc(100% + 8px);
    left: 0;
    width: 330px;
    background: #fff;
    border: 1px solid #e1e5eb;
    border-radius: 9px;
    box-shadow: 0 14px 34px rgb(31 43 61 / 17%);

    > label {
      display: flex;
      height: 46px;
      padding: 0 13px;
      align-items: center;
      gap: 9px;
      border-bottom: 1px solid #e1e5eb;
    }
    > label svg {
      width: 20px;
      height: 20px;
    }
    > label input {
      min-width: 0;
      flex: 1;
      border: 0;
      outline: 0;
      font: inherit;
    }
    > div {
      display: grid;
      max-height: 320px;
      padding: 8px;
      overflow-y: auto;
      gap: 2px;
    }
    > div button {
      display: flex;
      min-height: 40px;
      padding: 0 11px;
      align-items: center;
      gap: 9px;
      color: #263247;
      background: transparent;
      border: 0;
      border-radius: 7px;
      cursor: pointer;
      font: inherit;
    }
    > div button:hover {
      background: #edf1f5;
    }
    > div button svg {
      width: 18px;
      height: 18px;
      color: #536075;
    }
    p {
      margin: 22px 0;
      color: #9aa3af;
      text-align: center;
    }
  }
  &__rows {
    display: grid;
    gap: 12px;
  }
  &__error {
    margin: 5px 0 0;
    color: #ef5252;
    font-size: 13px;
  }
}
</style>
