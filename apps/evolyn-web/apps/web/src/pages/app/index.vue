<script setup lang="ts">
import type { Component } from 'vue';
import type {
  ApplicationAssetStarter,
  ApplicationAssetType,
} from '~/components/application/runtime/applicationAssetCatalog';
import type {
  ApplicationWorkspaceAsset,
  ApplicationWorkspaceAssetAction,
  ApplicationWorkspaceCreateAssetType,
  ApplicationWorkspaceMode,
} from '~/components/application/workspace/applicationWorkspace.types';
import type { ApplicationIconKey, FormType } from '~/types';
import { ApiError } from '@evolyn.do/utils';
import {
  RiArrowLeftLine,
  RiBookmark3Fill,
  RiBriefcase4Fill,
  RiCheckboxCircleFill,
  RiContactsBook3Fill,
  RiPieChart2Fill,
} from '@remixicon/vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { computed, markRaw, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { createApplicationMenuGroup } from '~/api/applications';
import { createForm } from '~/api/form';
import ApplicationEmptyState from '~/components/application/runtime/ApplicationEmptyState.vue';
import ApplicationWorkspaceShell from '~/components/application/workspace/ApplicationWorkspaceShell.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useApplicationHome } from '~/composables/useApplicationHome';
import { useApplicationMenu } from '~/composables/useApplicationMenu';
import { DEFAULT_APPLICATION_ICON, getApplicationIconName } from '~/types';

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
  menuRevision,
  status: menuStatus,
  errorMessage: menuErrorMessage,
  reload: reloadMenu,
} = useApplicationMenu(appCode);

const iconByKey: Record<ApplicationIconKey, Component> = {
  bookmark: markRaw(RiBookmark3Fill),
  briefcase: markRaw(RiBriefcase4Fill),
  contacts: markRaw(RiContactsBook3Fill),
  chart: markRaw(RiPieChart2Fill),
  check: markRaw(RiCheckboxCircleFill),
};

const applicationIcon = computed(
  () =>
    iconByKey[getApplicationIconName(application.value?.icon) as ApplicationIconKey] ??
    iconByKey.bookmark,
);
const workspacePreviewEnabled = computed(() => route.query.workspace === 'form');
// 设计器返回时携带表单公开编码，应用菜单加载完成后据此恢复对应节点选中态。
const requestedFormCode = computed(() => String(route.params.formCode ?? ''));
const hasMenuAssets = computed(() => menuAssets.value.length > 0);
// 菜单资产是应用已进入运行态的直接事实：即使历史应用 homeMode 尚未回填，
// 只要菜单接口返回可见节点也必须展示工作区，不能继续落入默认创建引导。
const showApplicationWorkspace = computed(
  () =>
    application.value?.homeMode === 'application' ||
    workspacePreviewEnabled.value ||
    hasMenuAssets.value,
);
const activeAssetCode = shallowRef('');
const workspaceMode = shallowRef<ApplicationWorkspaceMode>('fill');
const DEFAULT_FORM_NAME = '未命名表单';
/** 创建请求期间锁住所有入口，避免网络延迟下重复创建同一资产。 */
const creatingAssetType = shallowRef<ApplicationAssetType | null>(null);
/** 分组创建单独加锁，避免 Prompt 关闭后的请求窗口内再次提交。 */
const creatingGroup = shallowRef(false);

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

/** 递归按表单公开编码定位菜单节点，分组与其他资产不会参与匹配。 */
function findFormAsset(
  assets: ApplicationWorkspaceAsset[],
  formCode: string,
): ApplicationWorkspaceAsset | null {
  for (const asset of assets) {
    if (asset.type === 'form' && asset.targetCode === formCode) return asset;
    if (asset.children?.length) {
      const matched = findFormAsset(asset.children, formCode);
      if (matched) return matched;
    }
  }
  return null;
}

const activeWorkspaceAsset = computed(() => findAsset(menuAssets.value, activeAssetCode.value));

/** 菜单首次加载或当前选中节点消失时，默认选中第一项可操作资产。 */
function firstSelectableAsset(
  assets: ApplicationWorkspaceAsset[],
): ApplicationWorkspaceAsset | null {
  for (const asset of assets) {
    if (asset.type !== 'folder') return asset;
    const child = firstSelectableAsset(asset.children ?? []);
    if (child) return child;
  }
  return null;
}

watch(
  [menuAssets, requestedFormCode],
  ([assets, formCode]) => {
    const requestedAsset = findFormAsset(assets, formCode);
    if (requestedAsset) {
      activeAssetCode.value = requestedAsset.code;
      return;
    }
    if (findAsset(assets, activeAssetCode.value)) return;
    activeAssetCode.value = firstSelectableAsset(assets)?.code ?? '';
  },
  { immediate: true },
);

function returnToDashboard() {
  void router.push({ name: 'dashboard' });
}

/**
 * 新建表单（入口即创建）：以默认名称调 POST /forms 创建资产与空草稿
 * （后端同事务在应用菜单挂 form 节点，parentEntryCode 存在时挂到指定分组下）
 * → 携稳定 formCode 跳转设计器继续编辑。表单类型随创建请求持久化，设计器
 * 后续只从详情接口读取；表单名称进入设计器后通过属性面板修改。
 */
async function startNewForm(
  assetType: Extract<ApplicationAssetType, 'form' | 'workflow-form'>,
  parentEntryCode?: string,
) {
  if (creatingAssetType.value) return;

  const app = application.value;
  if (!app) {
    ElMessage.error('应用信息尚未就绪，请稍后重试');
    return;
  }

  const formType: FormType = assetType === 'workflow-form' ? 'workflow' : 'standard';
  creatingAssetType.value = assetType;
  let detail: Awaited<ReturnType<typeof createForm>>;
  try {
    detail = await createForm({
      applicationId: app.id,
      name: DEFAULT_FORM_NAME,
      formType,
      parentEntryCode,
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
    creatingAssetType.value = null;
    return;
  }

  try {
    await router.push({
      name: 'form-design',
      params: { appCode: appCode.value, formCode: detail.code },
    });
  } catch {
    // 资产已经创建成功，导航异常时不能提示“创建失败”，避免用户重试产生重复表单。
    ElMessage.error('表单已创建，但打开设计器失败，请从应用菜单重新进入');
  } finally {
    creatingAssetType.value = null;
  }
}

function openAssetStarter(starter: ApplicationAssetStarter) {
  if (starter.type === 'form') {
    void startNewForm('form');
    return;
  }

  if (starter.type === 'workflow-form') {
    void startNewForm('workflow-form');
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

  const formCode = asset.type === 'form' ? asset.targetCode : null;
  if (requestedFormCode.value === (formCode ?? '')) return;
  void router.replace({
    name: 'App',
    params: { appCode: appCode.value, formCode: formCode ?? '' },
  });
}

/**
 * 顶栏模式入口只对当前表单生效：填写留在应用运行态，编辑与数据管理
 * 携带菜单目标资产的公开 formCode 进入各自独立工作区。
 */
function updateWorkspaceMode(mode: ApplicationWorkspaceMode) {
  if (mode === 'fill') {
    workspaceMode.value = 'fill';
    return;
  }

  const asset = activeWorkspaceAsset.value;
  if (!asset || asset.type !== 'form' || !asset.targetCode) {
    ElMessage.info(`请先从左侧选择要${mode === 'design' ? '编辑' : '管理数据'}的表单`);
    return;
  }

  void router.push({
    name: mode === 'design' ? 'form-design' : 'form-data',
    params: { appCode: appCode.value, formCode: asset.targetCode },
  });
}

/**
 * 顶层与分组创建菜单共用同一处理链：表单复用现有设计器，分组通过
 * menuRevision 乐观锁持久化；仪表盘尚无资产接口时明确提示。
 */
async function createMenuGroup(parent?: ApplicationWorkspaceAsset) {
  if (creatingGroup.value) return;
  if (menuStatus.value !== 'ready' || menuRevision.value < 1) {
    ElMessage.warning('应用菜单尚未加载完成，请稍后重试');
    return;
  }

  let name = '';
  try {
    const result = await ElMessageBox.prompt(
      '',
      parent ? `在「${parent.label}」中新建分组` : '新建分组',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputPlaceholder: '请输入分组名称',
        inputValidator: (value) => {
          const normalized = value.trim();
          if (!normalized) return '请输入分组名称';
          if (Array.from(normalized).length > 128) return '分组名称不能超过 128 个字符';
          return true;
        },
        closeOnClickModal: false,
        showClose: false,
      },
    );
    name = result.value.trim();
  } catch {
    // 取消或关闭 Prompt 属于正常交互，不显示错误提示。
    return;
  }

  creatingGroup.value = true;
  try {
    await createApplicationMenuGroup(appCode.value, {
      name,
      parentEntryId: parent?.code,
      baseMenuRevision: menuRevision.value,
    });
    await reloadMenu();
    ElMessage.success('分组创建成功');
  } catch (error) {
    if (error instanceof ApiError && error.errCode === 'APP_MENU_VERSION_CONFLICT') {
      ElMessage.warning('应用菜单已更新，已为你刷新，请重新创建分组');
      await reloadMenu();
    } else if (error instanceof ApiError && error.errCode === 'APP_MENU_DEPTH_EXCEEDED') {
      ElMessage.error('分组最多支持两级，无法继续新建子分组');
    } else if (error instanceof ApiError && error.errCode === 'APP_MENU_PARENT_INVALID') {
      ElMessage.error('目标分组不存在或已删除，已为你刷新菜单');
      await reloadMenu();
    } else if (error instanceof ApiError && error.errCode === 'APP_MENU_NAME_INVALID') {
      ElMessage.error('分组名称不符合要求，请修改后重试');
    } else if (
      error instanceof ApiError &&
      (error.errCode === 'APP_STATUS_INVALID' || error.errCode === 'APP_PROVISIONING')
    ) {
      ElMessage.error('当前应用状态不支持新建分组');
    } else {
      ElMessage.error('创建分组失败，请稍后重试');
    }
  } finally {
    creatingGroup.value = false;
  }
}

function createWorkspaceAsset(payload: {
  parent?: ApplicationWorkspaceAsset;
  type: ApplicationWorkspaceCreateAssetType;
}) {
  if (payload.type === 'form') {
    void startNewForm('form', payload.parent?.code);
    return;
  }
  if (payload.type === 'workflow-form') {
    void startNewForm('workflow-form', payload.parent?.code);
    return;
  }
  if (payload.type === 'folder') {
    void createMenuGroup(payload.parent);
    return;
  }

  const assetLabel = '仪表盘';
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
    rename: '修改名称和图标',
    'switch-type': '切换表单类型',
    'reference-view': '查看引用视图',
    'copy-in-app': '复制到当前应用',
    'copy-cross-app': '复制到其他应用',
    move: '移动',
    favorite: '收藏',
    hide: '对成员隐藏',
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
          <el-button type="primary" @click="reloadMenu()"> 重新加载 </el-button>
        </template>
      </el-result>

      <ApplicationWorkspaceShell
        v-else
        :app-code="appCode"
        :application-name="applicationName"
        :application-icon="application?.icon ?? DEFAULT_APPLICATION_ICON"
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
          <el-button type="primary" @click="returnToDashboard"> 返回工作台 </el-button>
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
          <el-button type="primary" @click="reloadWorkspace()"> 重新加载 </el-button>
        </template>
      </el-result>

      <!-- 空应用维持独立引导；表单运行时由后续 @evolyn.do/form 包承接。 -->
      <ApplicationEmptyState
        v-else
        :creating-asset-type="creatingAssetType"
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
