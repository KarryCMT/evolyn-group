<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue';
import type { ComboWidget } from '../../../schema/types';
import { readWidgetOptions } from '../../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../types';
import { useFormRendererContext } from '../../store/injection';

/** 下拉单选（combo）：值协议 string|null；原生 select 最小实现。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();
const { runtime } = useFormRendererContext();

const widget = computed(() => props.item.widget as ComboWidget);
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const relatedOptions = ref<Array<{ label: string; value: string }>>([]);
const loading = shallowRef(false);
let controller: AbortController | undefined;
const placeholder = computed(() => widget.value.placeholder ?? '');
const modelValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const options = computed(() => {
  const result = widget.value.optionSource?.mode === 'related'
    ? relatedOptions.value
    : readWidgetOptions(widget.value);
  // 已存记录的历史值可能因源数据变更而不再候选，仍应可见但不会出现在新建选项中。
  if (modelValue.value && !result.some((option) => option.value === modelValue.value)) {
    return [{ label: modelValue.value, value: modelValue.value }, ...result];
  }
  return result;
});
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function onChange(event: Event): void {
  const raw = (event.target as HTMLSelectElement).value;
  emit('update:modelValue', raw === '' ? null : raw);
}
async function loadOptions(): Promise<void> {
  if (widget.value.optionSource?.mode !== 'related' || !runtime.value) return;
  controller?.abort(); controller = new AbortController(); loading.value = true;
  try { relatedOptions.value = await runtime.value.queryRelatedOptions(widget.value.widgetName, controller.signal); }
  catch { /* 保留已加载选项，不因短暂网络错误清空用户所见值。 */ }
  finally { if (!controller.signal.aborted) loading.value = false; }
}
onMounted(() => void loadOptions());
onBeforeUnmount(() => controller?.abort());
</script>

<template>
  <select
    :id="inputId"
    class="evf-select"
    :value="modelValue"
    :disabled="disabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="onChange"
    @focus="loadOptions"
    @blur="emit('blur')"
  >
    <option value="" :disabled="!item.widget.allowBlank">{{ loading ? '正在加载…' : (placeholder || '请选择') }}</option>
    <option v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </option>
  </select>
</template>
