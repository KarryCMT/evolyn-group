<template>
  <PluginDesignPanel
    v-model="pluginDesign"
    v-model:authentication="authentication"
    v-model:global-fields="globalFields"
    :plugin-id="pluginForm.id ?? null"
    :plugin-key="pluginForm.pluginKey"
    :plugin-version="pluginForm.version ?? null"
    @update="handleUpdateDesign"
    @update-function-info="handleUpdateFunctionInfo"
    @sort-functions="handleSortFunctions"
    @save="handleSaveDesign"
    @test="handleSaveTestDesign"
  />
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue';
import { ElNotification } from 'element-plus';
import {
  getPluginCenterDetail,
  updatePluginCenter,
  type OwnerPluginUpdateRequest,
  type PluginCenterForm,
  type PluginFunctionDetail,
  type PluginFunctionUpdateType,
} from '../api';
import type {
  PluginDesignAuthentication,
  PluginDesignConfig,
  PluginDesignField,
  PluginDesignFunction,
  PluginDesignFunctionUpdateComplete,
  PluginDesignSaveComplete,
  PluginDesignSaveValue,
} from '../types';
import { useFormDesignFactory } from '../hooks/useFormDesignFactory';
import { useFormFunctionUpdateQueue } from '../hooks/useFormFunctionUpdateQueue';
import {
  createFieldsFromTriggerTemplate,
  createResponseFieldsFromTrigger,
} from '../hooks/useFormDesignTemplates';
import { normalizeFormRuntime } from '../hooks/useFormRuntime';
import { createPluginFunctionRequestParameter } from '../utils/functionParameter';
import {
  createGlobalFieldsFromTemplate,
  createGlobalTemplateByFields,
} from '../utils/globalTemplate';
import PluginDesignPanel from './PluginDesignPanel.vue';

type DrawerTab = 'attrs' | 'design';
type OwnerPluginModule = 'auth' | 'common';
type OwnerPluginBaseUpdatePayload = Pick<
  OwnerPluginUpdateRequest,
  | 'pluginKey'
  | 'pluginName'
  | 'pluginIcon'
  | 'pluginIconColor'
  | 'pluginThemeColor'
  | 'pluginOverview'
  | 'pluginDetail'
  | 'pluginDetailFile'
>;
interface PluginNameUpdateTask {
  session: number;
  payload: OwnerPluginBaseUpdatePayload;
}
interface OpenDrawerOptions {
  /** 打开抽屉前是否已经产生需要同步到列表的持久化改动。 */
  hasPersistedChanges?: boolean;
}
type PluginAttributePayload = Pick<
  PluginCenterForm,
  | 'id'
  | 'pluginKey'
  | 'pluginIcon'
  | 'pluginThemeColor'
  | 'pluginIconColor'
  | 'pluginName'
  | 'pluginOverview'
  | 'pluginDetail'
  | 'pluginDetailFile'
  | 'helpDoc'
>;

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
});
const emits = defineEmits(['update:modelValue', 'refresh']);

const drawerVisible = ref(false);
// 插件抽屉首次渲染及每次重新打开时，默认进入插件设计页。
const activeTab = ref<DrawerTab>('design');
const detailLoading = ref(false);
// 等待 openDrawer 提供当前插件后再挂载内容，避免设计器在详情加载前提前请求元数据。
const drawerContentVisible = ref(false);
// 仅记录非函数数据的成功持久化，关闭抽屉时据此决定是否刷新插件列表。
const hasPersistedNonFunctionChanges = ref(false);
const { createDefaultAuthentication, createDefaultGlobalFields } = useFormDesignFactory();
const pluginDesign = ref<PluginDesignConfig | null>(null);
const authentication = ref<PluginDesignAuthentication>(createDefaultAuthentication());
const globalFields = ref<PluginDesignField[]>(createDefaultGlobalFields());
// 记录当前抽屉会话已持久化的插件名称，避免名称未变化时重复提交。
const lastSavedPluginName = ref('');
// 标题更新串行执行并只保留等待中的最新值，避免旧请求后完成覆盖新名称。
let pluginNameUpdateSession = 0;
// 分模块记录最近一次保存请求，防止旧响应覆盖新提交返回的字段 id。
const ownerPluginUpdateFlags: Record<OwnerPluginModule, number> = {
  auth: 0,
  common: 0,
};
// 每个函数独立记录最近一次保存请求，避免较早响应覆盖较新的回显数据。
const pluginFunctionUpdateFlags = new Map<string, number>();
// 抽屉重新打开后使用新的会话标识，阻止上一次会话的响应写入当前插件。
let pluginFunctionUpdateSession = 0;

const defaultForm = (): PluginCenterForm => ({
  id: '',
  pluginKey: '',
  version: null,
  pluginIcon: 'PersonalizationApp',
  pluginThemeColor: '#EAF3FF,#EAF3FF',
  pluginIconColor: '#1677FF',
  pluginName: '',
  pluginOverview: '',
  pluginDetail: '',
  pluginDetailFile: [],
  helpDoc: '',
  pluginDesign: null,
  authentication: createDefaultAuthentication(),
  globalTemplate: {
    fields: [],
  },
});
const pluginForm = reactive<PluginCenterForm>(defaultForm());
// 顶部标签使用语义图标，便于快速区分插件属性与插件设计。
// 抽屉标题支持直接编辑，并与插件名称表单字段保持一致。
/** 合并公共基础属性表单变更，并保留当前设计模块数据。 */

const normalizePluginDesign = (value: unknown): PluginDesignConfig | null => {
  if (!value) return null;
  if (typeof value === 'string') {
    try {
      return JSON.parse(value) as PluginDesignConfig;
    } catch (error) {
      return null;
    }
  }
  return value as PluginDesignConfig;
};

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return !!value && typeof value === 'object' && !Array.isArray(value);
};

const normalizeAuthentication = (value: unknown): PluginDesignAuthentication => {
  if (
    !isRecord(value) ||
    !isRecord(value.conf_template) ||
    !Array.isArray(value.conf_template.fields)
  ) {
    return createDefaultAuthentication();
  }
  return value as unknown as PluginDesignAuthentication;
};

/**
 * 从更新接口的 Axios 响应与业务响应外壳中提取完整插件数据。
 * @param response /plugin/center/update 接口响应。
 */
const getUpdatedPluginDetail = (response: unknown): Partial<PluginCenterForm> | null => {
  if (!isRecord(response)) return null;
  const responseBody = isRecord(response.data) ? response.data : response;
  const pluginDetail = isRecord(responseBody.data) ? responseBody.data : responseBody;
  return isRecord(pluginDetail) ? (pluginDetail as Partial<PluginCenterForm>) : null;
};

/**
 * 从函数更新接口的 Axios 响应与业务响应外壳中提取完整函数数据。
 * @param response /plugin/function/update 接口响应。
 */
const getUpdatedPluginFunction = (response: unknown): PluginFunctionDetail | null => {
  if (!isRecord(response)) return null;
  const responseBody = isRecord(response.data) ? response.data : response;
  const functionDetail = isRecord(responseBody.data) ? responseBody.data : responseBody;
  if (!isRecord(functionDetail) || functionDetail.id == null) return null;
  return functionDetail as unknown as PluginFunctionDetail;
};

/**
 * 判断接口返回的函数类型是否为设计器支持的类型。
 * @param value 接口返回的函数类型。
 */
const isPluginDesignFunctionType = (
  value: unknown,
): value is NonNullable<PluginDesignFunction['functionType']> => {
  return value === 'ai' || value === 'frontend' || value === 'backend';
};

/**
 * 将函数更新接口回显转换为设计器函数，并保留接口未返回的本地展示状态。
 * @param sourceFunction 本次请求提交的函数快照。
 * @param functionDetail 更新接口返回的完整函数数据。
 */
const createUpdatedPluginFunction = (
  sourceFunction: PluginDesignFunction,
  functionDetail: PluginFunctionDetail,
): PluginDesignFunction => {
  const requestFields = functionDetail.requestParameter?.fields;
  const responseFields = functionDetail.responseParameter?.fields;
  const nextFunctionId = String(functionDetail.id);
  return {
    ...sourceFunction,
    id: nextFunctionId,
    functionKey:
      typeof functionDetail.functionKey === 'string'
        ? functionDetail.functionKey
        : sourceFunction.functionKey,
    name:
      typeof functionDetail.functionName === 'string'
        ? functionDetail.functionName
        : sourceFunction.name,
    functionDescription:
      typeof functionDetail.functionDescription === 'string'
        ? functionDetail.functionDescription
        : sourceFunction.functionDescription,
    functionType: isPluginDesignFunctionType(functionDetail.functionType)
      ? functionDetail.functionType
      : sourceFunction.functionType,
    seq: typeof functionDetail.seq === 'number' ? functionDetail.seq : sourceFunction.seq,
    runtime:
      typeof functionDetail.runtime === 'string'
        ? normalizeFormRuntime(functionDetail.runtime)
        : sourceFunction.runtime,
    fields: Array.isArray(requestFields)
      ? createFieldsFromTriggerTemplate(requestFields)
      : sourceFunction.fields,
    responseParams: Array.isArray(responseFields)
      ? createResponseFieldsFromTrigger(responseFields)
      : sourceFunction.responseParams,
    code:
      typeof functionDetail.sourceCode === 'string'
        ? functionDetail.sourceCode
        : sourceFunction.code,
  };
};

/**
 * 使用更新接口回显覆盖本次保存的函数，使后端生成的字段 ID 进入当前设计数据。
 * @param functionId 本次发起保存的函数 ID。
 * @param updatedFunction 接口回显转换后的完整函数。
 */
const applyUpdatedPluginFunction = (functionId: string, updatedFunction: PluginDesignFunction) => {
  const currentDesign = pluginDesign.value;
  if (!currentDesign) return;
  const functionIndex = currentDesign.functions.findIndex((item) => item.id === functionId);
  if (functionIndex === -1) return;

  const nextFunctionId = updatedFunction.id;

  currentDesign.functions.splice(functionIndex, 1, updatedFunction);
  if (currentDesign.activeFunctionId === functionId) {
    currentDesign.activeFunctionId = nextFunctionId;
  }
};

type PluginFunctionSyncStatus = 'invalid' | 'stale' | 'success';

interface PluginFunctionSyncResult {
  /** 本次函数更新请求的有效状态。 */
  status: PluginFunctionSyncStatus;
  /** 接口确认并转换后的完整函数。 */
  updatedFunction?: PluginDesignFunction;
}

/**
 * 使用更新接口返回的当前模块覆盖设计器状态，使后端生成的字段 id 进入后续提交。
 * @param value 本次设计器提交的数据及当前菜单。
 * @param pluginDetail 更新接口返回的完整插件数据。
 */
const applyUpdatedPluginDetail = (
  value: PluginDesignSaveValue,
  pluginDetail: Partial<PluginCenterForm>,
) => {
  // 同步插件详情缓存，但只刷新当前保存模块，避免覆盖其他模块尚未提交的本地编辑。
  Object.assign(pluginForm, pluginDetail);
  if (value.activeMenu === 'common' && pluginDetail.globalTemplate) {
    globalFields.value = createGlobalFieldsFromTemplate(pluginDetail.globalTemplate);
    return;
  }
  if (value.activeMenu === 'auth' && pluginDetail.authentication) {
    authentication.value = normalizeAuthentication(pluginDetail.authentication);
    return;
  }
  if (pluginDetail.pluginDesign !== undefined) {
    pluginDesign.value = normalizePluginDesign(pluginDetail.pluginDesign);
  }
};

// 插件级保存只处理身份验证和通用参数，函数模块改走函数更新接口。
const getOwnerPluginModule = (
  activeMenu: PluginDesignSaveValue['activeMenu'],
): OwnerPluginModule => {
  if (activeMenu === 'auth') return 'auth';
  return 'common';
};

/**
 * 生成更新接口必需的完整基础属性，避免未传字段被后端置空。
 * @param pluginName 本次需要保存的插件名称，默认使用表单当前值。
 */
const createOwnerPluginBaseUpdatePayload = (
  pluginName = pluginForm.pluginName,
): OwnerPluginBaseUpdatePayload | null => {
  if (!pluginForm.pluginKey) return null;
  return {
    pluginKey: pluginForm.pluginKey,
    pluginName,
    pluginIcon: pluginForm.pluginIcon,
    pluginIconColor: pluginForm.pluginIconColor || '',
    pluginThemeColor: pluginForm.pluginThemeColor || '',
    pluginOverview: pluginForm.pluginOverview,
    pluginDetail: pluginForm.pluginDetail,
    pluginDetailFile: [...pluginForm.pluginDetailFile],
  };
};

// 身份验证和通用参数使用插件更新接口，并携带后端更新所需的完整基础属性。
const createOwnerPluginUpdatePayload = (value: PluginDesignSaveValue): PluginCenterForm | null => {
  const basePayload = createOwnerPluginBaseUpdatePayload();
  if (!basePayload) return null;
  if (value.activeMenu === 'auth') {
    return {
      ...basePayload,
      id: pluginForm.id,
      helpDoc: pluginForm.helpDoc,
      authentication: value.authentication,
    };
  }
  if (value.activeMenu === 'common') {
    return {
      ...basePayload,
      id: pluginForm.id,
      helpDoc: pluginForm.helpDoc,
      globalTemplate: createGlobalTemplateByFields(value.globalFields),
    };
  }
  return null;
};

const syncOwnerPluginDesign = async (value: PluginDesignSaveValue) => {
  const payload = createOwnerPluginUpdatePayload(value);
  if (!payload) return false;
  const module = getOwnerPluginModule(value.activeMenu);
  const currentRequestFlag = ++ownerPluginUpdateFlags[module];
  // 身份验证和通用参数按当前菜单更新，同时保留完整基础属性。
  const response = await updatePluginCenter(payload);
  hasPersistedNonFunctionChanges.value = true;
  if (currentRequestFlag !== ownerPluginUpdateFlags[module]) return true;
  const updatedPluginDetail = getUpdatedPluginDetail(response);
  if (updatedPluginDetail) applyUpdatedPluginDetail(value, updatedPluginDetail);
  return true;
};

/**
 * 将指定函数转换为更新接口参数，函数内容保存与拖拽排序共用同一字段白名单。
 * @param currentFunction 需要更新的完整函数数据。
 */
const createPluginFunctionUpdatePayloadByFunction = (
  currentFunction: PluginDesignFunction,
): PluginFunctionUpdateType | null => {
  if (!pluginForm.id) return null;
  return {
    id: currentFunction.id,
    pluginId: pluginForm.id,
    functionName: currentFunction.name,
    functionDescription: currentFunction.functionDescription,
    functionType: currentFunction.functionType || 'backend',
    seq: currentFunction.seq,
    requestParameter: createPluginFunctionRequestParameter(currentFunction.fields),
    responseParameter: {
      fields: currentFunction.responseParams,
    },
    runtime: currentFunction.runtime,
    sourceCode: currentFunction.code,
  };
};

/**
 * 排序请求执行时合并当前最新函数内容，只使用拖拽快照中的目标 seq。
 * @param sortedFunction 本次拖拽后需要更新排序号的函数快照。
 */
const createPluginFunctionSortUpdatePayload = (
  sortedFunction: PluginDesignFunction,
): PluginFunctionUpdateType | null => {
  const latestFunction = pluginDesign.value?.functions.find(
    (item) => item.id === sortedFunction.id,
  );
  return latestFunction
    ? createPluginFunctionUpdatePayloadByFunction({
        ...latestFunction,
        seq: sortedFunction.seq,
      })
    : null;
};

const {
  enqueueFunctionSort: handleSortFunctions,
  enqueueFunctionUpdate,
  resetFunctionSortQueue,
} = useFormFunctionUpdateQueue({
  createUpdatePayload: createPluginFunctionSortUpdatePayload,
  getSession: () => pluginFunctionUpdateSession,
});

/**
 * 将当前选中函数转换为函数更新接口参数。
 * @param value 设计器当前完整数据及选中菜单。
 */
const createPluginFunctionUpdatePayload = (
  value: PluginDesignSaveValue,
): PluginFunctionUpdateType | null => {
  // activeFunctionId 始终指向当前侧栏选中的函数，保存时只提交这一项。
  const currentFunction = value.pluginDesign.functions.find(
    (item) => item.id === value.pluginDesign.activeFunctionId,
  );
  return currentFunction ? createPluginFunctionUpdatePayloadByFunction(currentFunction) : null;
};

/**
 * 提交当前函数并返回接口确认的数据，统一处理同一函数的并发请求和抽屉会话切换。
 * @param value 设计器当前完整数据及选中菜单。
 */
const requestPluginFunctionUpdate = async (
  value: PluginDesignSaveValue,
): Promise<PluginFunctionSyncResult> => {
  const payload = createPluginFunctionUpdatePayload(value);
  if (!payload) return { status: 'invalid' };
  const functionId = String(payload.id);
  const sourceFunction = value.pluginDesign.functions.find((item) => item.id === functionId);
  if (!sourceFunction) return { status: 'invalid' };
  const currentRequestFlag = (pluginFunctionUpdateFlags.get(functionId) || 0) + 1;
  pluginFunctionUpdateFlags.set(functionId, currentRequestFlag);
  const currentSession = pluginFunctionUpdateSession;
  const response = await enqueueFunctionUpdate(payload);
  if (
    currentSession !== pluginFunctionUpdateSession ||
    currentRequestFlag !== pluginFunctionUpdateFlags.get(functionId)
  ) {
    return { status: 'stale' };
  }
  const functionDetail = getUpdatedPluginFunction(response);
  return {
    status: 'success',
    updatedFunction: functionDetail
      ? createUpdatedPluginFunction(sourceFunction, functionDetail)
      : sourceFunction,
  };
};

/**
 * 保存当前选中函数；请求参数不包含仅供前端切换菜单使用的 activeMenu。
 * @param value 设计器当前完整数据及选中菜单。
 */
const syncPluginFunction = async (value: PluginDesignSaveValue) => {
  const result = await requestPluginFunctionUpdate(value);
  if (result.status === 'success' && result.updatedFunction) {
    applyUpdatedPluginFunction(value.pluginDesign.activeFunctionId, result.updatedFunction);
  }
  // 导航期间被后续请求替代仍可继续切换；无有效函数数据时才阻止后续操作。
  return result.status !== 'invalid';
};

watch(
  () => props.modelValue,
  (value) => {
    drawerVisible.value = value;
  },
  { immediate: true },
);

watch(drawerVisible, (value) => {
  emits('update:modelValue', value);
});

/**
 * 打开插件抽屉，编辑时先拉取详情用于回显完整信息。
 * @param row 当前插件列表数据。
 * @param options 本次打开前已经发生的持久化状态。
 */
const openDrawer = async (row?: PluginCenterForm, options: OpenDrawerOptions = {}) => {
  resetPluginForm();
  // 新建插件在打开抽屉前已经完成新增请求，需要在关闭时同步一次列表。
  hasPersistedNonFunctionChanges.value = Boolean(options.hasPersistedChanges);
  activeTab.value = 'design';
  // 仅编辑已有插件时等待详情接口，并在抽屉出现前开启骨架屏，避免真实设计器短暂闪现。
  detailLoading.value = row?.pluginKey != null;
  drawerContentVisible.value = true;
  drawerVisible.value = true;
  if (!row || row.pluginKey == null) return;
  try {
    const res = await getPluginCenterDetail({
      pluginKey: row.pluginKey,
    });
    Object.assign(pluginForm, defaultForm(), res.data?.data || row);
    lastSavedPluginName.value = pluginForm.pluginName;
    pluginDesign.value = normalizePluginDesign(pluginForm.pluginDesign);
    authentication.value = normalizeAuthentication(pluginForm.authentication);
    globalFields.value = createGlobalFieldsFromTemplate(pluginForm.globalTemplate);
  } finally {
    detailLoading.value = false;
  }
};

const resetPluginForm = () => {
  pluginNameUpdateSession += 1;
  pluginFunctionUpdateSession += 1;
  pluginFunctionUpdateFlags.clear();
  resetFunctionSortQueue();
  lastSavedPluginName.value = '';
  hasPersistedNonFunctionChanges.value = false;
  Object.assign(pluginForm, defaultForm());
  pluginDesign.value = null;
  authentication.value = createDefaultAuthentication();
  globalFields.value = createDefaultGlobalFields();
};

// 设计器保存时先同步本地状态，再按当前菜单调用插件或函数更新接口。
const handleSaveDesign = async (
  value: PluginDesignSaveValue,
  onComplete: PluginDesignSaveComplete,
) => {
  try {
    // 代码、请求参数和返回参数都保存当前选中的同一个函数。
    const isFunctionMenu = ['code', 'request', 'response'].includes(value.activeMenu);
    const hasSyncedDesign = isFunctionMenu
      ? await syncPluginFunction(value)
      : await syncOwnerPluginDesign(value);
    if (!hasSyncedDesign) {
      onComplete(false);
      ElNotification({
        title: '成功',
        message: '插件设计已缓存，请完成插件属性后保存',
        type: 'success',
      });
      return;
    }
    onComplete(true);
    ElNotification({
      title: '成功',
      message: '插件设计保存成功',
      type: 'success',
    });
  } catch (error) {
    // 接口失败时通知设计器保持未保存状态，并交由现有请求错误处理继续提示。
    onComplete(false);
    throw error;
  }
};

/**
 * 函数名称或描述确认后立即提交更新，并将接口确认的完整函数返回设计器。
 * @param value 包含目标函数信息的完整设计快照。
 * @param onComplete 更新结果回调。
 */
const handleUpdateFunctionInfo = async (
  value: PluginDesignSaveValue,
  onComplete: PluginDesignFunctionUpdateComplete,
) => {
  try {
    const result = await requestPluginFunctionUpdate(value);
    onComplete(
      result.status === 'success',
      result.status === 'success' ? result.updatedFunction : undefined,
    );
  } catch (error) {
    onComplete(false);
    // 请求层负责统一展示接口错误，此处仅保留设计器中的旧函数信息。
    console.error(error);
  }
};

/**
 * 侧栏离开已修改模块时静默更新；成功后由设计器继续切换，失败则保留当前模块。
 * @param value 离开模块的数据快照。
 * @param onComplete 更新结果回调。
 */
const handleUpdateDesign = async (
  value: PluginDesignSaveValue,
  onComplete: PluginDesignSaveComplete,
) => {
  try {
    const isFunctionMenu = ['code', 'request', 'response'].includes(value.activeMenu);
    const updated = isFunctionMenu
      ? await syncPluginFunction(value)
      : await syncOwnerPluginDesign(value);
    onComplete(updated);
  } catch (error) {
    onComplete(false);
    // 请求层负责统一展示接口错误，此处只阻止切换并保留未保存数据。
    console.error(error);
  }
};

const handleSaveTestDesign = async (
  value: PluginDesignSaveValue,
  onComplete: PluginDesignSaveComplete,
) => {
  await handleSaveDesign(value, onComplete);
};

defineExpose({
  openDrawer,
});
</script>

<style lang="scss">
.plugin-center-drawer {
  position: relative;
  overflow: hidden;

  .el-drawer__body,
  .el-drawer__header {
    padding: 0 !important;
  }

  .el-drawer__header {
    margin-bottom: 0;
  }

  .el-drawer__body {
    height: calc(100vh - 48px);
    overflow: hidden;
  }

  .plugin-drawer {
    height: 100%;
    min-height: 0;
    overflow: hidden;

    &__body {
      box-sizing: border-box;
      height: 100%;
      min-height: 0;
      overflow: auto;
      background-color: var(--gp-fill-color-sm);
    }

    &__panel {
      box-sizing: border-box;
      width: min(1180px, 100%);
      min-height: 100%;
      padding-top: var(--gp-space-xl);
      margin: 0 auto;

      &--design {
        width: 100%;
        height: 100%;
        min-height: 0;
        padding-top: 0;
      }
    }
  }
}
</style>
