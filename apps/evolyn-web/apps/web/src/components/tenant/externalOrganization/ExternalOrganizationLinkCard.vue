<script setup lang="ts">
import { RiDeleteBin6Fill, RiEdit2Fill, RiQrCodeFill, RiShareForwardFill } from '@remixicon/vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import type { ExternalOrganizationLink } from './externalOrganization.types';

const props = defineProps<{ link: ExternalOrganizationLink }>();
const emit = defineEmits<{
  edit: [link: ExternalOrganizationLink];
  remove: [id: string];
  update: [link: ExternalOrganizationLink];
}>();

async function share() {
  try {
    await navigator.clipboard.writeText(props.link.url);
    ElMessage.success('邀请链接已复制');
  } catch {
    ElMessage.info('请手动复制邀请链接');
  }
}

async function remove() {
  await ElMessageBox.confirm('删除后该链接将失效，是否继续？', '删除链接', { type: 'warning' });
  emit('remove', props.link.id);
  ElMessage.success('链接已删除');
}
</script>

<template>
  <article class="external-organization-link-card">
    <header class="external-organization-link-card__header">
      <span class="external-organization-link-card__url">{{ props.link.url }}</span>
      <div class="external-organization-link-card__tools">
        <button type="button" aria-label="编辑链接" @click="emit('edit', props.link)">
          <RiEdit2Fill />
        </button>
        <button type="button" aria-label="删除链接" @click="remove"><RiDeleteBin6Fill /></button>
      </div>
    </header>
    <div class="external-organization-link-card__body">
      <p>通过此链接邀请的互联组织/互联对接人默认设置如下</p>
      <dl>
        <div>
          <dt>互联组织标签</dt>
          <dd>{{ props.link.label || '—' }}</dd>
        </div>
        <div>
          <dt>互联对接人角色</dt>
          <dd>{{ props.link.role || '—' }}</dd>
        </div>
        <div>
          <dt>通讯录权限</dt>
          <dd>
            <el-tag effect="light">{{ props.link.directoryPermission }}</el-tag>
          </dd>
        </div>
      </dl>
    </div>
    <footer class="external-organization-link-card__footer">
      <el-switch
        :model-value="props.link.enabled"
        @update:model-value="emit('update', { ...props.link, enabled: Boolean($event) })"
      />
      <div class="external-organization-link-card__actions">
        <button type="button" aria-label="查看二维码" @click="ElMessage.info('二维码功能即将开放')">
          <RiQrCodeFill />
        </button>
        <el-button type="primary" @click="share"><RiShareForwardFill />分享链接</el-button>
      </div>
    </footer>
  </article>
</template>

<style scoped lang="scss">
.external-organization-link-card {
  width: 528px;
  max-width: 100%;
  border: 1px solid var(--el-border-color-light);
  border-radius: 10px;
  overflow: hidden;

  &__header,
  &__footer,
  &__tools,
  &__actions {
    display: flex;
    align-items: center;
  }
  &__header {
    height: 56px;
    padding: 0 16px 0 20px;
    justify-content: space-between;
    background: var(--el-fill-color-lighter);
  }
  &__url {
    overflow: hidden;
    color: var(--el-text-color-regular);
    font-size: 16px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__tools,
  &__actions {
    gap: 10px;
  }
  &__tools button,
  &__actions button {
    display: inline-flex;
    width: 28px;
    height: 28px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;
  }
  &__tools button:hover,
  &__actions button:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__tools svg,
  &__actions svg {
    width: 18px;
    height: 18px;
  }
  &__body {
    padding: 16px 20px 10px;
  }
  &__body p {
    margin: 0 0 14px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
  &__body dl {
    margin: 0;
  }
  &__body dl div {
    display: grid;
    min-height: 34px;
    grid-template-columns: 146px 1fr;
    align-items: center;
  }
  &__body dt,
  &__body dd {
    margin: 0;
    color: var(--el-text-color-regular);
    font-size: 15px;
  }
  &__body dd {
    color: var(--el-text-color-secondary);
  }
  &__footer {
    height: 58px;
    margin: 0 20px;
    border-top: 1px solid var(--el-border-color-lighter);
    justify-content: space-between;
  }
  &__actions :deep(.el-button) {
    height: 34px;
    gap: 4px;
    font-size: 14px;
  }
}
</style>
