<script setup lang="ts">
import { RiAddLine, RiFileList3Line, RiFlashlightLine, RiStopCircleLine } from '@remixicon/vue';
import { type Component, computed } from 'vue';
import type { IntelligentNodeType } from '../schema';

defineOptions({ name: 'IntelligentNodeCard' });

interface NodeProperties {
  assistantType?: IntelligentNodeType;
  label?: string;
  description?: string;
  selected?: boolean;
}

const props = defineProps<{
  node: { properties?: NodeProperties };
}>();

const nodeType = computed(() => props.node.properties?.assistantType ?? 'action');
const icon = computed<Component>(() => {
  if (nodeType.value === 'trigger') return RiFileList3Line;
  if (nodeType.value === 'end') return RiStopCircleLine;
  return RiFlashlightLine;
});
</script>

<template>
  <article
    class="intelligent-node-card"
    :class="[
      `intelligent-node-card--${nodeType}`,
      { 'intelligent-node-card--selected': node.properties?.selected },
    ]"
  >
    <span class="intelligent-node-card__icon" aria-hidden="true">
      <component :is="icon" />
    </span>
    <span class="intelligent-node-card__copy">
      <strong>{{ node.properties?.label }}</strong>
      <small v-if="node.properties?.description">{{ node.properties.description }}</small>
    </span>
    <RiAddLine v-if="nodeType === 'action'" class="intelligent-node-card__marker" />
  </article>
</template>

<style scoped lang="scss">
.intelligent-node-card {
  display: flex;
  width: 100%;
  height: 100%;
  padding: 14px 16px;
  align-items: flex-start;
  gap: 12px;
  color: #172033;
  background: #fff;
  border: 2px solid transparent;
  border-radius: 10px;
  box-shadow: 0 5px 14px rgb(31 43 61 / 14%);
  box-sizing: border-box;
  cursor: pointer;
  transition: border-color 160ms ease, box-shadow 160ms ease;

  &:hover,
  &--selected {
    border-color: #00afa2;
    box-shadow: 0 7px 18px rgb(0 175 162 / 16%);
  }

  &__icon {
    display: inline-flex;
    width: 32px;
    height: 32px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    color: #2f7cf6;
    background: #eaf2ff;
    border-radius: 50%;

    svg {
      width: 19px;
      height: 19px;
    }
  }

  &__copy {
    display: flex;
    min-width: 0;
    flex: 1;
    flex-direction: column;
    gap: 6px;

    strong,
    small {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 15px;
      font-weight: 600;
      line-height: 22px;
    }

    small {
      color: #7b8594;
      font-size: 13px;
      line-height: 18px;
    }
  }

  &__marker {
    width: 16px;
    height: 16px;
    margin-top: 8px;
    color: #00afa2;
  }

  &--action &__icon {
    color: #00a86b;
    background: #e8f8f1;
  }

  &--end {
    padding: 0 24px;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: #121d30;
    border-radius: 999px;

    &:hover,
    &.intelligent-node-card--selected {
      border-color: #121d30;
    }
  }

  &--end &__icon,
  &--end small {
    display: none;
  }

  &--end &__copy {
    flex: 0 1 auto;
  }

  &--end strong {
    color: #fff;
    text-align: center;
  }
}
</style>
