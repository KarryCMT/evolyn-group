<script setup lang="ts">
import { computed, useSlots } from 'vue';
import type { ButtonEmits, ButtonProps } from './Button.types';

defineOptions({
  name: 'EvolynButton',
  inheritAttrs: false,
});

const props = withDefaults(defineProps<ButtonProps>(), {
  autofocus: false,
  autoInsertSpace: false,
  bg: false,
  circle: false,
  disabled: false,
  link: false,
  loading: false,
  nativeType: 'button',
  plain: false,
  round: false,
  size: 'default',
  text: false,
  type: 'default',
});
const emit = defineEmits<ButtonEmits>();
const slots = useSlots();

const isDisabled = computed(() => props.disabled || props.loading);
const hasIcon = computed(() => props.loading || Boolean(props.icon));
const shouldAddSpace = computed(() => {
  if (!props.autoInsertSpace || hasIcon.value) return false;
  const children = slots.default?.();
  const text =
    children?.length === 1 && typeof children[0]?.children === 'string' ? children[0].children : '';
  return /^[\u4e00-\u9fa5]{2}$/.test(text);
});

function handleClick(event: MouseEvent) {
  if (!isDisabled.value) emit('click', event);
}
</script>

<template>
  <button
    class="evolyn-button"
    :class="[
      `evolyn-button--${props.type}`,
      `evolyn-button--${props.size === 'medium' ? 'default' : props.size}`,
      {
        'is-plain': props.plain,
        'is-text': props.text,
        'is-link': props.link,
        'is-bg': props.bg,
        'is-round': props.round,
        'is-circle': props.circle,
        'is-loading': props.loading,
        'is-disabled': isDisabled,
      },
    ]"
    :autofocus="props.autofocus"
    :disabled="isDisabled"
    :type="props.nativeType"
    :aria-disabled="isDisabled"
    v-bind="$attrs"
    @click="handleClick"
  >
    <span v-if="props.loading || props.icon" class="evolyn-button__icon" aria-hidden="true">
      <component
        v-if="props.loading && props.loadingIcon"
        :is="props.loadingIcon"
        class="is-loading"
      />
      <svg v-else-if="props.loading" class="is-loading" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2.5" />
      </svg>
      <component v-else :is="props.icon" />
    </span>
    <span
      v-if="$slots.default"
      class="evolyn-button__content"
      :class="{ 'is-spaced': shouldAddSpace }"
    >
      <slot />
    </span>
  </button>
</template>

<!-- 外链样式在部分库构建器中收集不稳定，保留 @use 以便产物包含组件样式。 -->
<style lang="scss">
@use './Button.scss' as *;
</style>
