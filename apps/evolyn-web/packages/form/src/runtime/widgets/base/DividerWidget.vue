<script setup lang="ts">
import { computed } from 'vue';
import type { SeparatorWidget } from '../../../schema/types';
import type { RuntimeFieldProps } from '../../types';

/**
 * 分割线（separator）：纯布局项，无值、无校验、不进入提交载荷。
 * label 作为分割线文案（允许空串）；style 控制实线/虚线；整行渲染。
 */
const props = defineProps<RuntimeFieldProps>();

const widget = computed(() => props.item.widget as SeparatorWidget);
const hasLabel = computed(() => props.item.label.trim() !== '');
const dashed = computed(() => widget.value.style === 'dashed');
</script>

<template>
  <div
    class="evf-divider"
    :class="{ 'evf-divider--dashed': dashed }"
    role="separator"
    :aria-label="hasLabel ? item.label : undefined"
  >
    <span v-if="hasLabel" class="evf-divider__label">{{ item.label }}</span>
  </div>
</template>
