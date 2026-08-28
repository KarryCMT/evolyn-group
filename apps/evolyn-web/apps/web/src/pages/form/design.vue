<script setup lang="ts">
import {
  RiArrowDownBoxFill,
  RiCalendarScheduleFill,
  RiCheckDoubleFill,
  RiCloseFill,
  RiCheckboxMultipleFill,
  RiComputerFill,
  RiEyeFill,
  RiFileTextFill,
  RiHashtag,
  RiLightbulbFlashFill,
  RiMenuFill,
  RiRadioButtonFill,
  RiSave3Fill,
  RiShareForwardFill,
  RiSmartphoneFill,
  RiText,
  RiUploadCloud2Fill,
} from '@remixicon/vue';
import {
  type FormSchemaIssue,
  type FormSchemaPaletteGroup,
  FormSchemaCanvas,
  FormSchemaPalette,
  FormSchemaPropertyPanel,
  useFormSchemaEditor,
  validateFormSchema,
  validatePublishableFormSchema,
  WIDGET_GROUP_META,
  WIDGET_SPECS,
  type FormWidgetType,
} from '@evolyn.do/form/designer';
import { FormRenderer, type FormRuntimeAdapter } from '@evolyn.do/form/runtime';
import { ApiError } from '@evolyn.do/utils';
import { ElMessage } from 'element-plus';
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getApplicationByCode } from '~/api/applications';
import { createForm, publishForm, saveFormDraft } from '~/api/form';
import { useFormWorkspaceContext } from './workspace-context';
// 设计器内预览按运行时独立样式加载，避免依赖设计器样式副作用。
import '@evolyn.do/form/runtime/style.css';

defineOptions({ name: 'FormDesignPage' });

const route = useRoute();
const router = useRouter();
const workspace = useFormWorkspaceContext();

/** 编辑器状态：页面唯一持有 content.items（目标保存协议，ADR-010）。 */
const editor = useFormSchemaEditor();
const { document, items, selectedKey, selectedItem } = editor;

/** 路由参数：'new' 为历史兜底新建态；其余使用 form_ 前缀稳定公开编码。 */
const rawFormCode = computed(() => String(route.params.formCode ?? ''));
const isNewForm = computed(() => rawFormCode.value === 'new');
const formCode = computed(() => (rawFormCode.value.startsWith('form_') ? rawFormCode.value : ''));

/** 草稿口令与表单元信息：保存/发布回传后同步刷新。 */
const draftRevision = ref(0);
const formName = computed(() => workspace.detail.value?.name ?? '');
const publishedVersion = ref(0);
const loadFailed = ref(false);
const loading = ref(true);
const saving = ref(false);
const publishing = ref(false);
const renaming = computed(() => workspace.renaming.value);
const previewVisible = ref(false);
/** 预览画布设备：用真实宽度和栅格切换校验不同终端的填写体验。 */
const previewViewport = ref<'desktop' | 'mobile'>('desktop');
const unsupportedPreviewTypes = new Set<string>();

// 外壳是详情接口的唯一请求方；设计页仅消费共享响应，避免相同详情重复调用。
watch(
  [() => workspace.detail.value?.code, workspace.loading, workspace.loadFailed],
  ([loadedCode, workspaceLoading, workspaceFailed]) => {
    if (isNewForm.value) {
      if (!workspaceLoading && !loadedCode) void startNewForm();
      return;
    }
    if (!formCode.value) {
      loadFailed.value = true;
      loading.value = false;
      return;
    }
    loading.value = workspaceLoading;
    if (workspaceFailed) {
      loadFailed.value = true;
      loading.value = false;
      return;
    }
    const detail = workspace.detail.value;
    if (detail && detail.code === formCode.value) {
      adoptDetail(detail);
      loading.value = false;
    }
  },
  { immediate: true },
);

/**
 * 新建表单兜底：主入口已前移到应用工作台（点「新建表单/新建流程表单」直接
 * 使用「未命名表单」和持久化 formType 调 POST /forms，拿稳定 code 后跳转设计器，
 * 再在属性面板改名）；
 * 此处仅防御直接落地的 `/form/new/design` 路由——解析应用 ID → 使用默认名称创建资产与空草稿 →
 * 以稳定 code 替换路由。query 中的 parentEntryCode（目标分组编码）随创建
 * 请求消费；旧版 type 临时标记仅清理，不再参与类型判断。
 */
async function startNewForm(): Promise<void> {
  loading.value = true;
  try {
    const app = await getApplicationByCode(String(route.params.appCode ?? ''));
    // 兜底入口仅消费目标分组编码；表单类型不再从 query 推断。
    const parentEntryCode =
      typeof route.query.parentEntryCode === 'string' ? route.query.parentEntryCode : undefined;
    const detail = await createForm({
      applicationId: app.id,
      name: '未命名表单',
      formType: 'standard',
      parentEntryCode,
    });
    workspace.setDetail(detail);
    adoptDetail(detail);
    const { parentEntryCode: _parentEntryCode, type: _legacyType, ...restQuery } = route.query;
    await router.replace({
      name: 'form-design',
      params: { ...route.params, formCode: detail.code },
      query: restQuery,
    });
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'QUOTA_EXCEEDED') {
      ElMessage.error('表单数量已达套餐上限，请升级套餐或删除闲置表单');
    } else if (error instanceof ApiError && error.errCode === 'APP_MENU_PARENT_INVALID') {
      ElMessage.error('目标分组不存在或已删除，请返回工作台刷新后重试');
    } else {
      ElMessage.error('创建表单失败，请稍后重试');
    }
    loadFailed.value = true;
  } finally {
    loading.value = false;
  }
}

/** 以详情响应填充编辑态（加载与新建共用；草稿读取侧完成协议校验）。 */
function adoptDetail(detail: NonNullable<(typeof workspace.detail)['value']>): void {
  const result = validateFormSchema(detail.draft);
  if (!result.valid) {
    showIssues(result.issues);
    loadFailed.value = true;
    return;
  }
  editor.replaceDocument(result.document!);
  draftRevision.value = detail.draftRevision;
  publishedVersion.value = detail.publishedVersion;
  loadFailed.value = false;
}

/** 素材面板分组：基础字段可添加，其余分组置灰展示（后续阶段开放）。 */
const paletteGroups = computed<FormSchemaPaletteGroup[]>(() => {
  const iconOfType: Record<string, unknown> = {
    text: RiText,
    textarea: RiFileTextFill,
    number: RiHashtag,
    datetime: RiCalendarScheduleFill,
    radiogroup: RiRadioButtonFill,
    checkboxgroup: RiCheckboxMultipleFill,
    combo: RiArrowDownBoxFill,
    combocheck: RiCheckDoubleFill,
    separator: RiMenuFill,
  };
  return WIDGET_GROUP_META.map((group) => ({
    key: group.key,
    title: group.title,
    enabled: group.key === 'basic',
    entries: Object.entries(WIDGET_SPECS)
      .filter(([, spec]) => spec.group === group.key)
      .map(([type, spec]) => ({
        type,
        label: spec.label,
        icon: iconOfType[type] ?? RiMenuFill,
      })),
  }));
});

function onAddField(entry: { type: string }): void {
  editor.addItem(entry.type as FormWidgetType);
}

function onAddDragField(value: { type: string; index: number }): void {
  editor.addItem(value.type as FormWidgetType, value.index);
}

/** 表单属性面板改名：名称属于表单资产，而非草稿协议内容。 */
async function onUpdateFormName(name: string): Promise<void> {
  await workspace.rename(name);
}

/** 预览直接在当前工作台底部抽屉中打开，始终使用画布中的最新草稿。 */
function openPreview(): void {
  if (!items.value.length) {
    ElMessage.info('请先添加字段再预览');
    return;
  }
  previewVisible.value = true;
}

function setPreviewViewport(viewport: 'desktop' | 'mobile'): void {
  previewViewport.value = viewport;
}

/** 设计器预览不写入记录，仅模拟完成提交，方便即时验证字段交互。 */
const previewAdapter: FormRuntimeAdapter = {
  async submit() {
    return { accepted: true };
  },
};

function onUnsupportedPreviewField(info: { fieldKey: string; type: string }): void {
  if (unsupportedPreviewTypes.has(info.type)) return;
  unsupportedPreviewTypes.add(info.type);
  ElMessage.info(`字段类型「${info.type}」的填写能力尚未上线，预览中暂不可交互`);
}

function onPreviewSubmitSuccess(): void {
  ElMessage.success('预览提交成功');
}

/** 保存草稿：本地校验先行（前后端一致），服务端口令递增。 */
async function saveDraft(): Promise<void> {
  const local = validateFormSchema(document.value);
  if (!local.valid) {
    showIssues(local.issues);
    return;
  }
  saving.value = true;
  try {
    const result = await saveFormDraft(formCode.value, draftRevision.value, document.value);
    draftRevision.value = result.draftRevision;
    workspace.patchDetail({ draftRevision: result.draftRevision, draft: document.value });
    ElMessage.success('保存成功');
  } catch (error) {
    handleSaveError(error);
  } finally {
    saving.value = false;
  }
}

/** 发布：白名单 + 协议校验先行，成功提示版本号。 */
async function publish(): Promise<void> {
  const publishable = validatePublishableFormSchema(document.value);
  if (!publishable.valid) {
    showIssues(publishable.issues);
    return;
  }
  publishing.value = true;
  try {
    // 发布前先落草稿，保证发布的就是当前画布内容（口令以保存结果为准）。
    const saved = await saveFormDraft(formCode.value, draftRevision.value, document.value);
    const result = await publishForm(formCode.value, saved.draftRevision);
    draftRevision.value = saved.draftRevision;
    publishedVersion.value = result.publishedVersion;
    workspace.patchDetail({
      draftRevision: saved.draftRevision,
      draft: document.value,
      publishedVersion: result.publishedVersion,
    });
    ElMessage.success(`发布成功（版本 ${result.publishedVersion}）`);
  } catch (error) {
    handleSaveError(error);
  } finally {
    publishing.value = false;
  }
}

/** 保存/发布错误按 errCode 分支：结构问题展示路径清单，口令冲突刷新草稿。 */
function handleSaveError(error: unknown): void {
  if (error instanceof ApiError) {
    if (
      error.errCode === 'FORM_SCHEMA_INVALID' ||
      error.errCode === 'FORM_PUBLISH_UNSUPPORTED_FIELD'
    ) {
      const issues = (error.data as { issues?: FormSchemaIssue[] } | undefined)?.issues;
      showIssues(issues ?? []);
      return;
    }
    if (error.errCode === 'FORM_REVISION_CONFLICT') {
      ElMessage.warning('表单已被他人更新，正在重新加载');
      void workspace.reload().then((detail) => {
        if (detail) adoptDetail(detail);
      });
      return;
    }
  }
  ElMessage.error('操作失败，请稍后重试');
}

/** 展示 JSON Path 级问题清单（最多前 8 条，避免刷屏）。 */
function showIssues(issues: FormSchemaIssue[]): void {
  if (!issues.length) {
    ElMessage.error('操作失败，请稍后重试');
    return;
  }
  const lines = issues
    .slice(0, 8)
    .map((issue) => `${issue.path}：${issue.message}`)
    .join('\n');
  ElMessage({
    type: 'error',
    duration: 6000,
    showClose: true,
    message: issues.length > 8 ? `${lines}\n… 共 ${issues.length} 处问题` : lines,
  });
}

/** 保存等服务端能力尚未接入的占位交互。 */
function notifyUnavailable(action: string) {
  ElMessage.info(`${action}将在表单设计器接入后提供`);
}
</script>

<template>
  <section class="form-design-page" aria-label="表单设计工作台">
    <div class="form-design-page__toolbar" aria-label="表单设计操作">
      <div class="form-design-page__title">
        <span class="form-design-page__title-text">{{ formName || '表单设计' }}</span>
        <span v-if="publishedVersion > 0" class="form-design-page__title-tag">
          已发布 v{{ publishedVersion }}
        </span>
      </div>
      <div class="form-design-page__toolbar-actions">
        <button
          class="form-design-page__guide-button"
          type="button"
          @click="notifyUnavailable('新手引导')"
        >
          <RiLightbulbFlashFill />
          <span class="form-design-page__guide-label">查看新手引导</span>
        </button>
        <button
          class="form-design-page__action-button form-design-page__action-button--secondary"
          type="button"
          @click="openPreview"
        >
          <RiEyeFill />
          <span class="form-design-page__action-label">预览</span>
        </button>
        <button
          class="form-design-page__action-button form-design-page__action-button--secondary"
          type="button"
          :disabled="saving || loading || loadFailed"
          @click="saveDraft"
        >
          <RiSave3Fill />
          <span class="form-design-page__action-label">保存</span>
        </button>
        <button
          class="form-design-page__action-button form-design-page__action-button--primary"
          type="button"
          :disabled="publishing || loading || loadFailed"
          @click="publish"
        >
          <RiUploadCloud2Fill />
          <span class="form-design-page__action-label">发布</span>
        </button>
        <button
          class="form-design-page__icon-button form-design-page__share-button"
          type="button"
          aria-label="分享表单"
          @click="notifyUnavailable('分享')"
        >
          <RiShareForwardFill />
        </button>
      </div>
    </div>

    <div v-if="loading" class="form-design-page__state" role="status">正在加载表单…</div>
    <div
      v-else-if="loadFailed"
      class="form-design-page__state form-design-page__state--error"
      role="alert"
    >
      表单加载失败：请确认链接有效或刷新重试
    </div>

    <div v-else class="form-design-page__workspace">
      <FormSchemaPalette :groups="paletteGroups" @add-field="onAddField" />
      <FormSchemaCanvas
        :items="items"
        :selected-key="selectedKey"
        @select-item="editor.selectItem"
        @copy-item="editor.copyItem"
        @remove-item="editor.removeItem"
        @add-field="onAddDragField"
      />
      <FormSchemaPropertyPanel
        :item="selectedItem"
        :form-name="formName"
        :form-name-saving="renaming"
        @rename-key="editor.renameItemKey"
        @update-form-name="onUpdateFormName"
      />
    </div>

    <el-drawer
      v-model="previewVisible"
      append-to-body
      direction="btt"
      size="calc(100% - 40px)"
      :with-header="false"
      body-class="form-design-preview-drawer__body"
      class="form-design-preview-drawer"
    >
      <section class="form-design-preview" aria-label="表单预览">
        <header class="form-design-preview__header">
          <div class="form-design-preview__header-spacer" aria-hidden="true" />
          <div class="form-design-preview__viewport-switch" role="group" aria-label="预览设备">
            <button
              class="form-design-preview__viewport-button"
              :class="{
                'form-design-preview__viewport-button--active': previewViewport === 'desktop',
              }"
              type="button"
              :aria-pressed="previewViewport === 'desktop'"
              @click="setPreviewViewport('desktop')"
            >
              <RiComputerFill />
              <span>桌面端</span>
            </button>
            <button
              class="form-design-preview__viewport-button"
              :class="{
                'form-design-preview__viewport-button--active': previewViewport === 'mobile',
              }"
              type="button"
              :aria-pressed="previewViewport === 'mobile'"
              @click="setPreviewViewport('mobile')"
            >
              <RiSmartphoneFill />
              <span>移动端</span>
            </button>
          </div>
          <button
            class="form-design-preview__close"
            type="button"
            aria-label="关闭预览"
            @click="previewVisible = false"
          >
            <RiCloseFill />
          </button>
        </header>
        <el-scrollbar class="form-design-preview__body">
          <div
            class="form-design-preview__canvas"
            :class="`form-design-preview__canvas--${previewViewport}`"
          >
            <FormRenderer
              class="form-design-preview__runtime"
              :class="`form-design-preview__runtime--${previewViewport}`"
              :schema="document"
              :form-id="formCode"
              :adapter="previewAdapter"
              @unsupported-field="onUnsupportedPreviewField"
              @submit-success="onPreviewSubmitSuccess"
            />
          </div>
        </el-scrollbar>
      </section>
    </el-drawer>
  </section>
</template>

<style scoped lang="scss">
.form-design-page {
  display: flex;
  min-height: 0;
  margin: 0 var(--el-space-md) var(--el-space-md);
  overflow: hidden;
  flex: 1;
  flex-direction: column;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-large);
  box-shadow: var(--el-box-shadow-light);

  &__toolbar,
  &__toolbar-actions,
  &__guide-button,
  &__action-button {
    display: flex;
    align-items: center;
  }

  &__toolbar {
    height: 50px;
    min-height: 50px;
    padding: 0 var(--el-space-xl) 0 var(--el-space-3xl);
    justify-content: space-between;
  }

  &__title {
    display: flex;
    gap: var(--el-space-md);
    align-items: center;
    min-width: 0;
  }

  &__title-text {
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: var(--el-font-size-medium);
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__title-tag {
    padding: 0 var(--el-space-sm);
    color: var(--el-color-primary);
    font-size: var(--el-font-size-extra-small);
    font-weight: 500;
    line-height: 20px;
    background: var(--el-color-primary-light-9);
    border-radius: var(--el-border-radius-base);
  }

  &__guide-button,
  &__action-button,
  &__icon-button {
    border: 0;
    cursor: pointer;

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }

    &:disabled {
      cursor: not-allowed;
      opacity: 0.55;
    }
  }

  &__guide-button,
  &__action-button {
    justify-content: center;
    gap: var(--el-space-sm);
    font-size: var(--el-font-size-base);
    font-weight: 600;
  }

  &__guide-button {
    height: 32px;
    padding: 0 var(--el-space-md);
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 18px;
      height: 18px;
      color: var(--el-color-primary);
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__toolbar-actions {
    gap: var(--el-space-md);
  }

  &__action-button {
    min-width: 76px;
    height: 32px;
    padding: 0 var(--el-space-lg);
    border-radius: var(--el-border-radius-base);

    svg {
      width: 17px;
      height: 17px;
    }

    &--secondary {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      border: 1px solid var(--el-color-primary);

      &:hover:not(:disabled) {
        background: var(--el-color-primary-light-9);
      }
    }

    &--primary {
      color: var(--el-color-white);
      background: var(--el-color-primary);

      &:hover:not(:disabled) {
        background: var(--el-color-primary-light-3);
      }
    }
  }

  &__icon-button {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    border-radius: var(--el-border-radius-base);

    svg {
      width: 20px;
      height: 20px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  &__share-button {
    border: 1px solid var(--el-border-color);

    &:hover {
      border-color: var(--el-color-primary);
    }
  }

  &__state {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-base);

    &--error {
      color: var(--el-color-danger);
    }
  }

  &__workspace {
    display: flex;
    min-height: 0;
    flex: 1;
    overflow: hidden;
  }
}

// Drawer 传送到 body，使用唯一块类约束局部覆盖，避免影响其他抽屉。
:global(.form-design-preview-drawer) {
  border-radius: var(--el-border-radius-large) var(--el-border-radius-large) 0 0;
}

:global(.form-design-preview-drawer__body) {
  display: flex;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

.form-design-preview {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--el-bg-color-page);

  &__header {
    display: grid;
    height: 56px;
    min-height: 56px;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    padding: 0 var(--el-space-3xl);
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__viewport-switch {
    display: inline-flex;
    gap: var(--el-space-xs);
    padding: var(--el-space-xs);
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);
  }

  &__viewport-button {
    display: inline-flex;
    height: 28px;
    align-items: center;
    justify-content: center;
    gap: var(--el-space-sm);
    padding: 0 var(--el-space-lg);
    color: var(--el-text-color-secondary);
    font-size: var(--el-font-size-small);
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-small);
    cursor: pointer;

    svg {
      width: 16px;
      height: 16px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &--active {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      box-shadow: var(--el-box-shadow-lighter);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__close {
    display: inline-flex;
    width: 32px;
    height: 32px;
    margin-left: auto;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    cursor: pointer;

    svg {
      width: 22px;
      height: 22px;
    }

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__body {
    min-height: 0;
    flex: 1;
  }

  &__canvas {
    width: 704px;
    max-width: calc(100% - var(--el-space-4xl));
    min-height: 100%;
    margin: 0 auto;
    padding: var(--el-space-3xl) var(--el-space-md) var(--el-space-4xl);

    &--mobile {
      width: 375px;
    }
  }

  // 运行时主题映射到宿主 Element Plus 变量，暗色模式自动跟随。
  &__runtime {
    --evf-color-text: var(--el-text-color-primary);
    --evf-color-text-regular: var(--el-text-color-regular);
    --evf-color-text-secondary: var(--el-text-color-secondary);
    --evf-color-text-placeholder: var(--el-text-color-placeholder);
    --evf-color-text-disabled: var(--el-text-color-disabled);
    --evf-color-border: var(--el-border-color);
    --evf-color-border-light: var(--el-border-color-light);
    --evf-color-border-lighter: var(--el-border-color-lighter);
    --evf-color-fill-light: var(--el-fill-color-light);
    --evf-color-bg: var(--el-bg-color);
    --evf-color-primary: var(--el-color-primary);
    --evf-color-danger: var(--el-color-danger);
    --evf-font-size-base: var(--el-font-size-base);
    --evf-font-size-small: var(--el-font-size-small);
    --evf-font-size-extra-small: var(--el-font-size-extra-small);
    --evf-space-sm: var(--el-space-sm);
    --evf-space-md: var(--el-space-md);
    --evf-space-lg: var(--el-space-lg);
    --evf-space-xl: var(--el-space-xl);
    --evf-space-3xl: var(--el-space-3xl);
    --evf-radius-base: var(--el-border-radius-base);
    --evf-radius-medium: var(--el-border-radius-medium);
    display: flex;
    min-height: calc(100vh - 144px);
    flex-direction: column;
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-large);
    box-shadow: var(--el-box-shadow-light);

    :deep(.evf-form__body) {
      flex: 1;
    }

    &--mobile {
      --evf-columns: 1;
      --evf-control-height: 44px;
    }
  }
}

@media (max-width: 620px) {
  .form-design-page {
    margin: 0 var(--el-space-xs) var(--el-space-xs);
    border-radius: var(--el-border-radius-large);

    &__toolbar {
      padding: 0 var(--el-space-md) 0 var(--el-space-lg);
    }

    &__guide-button {
      padding: 0 var(--el-space-xs);
    }

    &__guide-label,
    &__action-label {
      display: none;
    }

    &__toolbar-actions {
      gap: var(--el-space-sm);
    }

    &__action-button {
      min-width: 34px;
      padding: 0 var(--el-space-md);
    }
  }

  .form-design-preview {
    &__header {
      height: 52px;
      min-height: 52px;
      padding: 0 var(--el-space-lg);
    }

    &__viewport-button {
      padding: 0 var(--el-space-md);

      span {
        display: none;
      }
    }

    &__canvas {
      padding: var(--el-space-lg) var(--el-space-xs) var(--el-space-3xl);

      &--desktop,
      &--mobile {
        width: 100%;
        max-width: 100%;
      }
    }

    &__runtime {
      min-height: calc(100vh - 124px);
    }
  }
}
</style>
