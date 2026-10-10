<script setup lang="ts">
import type {
  LabelRenderTaskDto,
  LabelRuntimeOutputPresetDto,
  LabelRuntimeProfileDto,
} from '~/api/label';
import { RiArrowLeftSLine, RiArrowRightSLine, RiQuestionLine } from '@remixicon/vue';
import { ElAlert, ElButton, ElDialog, ElEmpty, ElProgress, ElSkeleton } from 'element-plus';
import { computed, onBeforeUnmount, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  createLabelBatchRender,
  downloadLabelRenderTask,
  getFormLabelProfile,
  getLabelRenderTask,
  previewFormLabel,
} from '~/api/label';

defineOptions({ name: 'FormRecordLabelDownloadDialog' });

const props = defineProps<{
  formCode: string;
  recordIds: readonly number[];
}>();
const visible = defineModel<boolean>({ required: true });
const route = useRoute();
const router = useRouter();

type DialogState =
  | 'idle'
  | 'loading'
  | 'ready'
  | 'submitting'
  | 'processing'
  | 'success'
  | 'partial_success'
  | 'failed';

const state = shallowRef<DialogState>('idle');
const profile = shallowRef<LabelRuntimeProfileDto | null>(null);
const selectedPresetId = shallowRef('');
const currentIndex = shallowRef(0);
const previewUrl = shallowRef('');
const previewLoading = shallowRef(false);
const previewError = shallowRef('');
const task = shallowRef<LabelRenderTaskDto | null>(null);
const errorMessage = shallowRef('');
const previewCache = new Map<string, string>();
let previewController: AbortController | null = null;
let pollTimer: ReturnType<typeof setTimeout> | null = null;
let loadVersion = 0;

const currentRecordId = computed(() => props.recordIds[currentIndex.value]);
const selectedPreset = computed<LabelRuntimeOutputPresetDto | undefined>(() =>
  profile.value?.outputPresets.find((preset) => preset.id === selectedPresetId.value),
);
const processing = computed(() => state.value === 'submitting' || state.value === 'processing');
const hasUnpublishedTemplate = computed(
  () => Boolean(profile.value?.templateCode) && !profile.value?.available,
);
const unavailableDescription = computed(() =>
  hasUnpublishedTemplate.value
    ? '二维码标签配置尚未生效，请前往设置页保存后再预览'
    : '当前表单尚未保存二维码标签配置',
);
const actionLabel = computed(() => {
  if (state.value === 'submitting') return '正在创建任务…';
  if (state.value === 'processing') return `生成中 ${task.value?.progress ?? 0}%`;
  if (state.value === 'success' || state.value === 'partial_success') return '重新下载';
  return '下载文件';
});

function clearPoll(): void {
  if (pollTimer) clearTimeout(pollTimer);
  pollTimer = null;
}

function clearPreviewRequest(): void {
  previewController?.abort();
  previewController = null;
}

function revokePreviews(): void {
  for (const url of previewCache.values()) URL.revokeObjectURL(url);
  previewCache.clear();
  previewUrl.value = '';
}

function reset(): void {
  loadVersion += 1;
  clearPoll();
  clearPreviewRequest();
  revokePreviews();
  state.value = 'idle';
  profile.value = null;
  selectedPresetId.value = '';
  currentIndex.value = 0;
  task.value = null;
  errorMessage.value = '';
  previewError.value = '';
}

async function loadProfile(): Promise<void> {
  const version = ++loadVersion;
  state.value = 'loading';
  errorMessage.value = '';
  try {
    const next = await getFormLabelProfile(props.formCode);
    if (version !== loadVersion || !visible.value) return;
    profile.value = next;
    selectedPresetId.value = next.outputPresets[0]?.id ?? '';
    state.value = 'ready';
    if (next.available && selectedPresetId.value && props.recordIds.length > 0) {
      await loadPreview();
    }
  } catch (error) {
    if (version !== loadVersion) return;
    state.value = 'failed';
    errorMessage.value = error instanceof Error ? error.message : '二维码标签配置加载失败';
  }
}

async function loadPreview(): Promise<void> {
  const recordId = currentRecordId.value;
  if (!profile.value?.available || !recordId || !selectedPresetId.value) return;
  const cacheKey = `${recordId}:${selectedPresetId.value}`;
  const cached = previewCache.get(cacheKey);
  if (cached) {
    previewUrl.value = cached;
    previewError.value = '';
    return;
  }
  clearPreviewRequest();
  const controller = new AbortController();
  previewController = controller;
  previewLoading.value = true;
  previewError.value = '';
  try {
    const blob = await previewFormLabel(
      props.formCode,
      { recordId: String(recordId), outputPresetId: selectedPresetId.value },
      controller.signal,
    );
    if (controller.signal.aborted || !visible.value) return;
    const url = URL.createObjectURL(blob);
    previewCache.set(cacheKey, url);
    previewUrl.value = url;
  } catch (error) {
    if (controller.signal.aborted) return;
    previewError.value = error instanceof Error ? error.message : '标签预览失败';
  } finally {
    if (previewController === controller) previewController = null;
    previewLoading.value = false;
  }
}

function switchRecord(offset: number): void {
  const next = currentIndex.value + offset;
  if (next < 0 || next >= props.recordIds.length || processing.value) return;
  currentIndex.value = next;
  void loadPreview();
}

function selectPreset(id: string): void {
  if (processing.value || selectedPresetId.value === id) return;
  selectedPresetId.value = id;
  void loadPreview();
}

async function pollTask(taskId: string): Promise<void> {
  clearPoll();
  try {
    const next = await getLabelRenderTask(taskId);
    if (!visible.value || taskId !== task.value?.taskId) return;
    task.value = next;
    if (next.status === 'pending' || next.status === 'running') {
      state.value = 'processing';
      pollTimer = setTimeout(() => void pollTask(taskId), 1000);
      return;
    }
    if (next.status === 'success' || next.status === 'partial_success') {
      state.value = next.status;
      await downloadFile(next.taskId);
      return;
    }
    state.value = 'failed';
    errorMessage.value = next.errorMessage || '二维码标签生成失败';
  } catch (error) {
    if (!visible.value) return;
    state.value = 'failed';
    errorMessage.value = error instanceof Error ? error.message : '任务状态查询失败';
  }
}

async function downloadFile(taskId: string): Promise<void> {
  const blob = await downloadLabelRenderTask(taskId);
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `${profile.value?.templateName ?? '二维码标签'}.pdf`;
  anchor.click();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}

async function submit(): Promise<void> {
  if (processing.value || !profile.value?.available || !selectedPresetId.value) return;
  if ((state.value === 'success' || state.value === 'partial_success') && task.value) {
    await downloadFile(task.value.taskId);
    return;
  }
  state.value = 'submitting';
  errorMessage.value = '';
  try {
    const created = await createLabelBatchRender({
      formCode: props.formCode,
      recordIds: props.recordIds.map(String),
      outputPresetId: selectedPresetId.value,
    });
    task.value = {
      taskId: created.taskId,
      templateVersion: profile.value.publishedVersion ?? 0,
      format: 'pdf',
      outputPresetId: selectedPresetId.value,
      outputWidth: selectedPreset.value?.width ?? 0,
      outputHeight: selectedPreset.value?.height ?? 0,
      outputUnit: selectedPreset.value?.unit ?? 'mm',
      outputDpi: selectedPreset.value?.dpi ?? 300,
      status: created.status,
      totalCount: props.recordIds.length,
      successCount: 0,
      failedCount: 0,
      progress: 0,
      createdAt: '',
      updatedAt: '',
      items: [],
    };
    state.value = 'processing';
    await pollTask(created.taskId);
  } catch (error) {
    state.value = 'failed';
    errorMessage.value = error instanceof Error ? error.message : '二维码标签任务创建失败';
  }
}

function editTemplate(): void {
  visible.value = false;
  // 明确携带工作区参数，避免从弹窗按名称跳转时丢失当前应用或表单上下文。
  void router.push({
    name: 'form-extension-qrcode',
    params: { appCode: route.params.appCode, formCode: props.formCode },
  });
}

watch(
  visible,
  (open) => {
    if (open) void loadProfile();
    else reset();
  },
  { immediate: true },
);

onBeforeUnmount(reset);
</script>

<template>
  <ElDialog
    v-model="visible"
    class="label-download-dialog"
    width="min(800px, calc(100vw - 32px))"
    destroy-on-close
    :close-on-click-modal="!processing"
  >
    <template #header>
      <div class="label-download-dialog__title">
        <span>下载二维码标签</span>
        <RiQuestionLine aria-label="选择记录后按已发布模板生成多页 PDF" />
      </div>
    </template>

    <ElSkeleton v-if="state === 'loading'" :rows="7" animated />
    <ElEmpty
      v-else-if="profile && !profile.available"
      class="label-download-dialog__empty"
      :description="unavailableDescription"
      :image-size="96"
    >
      <ElButton v-if="profile.canManageTemplate" type="primary" plain @click="editTemplate">
        {{ hasUnpublishedTemplate ? '前往保存配置' : '配置二维码标签' }}
      </ElButton>
    </ElEmpty>
    <div v-else-if="profile?.available" class="label-download-dialog__body">
      <section class="label-download-dialog__preview-panel" aria-label="标签预览">
        <div class="label-download-dialog__pager">
          <ElButton
            circle
            :icon="RiArrowLeftSLine"
            :disabled="currentIndex <= 0 || processing"
            aria-label="上一条"
            @click="switchRecord(-1)"
          />
          <span>{{ currentIndex + 1 }}/{{ recordIds.length }}</span>
          <ElButton
            circle
            :icon="RiArrowRightSLine"
            :disabled="currentIndex >= recordIds.length - 1 || processing"
            aria-label="下一条"
            @click="switchRecord(1)"
          />
        </div>
        <p class="label-download-dialog__hint">
          字段内容超长时，将按模板规则自动截断，请认真核对标签展示样式！
        </p>
        <div class="label-download-dialog__canvas" :class="{ 'is-loading': previewLoading }">
          <img v-if="previewUrl" :src="previewUrl" alt="二维码标签预览" />
          <ElEmpty v-else-if="previewError" :description="previewError" :image-size="48" />
          <span v-else>正在生成预览…</span>
        </div>
        <button
          v-if="profile.canManageTemplate"
          type="button"
          class="label-download-dialog__edit"
          @click="editTemplate"
        >
          修改二维码标签
        </button>
      </section>

      <aside class="label-download-dialog__sizes" aria-label="尺寸选择">
        <h3>尺寸选择</h3>
        <button
          v-for="preset in profile.outputPresets"
          :key="preset.id"
          type="button"
          class="label-download-dialog__size"
          :class="{ 'is-selected': selectedPresetId === preset.id }"
          :disabled="processing"
          @click="selectPreset(preset.id)"
        >
          <strong
            >{{ preset.name }}（{{ preset.width }} X {{ preset.height }}{{ preset.unit }}）</strong
          >
          <span>{{ preset.pixelWidth }} X {{ preset.pixelHeight }} 像素</span>
          <i v-if="selectedPresetId === preset.id" aria-hidden="true">✓</i>
        </button>
      </aside>
    </div>

    <ElAlert
      v-if="state === 'partial_success' && task"
      class="label-download-dialog__alert"
      type="warning"
      :closable="false"
      :title="`已生成 ${task.successCount} 个标签，${task.failedCount} 个失败`"
    />
    <ElAlert
      v-else-if="state === 'failed' && errorMessage"
      class="label-download-dialog__alert"
      type="error"
      :closable="false"
      :title="errorMessage"
    />
    <ElProgress
      v-if="processing"
      class="label-download-dialog__progress"
      :percentage="task?.progress ?? 0"
    />

    <template #footer>
      <div class="label-download-dialog__footer">
        <span>本次下载标签 {{ recordIds.length }} 个</span>
        <ElButton
          type="primary"
          :loading="processing"
          :disabled="!profile?.available || !selectedPresetId || recordIds.length === 0"
          @click="submit"
        >
          {{ actionLabel }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<style scoped lang="scss">
.label-download-dialog {
  &__title,
  &__footer,
  &__pager {
    display: flex;
    align-items: center;
  }

  &__title {
    gap: var(--el-space-xs);
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-large);

    svg {
      width: var(--el-font-size-base);
      height: var(--el-font-size-base);
    }
  }

  &__body {
    display: grid;
    grid-template-columns: 480px 200px;
    gap: var(--el-space-2xl);
    align-items: start;
  }

  &__preview-panel {
    box-sizing: border-box;
    width: 480px;
    min-height: 428px;
    padding: var(--el-space-2xl) var(--el-space-4xl) var(--el-space-xl);
    text-align: center;
    background: var(--el-fill-color-lighter);
  }

  &__pager {
    justify-content: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-regular);
  }

  &__hint {
    margin: var(--el-space-md) 0 var(--el-space-lg);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }

  &__canvas {
    min-height: 280px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);

    img {
      max-width: 100%;
      max-height: 280px;
      box-shadow: var(--el-box-shadow-light);
    }

    &.is-loading {
      opacity: var(--el-disabled-opacity);
    }
  }

  &__edit {
    padding: var(--el-space-md);
    color: var(--el-color-primary);
    background: transparent;
    border: 0;
    cursor: pointer;

    &:hover {
      background: var(--el-color-primary-light-9);
      border-radius: var(--el-border-radius-base);
    }
  }

  &__sizes {
    width: 200px;

    h3 {
      margin: 0 0 var(--el-space-md);
      color: var(--el-text-color-primary);
      font-size: var(--el-font-size-large);
    }
  }

  &__size {
    position: relative;
    width: 100%;
    min-height: 80px;
    padding: var(--el-space-lg);
    display: flex;
    align-items: flex-start;
    flex-direction: column;
    gap: var(--el-space-xs);
    color: var(--el-text-color-primary);
    text-align: left;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    border-radius: var(--el-border-radius-base);
    cursor: pointer;

    & + & {
      margin-top: var(--el-space-lg);
    }

    &:hover:not(:disabled) {
      background: var(--el-fill-color-light);
      border-color: var(--el-color-primary-light-5);
    }

    span {
      color: var(--el-text-color-secondary);
    }

    i {
      position: absolute;
      right: var(--el-space-xs);
      bottom: var(--el-space-xs);
      color: var(--el-color-white);
      font-style: normal;
    }

    &.is-selected {
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 1px var(--el-color-primary) inset;

      &::after {
        position: absolute;
        right: 0;
        bottom: 0;
        width: 0;
        height: 0;
        content: '';
        border-style: solid;
        border-width: 0 0 28px 28px;
        border-color: transparent transparent var(--el-color-primary) transparent;
      }
    }
  }

  &__alert,
  &__progress {
    margin-top: var(--el-space-md);
  }

  &__footer {
    justify-content: space-between;
    color: var(--el-text-color-secondary);
  }
}

@media (max-width: 720px) {
  .label-download-dialog__body {
    grid-template-columns: 1fr;
  }

  .label-download-dialog__preview-panel,
  .label-download-dialog__sizes {
    width: 100%;
  }

  .label-download-dialog__sizes {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--el-space-sm);

    h3 {
      grid-column: 1 / -1;
    }
  }

  .label-download-dialog__size + .label-download-dialog__size {
    margin-top: 0;
  }
}
</style>
