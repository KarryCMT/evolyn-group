<script setup lang="ts">
import { RiDownload2Fill, RiUploadCloud2Fill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import type { UploadRequestOptions } from 'element-plus';
import type { MemberInvitationImportResult } from '~/api/member';

const props = defineProps<{
  submitting: boolean;
  result: MemberInvitationImportResult | null;
  upload: (file: File) => Promise<MemberInvitationImportResult>;
}>();

function downloadTemplate() {
  const link = document.createElement('a');
  link.href = '/templates/通讯录批量导入模板.xlsx';
  link.download = '通讯录批量导入模板.xlsx';
  link.click();
}

async function upload(options: UploadRequestOptions) {
  try {
    await props.upload(options.file);
    options.onSuccess?.({});
  } catch (error) {
    ElMessage.error('导入失败，请检查模板和成员信息');
  }
}
</script>

<template>
  <section class="organization-invite-batch" aria-label="批量导入成员">
    <ul class="organization-invite-batch__tips">
      <li>
        请下载模板，按格式修改后导入：<button type="button" @click="downloadTemplate">
          通讯录模板
        </button>
      </li>
      <li>每次最多导入 200 名成员，已经在通讯录内或信息有误的成员不导入</li>
    </ul>

    <el-upload
      class="organization-invite-batch__upload"
      drag
      accept=".xlsx"
      :show-file-list="false"
      :disabled="props.submitting"
      :http-request="upload"
    >
      <RiUploadCloud2Fill />
      <p>将文件拖拽到此区域，或 <span>点击添加</span></p>
      <small>文件大小不超过 5 MB</small>
    </el-upload>

    <section v-if="props.result" class="organization-invite-batch__result" aria-live="polite">
      <strong>导入完成：成功 {{ props.result.successCount }} 名成员</strong>
      <ul v-if="props.result.failedRows.length">
        <li v-for="message in props.result.failedRows" :key="message">{{ message }}</li>
      </ul>
      <span v-else>所有数据均已创建邀请。</span>
    </section>
  </section>
</template>

<style scoped lang="scss">
.organization-invite-batch {
  &__tips {
    margin: 0 0 var(--el-space-xl);
    padding: var(--el-space-lg) var(--el-space-2xl) var(--el-space-lg) var(--el-space-4xl);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-base);
    line-height: 24px;
  }

  &__tips button {
    padding: var(--el-space-xs) var(--el-space-sm);
    border: 0;
    border-radius: var(--el-border-radius-base);
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
  }

  &__tips button:hover {
    background: var(--el-color-primary-light-9);
  }

  &__upload :deep(.el-upload),
  &__upload :deep(.el-upload-dragger) {
    width: 100%;
  }

  &__upload :deep(.el-upload-dragger) {
    display: grid;
    min-height: 260px;
    border-color: var(--el-border-color);
    background: var(--el-bg-color);
    place-content: center;
  }

  &__upload :deep(svg) {
    width: 48px;
    height: 48px;
    margin: 0 auto var(--el-space-xl);
    color: var(--el-color-primary);
  }

  &__upload :deep(p) {
    margin: 0 0 var(--el-space-sm);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }

  &__upload :deep(p span) {
    color: var(--el-color-primary);
  }

  &__upload :deep(small) {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
  }

  &__result {
    margin-top: var(--el-space-2xl);
    padding: var(--el-space-xl) var(--el-space-2xl);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-medium);
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    line-height: 25px;
  }

  &__result strong {
    color: var(--el-text-color-primary);
  }
  &__result ul {
    margin: var(--el-space-md) 0 0;
    color: var(--el-color-danger);
  }
}
</style>
