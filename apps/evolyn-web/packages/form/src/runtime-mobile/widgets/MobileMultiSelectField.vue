<script setup lang="ts">
import {
  Button as VanButton,
  Cell as VanCell,
  Checkbox as VanCheckbox,
  CheckboxGroup as VanCheckboxGroup,
  Field as VanField,
  Popup as VanPopup,
} from 'vant';
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue';
import type { ComboCheckWidget } from '../../schema/types';
import { readWidgetOptions } from '../../schema/codec';
import { fieldAriaDescribedBy, fieldInputId } from '../../runtime/field-dom';
import { useFormRendererContext } from '../../runtime/store/injection';
import type { FormValue, RuntimeFieldEmits, RuntimeFieldProps } from '../../runtime/types';

const props = defineProps<RuntimeFieldProps>();
const emit = defineEmits<RuntimeFieldEmits>();
const { runtime } = useFormRendererContext();

const widget = computed(() => props.item.widget as ComboCheckWidget);
const inputId = computed(() => fieldInputId(widget.value.widgetName));
const relatedOptions = ref<Array<{ label: string; value: string }>>([]);
const draft = ref<string[]>([]);
const loading = shallowRef(false);
const open = shallowRef(false);
let controller: AbortController | undefined;

const modelValue = computed(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as string[]) : [],
);
const options = computed(() => {
  const result =
    widget.value.optionSource?.mode === 'related'
      ? relatedOptions.value
      : readWidgetOptions(widget.value);
  const missing = modelValue.value.filter((value) => !result.some((option) => option.value === value));
  return [...missing.map((value) => ({ label: value, value })), ...result];
});
const displayValue = computed(() => {
  const labels = new Map(options.value.map((option) => [option.value, option.label]));
  return modelValue.value.map((value) => labels.get(value) ?? value).join('、');
});
const describedBy = computed(() =>
  fieldAriaDescribedBy(
    widget.value.widgetName,
    props.item.description !== '',
    props.errors.length > 0,
  ),
);

watch(open, (visible) => {
  if (visible) draft.value = [...modelValue.value];
});

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
    // 保留既有候选项，等待用户下次打开时重试。
  } finally {
    if (!controller.signal.aborted) loading.value = false;
  }
}

function showPicker(): void {
  if (props.disabled || props.readonly) return;
  open.value = true;
  void loadOptions();
}

function confirm(): void {
  emit('update:modelValue', [...draft.value] as FormValue);
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
    <section class="evf-mobile-multi-select" :aria-label="item.label">
      <header class="evf-mobile-multi-select__header">
        <VanButton size="small" plain @click="open = false">取消</VanButton>
        <strong>{{ item.label }}</strong>
        <VanButton size="small" type="primary" @click="confirm">确定</VanButton>
      </header>
      <VanCheckboxGroup v-model="draft">
        <VanCell v-for="option in options" :key="option.value" clickable>
          <template #title>
            <VanCheckbox :name="option.value">{{ option.label }}</VanCheckbox>
          </template>
        </VanCell>
      </VanCheckboxGroup>
    </section>
  </VanPopup>
</template>

<style scoped>
.evf-mobile-multi-select {
  max-height: min(70vh, 560px);
  overflow-y: auto;
}

.evf-mobile-multi-select__header {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 56px;
  padding: 0 var(--van-padding-md);
  background: var(--van-background-2);
  border-bottom: 1px solid var(--van-border-color);
}
</style>
