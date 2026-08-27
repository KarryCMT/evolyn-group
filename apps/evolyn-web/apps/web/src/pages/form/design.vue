<script setup lang="ts">
import {
  RiArrowDownBoxFill,
  RiCalendarScheduleFill,
  RiCheckDoubleFill,
  RiCheckboxMultipleFill,
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
import { ApiError } from '@evolyn.do/utils';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getApplicationByCode } from '~/api/applications';
import { createForm, getForm, publishForm, saveFormDraft } from '~/api/form';
import { saveFormPreviewDocument } from './preview-storage';

defineOptions({ name: 'FormDesignPage' });

const route = useRoute();
const router = useRouter();

/** 编辑器状态：页面唯一持有 content.items（目标保存协议，ADR-010）。 */
const editor = useFormSchemaEditor();
const { document, items, selectedKey, selectedItem } = editor;

/** 路由参数：'new' 为工作台新建态（先创建资产再替换为真实 ID）；其余须为正整数。 */
const rawFormId = computed(() => String(route.params.formId ?? ''));
const isNewForm = computed(() => rawFormId.value === 'new');
const formId = computed(() => {
  const id = Number(rawFormId.value);
  return Number.isInteger(id) && id > 0 ? id : 0;
});

/** 草稿口令与表单元信息：保存/发布回传后同步刷新。 */
const draftRevision = ref(0);
const formName = ref('');
const publishedVersion = ref(0);
const loadFailed = ref(false);
const loading = ref(true);
const saving = ref(false);
const publishing = ref(false);

onMounted(() => {
  if (isNewForm.value) {
    void startNewForm();
    return;
  }
  if (!formId.value) {
    // 非法路由参数（NaN/0/负数）不发请求，直接进入受控错误态。
    loadFailed.value = true;
    loading.value = false;
    return;
  }
  void loadDraft();
});

/**
 * 新建表单兜底：主入口已前移到应用工作台（点「新建表单/新建流程表单」即调
 * POST /forms 拿真实 ID 后携 ID 跳转设计器）；此处仅防御直接落地的
 * `/form/new/design` 路由——命名 → 解析应用 ID → 创建资产与空草稿 →
 * 以真实 ID 替换路由。query 中的 parentEntryCode（目标分组编码）随创建
 * 请求消费，路由替换后不再携带。
 */
async function startNewForm(): Promise<void> {
  let name: string;
  try {
    const { value } = await ElMessageBox.prompt('请输入表单名称', '新建表单', {
      inputValue: '未命名表单',
      inputValidator: (input: string) =>
        input && input.trim() ? true : '请输入 1–128 个字符的表单名称',
      confirmButtonText: '创建',
      cancelButtonText: '取消',
      closeOnClickModal: false,
    });
    name = value.trim();
  } catch {
    // 取消命名即放弃创建，返回来源页（应用工作台）。
    void router.push({ name: 'App', params: { appCode: route.params.appCode } });
    return;
  }

  loading.value = true;
  try {
    const app = await getApplicationByCode(String(route.params.appCode ?? ''));
    // 兜底入口允许经 query 携带目标分组编码（type=workflow 等其余 query 原样保留）
    const parentEntryCode =
      typeof route.query.parentEntryCode === 'string' ? route.query.parentEntryCode : undefined;
    const detail = await createForm({ applicationId: app.id, name, parentEntryCode });
    adoptDetail(detail);
    const { parentEntryCode: _consumed, ...restQuery } = route.query;
    await router.replace({
      name: 'form-design',
      params: { ...route.params, formId: String(detail.id) },
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
function adoptDetail(detail: Awaited<ReturnType<typeof getForm>>): void {
  const result = validateFormSchema(detail.draft);
  if (!result.valid) {
    showIssues(result.issues);
    loadFailed.value = true;
    return;
  }
  editor.replaceDocument(result.document!);
  draftRevision.value = detail.draftRevision;
  formName.value = detail.name;
  publishedVersion.value = detail.publishedVersion;
  loadFailed.value = false;
}

/** 加载草稿：读取侧完成校验，无效配置进入受控错误态。 */
async function loadDraft(): Promise<void> {
  loading.value = true;
  try {
    adoptDetail(await getForm(formId.value));
  } catch {
    loadFailed.value = true;
  } finally {
    loading.value = false;
  }
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

/** 预览：草稿全文经会话存储传递，预览页已发布时优先走 bootstrap。 */
function openPreview(): void {
  if (!items.value.length) {
    ElMessage.info('请先添加字段再预览');
    return;
  }
  saveFormPreviewDocument(String(formId.value), document.value);
  router.push({ name: 'form-preview', params: { ...route.params, formId: String(formId.value) } });
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
    const result = await saveFormDraft(formId.value, draftRevision.value, document.value);
    draftRevision.value = result.draftRevision;
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
    const saved = await saveFormDraft(formId.value, draftRevision.value, document.value);
    const result = await publishForm(formId.value, saved.draftRevision);
    draftRevision.value = saved.draftRevision;
    publishedVersion.value = result.publishedVersion;
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
      void loadDraft();
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
      <FormSchemaPropertyPanel :item="selectedItem" @rename-key="editor.renameItemKey" />
    </div>
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
    border-bottom: 1px solid var(--el-border-color-lighter);
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
