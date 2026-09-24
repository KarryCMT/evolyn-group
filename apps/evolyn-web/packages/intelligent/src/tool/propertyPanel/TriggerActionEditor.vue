<script setup lang="ts">
import { RiAddLine, RiDeleteBin6Line } from '@remixicon/vue';
import { onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue';
import type { IntelligentFormFieldOption } from '../../mock/nodeTemplates';
import type {
  FormTriggerAction,
  FormTriggerActionType,
  FormTriggerUpdateScope,
} from '../../schema';
import TriggerFieldSelect from './TriggerFieldSelect.vue';
import TriggerOptionSelect from './TriggerOptionSelect.vue';

defineOptions({ name: 'TriggerActionEditor' });

defineProps<{
  fields: readonly IntelligentFormFieldOption[];
}>();

const actions = defineModel<FormTriggerAction[]>({ default: () => [] });
const rootRef = useTemplateRef<HTMLElement>('rootRef');
const menuOpen = shallowRef(false);
const actionOptions: readonly { label: string; value: FormTriggerActionType }[] = [
  { value: 'create', label: '新增数据时' },
  { value: 'update', label: '修改数据时' },
  { value: 'delete', label: '删除数据时' },
];
const updateScopeOptions = [
  { value: 'any-field', label: '任意字段' },
  { value: 'specified-fields', label: '任意指定字段' },
] as const;
function createId(): string {
  return `trigger_action_${globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2)}`;
}

// 同类动作允许重复配置，例如针对不同字段分别建立多条“修改数据时”规则。
function addAction(type: FormTriggerActionType): void {
  actions.value = [
    ...actions.value,
    type === 'update'
      ? {
          id: createId(),
          type,
          updateScope: 'specified-fields',
          fieldIds: ['employee_name'],
        }
      : { id: createId(), type },
  ];
  menuOpen.value = false;
}

function removeAction(actionId: string): void {
  actions.value = actions.value.filter((action) => action.id !== actionId);
}

function updateAction(actionId: string, patch: Partial<FormTriggerAction>): void {
  actions.value = actions.value.map((action) =>
    action.id === actionId
      ? {
          ...action,
          ...patch,
          fieldIds: patch.fieldIds ? [...patch.fieldIds] : action.fieldIds,
        }
      : action,
  );
}

function updateScope(actionId: string, value: string): void {
  const updateScope = value as FormTriggerUpdateScope;
  updateAction(actionId, {
    updateScope,
    fieldIds: updateScope === 'specified-fields' ? ['employee_name'] : [],
  });
}

function updateField(actionId: string, fieldId: string): void {
  updateAction(actionId, { fieldIds: fieldId ? [fieldId] : [] });
}

function actionLabel(type: FormTriggerActionType): string {
  return actionOptions.find((option) => option.value === type)?.label ?? '';
}

function closeOnOutside(event: PointerEvent): void {
  if (!rootRef.value?.contains(event.target as Node)) menuOpen.value = false;
}

onMounted(() => document.addEventListener('pointerdown', closeOnOutside));
onBeforeUnmount(() => document.removeEventListener('pointerdown', closeOnOutside));
</script>

<template>
  <section ref="rootRef" class="trigger-action-editor">
    <h3><span>*</span>触发动作</h3>
    <div class="trigger-action-editor__add-wrap">
      <button
        type="button"
        class="trigger-action-editor__add"
        :aria-expanded="menuOpen"
        @click="menuOpen = !menuOpen"
      >
        <RiAddLine />添加动作
      </button>
      <div v-if="menuOpen" class="trigger-action-editor__menu">
        <small>当表单事件中</small>
        <button
          v-for="option in actionOptions"
          :key="option.value"
          type="button"
          @click="addAction(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
    </div>

    <p v-if="actions.length === 0" class="trigger-action-editor__empty">请至少添加一个触发动作</p>

    <div
      v-for="(action, index) in actions"
      :key="action.id"
      class="trigger-action-editor__row"
      :class="{ 'is-update': action.type === 'update' }"
    >
      <span class="trigger-action-editor__join">{{ index === 0 ? '当' : '或' }}</span>
      <strong>{{ actionLabel(action.type) }}</strong>
      <template v-if="action.type === 'update'">
        <TriggerOptionSelect
          :model-value="action.updateScope ?? 'specified-fields'"
          :options="updateScopeOptions"
          control-label="修改字段范围"
          @update:model-value="updateScope(action.id, $event)"
        />
        <TriggerFieldSelect
          v-if="action.updateScope !== 'any-field'"
          :model-value="action.fieldIds?.[0] ?? ''"
          :options="fields"
          control-label="选择修改字段"
          @update:model-value="updateField(action.id, $event)"
        />
        <span>修改时</span>
      </template>
      <button
        type="button"
        class="trigger-action-editor__remove"
        :aria-label="`删除${actionLabel(action.type)}`"
        @click="removeAction(action.id)"
      >
        <RiDeleteBin6Line />
      </button>
    </div>
  </section>
</template>

<style scoped lang="scss">
.trigger-action-editor {
  position: relative;
  padding: 26px 32px 30px;

  h3 { margin: 0 0 20px; font-size: 17px; font-weight: 650; }
  h3 span { color: #f04f4f; }

  &__add-wrap { position: relative; width: max-content; }
  &__add {
    display: inline-flex;
    padding: 4px 0;
    align-items: center;
    gap: 5px;
    color: #00a99d;
    background: transparent;
    border: 0;
    cursor: pointer;
    font: inherit;
    font-size: 16px;

    svg { width: 22px; height: 22px; }
  }

  &__menu {
    position: absolute;
    z-index: 32;
    top: calc(100% + 8px);
    left: 0;
    display: grid;
    width: 220px;
    padding: 12px;
    background: #fff;
    border: 1px solid #e2e6ec;
    border-radius: 8px;
    box-shadow: 0 12px 30px rgb(31 43 61 / 17%);

    small { padding: 4px 8px 8px; color: #687386; }
    button {
      padding: 9px 8px;
      color: #8993a1;
      background: transparent;
      border: 0;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;
      text-align: left;

      &:hover { color: #263247; background: #eef8f7; }
    }
  }

  &__empty { margin: 18px 0 0; color: #e05252; font-size: 13px; }

  &__row {
    display: grid;
    min-height: 46px;
    margin-top: 8px;
    align-items: center;
    grid-template-columns: 30px minmax(130px, 1fr) 28px;
    gap: 8px;
    color: #253147;

    &.is-update {
      grid-template-columns: 30px 125px minmax(150px, 0.9fr) minmax(180px, 1.2fr) auto 28px;
    }

    strong { font-weight: 500; }
  }

  &__join { color: #697487; }
  &__remove {
    display: inline-flex;
    width: 28px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: #667286;
    background: transparent;
    border: 0;
    cursor: pointer;

    svg { width: 18px; height: 18px; }
  }
}

@media (max-width: 980px) {
  .trigger-action-editor {
    padding: 22px 20px;

    &__row,
    &__row.is-update { grid-template-columns: 26px minmax(0, 1fr) 28px; }
    &__row.is-update > :not(.trigger-action-editor__join, strong, .trigger-action-editor__remove) {
      grid-column: 2;
    }
  }
}
</style>
