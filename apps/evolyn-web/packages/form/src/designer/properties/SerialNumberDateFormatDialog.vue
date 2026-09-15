<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue';
import {
  ElButton,
  ElDialog,
  ElFormItem,
  ElInput,
  ElOption,
  ElRadio,
  ElRadioGroup,
  ElSelect,
} from 'element-plus';
import type { SnDateFormatType, SnSubmittedAtRulePart } from '../../schema/types';

interface DateFormatOption {
  format: string;
  preview: string;
}

const PRESET_FORMATS: readonly DateFormatOption[] = [
  { format: 'yyyy', preview: '2015' },
  { format: 'yyyyMM', preview: '201501' },
  { format: 'yyyy-MM', preview: '2015-01' },
  { format: 'yyyy/MM', preview: '2015/01' },
  { format: 'yyyyMMdd', preview: '20150101' },
  { format: 'yyyy-MM-dd', preview: '2015-01-01' },
  { format: 'yyyy/MM/dd', preview: '2015/01/01' },
  { format: 'MMdd', preview: '0101' },
  { format: 'MM-dd', preview: '01-01' },
  { format: 'MM/dd', preview: '01/01' },
];

const props = defineProps<{ modelValue: boolean; rule: SnSubmittedAtRulePart | null }>();
const emit = defineEmits<{
  'update:modelValue': [visible: boolean];
  confirm: [rule: Pick<SnSubmittedAtRulePart, 'format' | 'formatType'>];
}>();

const formatType = shallowRef<SnDateFormatType>('preset');
const selectedPreset = shallowRef('yyyyMMdd');
const customFormat = shallowRef('yyyyMMdd');
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
});
const activeFormat = computed(() =>
  formatType.value === 'preset' ? selectedPreset.value : customFormat.value.trim(),
);
const formatPreview = computed(() => previewDateFormat(activeFormat.value));
const customFormatValid = computed(() => isDateFormat(activeFormat.value));

watch(
  () => props.modelValue,
  (opened) => {
    if (!opened || !props.rule) return;
    formatType.value = props.rule.formatType === 'custom' ? 'custom' : 'preset';
    selectedPreset.value = PRESET_FORMATS.some((option) => option.format === props.rule?.format)
      ? props.rule.format
      : 'yyyyMMdd';
    customFormat.value = props.rule.format;
  },
);

function isDateFormat(format: string): boolean {
  if (!format || format.length > 32) return false;
  const tokens = format.match(/yyyy|MM|dd|[-/]/g);
  return tokens?.join('') === format && /yyyy|MM|dd/.test(format);
}

function previewDateFormat(format: string): string {
  return format.replace(
    /yyyy|MM|dd/g,
    (token) => ({ yyyy: '2015', MM: '01', dd: '01' })[token] ?? token,
  );
}

function close(): void {
  visible.value = false;
}

function confirm(): void {
  if (!customFormatValid.value) return;
  emit('confirm', { format: activeFormat.value, formatType: formatType.value });
  close();
}
</script>

<template>
  <el-dialog v-model="visible" title="日期格式设置" width="640px" append-to-body>
    <div class="serial-date-format-dialog">
      <el-radio-group v-model="formatType" class="serial-date-format-dialog__mode">
        <el-radio value="preset">预定义格式</el-radio>
        <el-radio value="custom">自定义格式</el-radio>
      </el-radio-group>
      <el-form-item v-if="formatType === 'preset'" class="serial-date-format-dialog__control">
        <el-select v-model="selectedPreset" aria-label="预定义日期格式">
          <el-option
            v-for="option in PRESET_FORMATS"
            :key="option.format"
            :label="option.preview"
            :value="option.format"
          />
        </el-select>
      </el-form-item>
      <el-form-item
        v-else
        class="serial-date-format-dialog__control"
        :error="customFormatValid ? undefined : '仅支持 yyyy、MM、dd 与 -、/'"
      >
        <el-input v-model="customFormat" maxlength="32" aria-label="自定义日期格式" />
      </el-form-item>
      <p class="serial-date-format-dialog__preview">格式预览：{{ formatPreview || '—' }}</p>
    </div>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" :disabled="!customFormatValid" @click="confirm">确定</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.serial-date-format-dialog {
  display: grid;
  gap: 18px;

  &__mode {
    display: grid;
    justify-items: start;
    gap: 18px;
  }

  &__control {
    margin: 0 0 0 34px;
  }
  &__control :deep(.el-select),
  &__control :deep(.el-input) {
    width: 320px;
  }
  &__preview {
    margin: -12px 0 0 34px;
    font-size: 14px;
    color: var(--el-text-color-secondary);
  }
}
</style>
