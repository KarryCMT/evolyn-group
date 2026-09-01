<script setup lang="ts">
import { ApiError } from '@evolyn.do/utils';
import {
  WorkflowDesigner,
  createWorkflowDocument,
  normalizeWorkflowDocument,
  type WorkflowActorOptions,
  type WorkflowDepartmentOption,
  type WorkflowDocument,
  type WorkflowField,
  type WorkflowIssue,
} from '@evolyn.do/workflow';
import { RiFullscreenFill, RiHistoryFill, RiSave3Fill, RiUpload2Fill } from '@remixicon/vue';
import { ElAlert, ElButton, ElDrawer, ElMessage, ElTable, ElTableColumn } from 'element-plus';
import { computed, onMounted, shallowRef, useTemplateRef } from 'vue';
import {
  createWorkflow,
  getWorkflow,
  getWorkflowVersion,
  listWorkflowVersions,
  listWorkflows,
  publishWorkflow,
  saveWorkflowDraft,
} from '~/api/workflow';
import { getDepartmentTree } from '~/api/department';
import { listMembers } from '~/api/member';
import { getOrganizationRoleTree } from '~/api/role';
import type { WorkflowDetailDto, WorkflowVersionDto } from '~/types';
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

const isWorkflowForm = computed(() => detail.value?.formType === 'workflow');
const publishedVersion = computed(() => definition.value?.publishedVersion ?? 0);

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

onMounted(async () => {
  await Promise.all([loadDefinition(), loadActorOptions()]);
  loading.value = false;
});

/** 按绑定表单定位定义（一条表单至多一条）；未绑定时保持空文档，首次保存懒建 */
async function loadDefinition() {
  const formCode = detail.value?.code;
  if (!formCode) return;
  try {
    const page = await listWorkflows({ formCode, limit: 1 });
    if (page.items.length === 0) return;
    const loaded = await getWorkflow(page.items[0].code);
    applyDefinition(loaded);
  } catch {
    loadFailed.value = true;
  }
}

function applyDefinition(loaded: WorkflowDetailDto) {
  definition.value = loaded;
  draftDocument.value = normalizeWorkflowDocument(loaded.draft) ?? createWorkflowDocument();
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
      const created = await createWorkflow({
        name: `${detail.value?.name ?? '未命名'}审批流程`,
        formCode: detail.value?.code,
      });
      applyDefinition(created);
      target = created;
    }
    const result = await saveWorkflowDraft(target.code, {
      draftRevision: target.draftRevision,
      draft: draftDocument.value,
    });
    definition.value = { ...target, draftRevision: result.draftRevision };
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
    definition.value = { ...target, publishedVersion: result.versionNo };
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

const versionsDrawerVisible = shallowRef(false);
const versions = shallowRef<WorkflowVersionDto[]>([]);
const versionsLoading = shallowRef(false);
const previewVisible = shallowRef(false);
const previewDocument = shallowRef<WorkflowDocument>(createWorkflowDocument());
const previewTitle = shallowRef('版本预览');

async function openVersions() {
  versionsDrawerVisible.value = true;
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

/** 版本只读预览：加载不可变快照，画布隐藏素材/属性面板 */
async function openVersionPreview(version: WorkflowVersionDto) {
  if (!definition.value) return;
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
        <span
          class="form-workflow-page__version"
          :class="{ 'form-workflow-page__version--draft': publishedVersion === 0 }"
        >
          <i />{{ publishedVersion > 0 ? `流程版本（V${publishedVersion}）` : '未发布' }}
        </span>
        <ElButton v-if="publishedVersion > 0" :icon="RiHistoryFill" @click="openVersions">
          版本历史
        </ElButton>
        <ElButton
          type="primary"
          plain
          :icon="RiSave3Fill"
          :loading="saving"
          :disabled="isWorkflowForm === false"
          @click="saveDraft"
        >
          {{ dirty ? '保存' : '已保存' }}
        </ElButton>
        <ElButton
          type="primary"
          :icon="RiUpload2Fill"
          :loading="publishing"
          :disabled="isWorkflowForm === false"
          @click="publish"
        >
          发布
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
      <span class="form-workflow-page__issues-title"
        >未通过发布校验（{{ publishIssues.length }} 项）：</span
      >
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
      class="form-workflow-page__workspace"
      :document="draftDocument"
      :fields="workflowFields"
      :actor-options="actorOptions"
      :issues="publishIssues"
      :readonly="isWorkflowForm === false"
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

    <ElDrawer
      v-model="versionsDrawerVisible"
      title="版本历史"
      direction="rtl"
      size="380px"
      :append-to-body="true"
    >
      <ElTable
        v-loading="versionsLoading"
        :data="versions"
        size="small"
        empty-text="暂无发布版本"
        @row-click="(row: WorkflowVersionDto) => openVersionPreview(row)"
      >
        <ElTableColumn prop="versionNo" label="版本" width="70">
          <template #default="{ row }">V{{ row.versionNo }}</template>
        </ElTableColumn>
        <ElTableColumn prop="publishedAt" label="发布时间" min-width="150" />
        <ElTableColumn label="操作" width="70">
          <template #default>
            <ElButton link type="primary" size="small">预览</ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElDrawer>

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
    min-height: 50px;
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
    align-items: center;
    color: var(--el-text-color-primary);
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
      color: var(--el-text-color-secondary);

      i {
        background: var(--el-text-color-placeholder);
      }
    }
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
