<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue';
import type { ComboCheckWidget } from '../../../schema/types';
import { readWidgetOptions } from '../../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { FormValue, RuntimeFieldEmits, RuntimeFieldProps } from '../../types';
import { useFormRendererContext } from '../../store/injection';

/** 下拉多选（combocheck）：值协议 string[]；原生 multiple select 最小实现。 */
const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();
const { runtime } = useFormRendererContext();

const widget = computed(() => props.item.widget as ComboCheckWidget);
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const relatedOptions = ref<Array<{ label: string; value: string }>>([]);
const loading = shallowRef(false);
let controller: AbortController | undefined;
const modelValue = computed(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as string[]) : [],
);
const options = computed(() => {
  const result = widget.value.optionSource?.mode === 'related'
    ? relatedOptions.value
    : readWidgetOptions(widget.value);
  const known = new Set(result.map((option) => option.value));
  const historical = modelValue.value
    .filter((value) => !known.has(value))
    .map((value) => ({ label: value, value }));
  return [...historical, ...result];
});
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    props.item.widget.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

function onChange(event: Event): void {
  const selected = Array.from((event.target as HTMLSelectElement).selectedOptions).map(
    (option) => option.value,
  );
  emit('update:modelValue', selected as FormValue);
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
    class="evf-select evf-select--multiple"
    multiple
    :value="modelValue"
    :disabled="disabled"
    :aria-required="!item.widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @change="onChange"
    @focus="loadOptions"
    @blur="emit('blur')"
  >
    <option v-if="loading" disabled value="">正在加载…</option>
    <option v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </option>
  </select>
</template>
