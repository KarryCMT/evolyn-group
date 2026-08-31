<script setup lang="ts">
import { computed } from 'vue';
import type { SeparatorWidget } from '../../../schema/types';
import { sanitizeRichTextDescription } from '../../../schema/richTextDescription';
import type { RuntimeFieldProps } from '../../types';

/**
 * 分割线（separator）：纯布局项，无值、无校验、不进入提交载荷。
 * 描述信息优先作为分割线文案；未填写描述时再回退至分割线自身文案。
 * 其余配置与 Element Plus Divider 三个属性对齐。
 */
const props = defineProps<RuntimeFieldProps>();

const widget = computed(() => props.item.widget as SeparatorWidget);
const descriptionHtml = computed(() => sanitizeRichTextDescription(props.item.description));
const hasDescription = computed(() => descriptionHtml.value !== '');
const hasLabel = computed(() => hasDescription.value || Boolean(widget.value.content?.trim()));
const direction = computed(() => widget.value.direction ?? 'horizontal');
const borderStyle = computed(() => widget.value.borderStyle ?? widget.value.style ?? 'solid');
const contentPosition = computed(() => widget.value.contentPosition ?? 'center');
</script>

<template>
  <div
    class="evf-divider"
    :class="[`evf-divider--${direction}`, `evf-divider--content-${contentPosition}`]"
    :style="{ '--evf-divider-border-style': borderStyle }"
    role="separator"
    :aria-label="hasDescription ? undefined : (hasLabel ? widget.content : undefined)"
  >
    <span
      v-if="hasDescription"
      class="evf-divider__label"
      v-html="descriptionHtml"
    />
    <span v-else-if="hasLabel" class="evf-divider__label">{{ widget.content }}</span>
  </div>
</template>
