<script setup lang="ts">
import { RiInformationLine } from '@remixicon/vue';
import { computed, shallowRef } from 'vue';
import {
  ElIcon,
  ElMessage,
  ElMessageBox,
  ElTooltip,
} from 'element-plus';
import type { FormItem } from '../schema/types';
import FormSchemaFrontendEventDrawer from './FormSchemaFrontendEventDrawer.vue';
import FormSchemaFrontendEventDebugDrawer from './FormSchemaFrontendEventDebugDrawer.vue';
import FormSchemaFrontendEventDialog from './FormSchemaFrontendEventDialog.vue';
import {
  formEventFieldOptions,
  FORM_EVENT_LIMITS,
  type FormEvent,
} from './frontend-events';

const formEvents = defineModel<FormEvent[]>({ required: true });
const props = defineProps<{ items: readonly FormItem[] }>();
const panelOpen = shallowRef(false);
const editorOpen = shallowRef(false);
const debugOpen = shallowRef(false);
const editingID = shallowRef<string | null>(null);
const debuggingEvent = shallowRef<FormEvent | undefined>();
const fields = computed(() => formEventFieldOptions(props.items));
const editingEvent = computed(() => formEvents.value.find((event) => event.id === editingID.value));
const existingNames = computed(() =>
  formEvents.value.filter((event) => event.id !== editingID.value).map((event) => event.name),
);
const configuredLabel = computed(() =>
  formEvents.value.length === 0 ? '设置' : `已设置 ${formEvents.value.length} 个前端事件`,
);

function openCreate(): void {
  if (formEvents.value.length >= FORM_EVENT_LIMITS.maxEvents) {
    ElMessage.warning(`每张表单最多配置 ${FORM_EVENT_LIMITS.maxEvents} 个前端事件`);
    return;
  }
  editingID.value = null;
  editorOpen.value = true;
}

function openEdit(event: FormEvent): void {
  editingID.value = event.id;
  editorOpen.value = true;
}

function saveEvent(event: FormEvent): void {
  const index = formEvents.value.findIndex((item) => item.id === event.id);
  if (index < 0) formEvents.value = [...formEvents.value, event];
  else formEvents.value = formEvents.value.map((item) => (item.id === event.id ? event : item));
}

function saveAndDebug(event: FormEvent): void {
  saveEvent(event);
  debugEvent(event);
}

function toggleEvent(event: FormEvent, enabled: boolean | string | number): void {
  formEvents.value = formEvents.value.map((item) =>
    item.id === event.id ? { ...item, enabled: enabled === true } : item,
  );
}

async function removeEvent(event: FormEvent): Promise<void> {
  try {
    await ElMessageBox.confirm(`删除“${event.name}”后将无法恢复，是否继续？`, '删除前端事件', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    });
  } catch {
    return;
  }
  formEvents.value = formEvents.value.filter((item) => item.id !== event.id);
}

function debugEvent(event: FormEvent): void {
  debuggingEvent.value = event;
  debugOpen.value = true;
}

</script>

<template>
  <section class="form-frontend-events" aria-label="前端事件">
    <div class="form-frontend-events__heading">
      <span class="form-frontend-events__title">前端事件</span>
      <el-tooltip content="在填写或编辑数据时，由字段值变化触发自动操作" placement="top">
        <el-icon class="form-frontend-events__help" aria-label="前端事件说明"><RiInformationLine /></el-icon>
      </el-tooltip>
    </div>
    <p class="form-frontend-events__description">让表单字段值的变化触发一系列自动操作。</p>
    <button class="form-frontend-events__entry" type="button" @click="panelOpen = true">
      <span>{{ configuredLabel }}</span><span aria-hidden="true">›</span>
    </button>

    <FormSchemaFrontendEventDrawer
      v-model="panelOpen"
      :events="formEvents"
      :fields="fields"
      @create="openCreate"
      @edit="openEdit"
      @debug="debugEvent"
      @remove="removeEvent"
      @toggle="toggleEvent"
    />

    <FormSchemaFrontendEventDialog
      v-model="editorOpen"
      :event="editingEvent"
      :fields="fields"
      :existing-names="existingNames"
      @save="saveEvent"
      @save-and-debug="saveAndDebug"
    />
    <FormSchemaFrontendEventDebugDrawer
      v-model="debugOpen"
      :event="debuggingEvent"
      :fields="fields"
    />
  </section>
</template>

<style scoped lang="scss">
.form-frontend-events { display: flex; flex-direction: column; gap: 10px; width: 100%; padding-top: 2px; }
.form-frontend-events__heading { display: inline-flex; gap: 6px; align-items: center; }
.form-frontend-events__title { color: var(--el-text-color-primary); font-size: 15px; font-weight: 600; }
.form-frontend-events__help { color: var(--el-text-color-secondary); cursor: help; }
.form-frontend-events__description { margin: -2px 0 0; color: var(--el-text-color-secondary); font-size: 13px; line-height: 1.6; }
.form-frontend-events__entry { display: flex; width: 100%; min-height: 38px; align-items: center; justify-content: space-between; padding: 8px 11px; color: var(--el-text-color-regular); font: inherit; text-align: left; cursor: pointer; background: var(--el-bg-color); border: 1px solid var(--el-border-color); border-radius: var(--el-border-radius-base); }
.form-frontend-events__entry:hover { color: var(--el-color-primary); border-color: var(--el-color-primary); }
</style>
