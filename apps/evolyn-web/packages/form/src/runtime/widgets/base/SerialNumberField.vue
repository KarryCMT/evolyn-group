<script setup lang="ts">
import { computed } from 'vue';
import { fieldAriaDescribedBy, fieldInputId } from '../../field-dom';
import type { RuntimeFieldProps } from '../../types';

/** 流水号只展示后端已生成值；新建记录不预占号，因此保持空的禁用态占位。 */
const props = defineProps<RuntimeFieldProps>();
const inputId = computed(() => fieldInputId(props.item.widget.widgetName));
const value = computed(() => (typeof props.modelValue === 'string' ? props.modelValue : ''));
const describedBy = computed(() =>
  fieldAriaDescribedBy(props.item.widget.widgetName, props.item.description !== '', props.errors.length > 0),
);
</script>

<template>
  <input
    :id="inputId"
    class="evf-input"
    type="text"
    :value="value"
    disabled
    placeholder="自动生成无需填写"
    :aria-describedby="describedBy"
  />
</template>
