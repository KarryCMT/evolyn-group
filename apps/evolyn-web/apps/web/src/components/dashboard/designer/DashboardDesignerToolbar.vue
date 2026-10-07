<script setup lang="ts">
import type { DashboardDesignerSaveStatus } from '~/composables/useDashboardDesigner';
import { RiArrowLeftLine, RiEyeLine, RiSave3Line } from '@remixicon/vue';
import { computed } from 'vue';

const props = defineProps<{
  name: string;
  revision: number;
  dirty: boolean;
  saveStatus: DashboardDesignerSaveStatus;
}>();
const emit = defineEmits<{ back: []; save: []; preview: [] }>();

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
    <div class="designer-toolbar__identity">
      <span class="designer-toolbar__kicker">DASHBOARD STUDIO</span>
      <strong>{{ name }}</strong>
    </div>
    <div class="designer-toolbar__revision">
      DRAFT / {{ revision }}
    </div>
    <span
      class="designer-toolbar__status"
      :class="`designer-toolbar__status--${saveStatus}`"
      data-testid="dashboard-save-status"
    >
      <i />{{ statusLabel }}
    </span>
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
.designer-toolbar {
  display: grid;
  grid-template-columns: 36px minmax(180px, 1fr) auto auto auto;
  align-items: center;
  gap: 16px;
  min-height: 64px;
  padding: 0 20px;
  color: #152033;
  background: rgba(255, 255, 255, 0.96);
  border-bottom: 1px solid rgba(26, 39, 62, 0.09);
  box-shadow: 0 8px 28px rgba(26, 39, 62, 0.05);
  backdrop-filter: blur(14px);
}
.designer-toolbar__back {
  display: grid;
  width: 34px;
  height: 34px;
  padding: 0;
  place-items: center;
  color: #445067;
  cursor: pointer;
  background: #f3f6f8;
  border: 0;
  border-radius: 10px;
  font-size: 19px;
}
.designer-toolbar__back:hover {
  color: #0f8f84;
  background: #e9f6f4;
}
.designer-toolbar__identity {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}
.designer-toolbar__identity strong {
  overflow: hidden;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.designer-toolbar__kicker {
  color: #0f8f84;
  font:
    800 9px/1 ui-monospace,
    monospace;
  letter-spacing: 0.14em;
}
.designer-toolbar__revision {
  color: #7d8798;
  font:
    700 11px/1 ui-monospace,
    monospace;
  letter-spacing: 0.08em;
}
.designer-toolbar__status {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: #69758a;
  font-size: 12px;
}
.designer-toolbar__status i {
  width: 7px;
  height: 7px;
  background: #9aa4b4;
  border-radius: 50%;
}
.designer-toolbar__status--idle i {
  background: #e6a23c;
}
.designer-toolbar__status--saving i {
  background: #409eff;
  animation: pulse 1s infinite;
}
.designer-toolbar__status--saved i {
  background: #0f9f74;
}
.designer-toolbar__status--error i,
.designer-toolbar__status--conflict i {
  background: #d6533c;
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
@media (max-width: 760px) {
  .designer-toolbar {
    grid-template-columns: 36px 1fr auto;
  }
  .designer-toolbar__revision,
  .designer-toolbar__status {
    display: none;
  }
}
</style>
