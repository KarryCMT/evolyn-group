<script setup lang="ts">
import type { DashboardDesignerSaveStatus } from '~/composables/useDashboardDesigner';
import { RiArrowLeftLine, RiEyeLine, RiSave3Line } from '@remixicon/vue';
import { computed } from 'vue';
import WorkspaceTitleEditor from '~/components/WorkspaceTitleEditor.vue';

const props = defineProps<{
  name: string;
  revision: number;
  dirty: boolean;
  saveStatus: DashboardDesignerSaveStatus;
  renaming?: boolean;
}>();
const emit = defineEmits<{
  back: [];
  save: [];
  preview: [];
  rename: [name: string, onSuccess: () => void];
}>();

const statusLabel = computed(() => {
  if (props.saveStatus === 'saving') return '正在保存';
  if (props.saveStatus === 'conflict') return '版本冲突';
  if (props.saveStatus === 'error') return '保存失败';
  if (props.dirty) return '未保存';
  if (props.saveStatus === 'saved') return '已保存';
  return '已同步';
});
</script>

<template>
  <header class="designer-toolbar">
    <button
      class="designer-toolbar__back"
      type="button"
      aria-label="返回应用"
      @click="emit('back')"
    >
      <RiArrowLeftLine />
    </button>
    <WorkspaceTitleEditor
      class="designer-toolbar__identity"
      :name="name"
      resource-label="仪表盘"
      :saving="renaming"
      @submit="(name, onSuccess) => emit('rename', name, onSuccess)"
    />
    <nav class="designer-toolbar__tabs" aria-label="仪表盘设计导航">
      <button class="designer-toolbar__tab is-active" type="button">
        仪表盘设计
      </button>
      <button class="designer-toolbar__tab" type="button" disabled title="扩展能力即将开放">
        扩展功能
      </button>
      <button class="designer-toolbar__tab" type="button" disabled title="发布能力即将开放">
        仪表盘发布
      </button>
    </nav>
    <div class="designer-toolbar__meta">
      <span class="designer-toolbar__revision">草稿 / {{ revision }}</span>
      <span
        class="designer-toolbar__status"
        :class="`designer-toolbar__status--${saveStatus}`"
        data-testid="dashboard-save-status"
      >
        <i />{{ statusLabel }}
      </span>
    </div>
    <div class="designer-toolbar__actions">
      <el-button :disabled="saveStatus === 'saving'" @click="emit('preview')">
        <RiEyeLine />预览
      </el-button>
      <el-button
        type="primary"
        :disabled="!dirty || saveStatus === 'saving'"
        :loading="saveStatus === 'saving'"
        @click="emit('save')"
      >
        <RiSave3Line />保存
      </el-button>
    </div>
  </header>
</template>

<style scoped>
/* Vue 的 :deep() 用于统一 Element Plus 按钮内图标尺寸。 */
/* stylelint-disable selector-pseudo-class-no-unknown */
.designer-toolbar {
  display: grid;
  grid-template-columns: 36px minmax(180px, 1fr) auto minmax(180px, 1fr) auto;
  gap: 12px;
  align-items: center;
  min-height: 56px;
  padding: 0 18px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.designer-toolbar__back {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  padding: 0;
  font-size: 19px;
  color: var(--el-text-color-regular);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;
}

.designer-toolbar__back:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.designer-toolbar__identity {
  min-width: 0;
}

.designer-toolbar__tabs {
  display: flex;
  gap: 38px;
  align-items: stretch;
  justify-content: center;
  height: 56px;
}

.designer-toolbar__tab {
  position: relative;
  padding: 0;
  font-size: 15px;
  color: var(--el-text-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
}

.designer-toolbar__tab.is-active {
  font-weight: 600;
  color: var(--el-color-primary);
}

.designer-toolbar__tab.is-active::after {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 3px;
  content: '';
  background: var(--el-color-primary);
}

.designer-toolbar__tab:disabled {
  color: var(--el-text-color-disabled);
  cursor: not-allowed;
}

.designer-toolbar__meta {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: flex-end;
}

.designer-toolbar__revision {
  font:
    700 11px/1 ui-monospace,
    monospace;
  color: var(--el-text-color-secondary);
  letter-spacing: 0.04em;
}

.designer-toolbar__status {
  display: inline-flex;
  gap: 7px;
  align-items: center;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.designer-toolbar__status i {
  width: 7px;
  height: 7px;
  background: var(--el-text-color-placeholder);
  border-radius: 50%;
}

.designer-toolbar__status--idle i {
  background: var(--el-color-warning);
}

.designer-toolbar__status--saving i {
  background: var(--el-color-primary);
  animation: pulse 1s infinite;
}

.designer-toolbar__status--saved i {
  background: var(--el-color-success);
}

.designer-toolbar__status--error i,
.designer-toolbar__status--conflict i {
  background: var(--el-color-danger);
}

.designer-toolbar__actions {
  display: flex;
  gap: 8px;
}

.designer-toolbar__actions :deep(svg) {
  width: 16px;
  margin-right: 5px;
}

@keyframes pulse {
  50% {
    opacity: 0.35;
    transform: scale(0.8);
  }
}

@media (width <= 760px) {
  .designer-toolbar {
    grid-template-columns: 36px 1fr auto;
  }

  .designer-toolbar__tabs,
  .designer-toolbar__revision,
  .designer-toolbar__status {
    display: none;
  }

  .designer-toolbar__meta {
    display: none;
  }
}
</style>
