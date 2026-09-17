<script setup lang="ts">
import { RiAddLine, RiBugLine, RiDeleteBin6Line, RiEditLine } from '@remixicon/vue';
import { computed } from 'vue';
import { ElButton, ElDrawer, ElSwitch } from 'element-plus';
import type { FormEvent, FormEventFieldOption } from './frontend-events';

/** 前端事件列表的底部抽屉，只负责展示列表并将用户操作上抛。 */
const open = defineModel<boolean>({ required: true });
const props = defineProps<{
  events: readonly FormEvent[];
  fields: readonly FormEventFieldOption[];
}>();
const emit = defineEmits<{
  create: [];
  edit: [event: FormEvent];
  debug: [event: FormEvent];
  remove: [event: FormEvent];
  toggle: [event: FormEvent, enabled: boolean | string | number];
}>();

const hasEvents = computed(() => props.events.length > 0);

function eventSummary(event: FormEvent): string {
  const trigger = props.fields.find((field) => field.key === event.trigger)?.label || event.trigger;
  const targets = event.action
    .map(
      (action) => props.fields.find((field) => field.key === action.field)?.label || action.field,
    )
    .filter(Boolean);
  return `当「${trigger || '未选择字段'}」值发生变化时，发送 ${event.request.method.toUpperCase()} 请求${targets.length ? `，写入「${targets.join('、')}」` : ''}。`;
}
</script>

<template>
  <el-drawer
    v-model="open"
    class="form-frontend-event-drawer"
    direction="btt"
    size="90%"
    title="前端事件"
    append-to-body
  >
    <div class="form-frontend-event-drawer__content">
      <div class="form-frontend-event-drawer__toolbar">
        <el-button type="primary" :icon="RiAddLine" @click="emit('create')">添加前端事件</el-button>
        <p>前端事件可以在填写或编辑数据时，让表单字段值的变化触发一系列自动操作。</p>
      </div>

      <div v-if="!hasEvents" class="form-frontend-event-drawer__empty">
        <strong>还没有前端事件</strong>
        <span>添加事件后，可在字段值变化时自动请求并回填数据。</span>
      </div>

      <div v-else class="form-frontend-event-drawer__list">
        <article
          v-for="event in events"
          :key="event.id"
          class="form-frontend-event-drawer__event-card"
        >
          <div class="form-frontend-event-drawer__event-copy">
            <strong>{{ event.name }}</strong>
            <p v-if="event.description">{{ event.description }}</p>
            <p class="form-frontend-event-drawer__summary">{{ eventSummary(event) }}</p>
          </div>
          <div class="form-frontend-event-drawer__event-actions">
            <button type="button" @click="emit('edit', event)"><RiEditLine />编辑</button>
            <button type="button" @click="emit('debug', event)"><RiBugLine />调试</button>
            <button class="is-danger" type="button" @click="emit('remove', event)">
              <RiDeleteBin6Line />删除
            </button>
            <el-switch
              :model-value="event.enabled"
              :aria-label="`${event.name}启用状态`"
              @update:model-value="emit('toggle', event, $event)"
            />
          </div>
        </article>
      </div>
    </div>
  </el-drawer>
</template>

<style scoped lang="scss">
.form-frontend-event-drawer__content {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
}
.form-frontend-event-drawer__toolbar {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.form-frontend-event-drawer__toolbar p {
  margin: 15px 0 0;
  color: var(--el-text-color-secondary);
}
.form-frontend-event-drawer__empty {
  display: flex;
  flex: 1;
  min-height: 260px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
}
.form-frontend-event-drawer__empty strong {
  color: var(--el-text-color-primary);
  font-size: 16px;
}
.form-frontend-event-drawer__list {
  min-height: 260px;
}
.form-frontend-event-drawer__event-card {
  display: flex;
  min-height: 110px;
  align-items: flex-start;
  justify-content: space-between;
  gap: 28px;
  padding: 20px 4px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.form-frontend-event-drawer__event-copy {
  min-width: 0;
}
.form-frontend-event-drawer__event-copy strong {
  color: var(--el-text-color-primary);
  font-size: 16px;
}
.form-frontend-event-drawer__event-copy p {
  margin: 7px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.form-frontend-event-drawer__summary {
  color: var(--el-text-color-regular) !important;
}
.form-frontend-event-drawer__event-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}
.form-frontend-event-drawer__event-actions button {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  padding: 5px;
  color: var(--el-color-primary);
  font: inherit;
  cursor: pointer;
  background: transparent;
  border: 0;
}
.form-frontend-event-drawer__event-actions button.is-danger {
  color: var(--el-color-danger);
}
:global(.form-frontend-event-drawer) {
  border-radius: 12px 12px 0 0;
}
:global(.form-frontend-event-drawer .el-drawer__header) {
  margin-bottom: 0;
  padding: 22px 24px 18px;
  color: var(--el-text-color-primary);
  font-size: 20px;
  font-weight: 600;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
:global(.form-frontend-event-drawer .el-drawer__body) {
  min-height: 0;
  padding: 24px;
  overflow: hidden;
}
@media (max-width: 760px) {
  .form-frontend-event-drawer__event-card {
    flex-direction: column;
  }
  .form-frontend-event-drawer__event-actions {
    flex-wrap: wrap;
  }
}
</style>
