<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { RiQuestionFill } from '@remixicon/vue';
import {
  ElButton,
  ElDialog,
  ElFormItem,
  ElIcon,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTooltip,
} from 'element-plus';
import type { SnCounterRulePart } from '../../schema/types';

const props = defineProps<{ modelValue: boolean; rule: SnCounterRulePart | null }>();
const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  confirm: [rule: SnCounterRulePart];
}>();

const draft = ref<SnCounterRulePart | null>(null);
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
});

watch(
  () => props.modelValue,
  (opened) => {
    if (opened && props.rule) draft.value = { ...props.rule };
    if (!opened) draft.value = null;
  },
);

function close(): void {
  visible.value = false;
}

function confirm(): void {
  if (!draft.value) return;
  emit('confirm', draft.value);
  close();
}
</script>

<template>
  <el-dialog v-model="visible" title="计数设置" width="640px" append-to-body>
    <template v-if="draft">
      <el-form-item label="计数位数">
        <el-input-number v-model="draft.digits" :min="3" :max="8" :step="1" controls-position="right" />
      </el-form-item>
      <el-form-item>
        <template #label>
          <span class="serial-counter-dialog__label">位数固定
            <el-tooltip content="开启后，会根据计数位数显示计数值" placement="top">
              <el-icon><RiQuestionFill /></el-icon>
            </el-tooltip>
          </span>
        </template>
        <el-switch v-model="draft.fixedWidth" />
      </el-form-item>
      <el-form-item label="重置周期">
        <el-select v-model="draft.resetCycle">
          <el-option label="不自动重置" value="none" />
          <el-option label="每日重置" value="daily" />
          <el-option label="每月重置" value="monthly" />
          <el-option label="每年重置" value="yearly" />
        </el-select>
      </el-form-item>
      <el-form-item label="初始值">
        <el-input-number v-model="draft.initialValue" :min="1" :max="99999999" :step="1" controls-position="right" />
      </el-form-item>
    </template>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" @click="confirm">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.serial-counter-dialog {
  &__label { display: inline-flex; align-items: center; gap: 4px; }
  &__label :deep(.el-icon) { color: var(--el-text-color-secondary); cursor: help; }
}
</style>
