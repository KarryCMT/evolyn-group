<script setup lang="ts">
import type {
  FormSchemaIssue,
  FormSchemaPaletteGroup,
  FormTabStyle,
  FormWidgetType,
} from '@evolyn.do/form/designer';
import type { FormRuntimeAdapter } from '@evolyn.do/form/runtime-core';
import {
  FORM_PROTOCOL_VERSION,
  FormSchemaCanvas,
  FormSchemaPalette,
  FormSchemaPropertyPanel,
  migrateFormSchema,
  useFormSchemaEditor,
  validateFormSchema,
  validatePublishableFormSchema,
  WIDGET_GROUP_META,
  WIDGET_SPECS,
} from '@evolyn.do/form/designer';
import { ApiError } from '@evolyn.do/utils';
import {
  RiArrowDownBoxFill,
  RiCalendarScheduleFill,
  RiCheckboxMultipleFill,
  RiCheckDoubleFill,
  RiEyeFill,
  RiFileTextFill,
  RiHashtag,
  RiLightbulbFlashFill,
  RiMenuFill,
  RiRadioButtonFill,
  RiSave3Fill,
  RiShareForwardFill,
  RiText,
  RiUploadCloud2Fill,
} from '@remixicon/vue';
import { ElMessage } from 'element-plus';
import { computed, ref, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getApplicationByCode } from '~/api/applications';
import { createForm, publishForm, saveFormDraft } from '~/api/form';
import FormDesignPreviewDrawer from '~/components/form/FormDesignPreviewDrawer.vue';
import { useFormWorkspaceContext } from './workspace-context';

defineOptions({ name: 'FormDesignPage' });

const route = useRoute();
const router = useRouter();
const workspace = useFormWorkspaceContext();

/** 编辑器状态：页面唯一持有字段定义与布局引用（目标保存协议 v2）。 */
const editor = useFormSchemaEditor();
const { document, items, selectedKey, selectedItem, selectedLayout } = editor;

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
const previewVisible = shallowRef(false);
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
  const result = migrateFormSchema(detail.draft, detail.protocolVersion);
  if (!result.document) {
    showIssues(result.issues);
    loadFailed.value = true;
    return;
  }
  editor.replaceDocument(result.document);
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
  return [
    ...WIDGET_GROUP_META.map((group) => ({
      key: group.key,
      title: group.title,
      enabled: group.key === 'basic',
      entries: Object.entries(WIDGET_SPECS)
        .filter(([, spec]) => spec.group === group.key)
        .map(([type, spec]) => ({
          type,
          label: spec.label,
          icon: iconOfType[type] ?? RiMenuFill,
          // P4 首先开放子表单设计能力；同组关联字段仍按原计划保持不可添加。
          enabled: group.key === 'basic' || type === 'subform',
        })),
    })),
    {
      key: 'layout',
      title: '布局组件',
      enabled: true,
      entries: [{ type: 'multitab', label: '标签页', icon: RiMenuFill }],
    },
  ];
});

function onAddField(entry: { type: string }): void {
  if (entry.type === 'multitab') {
    editor.addMultitab();
    return;
  }
  editor.addItem(entry.type as FormWidgetType);
}

function setSelectedTabStyle(style: FormTabStyle): void {
  if (selectedLayout.value) editor.setTabStyle(selectedLayout.value.name, style);
}

function addSelectedTab(): void {
  if (selectedLayout.value) editor.addTab(selectedLayout.value.name);
}

function removeSelectedTab(tabName: string): void {
  if (selectedLayout.value) editor.removeTab(selectedLayout.value.name, tabName);
}

function duplicateSelectedTab(tabName: string): void {
  if (selectedLayout.value) editor.duplicateTab(selectedLayout.value.name, tabName);
}

function renameSelectedTab(tabName: string, title: string): void {
  if (selectedLayout.value) editor.renameTab(selectedLayout.value.name, tabName, title);
}

function reorderSelectedTabs(tabNames: string[]): void {
  if (selectedLayout.value) editor.reorderTabs(selectedLayout.value.name, tabNames);
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

/** 设计器预览不写入记录，仅模拟完成提交，方便即时验证字段交互。 */
const previewAdapter: FormRuntimeAdapter = {
  async submit() {
    return { accepted: true };
  },
  async saveDraft() {
    // 仅验证填写草稿交互，不调用设计结构草稿接口，也不持久化用户填写值。
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

function onPreviewDraftSuccess(): void {
  ElMessage.info('预览模式不会保存填写草稿');
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
    const result = await saveFormDraft(
      formCode.value,
      draftRevision.value,
      FORM_PROTOCOL_VERSION,
      document.value,
    );
    draftRevision.value = result.draftRevision;
    workspace.patchDetail({
      draftRevision: result.draftRevision,
      protocolVersion: FORM_PROTOCOL_VERSION,
      draft: document.value,
    });
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
    const saved = await saveFormDraft(
      formCode.value,
      draftRevision.value,
      FORM_PROTOCOL_VERSION,
      document.value,
    );
    const result = await publishForm(formCode.value, saved.draftRevision);
    draftRevision.value = saved.draftRevision;
    publishedVersion.value = result.publishedVersion;
    workspace.patchDetail({
      draftRevision: saved.draftRevision,
      protocolVersion: FORM_PROTOCOL_VERSION,
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
        :document="document"
        :selected-key="selectedKey"
        @select-item="editor.selectItem"
        @select-layout="editor.selectLayout"
        @copy-item="editor.copyItem"
        @remove-item="editor.removeItem"
        @select-subform-item="editor.selectSubformItem"
        @copy-subform-item="editor.copySubformItem"
        @remove-subform-item="editor.removeSubformItem"
        @replace-subform-items="editor.replaceSubformItems"
        @replace-references="editor.replaceReferences($event.target, $event.entries)"
        @remove-layout="editor.removeMultitab"
      />
      <FormSchemaPropertyPanel
        :item="selectedItem"
        :layout="selectedLayout"
        :form-layout="document.content.layout"
        :form-name="formName"
        :form-name-saving="renaming"
        @rename-key="editor.renameItemKey"
        @update-item="editor.updateSelectedItem"
        @update-form-name="onUpdateFormName"
        @update-form-layout="editor.setFormLayout"
        @set-tab-style="setSelectedTabStyle"
        @add-tab="addSelectedTab"
        @remove-tab="removeSelectedTab"
        @duplicate-tab="duplicateSelectedTab"
        @rename-tab="renameSelectedTab"
        @reorder-tabs="reorderSelectedTabs"
      />
    </div>

    <FormDesignPreviewDrawer
      v-model="previewVisible"
      :schema="document"
      :form-id="formCode"
      :adapter="previewAdapter"
      @unsupported-field="onUnsupportedPreviewField"
      @submit-success="onPreviewSubmitSuccess"
      @draft-success="onPreviewDraftSuccess"
    />
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
}
</style>
