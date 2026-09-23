<script setup lang="ts">
import type {
  WorkflowActorOptions,
  WorkflowDepartmentOption,
  WorkflowDocument,
  WorkflowField,
  WorkflowIssue,
} from '@evolyn.do/workflow';
import type { WorkflowDetailDto, WorkflowVersionDto } from '~/types';
import { ApiError } from '@evolyn.do/utils';
import { createWorkflowDocument, normalizeWorkflowDocument, WorkflowDesigner } from '@evolyn.do/workflow';
import {
  RiArrowDownSLine,
  RiFullscreenFill,
  RiHistoryFill,
  RiSave3Fill,
  RiUpload2Fill,
} from '@remixicon/vue';
import {
  ElAlert,
  ElButton,
  ElDialog,
  ElMessage,
  ElPopover,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';
import { computed, onMounted, onUnmounted, shallowRef, useTemplateRef, watch } from 'vue';
import { onBeforeRouteLeave } from 'vue-router';
import { getDepartmentTree } from '~/api/department';
import { listMembers } from '~/api/member';
import { getOrganizationRoleTree } from '~/api/role';
import {
  createWorkflow,
  createWorkflowDraftVersion,
  getWorkflow,
  getWorkflowVersion,
  listWorkflows,
  listWorkflowVersions,
  publishWorkflow,
  saveWorkflowDraft,
} from '~/api/workflow';
import { useFormWorkspaceContext } from './workspace-context';

defineOptions({ name: 'FormWorkflowPage' });

/**
 * 流程设计工作区（Phase 9）：以绑定表单（form_code）定位/懒建流程定义，
 * DSL 草稿与发布走 /workflows 域接口；表单字段由当前表单草稿投影注入，
 * 组织对象（成员/角色/部门）供审批人配置选择。
 */
const { detail } = useFormWorkspaceContext();

const draftDocument = shallowRef<WorkflowDocument>(createWorkflowDocument());
const definition = shallowRef<WorkflowDetailDto | null>(null);
const dirty = shallowRef(false);
const saving = shallowRef(false);
const publishing = shallowRef(false);
const loading = shallowRef(true);
const loadFailed = shallowRef(false);
/** 后端发布校验回传的 issues（画布高亮 + 错误面板双消费） */
const publishIssues = shallowRef<WorkflowIssue[]>([]);
const actorOptions = shallowRef<WorkflowActorOptions>({ members: [], roles: [], departments: [] });
/** 当前画布正在查看的版本。设计版本可编辑，其余启用/历史版本均只读。 */
const currentVersionNo = shallowRef(1);
const versionMenuVisible = shallowRef(false);
const versionsDialogVisible = shallowRef(false);

const isWorkflowForm = computed(() => detail.value?.formType === 'workflow');
const publishedVersion = computed(() => definition.value?.publishedVersion ?? 0);
const editingVersionNo = computed(() => definition.value?.draftVersionNo ?? 1);
const isEditingDraft = computed(
  () =>
    definition.value === null ||
    (definition.value.hasDraft === true && currentVersionNo.value === editingVersionNo.value),
);
const designerReadonly = computed(
  () => isWorkflowForm.value === false || (definition.value !== null && !isEditingDraft.value),
);
/** 版本/编辑模式切换即开启新的画布会话，杜绝跨版本复用选区与临时坐标。 */
const designerSessionKey = computed(
  () =>
    `${definition.value?.code ?? 'new'}:${currentVersionNo.value}:${designerReadonly.value ? 'readonly' : 'editing'}`,
);
// immediate watcher 会在 setup 内同步执行，递增令牌必须先完成初始化。
let definitionLoadVersion = 0;

/** 从当前表单草稿投影流程字段契约：可发布控件、成员类字段打 userField 标 */
const workflowFields = computed<WorkflowField[]>(() => {
  const items = detail.value?.draft?.content?.items ?? [];
  return items
    .filter((item) => !['separator', 'button'].includes(item.widget.type))
    .map((item) => ({
      widgetName: item.widget.widgetName,
      label: item.label || item.widget.widgetName,
      required: item.widget.allowBlank === false,
      userField: item.widget.type === 'user' || item.widget.type === 'usergroup',
    }));
});

// 外壳异步加载表单详情，子路由可能先挂载。监听绑定编码而不是只在 mounted
// 时读取，确保详情抵达后能够加载已存在的流程定义，避免误发重复创建请求。
watch(
  () => detail.value?.code,
  (formCode) => {
    if (!formCode) return;
    void loadDefinition(formCode);
  },
  { immediate: true },
);

onMounted(() => {
  void loadActorOptions();
  window.addEventListener('beforeunload', confirmBrowserLeave);
});

onUnmounted(() => {
  window.removeEventListener('beforeunload', confirmBrowserLeave);
});

/** 按绑定表单定位定义（一条表单至多一条）；未绑定时保持空文档，首次保存懒建 */
async function loadDefinition(formCode = detail.value?.code): Promise<WorkflowDetailDto | null> {
  if (!formCode) return null;
  const requestVersion = ++definitionLoadVersion;
  loading.value = true;
  loadFailed.value = false;
  try {
    const page = await listWorkflows({ formCode, limit: 1 });
    if (requestVersion !== definitionLoadVersion) return null;
    if (page.items.length === 0) return null;
    const loaded = await getWorkflow(page.items[0].code);
    if (requestVersion !== definitionLoadVersion) return null;
    applyDefinition(loaded);
    return loaded;
  } catch {
    if (requestVersion === definitionLoadVersion) loadFailed.value = true;
    return null;
  } finally {
    if (requestVersion === definitionLoadVersion) loading.value = false;
  }
}

function applyDefinition(loaded: WorkflowDetailDto) {
  definition.value = loaded;
  draftDocument.value = normalizeWorkflowDocument(loaded.draft) ?? createWorkflowDocument();
  currentVersionNo.value = loaded.hasDraft ? loaded.draftVersionNo : loaded.publishedVersion || 1;
  dirty.value = false;
  publishIssues.value = [];
}

/** 审批人可选对象：成员/角色/部门一次性装载（角色名即 roleCode 语义） */
async function loadActorOptions() {
  try {
    const [memberPage, roleTree, departmentTree] = await Promise.all([
      listMembers({ page: 1, pageSize: 200 }),
      getOrganizationRoleTree(),
      getDepartmentTree(),
    ]);
    actorOptions.value = {
      members: memberPage.items.map((member) => ({ id: member.id, label: member.name })),
      roles: roleTree.groups.flatMap((group) =>
        group.roles.map((role) => ({ code: role.name, label: role.name })),
      ),
      departments: mapDepartments(departmentTree),
    };
  } catch {
    // 组织对象加载失败不阻塞流程编辑：审批人选择降级为手填，保存/发布不受影响
  }
}

function mapDepartments(
  nodes: Array<{ id: number; name: string; children?: unknown[] }>,
): WorkflowDepartmentOption[] {
  return nodes.map((node) => ({
    id: node.id,
    label: node.name,
    children: Array.isArray(node.children)
      ? mapDepartments(node.children as Array<{ id: number; name: string; children?: unknown[] }>)
      : undefined,
  }));
}

function updateDocument(next: WorkflowDocument) {
  draftDocument.value = next;
  dirty.value = true;
  publishIssues.value = [];
}

/** 保存草稿：未绑定时先懒建定义（名称取「表单名 + 审批流程」）再保存；返回是否成功供发布链路判断 */
async function saveDraft(): Promise<boolean> {
  if (saving.value) return false;
  saving.value = true;
  try {
    let target = definition.value;
    if (!target) {
      // 创建定义会返回最小草稿，先保留用户已经在本地画布完成的编辑。
      const pendingDocument = draftDocument.value;
      try {
        const created = await createWorkflow({
          name: `${detail.value?.name ?? '未命名'}审批流程`,
          formCode: detail.value?.code,
        });
        applyDefinition(created);
        draftDocument.value = pendingDocument;
        target = created;
      } catch (error) {
        // 多窗口首次保存可能并发创建同一表单的绑定定义；读取胜出的定义后
        // 继续当前保存，用户无需手动刷新或再次点击。
        if (!(error instanceof ApiError) || error.errCode !== 'WORKFLOW_FORM_ALREADY_BOUND') {
          throw error;
        }
        target = await loadDefinition();
        if (!target) throw error;
        draftDocument.value = pendingDocument;
        dirty.value = true;
      }
    }
    if (!target.hasDraft) {
      ElMessage.warning('当前版本已启用，请先添加新版本再编辑');
      return false;
    }
    const result = await saveWorkflowDraft(target.code, {
      draftRevision: target.draftRevision,
      draft: draftDocument.value,
    });
    definition.value = {
      ...target,
      draftRevision: result.draftRevision,
      draft: draftDocument.value,
    };
    dirty.value = false;
    ElMessage.success('流程草稿已保存');
    return true;
  } catch (error) {
    notifySaveError(error);
    return false;
  } finally {
    saving.value = false;
  }
}

/** 乐观锁冲突提示刷新；其余按 errCode 文案透出，不做 message 匹配 */
function notifySaveError(error: unknown) {
  if (error instanceof ApiError && error.errCode === 'WORKFLOW_REVISION_CONFLICT') {
    ElMessage.warning('流程已被他人更新，正在重新加载最新草稿');
    void loadDefinition();
    return;
  }
  if (error instanceof ApiError && error.errCode === 'WORKFLOW_DEFINITION_INVALID') {
    applyDefinitionIssues(error);
    ElMessage.error('流程定义未通过协议校验，请按错误标注修正后重试');
    return;
  }
  ElMessage.error('保存失败，请稍后重试');
}

/** 发布：先保存当前编辑（失败即中止），再按口令发布；失败把 issues 交给画布定位 */
async function publish() {
  if (publishing.value) return;
  publishing.value = true;
  try {
    const saved = await saveDraft();
    if (!saved) return;
    const target = definition.value;
    if (!target) return;
    const result = await publishWorkflow(target.code, {
      draftRevision: target.draftRevision,
    });
    definition.value = {
      ...target,
      publishedVersion: result.versionNo,
      draftVersionNo: result.versionNo,
      hasDraft: false,
    };
    currentVersionNo.value = result.versionNo;
    publishIssues.value = [];
    ElMessage.success(`流程已发布，版本 V${result.versionNo}`);
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'WORKFLOW_DEFINITION_INVALID') {
      applyDefinitionIssues(error);
      ElMessage.error('发布未通过校验，请按画布错误标注修正后重试');
    } else {
      ElMessage.error('发布失败，请稍后重试');
    }
  } finally {
    publishing.value = false;
  }
}

/** 从 BizError data 负载提取 issues（path/code/message 与前端即时校验同形） */
function applyDefinitionIssues(error: ApiError) {
  const payload = error.data as { issues?: WorkflowIssue[] } | undefined;
  publishIssues.value = Array.isArray(payload?.issues) ? payload.issues : [];
}

/* ---- 版本历史与只读预览 ---- */

const versions = shallowRef<WorkflowVersionDto[]>([]);
const versionsLoading = shallowRef(false);
const previewVisible = shallowRef(false);
const previewDocument = shallowRef<WorkflowDocument>(createWorkflowDocument());
const previewTitle = shallowRef('版本预览');

async function loadVersions() {
  if (!definition.value) return;
  versionsLoading.value = true;
  try {
    versions.value = await listWorkflowVersions(definition.value.code);
  } catch {
    ElMessage.error('版本历史加载失败');
  } finally {
    versionsLoading.value = false;
  }
}

async function openVersions() {
  versionsDialogVisible.value = true;
  versionMenuVisible.value = false;
  await loadVersions();
}

/** 从当前启用版本（或指定历史版本）创建下一设计版本。 */
async function addDraftVersion(baseVersionNo = publishedVersion.value) {
  const target = definition.value;
  if (!target) return;
  if (target.hasDraft) {
    await switchVersion(target.draftVersionNo);
    return;
  }
  try {
    const created = await createWorkflowDraftVersion(target.code, { baseVersionNo });
    applyDefinition(created);
    versionMenuVisible.value = false;
    versionsDialogVisible.value = false;
    await loadVersions();
    ElMessage.success(`已创建流程版本 V${created.draftVersionNo}`);
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'WORKFLOW_DRAFT_ALREADY_EXISTS') {
      await loadDefinition();
      ElMessage.warning('已存在设计中的版本，已为你切换到最新内容');
      return;
    }
    ElMessage.error('添加新版本失败，请稍后重试');
  }
}

/** 切换版本前统一处理未保存修改；启用/历史快照始终以只读方式加载。 */
async function switchVersion(versionNo: number) {
  if (versionNo === currentVersionNo.value) {
    versionMenuVisible.value = false;
    return;
  }
  if (!(await confirmUnsavedChanges())) return;
  const target = definition.value;
  if (!target) return;
  if (target.hasDraft && versionNo === target.draftVersionNo) {
    draftDocument.value = normalizeWorkflowDocument(target.draft) ?? createWorkflowDocument();
  } else {
    try {
      const snapshot = await getWorkflowVersion(target.code, versionNo);
      draftDocument.value = normalizeWorkflowDocument(snapshot.dsl) ?? createWorkflowDocument();
    } catch {
      ElMessage.error('版本快照加载失败');
      return;
    }
  }
  currentVersionNo.value = versionNo;
  publishIssues.value = [];
  versionMenuVisible.value = false;
}

/** 版本只读预览：加载不可变快照，画布隐藏素材/属性面板 */
async function openVersionPreview(row: unknown) {
  if (!definition.value) return;
  const version = row as WorkflowVersionDto;
  try {
    const detailVersion = await getWorkflowVersion(definition.value.code, version.versionNo);
    const normalized = normalizeWorkflowDocument(detailVersion.dsl);
    if (normalized) {
      previewDocument.value = normalized;
      previewTitle.value = `V${version.versionNo} 版本预览（发布于 ${version.publishedAt}）`;
      previewVisible.value = true;
    }
  } catch {
    ElMessage.error('版本快照加载失败');
  }
}

function manageVersion(row: unknown) {
  const version = row as WorkflowVersionDto;
  if (version.status === 'active' && !definition.value?.hasDraft) {
    void addDraftVersion(version.versionNo);
    return;
  }
  void openVersionPreview(version);
}

function openCurrentPreview() {
  previewDocument.value = draftDocument.value;
  previewTitle.value = `流程版本 V${currentVersionNo.value} 预览`;
  previewVisible.value = true;
}

function testWorkflow() {
  ElMessage.info('流程测试将在预览与运行态联调阶段开放');
}

/* ---- 未保存修改保护 ---- */

const unsavedDialogVisible = shallowRef(false);
let unsavedResolver: ((allow: boolean) => void) | undefined;

function confirmBrowserLeave(event: BeforeUnloadEvent) {
  if (!dirty.value) return;
  event.preventDefault();
  event.returnValue = '';
}

function confirmUnsavedChanges(): Promise<boolean> {
  if (!dirty.value) return Promise.resolve(true);
  unsavedDialogVisible.value = true;
  return new Promise((resolve) => {
    unsavedResolver = resolve;
  });
}

function stayOnPage() {
  unsavedDialogVisible.value = false;
  unsavedResolver?.(false);
  unsavedResolver = undefined;
}

function discardAndContinue() {
  dirty.value = false;
  unsavedDialogVisible.value = false;
  unsavedResolver?.(true);
  unsavedResolver = undefined;
}

async function saveAndContinue() {
  if (!(await saveDraft())) return;
  unsavedDialogVisible.value = false;
  unsavedResolver?.(true);
  unsavedResolver = undefined;
}

onBeforeRouteLeave(() => confirmUnsavedChanges());

/* ---- 全屏 ---- */

const rootRef = useTemplateRef<HTMLElement>('rootRef');

async function toggleFullscreen() {
  const element = rootRef.value;
  if (!element) return;
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen();
    } else {
      await element.requestFullscreen();
    }
  } catch {
    ElMessage.warning('当前环境不支持全屏显示');
  }
}
</script>

<template>
  <section ref="rootRef" class="form-workflow-page" aria-label="流程设计工作台">
    <div class="form-workflow-page__toolbar" aria-label="流程设计操作">
      <ElAlert
        v-if="isWorkflowForm === false"
        class="form-workflow-page__type-alert"
        type="info"
        :closable="false"
        show-icon
        title="当前表单为普通表单：审批流程需要在表单设置中切换为流程型表单后配置"
      />
      <div class="form-workflow-page__toolbar-actions">
        <ElPopover
          v-model:visible="versionMenuVisible"
          placement="bottom-start"
          :width="410"
          trigger="click"
          popper-class="workflow-version-popper"
          @before-enter="loadVersions"
        >
          <template #reference>
            <button
              type="button"
              class="form-workflow-page__version"
              :class="{ 'form-workflow-page__version--draft': isEditingDraft }"
            >
              <i />流程版本（V{{ currentVersionNo }}）
              <RiArrowDownSLine aria-hidden="true" />
            </button>
          </template>
          <div class="form-workflow-page__version-menu">
            <button
              v-if="!definition || definition.hasDraft"
              type="button"
              class="form-workflow-page__version-row"
              :class="{ 'is-current': currentVersionNo === (definition?.draftVersionNo ?? 1) }"
              @click="switchVersion(definition?.draftVersionNo ?? 1)"
            >
              <span class="form-workflow-page__check">✓</span>
              <span>流程版本（V{{ definition?.draftVersionNo ?? 1 }}）</span>
              <ElTag type="warning" effect="plain">
                设计中
              </ElTag>
            </button>
            <button
              v-for="version in versions"
              :key="version.versionNo"
              type="button"
              class="form-workflow-page__version-row"
              :class="{ 'is-current': currentVersionNo === version.versionNo }"
              @click="switchVersion(version.versionNo)"
            >
              <span class="form-workflow-page__check">✓</span>
              <span>流程版本（V{{ version.versionNo }}）</span>
              <ElTag v-if="version.status === 'active'" type="success" effect="plain">
                启用中
              </ElTag>
            </button>
            <div class="form-workflow-page__version-actions">
              <button type="button" :disabled="publishedVersion === 0" @click="addDraftVersion()">
                ＋ 添加新版本
              </button>
              <button type="button" @click="openVersions">
                <RiHistoryFill aria-hidden="true" /> 管理已有版本
              </button>
            </div>
          </div>
        </ElPopover>
        <ElButton plain @click="openCurrentPreview">
          预览
        </ElButton>
        <ElButton plain @click="testWorkflow">
          测试
        </ElButton>
        <ElButton
          v-if="isEditingDraft"
          type="primary"
          plain
          :icon="RiSave3Fill"
          :loading="saving"
          :disabled="designerReadonly"
          @click="saveDraft"
        >
          {{ dirty ? '保存' : '已保存' }}
        </ElButton>
        <ElButton
          v-if="isEditingDraft"
          type="primary"
          :icon="RiUpload2Fill"
          :loading="publishing"
          :disabled="designerReadonly"
          @click="publish"
        >
          启用流程
        </ElButton>
        <ElButton
          class="form-workflow-page__icon-button"
          :icon="RiFullscreenFill"
          @click="toggleFullscreen"
        />
      </div>
    </div>

    <div
      v-if="publishIssues.length > 0"
      class="form-workflow-page__issues"
      aria-label="发布校验问题"
    >
      <span class="form-workflow-page__issues-title">未通过发布校验（{{ publishIssues.length }} 项）：</span>
      <span
        v-for="item in publishIssues"
        :key="`${item.path}-${item.code}`"
        class="form-workflow-page__issue"
      >
        {{ item.message }}
      </span>
    </div>

    <WorkflowDesigner
      v-if="loading === false && loadFailed === false"
      :key="designerSessionKey"
      class="form-workflow-page__workspace"
      :document="draftDocument"
      :fields="workflowFields"
      :actor-options="actorOptions"
      :issues="publishIssues"
      :readonly="designerReadonly"
      @update-document="updateDocument"
    />
    <ElAlert
      v-else-if="loadFailed"
      class="form-workflow-page__load-alert"
      type="error"
      :closable="false"
      show-icon
      title="流程定义加载失败"
      description="请检查网络后刷新页面重试"
    />

    <ElDialog
      v-model="versionsDialogVisible"
      title="管理已有版本"
      width="72%"
      append-to-body
      class="form-workflow-page__versions-dialog"
    >
      <ElTable
        v-loading="versionsLoading"
        :data="versions"
        empty-text="暂无发布版本"
      >
        <ElTableColumn prop="versionNo" label="流程版本" min-width="200">
          <template #default="{ row }">
            流程版本（V{{ row.versionNo }}）
          </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="120">
          <template #default="{ row }">
            <ElTag v-if="row.status === 'active'" type="success" effect="plain">
              启用中
            </ElTag>
            <span v-else>历史版本</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="publishedAt" label="启用时间" min-width="180" />
        <ElTableColumn label="操作" width="110" align="right">
          <template #default="{ row }">
            <ElButton
              link
              type="primary"
              @click="manageVersion(row)"
            >
              {{ row.status === 'active' && !definition?.hasDraft ? '编辑' : '查看' }}
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElDialog>

    <ElDialog
      v-model="unsavedDialogVisible"
      width="640px"
      append-to-body
      :show-close="false"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="form-workflow-page__unsaved-dialog"
    >
      <div class="form-workflow-page__unsaved-content">
        <span class="form-workflow-page__unsaved-icon">?</span>
        <div>
          <h3>流程设定有修改，是否保存？</h3>
          <p>你修改了流程设定但没有保存，是否需要保存流程设定并继续？</p>
        </div>
      </div>
      <template #footer>
        <div class="form-workflow-page__unsaved-footer">
          <ElButton link type="primary" @click="stayOnPage">
            留在此页
          </ElButton>
          <div>
            <ElButton @click="discardAndContinue">
              不保存
            </ElButton>
            <ElButton type="primary" :loading="saving" @click="saveAndContinue">
              保存并继续
            </ElButton>
          </div>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="previewVisible"
      :title="previewTitle"
      fullscreen
      append-to-body
      destroy-on-close
      class="form-workflow-page__preview-dialog"
    >
      <WorkflowDesigner :document="previewDocument" :fields="workflowFields" readonly />
    </ElDialog>
  </section>
</template>

<style scoped lang="scss">
.form-workflow-page {
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

  &:fullscreen {
    margin: 0;
    border-radius: 0;
  }

  &__toolbar {
    display: flex;
    min-height: 64px;
    padding: 0 var(--el-space-xl);
    align-items: center;
    justify-content: space-between;
    gap: var(--el-space-md);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  &__type-alert {
    flex: 1;
    padding: var(--el-space-xs) var(--el-space-sm);
  }

  &__toolbar-actions {
    display: flex;
    align-items: center;
    gap: var(--el-space-sm);
    margin-left: auto;
  }

  &__icon-button {
    width: 32px;
    padding: 0;
  }

  &__version {
    display: flex;
    margin-right: var(--el-space-sm);
    padding: 8px 10px;
    align-items: center;
    appearance: none;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    color: var(--el-text-color-primary);
    cursor: pointer;
    font-size: var(--el-font-size-base);
    font-weight: 600;
    gap: var(--el-space-sm);
    white-space: nowrap;

    i {
      display: block;
      width: 8px;
      height: 8px;
      background: var(--el-color-success);
      border-radius: var(--el-border-radius-half);
    }

    &--draft {
      i {
        background: var(--el-color-warning);
      }
    }
  }

  &__version-menu {
    margin: -12px;
  }

  &__version-row,
  &__version-actions button {
    display: flex;
    width: 100%;
    min-height: 52px;
    padding: 0 18px;
    align-items: center;
    gap: 10px;
    appearance: none;
    background: var(--el-bg-color);
    border: 0;
    color: var(--el-text-color-primary);
    cursor: pointer;
    font: inherit;
    text-align: left;

    &:hover {
      background: var(--el-fill-color-light);
    }

    .el-tag {
      margin-left: auto;
    }
  }

  &__version-row:not(.is-current) &__check {
    visibility: hidden;
  }

  &__check {
    color: var(--el-color-primary);
    font-weight: 700;
  }

  &__version-actions {
    padding: 10px 0;
    border-top: 1px solid var(--el-border-color-lighter);

    button {
      min-height: 46px;

      &:disabled {
        color: var(--el-text-color-disabled);
        cursor: not-allowed;
      }

      svg {
        width: 18px;
      }
    }
  }

  &__unsaved-content {
    display: flex;
    padding: 6px 10px 20px;
    gap: 18px;

    h3 {
      margin: 0 0 12px;
      color: var(--el-text-color-primary);
      font-size: 20px;
    }

    p {
      margin: 0;
      color: var(--el-text-color-regular);
      font-size: 16px;
    }
  }

  &__unsaved-icon {
    display: grid;
    width: 36px;
    height: 36px;
    flex: 0 0 36px;
    place-items: center;
    background: var(--el-color-primary);
    border-radius: 50%;
    color: white;
    font-size: 22px;
  }

  &__unsaved-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__issues {
    display: flex;
    padding: var(--el-space-xs) var(--el-space-xl);
    flex-wrap: wrap;
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-color-danger);
    font-size: 13px;
    background: var(--el-color-danger-light-9);
    border-bottom: 1px solid var(--el-color-danger-light-8);
  }

  &__issues-title {
    font-weight: 600;
  }

  &__issue {
    padding: 0 var(--el-space-xs);
    background: var(--el-bg-color);
    border-radius: var(--el-border-radius-base);
  }

  &__workspace {
    min-height: 0;
    flex: 1;
  }

  &__load-alert {
    margin: var(--el-space-lg);
  }
}

@media (max-width: 760px) {
  .form-workflow-page {
    margin: 0 var(--el-space-xs) var(--el-space-xs);

    &__toolbar {
      padding: 0 var(--el-space-md);
    }

    &__version {
      display: none;
    }
  }
}
</style>
