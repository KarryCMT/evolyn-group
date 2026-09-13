<script setup lang="ts">
import type { FormRecordMemberCard, FormRecordMemberReference } from '~/types';
import { computed, shallowRef, watch } from 'vue';
import { getFormRecordMemberCard } from '~/api/form';

defineOptions({ name: 'FormRecordMemberCardPopover' });

const props = defineProps<{
  formCode: string;
  references: FormRecordMemberReference[];
  position: { x: number; y: number } | null;
}>();

const visible = defineModel<boolean>({ required: true });
const selectedReference = shallowRef<string | null>(null);
const card = shallowRef<FormRecordMemberCard | null>(null);
const loading = shallowRef(false);
const errorMessage = shallowRef('');

const panelStyle = computed(() => {
  const x = props.position?.x ?? 24;
  const y = props.position?.y ?? 24;
  return {
    left: `${Math.max(16, Math.min(x, window.innerWidth - 360))}px`,
    top: `${Math.max(16, Math.min(y + 12, window.innerHeight - 360))}px`,
  };
});

const initials = computed(() => card.value?.name.trim().slice(-2) || '成员');

watch(
  () => [visible.value, props.references] as const,
  ([isVisible, references]) => {
    if (!isVisible) return;
    selectedReference.value = references[0]?.memberCode ?? null;
  },
  { immediate: true, deep: true },
);

watch(
  () => [visible.value, props.formCode, selectedReference.value] as const,
  async ([isVisible, formCode, reference], _, onCleanup) => {
    card.value = null;
    errorMessage.value = '';
    if (!isVisible || !formCode || !reference) return;

    const controller = new AbortController();
    onCleanup(() => controller.abort());
    loading.value = true;
    try {
      card.value = await getFormRecordMemberCard(formCode, reference, controller.signal);
    } catch {
      if (!controller.signal.aborted) {
        errorMessage.value = '成员信息暂时无法加载';
      }
    } finally {
      if (!controller.signal.aborted) loading.value = false;
    }
  },
  { immediate: true },
);

function selectReference(reference: string) {
  if (reference !== selectedReference.value) selectedReference.value = reference;
}
</script>

<template>
  <Teleport to="body">
    <section
      v-if="visible"
      class="member-card-popover"
      :style="panelStyle"
      role="dialog"
      aria-label="成员信息"
    >
      <button class="member-card-popover__close" type="button" aria-label="关闭" @click="visible = false">
        ×
      </button>
      <div v-if="references.length > 1" class="member-card-popover__members" aria-label="选择成员">
        <button
          v-for="reference in references"
          :key="reference.memberCode"
          type="button"
          :class="{ 'is-active': selectedReference === reference.memberCode }"
          @click="selectReference(reference.memberCode)"
        >
          {{ reference.name }}
        </button>
      </div>
      <p v-if="loading" class="member-card-popover__hint" aria-live="polite">
        正在加载成员信息…
      </p>
      <p v-else-if="errorMessage" class="member-card-popover__hint member-card-popover__hint--error" role="alert">
        {{ errorMessage }}
      </p>
      <template v-else-if="card">
        <header class="member-card-popover__header">
          <img v-if="card.avatar" class="member-card-popover__avatar" :src="card.avatar" alt="">
          <span v-else class="member-card-popover__avatar" aria-hidden="true">{{ initials }}</span>
          <div>
            <strong>{{ card.name }}</strong>
            <span>{{ card.status === 'disabled' ? '已停用成员' : '内部成员' }}</span>
          </div>
        </header>
        <dl class="member-card-popover__details">
          <div>
            <dt>成员编号</dt>
            <dd>{{ card.memberCode }}</dd>
          </div>
          <div>
            <dt>部门</dt>
            <dd>{{ card.departments.join('、') || '未加入部门' }}</dd>
          </div>
        </dl>
      </template>
    </section>
  </Teleport>
</template>

<style scoped lang="scss">
.member-card-popover {
  position: fixed;
  z-index: 3000;
  width: min(328px, calc(100vw - 32px));
  padding: 20px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  box-shadow: var(--el-box-shadow-dark);
}

.member-card-popover__close {
  position: absolute;
  top: 8px;
  right: 10px;
  width: 28px;
  height: 28px;
  padding: 0;
  color: var(--el-text-color-secondary);
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 50%;
}

.member-card-popover__close:hover { background: var(--el-fill-color-light); }

.member-card-popover__members {
  display: flex;
  margin: -4px 28px 16px -4px;
  gap: 4px;
  flex-wrap: wrap;
}

.member-card-popover__members button {
  padding: 3px 8px;
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;
}

.member-card-popover__members button.is-active { background: var(--el-color-primary-light-9); }

.member-card-popover__header {
  display: flex;
  min-height: 56px;
  align-items: center;
  gap: 14px;
}

.member-card-popover__avatar {
  display: grid;
  width: 56px;
  height: 56px;
  place-items: center;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  background: linear-gradient(135deg, #39a75d, #8ac600);
  border-radius: 50%;
}

.member-card-popover__header strong,
.member-card-popover__header > div > span { display: block; }
.member-card-popover__header strong { font-size: 18px; }
.member-card-popover__header div span {
  width: fit-content;
  margin-top: 6px;
  padding: 2px 6px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 3px;
}

.member-card-popover__details {
  margin: 18px 0 0;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.member-card-popover__details div { display: grid; grid-template-columns: 84px 1fr; gap: 12px; }
.member-card-popover__details div + div { margin-top: 12px; }
.member-card-popover__details dt { color: var(--el-text-color-secondary); }
.member-card-popover__details dd { margin: 0; overflow-wrap: anywhere; }
.member-card-popover__hint { margin: 28px 0 8px; color: var(--el-text-color-secondary); }
.member-card-popover__hint--error { color: var(--el-color-danger); }
</style>
