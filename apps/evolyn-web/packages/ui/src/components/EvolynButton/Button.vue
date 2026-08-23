<template>
  <button
    class="v-button"
    :class="[
      type ? `v-button--${type}` : '',
      size ? `v-button--${size}` : '',
      {
        'is-round': round,
        'is-disabled': disabled,
      },
    ]"
    :disabled="disabled"
    @click="handleClick"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
import type { ButtonEmits, ButtonProps } from './Button.types';

defineOptions({
  name: 'VButton',
});

const {
  type = undefined,
  size = 'medium',
  disabled = false,
  round = false,
} = defineProps<ButtonProps>();

const emit = defineEmits<ButtonEmits>();

const handleClick = (event: MouseEvent) => {
  if (!disabled) {
    emit('click', event);
  }
};
</script>

<!-- <style src="./Button.scss"></style> 这种外链写法在部分库构建器里不稳定，所以保留内联 @use 写法让样式跟随组件入口被收集。-->
<style lang="scss">
@use './Button.scss' as *;
</style>
