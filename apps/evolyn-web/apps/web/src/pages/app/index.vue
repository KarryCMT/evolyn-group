<script setup lang="ts">
import {
  RiArrowLeftLine,
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCheckboxCircleFill,
  RiContactsBook3Fill,
  RiPieChart2Fill,
} from '@remixicon/vue';
import { ApiError } from '@evolyn.do/utils';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, markRaw, shallowRef, watch, type Component } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { createForm } from '~/api/form';
import ApplicationEmptyState from '~/components/application/runtime/ApplicationEmptyState.vue';
import type { ApplicationAssetStarter } from '~/components/application/runtime/applicationAssetCatalog';
import ApplicationWorkspaceShell from '~/components/application/workspace/ApplicationWorkspaceShell.vue';
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceAssetAction,
  ApplicationWorkspaceCreateAssetType,
  ApplicationWorkspaceMode,
} from '~/components/application/workspace/applicationWorkspace.types';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useApplicationHome } from '~/composables/useApplicationHome';
import { useApplicationMenu } from '~/composables/useApplicationMenu';
import type { ApplicationIcon } from '~/types';

defineOptions({ name: 'ApplicationHomePage' });

const route = useRoute();
const router = useRouter();
const appCode = computed(() => String(route.params.appCode ?? ''));
const { application, applicationName, errorMessage, reload, status } = useApplicationHome(appCode);

// 应用详情异步加载完成后，以业务名称覆盖路由的通用「应用」标题。
watch(
  [applicationName, () => route.name],
  ([title, routeName]) => {
    if (routeName === 'App') document.title = title;
  },
  { immediate: true },
);

// 工作区侧栏资产树：来自应用菜单接口（M2-菜单-2），替换硬编码预览数据
const {
  assets: menuAssets,
  status: menuStatus,
  errorMessage: menuErrorMessage,
  reload: reloadMenu,
} = useApplicationMenu(appCode);

const iconByKey: Record<ApplicationIcon, Component> = {
  bookmark: markRaw(RiBookmark3Fill),
  briefcase: markRaw(RiBriefcase4Fill),
  contacts: markRaw(RiContactsBook3Fill),
  chart: markRaw(RiPieChart2Fill),
  check: markRaw(RiCheckboxCircleFill),
};

const applicationIcon = computed(() => iconByKey[application.value?.icon ?? 'bookmark']);
const workspacePreviewEnabled = computed(() => route.query.workspace === 'form');
// 首页形态以应用持久化状态为准；workspace 查询参数仅保留给当前表单设计器
// 的回跳预览。当前成员无菜单权限时仍保留运行时壳，不会退回构建引导。
const showApplicationWorkspace = computed(
  () => application.value?.homeMode === 'application' || workspacePreviewEnabled.value,
);
const activeAssetCode = shallowRef('');
const workspaceMode = shallowRef<ApplicationWorkspaceMode>('fill');

/** 递归按编码定位资产节点；菜单为空或未选中时为 null（内容区渲染空态） */
function findAsset(
  assets: ApplicationWorkspaceAsset[],
  code: string,
): ApplicationWorkspaceAsset | null {
  for (const asset of assets) {
    if (asset.code === code) return asset;
    if (asset.children?.length) {
      const matched = findAsset(asset.children, code);
      if (matched) return matched;
    }
  }
  return null;
}

const activeWorkspaceAsset = computed(() => findAsset(menuAssets.value, activeAssetCode.value));

function returnToDashboard() {
  void router.push({ name: 'dashboard' });
}

/**
 * 新建表单（入口即创建）：命名 → 以当前应用调 POST /forms 创建资产与空草稿
 * （后端同事务在应用菜单挂 form 节点，parentEntryCode 存在时挂到指定分组下）
 * → 携真实 formId 跳转设计器继续编辑。流程表单通过 query 标记类型，
 * 供设计器展示流程设计入口。
 */
async function startNewForm(workflow = false, parentEntryCode?: string) {
  const app = application.value;
  if (!app) {
    ElMessage.error('应用信息尚未就绪，请稍后重试');
    return;
  }

  let name: string;
  try {
    const { value } = await ElMessageBox.prompt('请输入表单名称', '新建表单', {
      inputValue: '未命名表单',
      inputValidator: (input: string) =>
        input && input.trim() ? true : '请输入 1–128 个字符的表单名称',
      confirmButtonText: '创建并设计',
      cancelButtonText: '取消',
      closeOnClickModal: false,
    });
    name = value.trim();
  } catch {
    // 取消命名即放弃创建，停留在应用工作台。
    return;
  }

  try {
    const detail = await createForm({ applicationId: app.id, name, parentEntryCode });
    void router.push({
      name: 'form-design',
      params: { appCode: appCode.value, formId: String(detail.id) },
      query: workflow ? { type: 'workflow' } : undefined,
    });
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'QUOTA_EXCEEDED') {
      ElMessage.error('表单数量已达套餐上限，请升级套餐或删除闲置表单');
    } else if (error instanceof ApiError && error.errCode === 'APP_MENU_PARENT_INVALID') {
      // 目标分组已被删除或不可用：菜单可能已过期，提示刷新重选
      ElMessage.error('目标分组不存在或已删除，请刷新页面后重试');
      reloadMenu();
    } else {
      ElMessage.error('创建表单失败，请稍后重试');
    }
  }
}

function openAssetStarter(starter: ApplicationAssetStarter) {
  if (starter.type === 'form') {
    startNewForm();
    return;
  }

  if (starter.type === 'workflow-form') {
    startNewForm(true);
    return;
  }

  const labels: Record<ApplicationAssetStarter['type'], string> = {
    'workflow-form': '流程表单',
    form: '表单',
    dashboard: '仪表盘',
  };
  ElMessage.info(`${labels[starter.type]}能力将在后续版本接入`);
}

function showAssetGuide() {
  ElMessage.info('表单和仪表盘能力将在后续版本接入');
}

/** 应用后台已具备基础壳，入口保留当前应用编码以维持同一应用上下文。 */
function openApplicationManagement() {
  void router.push({ name: 'app-setting-permissions', params: { appCode: appCode.value } });
}

function selectWorkspaceAsset(asset: ApplicationWorkspaceAsset) {
  activeAssetCode.value = asset.code;
  workspaceMode.value = 'fill';
}

/** 顶栏“编辑”只对当前表单生效，并携带菜单目标资产的公开 formId 进入设计器。 */
function updateWorkspaceMode(mode: ApplicationWorkspaceMode) {
  if (mode !== 'design') {
    workspaceMode.value = mode;
    return;
  }

  const asset = activeWorkspaceAsset.value;
  if (!asset || asset.type !== 'form' || !asset.targetId) {
    ElMessage.info('请先从左侧选择要编辑的表单');
    return;
  }

  void router.push({
    name: 'form-design',
    params: { appCode: appCode.value, formId: asset.targetId },
  });
}

/**
 * 顶层与分组创建菜单共用同一处理链：表单先复用现有设计器，父级编码随路由携带；
 * 仪表盘与分组尚无菜单写接口时只提示，不在前端伪造菜单数据。
 */
function createWorkspaceAsset(payload: {
  parent?: ApplicationWorkspaceAsset;
  type: ApplicationWorkspaceCreateAssetType;
}) {
  if (payload.type === 'form') {
    startNewForm(false, payload.parent?.code);
    return;
  }
  if (payload.type === 'workflow-form') {
    startNewForm(true, payload.parent?.code);
    return;
  }

  const assetLabel = payload.type === 'dashboard' ? '仪表盘' : payload.parent ? '子分组' : '分组';
  const location = payload.parent ? `在「${payload.parent.label}」中` : '';
  ElMessage.info(`${location}新建${assetLabel}的能力将在后续版本接入`);
}

/** 菜单写接口尚未落地，浮窗入口先明确反馈能力建设状态，避免静默失效。 */
function handleWorkspaceAssetAction(payload: {
  asset: ApplicationWorkspaceAsset;
  action: ApplicationWorkspaceAssetAction;
}) {
  const actionLabels: Record<ApplicationWorkspaceAssetAction, string> = {
    edit: '编辑',
    move: '移动',
    favorite: '收藏',
    delete: '删除',
  };
  ElMessage.info(`${actionLabels[payload.action]}「${payload.asset.label}」功能将在后续版本接入`);
}

function reloadWorkspace() {
  reload();
  reloadMenu();
}
</script>

<template>
  <div class="application-home-page">
    <!-- 菜单型应用直接进入运行时壳；表单设计器的 workspace=form 回跳仍兼容。 -->
    <template v-if="showApplicationWorkspace && status === 'ready'">
      <!-- 应用元信息已就绪但菜单加载失败：错误态统一在页面层拦截并重试，
           侧栏/内容组件不感知后端错误码（应用菜单方案 §11）。 -->
      <el-result
        v-if="menuStatus === 'error'"
        class="application-home-page__result"
        icon="error"
        title="加载应用菜单失败"
        :sub-title="menuErrorMessage"
      >
        <template #extra>
          <el-button type="primary" @click="reloadMenu()">重新加载</el-button>
        </template>
      </el-result>

      <ApplicationWorkspaceShell
        v-else
        :application-name="applicationName"
        :application-icon="application?.icon ?? 'bookmark'"
        :assets="menuAssets"
        :active-asset="activeWorkspaceAsset"
        :mode="workspaceMode"
        :menu-status="menuStatus"
        @back="returnToDashboard"
        @create-asset="createWorkspaceAsset"
        @asset-guide="showAssetGuide"
        @select-asset="selectWorkspaceAsset"
        @asset-action="handleWorkspaceAssetAction"
        @open-management="openApplicationManagement"
        @update-mode="updateWorkspaceMode"
      />
    </template>

    <template v-else>
      <TopNavigation :title="applicationName" :show-default-navigation="false" surface="surface">
        <template #leading>
          <button
            class="application-home-page__back"
            type="button"
            aria-label="返回工作台"
            @click="returnToDashboard"
          >
            <RiArrowLeftLine />
          </button>
        </template>
        <template #title>
          <span class="application-home-page__title">
            <span class="application-home-page__icon" aria-hidden="true">
              <component :is="applicationIcon" />
            </span>
            <strong>{{ applicationName }}</strong>
          </span>
        </template>
      </TopNavigation>

      <section v-if="status === 'loading'" v-loading="true" class="application-home-page__status" />

      <el-result
        v-else-if="status === 'not-found'"
        class="application-home-page__result"
        icon="warning"
        title="应用不存在或已不可访问"
        sub-title="请返回工作台后重新选择应用。"
      >
        <template #extra>
          <el-button type="primary" @click="returnToDashboard">返回工作台</el-button>
        </template>
      </el-result>

      <el-result
        v-else-if="status === 'error'"
        class="application-home-page__result"
        icon="error"
        title="加载应用失败"
        :sub-title="errorMessage"
      >
        <template #extra>
          <el-button type="primary" @click="reloadWorkspace()">重新加载</el-button>
        </template>
      </el-result>

      <!-- 空应用维持独立引导；表单运行时由后续 @evolyn.do/form 包承接。 -->
      <ApplicationEmptyState
        v-else
        @select-asset="openAssetStarter"
        @learn-more="showAssetGuide"
        @open-management="openApplicationManagement"
      />
    </template>
  </div>
</template>

<style scoped lang="scss">
.application-home-page {
  display: flex;
  height: 100vh;
  overflow: hidden;
  flex-direction: column;
  background: var(--el-fill-color-lighter);

  &__back {
    display: inline-flex;
    width: 32px;
    height: 32px;
    padding: 0;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-regular);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: var(--el-border-radius-base);
    font-size: var(--el-font-size-medium);

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: 2px;
    }
  }

  &__title {
    display: inline-flex;
    min-width: 0;
    align-items: center;
    gap: var(--el-space-md);
    color: var(--el-text-color-primary);

    strong {
      overflow: hidden;
      font-size: var(--el-font-size-medium);
      font-weight: 650;
      line-height: 26px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  &__icon {
    display: inline-flex;
    width: 30px;
    height: 30px;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    color: var(--el-color-white);
    background: var(--el-color-primary);
    border-radius: var(--el-border-radius-medium);
    font-size: var(--el-font-size-large);
  }

  &__status,
  &__result {
    flex: 1;
  }

  &__status {
    min-height: 0;
  }

  &__result {
    display: grid;
    place-content: center;
    background: var(--el-fill-color-lighter);
  }
}
</style>
