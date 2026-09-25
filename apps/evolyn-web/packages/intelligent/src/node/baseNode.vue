<script setup lang="ts">
import { computed } from 'vue';
import type { IntelligentActionType, IntelligentNodeType } from '../schema';
import { ACTION_COLORS } from '../util';
import EventIcon from '../icon/eventIcon.vue';
import ReactionIcon from '../icon/reactionIcon.vue';

defineOptions({ name: 'IntelligentBaseNode' });

interface NodeProperties {
  assistantType?: IntelligentNodeType;
  actionType?: IntelligentActionType;
  configured?: boolean;
  label?: string;
  description?: string;
  selected?: boolean;
}

const props = defineProps<{
  node: { properties?: NodeProperties };
}>();

const nodeType = computed(() => props.node.properties?.assistantType ?? 'action');
const isInvalid = computed(
  () => nodeType.value !== 'end' && props.node.properties?.configured === false,
);
const actionColor = computed(() => {
  const actionType = props.node.properties?.actionType ?? 'create-record';
  return ACTION_COLORS[actionType];
});
</script>

<template>
  <article
    class="intelligent-base-node"
    :class="[
      `intelligent-base-node--${nodeType}`,
      {
        'intelligent-base-node--invalid': isInvalid,
        'intelligent-base-node--selected': node.properties?.selected,
      },
    ]"
    :aria-invalid="isInvalid"
  >
    <span
      v-if="nodeType !== 'end'"
      class="intelligent-base-node__icon"
      :style="{
        color: nodeType === 'trigger' ? '#2f7cf6' : actionColor.foreground,
        backgroundColor: nodeType === 'trigger' ? '#eaf2ff' : actionColor.background,
      }"
      aria-hidden="true"
    >
      <EventIcon v-if="nodeType === 'trigger'" size="18" />
      <ReactionIcon v-else size="18" />
    </span>
    <span class="intelligent-base-node__copy">
      <strong>{{ node.properties?.label }}</strong>
      <small v-if="node.properties?.description">{{ node.properties.description }}</small>
    </span>
    <span v-if="nodeType === 'action'" class="intelligent-base-node__add" aria-hidden="true">+</span>
  </article>
</template>

<style scoped lang="scss">
.intelligent-base-node {
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
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;

  &:hover {
    border-color: #2f7cf6;
    box-shadow: 0 0 0 5px rgb(47 124 246 / 12%), 0 8px 20px rgb(31 43 61 / 16%);
    transform: translateY(-1px);
  }

  &--selected {
    border-color: #2f7cf6;
    box-shadow: 0 0 0 5px rgb(47 124 246 / 14%), 0 8px 20px rgb(31 43 61 / 18%);
  }

  &--invalid {
    border-color: #ff4d4f;
  }

  &--invalid:hover,
  &--invalid.intelligent-base-node--selected {
    // 蓝色外环表达交互状态，红色描边持续表达校验失败，两个状态互不覆盖。
    border-color: #ff4d4f;
    box-shadow: 0 0 0 5px rgb(47 124 246 / 14%), 0 8px 20px rgb(31 43 61 / 18%);
  }

  &__icon {
    display: inline-flex;
    width: 32px;
    height: 32px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
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

    strong { font-size: 15px; font-weight: 600; line-height: 22px; }
    small { color: #7b8594; font-size: 13px; line-height: 18px; }
  }

  &__add {
    margin-top: 5px;
    color: #00a99d;
    font-size: 20px;
    line-height: 20px;
  }

  &--end {
    padding: 0 24px;
    align-items: center;
    justify-content: center;
    color: #fff;
    background: #121d30;
    border-radius: 999px;
  }

  &--end:hover,
  &--end.intelligent-base-node--selected {
    border-color: #2f7cf6;
  }

  &--end &__copy { flex: 0 1 auto; }
  &--end strong { color: #fff; text-align: center; }
}
</style>
