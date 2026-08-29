<script setup lang="ts">
import { ElButton, ElDropdown, ElDropdownItem, ElDropdownMenu } from 'element-plus';
import { RiMoreFill } from '@remixicon/vue';
import { computed } from 'vue';
import type { FormIssue } from '../types';
import type {
  FormRuntimeActionDefinition,
  FormRuntimeActionIntent,
  FormRuntimeLayout,
} from '../actions/types';

defineOptions({ name: 'FormRuntimeActionBar' });

const props = withDefaults(
  defineProps<{
    actions: readonly FormRuntimeActionDefinition[];
    issues?: readonly FormIssue[];
    layout?: FormRuntimeLayout;
    formDomId: string;
    contentWidth?: string;
  }>(),
  { issues: () => [], layout: 'auto', contentWidth: '860px' },
);

const emit = defineEmits<{
  action: [action: FormRuntimeActionDefinition];
}>();

defineSlots<{
  actionLabel(props: { action: FormRuntimeActionDefinition }): unknown;
}>();

const orderedActions = computed(() =>
  props.actions
    .filter((action) => action.visible !== false)
    .slice()
    .sort((left, right) => (left.order ?? 0) - (right.order ?? 0)),
);

const desktopDirectActions = computed(() => pickDirectActions(orderedActions.value, 3, false));
const desktopOverflowActions = computed(() =>
  orderedActions.value.filter(
    (action) => !desktopDirectActions.value.some((direct) => direct.key === action.key),
  ),
);
const mobileDirectActions = computed(() => pickDirectActions(orderedActions.value, 2, true));
const mobileOverflowActions = computed(() =>
  orderedActions.value.filter(
    (action) => !mobileDirectActions.value.some((direct) => direct.key === action.key),
  ),
);

const actionBarClasses = computed(() => [
  'evf-runtime-action-bar',
  `evf-runtime-action-bar--${props.layout}`,
]);

function pickDirectActions(
  actions: readonly FormRuntimeActionDefinition[],
  limit: number,
  mobile: boolean,
): FormRuntimeActionDefinition[] {
  const candidates = mobile
    ? actions.filter((action) => action.mobilePresentation !== 'overflow')
    : [...actions];
  if (candidates.length <= limit) return candidates;

  // 主要动作始终保留在可见区域，再按业务顺序补齐剩余位置。
  const primary = candidates.filter((action) => action.intent === 'primary');
  const selected = [...primary.slice(-1)];
  for (const action of candidates) {
    if (selected.length >= limit) break;
    if (!selected.some((item) => item.key === action.key)) selected.push(action);
  }
  return selected.sort((left, right) => (left.order ?? 0) - (right.order ?? 0));
}

function elementType(intent: FormRuntimeActionIntent | undefined) {
  if (intent === 'primary') return 'primary';
  if (intent === 'danger') return 'danger';
  return 'default';
}

function triggerAction(action: FormRuntimeActionDefinition, event?: Event): void {
  event?.preventDefault();
  if (action.disabled || action.loading) return;
  emit('action', action);
}

function triggerByKey(key: string, actions: readonly FormRuntimeActionDefinition[]): void {
  const action = actions.find((item) => item.key === key);
  if (action) triggerAction(action);
}
</script>

<template>
  <footer :class="actionBarClasses" aria-label="表单操作">
    <div
      v-if="props.issues.length"
      class="evf-runtime-action-bar__issues"
      :style="{ maxWidth: props.contentWidth }"
      aria-live="polite"
    >
      <p
        v-for="(issue, index) in props.issues"
        :key="`${issue.message}-${index}`"
        class="evf-runtime-action-bar__issue"
      >
        {{ issue.message }}
      </p>
    </div>

    <div class="evf-runtime-action-bar__content" :style="{ maxWidth: props.contentWidth }">
      <div class="evf-runtime-action-bar__layout evf-runtime-action-bar__layout--desktop">
        <div class="evf-runtime-action-bar__group">
          <ElButton
            v-for="action in desktopDirectActions"
            :key="action.key"
            :type="elementType(action.intent)"
            :plain="action.intent === 'secondary' || action.intent === 'plain'"
            :disabled="action.disabled"
            :loading="action.loading"
            :native-type="action.behavior === 'submit' ? 'submit' : 'button'"
            :form="action.behavior === 'submit' ? props.formDomId : undefined"
            :data-action-key="action.key"
            @click="triggerAction(action, $event)"
          >
            <slot name="actionLabel" :action="action">{{ action.label }}</slot>
          </ElButton>
        </div>

        <ElDropdown
          v-if="desktopOverflowActions.length"
          trigger="click"
          @command="triggerByKey($event, desktopOverflowActions)"
        >
          <ElButton class="evf-runtime-action-bar__more" aria-label="更多表单操作">
            <RiMoreFill aria-hidden="true" />
            <span>更多</span>
          </ElButton>
          <template #dropdown>
            <ElDropdownMenu>
              <ElDropdownItem
                v-for="action in desktopOverflowActions"
                :key="action.key"
                :command="action.key"
                :disabled="action.disabled || action.loading"
              >
                <slot name="actionLabel" :action="action">{{ action.label }}</slot>
              </ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>
      </div>

      <div class="evf-runtime-action-bar__layout evf-runtime-action-bar__layout--mobile">
        <ElDropdown
          v-if="mobileOverflowActions.length"
          trigger="click"
          @command="triggerByKey($event, mobileOverflowActions)"
        >
          <ElButton class="evf-runtime-action-bar__mobile-more" circle aria-label="更多表单操作">
            <RiMoreFill aria-hidden="true" />
          </ElButton>
          <template #dropdown>
            <ElDropdownMenu>
              <ElDropdownItem
                v-for="action in mobileOverflowActions"
                :key="action.key"
                :command="action.key"
                :disabled="action.disabled || action.loading"
              >
                <slot name="actionLabel" :action="action">{{ action.label }}</slot>
              </ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>

        <ElButton
          v-for="action in mobileDirectActions"
          :key="action.key"
          class="evf-runtime-action-bar__mobile-action"
          :class="{
            'evf-runtime-action-bar__mobile-action--primary': action.intent === 'primary',
          }"
          :type="elementType(action.intent)"
          :plain="action.intent === 'secondary' || action.intent === 'plain'"
          :disabled="action.disabled"
          :loading="action.loading"
          :native-type="action.behavior === 'submit' ? 'submit' : 'button'"
          :form="action.behavior === 'submit' ? props.formDomId : undefined"
          :data-action-key="action.key"
          @click="triggerAction(action, $event)"
        >
          <slot name="actionLabel" :action="action">{{ action.label }}</slot>
        </ElButton>
      </div>
    </div>
  </footer>
</template>
