<script setup lang="ts">
import { RiFileList3Line } from '@remixicon/vue';
import { computed } from 'vue';
import { mockFormFields } from '../../mock/nodeTemplates';
import type {
  FormTriggerAction,
  FormTriggerCondition,
  FormTriggerConditionMode,
  IntelligentTrigger,
} from '../../schema';
import TriggerActionEditor from './TriggerActionEditor.vue';
import TriggerConditionEditor from './TriggerConditionEditor.vue';

defineOptions({ name: 'FormTriggerPropertyPanel' });

const props = defineProps<{
  trigger: IntelligentTrigger;
}>();

const emit = defineEmits<{
  update: [patch: Partial<IntelligentTrigger>];
}>();

// 子编辑器只操作各自片段，统一通过 trigger patch 回写画布文档事实源。
const actions = computed<FormTriggerAction[]>({
  get: () => props.trigger.actions,
  set: (value) => emit('update', { actions: value }),
});
const conditions = computed<FormTriggerCondition[]>({
  get: () => props.trigger.conditions,
  set: (value) => emit('update', { conditions: value }),
});
const conditionMode = computed<FormTriggerConditionMode>({
  get: () => props.trigger.conditionMode,
  set: (value) => emit('update', { conditionMode: value }),
});
</script>

<template>
  <div class="form-trigger-property-panel">
    <section class="form-trigger-property-panel__form">
      <h3>触发表单</h3>
      <div class="form-trigger-property-panel__form-value">
        <span>{{ trigger.formName || '当前表单' }}</span>
        <RiFileList3Line aria-hidden="true" />
      </div>
    </section>

    <TriggerActionEditor v-model="actions" :fields="mockFormFields" />
    <TriggerConditionEditor
      v-model:conditions="conditions"
      v-model:mode="conditionMode"
      :actions="actions"
      :fields="mockFormFields"
    />
  </div>
</template>

<style scoped lang="scss">
.form-trigger-property-panel {
  min-height: 100%;
  color: #172033;
  background: #fff;

  &__form { padding: 26px 32px 0; }
  &__form h3 { margin: 0 0 14px; font-size: 17px; font-weight: 650; }
  &__form-value {
    display: flex;
    height: 44px;
    padding: 0 14px;
    align-items: center;
    justify-content: space-between;
    color: #9aa3af;
    background: #f7f8fa;
    border: 1px solid #d9dfe8;
    border-radius: 6px;

    svg { width: 19px; height: 19px; color: #536075; }
  }
}

@media (max-width: 980px) {
  .form-trigger-property-panel__form { padding: 22px 20px 0; }
}
</style>
