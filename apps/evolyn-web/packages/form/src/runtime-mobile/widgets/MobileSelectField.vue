<script setup lang="ts">
import { Field as VanField, Picker as VanPicker, Popup as VanPopup } from 'vant';
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue';
import type { ComboWidget } from '../../schema/types';
import { readWidgetOptions } from '../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId } from '../../runtime/field-dom';
import { useFormRendererContext } from '../../runtime/store/injection';
import type { RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';

interface PickerConfirmPayload {
  selectedValues: Array<string | number>;
}

const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();
const { runtime } = useFormRendererContext();

const widget = computed(() => props.item.widget as ComboWidget);
const inputId = computed(() => fieldInputId(widget.value.widgetName));
const relatedOptions = ref<Array<{ label: string; value: string }>>([]);
const loading = shallowRef(false);
const open = shallowRef(false);
let controller: AbortController | undefined;

const modelValue = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const options = computed(() => {
  const result =
    widget.value.optionSource?.mode === 'related'
      ? relatedOptions.value
      : readWidgetOptions(widget.value);
  if (modelValue.value && !result.some((option) => option.value === modelValue.value)) {
    return [{ label: modelValue.value, value: modelValue.value }, ...result];
  }
  return result;
});
const columns = computed(() => options.value.map((option) => ({ text: option.label, value: option.value })));
const displayValue = computed(
  () => options.value.find((option) => option.value === modelValue.value)?.label ?? modelValue.value,
);
const selectedValues = computed(() => (modelValue.value ? [modelValue.value] : []));
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    widget.value.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

async function loadOptions(): Promise<void> {
  if (widget.value.optionSource?.mode !== 'related' || !runtime.value) return;
  controller?.abort();
  controller = new AbortController();
  loading.value = true;
  try {
    relatedOptions.value = await runtime.value.queryRelatedOptions(
      widget.value.widgetName,
      controller.signal,
    );
  } catch {
    // 短暂网络错误不清空已加载内容，避免打开选择器时历史值突然消失。
  } finally {
    if (!controller.signal.aborted) loading.value = false;
  }
}

function showPicker(): void {
  if (props.disabled || props.readonly) return;
  open.value = true;
  void loadOptions();
}

function confirm(payload: PickerConfirmPayload): void {
  const value = payload.selectedValues[0];
  emit('update:modelValue', value === undefined ? null : String(value));
  emit('blur');
  open.value = false;
}

onMounted(() => void loadOptions());
onBeforeUnmount(() => controller?.abort());
</script>

<template>
  <VanField
    :id="inputId"
    class="evf-mobile-field"
    :model-value="displayValue"
    :placeholder="loading ? '正在加载…' : widget.placeholder || '请选择'"
    :disabled="disabled"
    readonly
    is-link
    :aria-required="!widget.allowBlank || undefined"
    :aria-invalid="errors.length > 0 || undefined"
    :aria-describedby="describedBy"
    @click="showPicker"
  />
  <VanPopup v-model:show="open" position="bottom" round teleport="body">
    <VanPicker
      :title="item.label"
      :columns="columns"
      :model-value="selectedValues"
      @confirm="confirm"
      @cancel="open = false"
    />
  </VanPopup>
</template>
