<script setup lang="ts">
import { Button as VanButton } from 'vant';
import { computed } from 'vue';
import type { FormIssue } from '../runtime/types';
import type { FormRuntimeActionDefinition } from '../runtime/actions/types';

const props = withDefaults(
  defineProps<{
    actions: readonly FormRuntimeActionDefinition[];
    issues?: readonly FormIssue[];
  }>(),
  { issues: () => [] },
);

const emit = defineEmits<{ action: [action: FormRuntimeActionDefinition] }>();

const orderedActions = computed(() =>
  props.actions
    .filter((action) => action.visible !== false)
    .slice()
    .sort((left, right) => (left.order ?? 0) - (right.order ?? 0)),
);
const directActions = computed(() =>
  orderedActions.value.filter((action) => action.mobilePresentation !== 'overflow'),
);

function emitAction(action: FormRuntimeActionDefinition): void {
  if (action.disabled || action.loading) return;
  emit('action', action);
}
</script>

<template>
  <footer class="evf-mobile-action-bar" aria-label="表单操作">
    <p v-for="(issue, index) in props.issues" :key="index" class="evf-mobile-action-bar__issue">
      {{ issue.message }}
    </p>
    <div class="evf-mobile-action-bar__actions">
      <VanButton
        v-for="action in directActions"
        :key="action.key"
        class="evf-mobile-action-bar__action"
        :type="action.intent === 'primary' ? 'primary' : action.intent === 'danger' ? 'danger' : 'default'"
        size="large"
        round
        :disabled="action.disabled || action.loading"
        :loading="action.loading"
        :data-action-key="action.key"
        @click="emitAction(action)"
      >
        {{ action.label }}
      </VanButton>
    </div>
  </footer>
</template>

<style scoped lang="scss">
.evf-mobile-action-bar {
  display: flex;
  flex: 0 0 auto;
  flex-direction: column;
  gap: var(--evf-space-md);
  padding: var(--evf-space-lg) var(--evf-space-xl)
    max(var(--evf-space-lg), env(safe-area-inset-bottom));
  background: var(--evf-color-bg);
  border-top: 1px solid var(--evf-color-border-lighter);
}

.evf-mobile-action-bar__issue {
  margin: 0;
  font-size: var(--evf-font-size-extra-small);
  color: var(--evf-color-danger);
}

.evf-mobile-action-bar__actions {
  display: flex;
  gap: var(--evf-space-lg);
}

.evf-mobile-action-bar__action {
  flex: 1;
  min-width: 0;
}

.evf-mobile-action-bar__action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
</style>
