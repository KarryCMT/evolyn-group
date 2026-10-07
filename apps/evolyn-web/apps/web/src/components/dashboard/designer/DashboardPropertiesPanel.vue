<script setup lang="ts">
import type { BusinessDashboardIssue, BusinessDashboardWidget } from '@evolyn.do/dashboard';
import { computed, shallowRef, watch } from 'vue';

const props = defineProps<{
  widget: BusinessDashboardWidget | null;
  issues: BusinessDashboardIssue[];
  focusedIssuePath: string;
}>();
const emit = defineEmits<{
  update: [id: string, patch: Partial<Pick<BusinessDashboardWidget, 'title' | 'settings'>>];
  remove: [id: string];
  focusIssue: [issue: BusinessDashboardIssue];
}>();

const title = shallowRef('');
const selectedIssue = computed(() =>
  props.issues.find((issue) => issue.path === props.focusedIssuePath),
);

watch(
  () => props.widget,
  (widget) => {
    title.value = widget?.title ?? '';
  },
  { immediate: true },
);

function commitTitle() {
  const value = title.value.trim();
  if (!props.widget || !value || value === props.widget.title) return;
  emit('update', props.widget.id, { title: value });
}
</script>

<template>
  <aside class="properties-panel" aria-label="组件属性">
    <header class="properties-panel__header">
      <span>INSPECTOR</span>
      <strong>属性面板</strong>
    </header>
    <div v-if="widget" class="properties-panel__body">
      <div class="properties-panel__type">
        <span>{{ widget.type === 'chart' ? '统计图' : '明细表' }}</span>
        <code>{{ widget.id }}</code>
      </div>
      <el-form label-position="top">
        <el-form-item
          label="组件标题"
          :class="{ 'is-issue': selectedIssue?.path.endsWith('.title') }"
        >
          <el-input v-model="title" maxlength="60" show-word-limit @change="commitTitle" />
        </el-form-item>
        <div class="properties-panel__layout">
          <span v-for="key in ['x', 'y', 'w', 'h'] as const" :key="key">
            <small>{{ key.toUpperCase() }}</small><strong>{{ widget.layout[key] }}</strong>
          </span>
        </div>
        <p class="properties-panel__hint">
          拖动画布组件改变位置，使用边缘手柄调整尺寸。
        </p>
      </el-form>
      <el-button
        class="properties-panel__delete"
        text
        type="danger"
        @click="emit('remove', widget.id)"
      >
        删除组件
      </el-button>
    </div>
    <div v-else class="properties-panel__empty">
      <span>SELECT A WIDGET</span>
      <strong>选择画布中的组件</strong>
      <p>这里会显示标题、布局和后续的数据配置。</p>
    </div>
    <div v-if="issues.length" class="properties-panel__issues">
      <strong>需要处理的问题</strong>
      <button
        v-for="issue in issues"
        :key="`${issue.path}-${issue.code}`"
        type="button"
        @click="emit('focusIssue', issue)"
      >
        <code>{{ issue.path }}</code><span>{{ issue.message }}</span>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.properties-panel {
  display: flex;
  flex: 0 0 286px;
  flex-direction: column;
  min-height: 0;
  color: #172033;
  background: #fff;
  border-left: 1px solid rgba(23, 32, 51, 0.09);
}
.properties-panel__header {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 22px 20px 17px;
  border-bottom: 1px solid rgba(23, 32, 51, 0.07);
}
.properties-panel__header span,
.properties-panel__empty > span {
  color: #0f8f84;
  font:
    800 9px/1 ui-monospace,
    monospace;
  letter-spacing: 0.15em;
}
.properties-panel__header strong {
  font-size: 16px;
}
.properties-panel__body {
  padding: 20px;
}
.properties-panel__type {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 14px;
  margin-bottom: 20px;
  background: #f4f7f9;
  border-radius: 11px;
}
.properties-panel__type span {
  font-size: 13px;
  font-weight: 700;
}
.properties-panel__type code {
  overflow: hidden;
  color: #8994a5;
  font-size: 10px;
  text-overflow: ellipsis;
}
.properties-panel__layout {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}
.properties-panel__layout span {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 9px;
  text-align: center;
  background: #f6f8fa;
  border-radius: 8px;
}
.properties-panel__layout small {
  color: #99a3b2;
  font:
    700 9px/1 ui-monospace,
    monospace;
}
.properties-panel__layout strong {
  font-size: 13px;
}
.properties-panel__hint {
  color: #8a95a7;
  font-size: 11px;
  line-height: 1.6;
}
.properties-panel__delete {
  margin-top: 12px;
}
.properties-panel__empty {
  display: flex;
  padding: 34px 24px;
  flex-direction: column;
  gap: 9px;
}
.properties-panel__empty strong {
  font-size: 15px;
}
.properties-panel__empty p {
  margin: 0;
  color: #8b95a7;
  font-size: 12px;
  line-height: 1.6;
}
.properties-panel__issues {
  display: flex;
  max-height: 230px;
  padding: 16px 18px;
  margin-top: auto;
  overflow: auto;
  flex-direction: column;
  gap: 8px;
  background: #fff7f3;
  border-top: 1px solid #f1d0c5;
}
.properties-panel__issues > strong {
  color: #a74731;
  font-size: 12px;
}
.properties-panel__issues button {
  display: flex;
  padding: 9px;
  text-align: left;
  color: #7f3827;
  cursor: pointer;
  background: #fff;
  border: 1px solid #efd5cc;
  border-radius: 8px;
  flex-direction: column;
  gap: 4px;
}
.properties-panel__issues code {
  font-size: 9px;
}
.properties-panel__issues span {
  font-size: 11px;
}
.is-issue :deep(.el-input__wrapper) {
  box-shadow: 0 0 0 1px #d6533c inset;
}
</style>
