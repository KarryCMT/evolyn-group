<script setup lang="ts">
import { RiAddFill, RiCloseFill, RiDownload2Fill, RiUpload2Fill } from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import type { UploadRequestOptions } from 'element-plus';
import ExternalOrganizationLinkCard from './ExternalOrganizationLinkCard.vue';
import ExternalOrganizationLinkEditor from './ExternalOrganizationLinkEditor.vue';
import type {
  ExternalOrganizationInviteMode,
  ExternalOrganizationLink,
} from './externalOrganization.types';

const props = defineProps<{
  links: ExternalOrganizationLink[];
  mode: ExternalOrganizationInviteMode;
}>();

const visible = defineModel<boolean>('visible', { required: true });
const emit = defineEmits<{
  'update:mode': [mode: ExternalOrganizationInviteMode];
  add: [];
  update: [link: ExternalOrganizationLink];
  remove: [id: string];
}>();

const editingLink = defineModel<ExternalOrganizationLink | null>('editingLink', { required: true });

function close() {
  visible.value = false;
}
function openEditor(link: ExternalOrganizationLink) {
  editingLink.value = link;
}
function updateLink(link: ExternalOrganizationLink) {
  emit('update', link);
}
function selectFile(options: UploadRequestOptions) {
  ElMessage.success(`已选择 ${options.file.name}，导入结果将在此展示`);
  options.onSuccess?.({});
  return Promise.resolve();
}
function downloadTemplate() {
  ElMessage.info('Excel 模板下载功能即将开放');
}
</script>

<template>
  <el-dialog
    v-model="visible"
    class="external-organization-invite-dialog"
    fullscreen
    :show-close="false"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <template #header>
      <header class="external-organization-invite-dialog__header">
        <h1>新建连接</h1>
        <button type="button" aria-label="关闭新建连接" @click="close"><RiCloseFill /></button>
      </header>
    </template>

    <el-scrollbar class="external-organization-invite-dialog__scrollbar">
      <section class="external-organization-invite-dialog__surface" aria-label="互联组织邀请设置">
        <el-tabs
          :model-value="props.mode"
          class="external-organization-invite-dialog__tabs"
          @update:model-value="emit('update:mode', $event as ExternalOrganizationInviteMode)"
        >
          <el-tab-pane label="公开链接邀请" name="public">
            <section class="external-organization-invite-dialog__panel">
              <h2>公开链接邀请适用场景</h2>
              <ul class="external-organization-invite-dialog__tips">
                <li>当合作伙伴「已有账号」时，只能用公开链接邀请。推荐通过微信发送邀请链接。</li>
                <li>
                  当合作伙伴「想在钉钉/企业微信/飞书上使用」时，需先在对应平台上安装注册，再用公开链接邀请。
                </li>
                <li>
                  当合作伙伴「没有账号」时，可以用公开链接邀请对方注册。<button type="button">
                    了解更多
                  </button>
                </li>
              </ul>
              <h2 class="external-organization-invite-dialog__section-title">设置链接</h2>
              <p class="external-organization-invite-dialog__intro">
                将下方链接分享给合作伙伴，对方点击即可建立连接关系；建议预设互联组织标签、角色等属性后分享。
              </p>
              <button
                type="button"
                class="external-organization-invite-dialog__add"
                @click="emit('add')"
              >
                <RiAddFill />添加链接
              </button>
              <div class="external-organization-invite-dialog__cards">
                <ExternalOrganizationLinkCard
                  v-for="link in props.links"
                  :key="link.id"
                  :link="link"
                  @edit="openEditor"
                  @update="updateLink"
                  @remove="emit('remove', $event)"
                />
              </div>
            </section>
          </el-tab-pane>
          <el-tab-pane label="批量导入邀请" name="batch">
            <section class="external-organization-invite-dialog__panel">
              <h2>批量导入邀请适用场景</h2>
              <p class="external-organization-invite-dialog__intro">
                当合作伙伴「没有账号且不想自己注册」时，可以使用批量导入的方式。<button
                  type="button"
                  class="external-organization-invite-dialog__text-link"
                >
                  了解更多
                </button>
              </p>
              <h2 class="external-organization-invite-dialog__section-title">导入Excel</h2>
              <el-upload
                class="external-organization-invite-dialog__upload"
                drag
                accept=".xlsx,.xls"
                :show-file-list="false"
                :http-request="selectFile"
              >
                <RiUpload2Fill />
                <span>选择或拖拽上传文件, 单个5MB以内</span>
                <template #tip
                  ><button type="button" @click.stop="downloadTemplate">
                    <RiDownload2Fill />下载导入模板
                  </button></template
                >
              </el-upload>
              <p class="external-organization-invite-dialog__batch-notice">
                导入成功后会自动发送邀请短信，合作伙伴点击短信即可连接<br />你也可以通过微信，直接分享下方链接给合作互联组织创建者，绑定官方公众号并建立连接
              </p>
              <div class="external-organization-invite-dialog__batch-result">
                成功导入Excel后生成
              </div>
            </section>
          </el-tab-pane>
        </el-tabs>
      </section>
    </el-scrollbar>

    <ExternalOrganizationLinkEditor
      :visible="Boolean(editingLink)"
      :link="editingLink"
      @save="updateLink"
      @update:visible="
        (value) => {
          if (!value) editingLink = null;
        }
      "
    />
  </el-dialog>
</template>

<style scoped lang="scss">
.external-organization-invite-dialog {
  &__header {
    display: flex;
    height: 80px;
    align-items: center;
    justify-content: center;
  }
  &__header h1 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 650;
    line-height: 28px;
  }
  &__header button {
    position: absolute;
    right: 28px;
    display: inline-flex;
    width: 36px;
    height: 36px;
    padding: 0;
    border: 0;
    border-radius: var(--el-border-radius-medium);
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-primary);
    background: transparent;
    cursor: pointer;
  }
  &__header button:hover {
    background: var(--el-fill-color-light);
  }
  &__header button svg {
    width: 24px;
    height: 24px;
  }
  &__scrollbar {
    height: calc(100vh - 80px);
    background: var(--el-fill-color-lighter);
  }
  &__surface {
    min-height: calc(100vh - 140px);
    margin: var(--el-space-3xl) clamp(var(--el-space-3xl), 7vw, 140px);
    padding: var(--el-space-4xl);
    border-radius: var(--el-border-radius-large);
    background: var(--el-bg-color);
  }
  &__tabs :deep(.el-tabs__header) {
    margin-bottom: var(--el-space-4xl);
  }
  &__tabs :deep(.el-tabs__item) {
    height: 56px;
    padding: 0 var(--el-space-5xl);
    border: 1px solid var(--el-border-color-light);
    border-bottom: 0;
    border-radius: var(--el-border-radius-medium) var(--el-border-radius-medium) 0 0;
    font-size: var(--el-font-size-medium);
  }
  &__tabs :deep(.el-tabs__item.is-active) {
    color: var(--el-color-primary);
    background: var(--el-bg-color);
  }
  &__tabs :deep(.el-tabs__active-bar) {
    display: none;
  }
  &__panel {
    max-width: 1220px;
  }
  &__panel h2 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 650;
    line-height: 26px;
  }
  &__tips {
    margin: var(--el-space-md) 0 var(--el-space-3xl);
    padding-left: var(--el-space-2xl);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 28px;
  }
  &__tips button,
  &__text-link {
    padding: 0 var(--el-space-xs);
    border: 0;
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font: inherit;
  }
  &__section-title {
    margin-top: var(--el-space-3xl) !important;
  }
  &__intro,
  &__batch-notice {
    margin: var(--el-space-md) 0 var(--el-space-3xl);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);
    line-height: 26px;
  }
  &__add {
    display: inline-flex;
    height: 34px;
    padding: 0;
    border: 0;
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font-size: var(--el-font-size-medium);
  }
  &__add:hover {
    background: var(--el-color-primary-light-9);
  }
  &__add svg {
    width: 20px;
    height: 20px;
  }
  &__cards {
    display: grid;
    margin-top: var(--el-space-lg);
    gap: var(--el-space-xl);
  }
  &__upload :deep(.el-upload-dragger) {
    display: grid;
    width: 1168px;
    max-width: 100%;
    min-height: 320px;
    padding: 0;
    border-color: var(--el-border-color-light);
    place-content: center;
  }
  &__upload :deep(.el-upload-dragger > svg) {
    width: 26px;
    height: 26px;
    margin: 0 auto var(--el-space-md);
    color: var(--el-color-primary);
  }
  &__upload :deep(.el-upload-dragger > span) {
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-medium);
  }
  &__upload :deep(.el-upload__tip) {
    margin-top: var(--el-space-2xl);
    text-align: center;
  }
  &__upload :deep(.el-upload__tip button) {
    border: 0;
    color: var(--el-color-primary);
    background: transparent;
    cursor: pointer;
    font-size: var(--el-font-size-base);
    text-decoration: underline;
  }
  &__upload :deep(.el-upload__tip svg) {
    width: 16px;
    height: 16px;
    vertical-align: -3px;
  }
  &__batch-notice {
    margin-top: var(--el-space-4xl);
  }
  &__batch-result {
    display: grid;
    width: 1168px;
    max-width: 100%;
    height: 78px;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    font-size: var(--el-font-size-medium);
    place-items: center;
  }
}

:global(.external-organization-invite-dialog) {
  /* 全屏邀请层附加在 body，需与固定浅色的管理后台保持一致。 */
  --el-bg-color-overlay: #ffffff;
  --el-fill-color: #f4f6fa;
  --el-fill-color-light: #f7f8fc;
  --el-fill-color-lighter: #fafbfc;
  --el-fill-color-blank: #ffffff;
  --el-text-color-primary: #202938;
  --el-text-color-regular: #515968;
  --el-text-color-secondary: #8a94a6;
  --el-border-color: #dbe1eb;
  --el-border-color-light: #e7eaf0;
  width: 100vw;
  height: calc(100vh - 62px);
  margin: 62px 0 0;
  border-radius: 0;
  background: var(--el-bg-color);
  color-scheme: light;
}
:global(.external-organization-invite-dialog .el-dialog__header) {
  margin: 0;
  padding: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
:global(.external-organization-invite-dialog .el-dialog__body) {
  padding: 0;
}

@media (max-width: 720px) {
  .external-organization-invite-dialog {
    &__surface {
      margin: var(--el-space-xl);
      padding: var(--el-space-2xl);
    }
    &__tabs :deep(.el-tabs__item) {
      padding: 0 var(--el-space-xl);
    }
  }
}
</style>
