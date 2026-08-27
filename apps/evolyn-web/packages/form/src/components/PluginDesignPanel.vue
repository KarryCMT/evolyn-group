<template>
  <div class="plugin-design">
    <div class="plugin-design__workspace">
      <FormDesignPalette v-if="showPalette" :fields="paletteFields" @add-field="addField" />

      <main
        class="plugin-design__content"
        :class="{
          'plugin-design__content--bordered': ['auth', 'response'].includes(activeMenu),
        }"
        @click="handleContentClick"
      >
        <!-- 内容区滚动统一交给 Element Plus Scrollbar，避免出现浏览器原生滚动条样式。 -->
        <el-scrollbar class="plugin-design__content-scrollbar" :view-class="contentViewClass">
          <FormDesignCodeView
            v-if="isCodePage"
            ref="codeViewRef"
            :function-data="currentFunction"
            :global-fields="globalFieldsData"
            :active-menu="activeMenu"
            :diagnostics="codeDiagnostics"
            :diagnostic-focus-key="diagnosticFocusKey"
          />
          <FormFieldCanvas
            v-else
            :fields="activeFields"
            :selected-field-key="selectedFieldKey"
            :selected-subform-child-field-key="selectedSubformChildFieldKey"
            @select-field="selectField"
            @select-subform-child="selectSubformChildField"
            @copy-field="copyField"
            @copy-subform-child="copySubformChildField"
            @remove-field="removeField"
            @remove-subform-child="removeSubformChildField"
            @add-drag-field="addDragField"
            @add-subform-drag-field="addSubformDragField"
            @update-field-default-value="updateFieldDefaultValue"
          />
        </el-scrollbar>
      </main>

      <FormDesignPropertyPanel
        v-if="showPropertyPanel"
        :field="selectedField"
        :widget-name-label="selectedWidgetNameLabel"
        :selected-subform-child-field-key="selectedSubformChildFieldKey"
        @add-option="addOption"
        @remove-option="removeOption"
        @update-default-value="updateSelectedFieldDefaultValue"
        @update-field-key="updateSelectedFieldKey"
        @update-option="updateOption"
        @update-subform-child-selection="selectedSubformChildFieldKey = $event"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue';
import { ElMessage, ElScrollbar } from 'element-plus';
import type { PluginApplet, PluginTrigger } from '../api';
import type {
  PluginCodeDiagnostic,
  PluginDesignAuthentication,
  PluginDesignConfig,
  PluginDesignDragField,
  PluginDesignField,
  PluginDesignFunction,
  PluginDesignFunctionSortComplete,
  PluginDesignFunctionSortPayload,
  PluginDesignFunctionUpdateComplete,
  PluginDesignFunctionUpdatePayload,
  PluginDesignMenuKey,
  PluginDesignPaletteItem,
  PluginDesignSaveComplete,
  PluginDesignSaveValue,
  PluginDesignTemplateField,
  PluginDesignViewMode,
  PluginRuntime,
} from '../types';
import {
  cloneFormDesign,
  createDefaultCode,
  formDesignRuntimeOptions,
  useFormDesignFactory,
} from '../hooks/useFormDesignFactory';
import { useFormAuthFieldActions } from '../hooks/useFormAuthFieldActions';
import { useFormDesignMetadata } from '../hooks/useFormDesignMetadata';
import { useFormDesignPalette } from '../hooks/useFormDesignPalette';
import { useFormDesignTemplates } from '../hooks/useFormDesignTemplates';
import { useFormFieldActions } from '../hooks/useFormFieldActions';
import { useFormFunctionActions } from '../hooks/useFormFunctionActions';
import { validateFormCode, validateFormDesignCode } from '../hooks/useFormCodeValidation';
import { useFormResponseParams } from '../hooks/useFormResponseParams';
import { normalizeFormRuntime } from '../hooks/useFormRuntime';
import {
  normalizePluginDesignAuthentication,
  normalizePluginDesignFields,
} from '../utils/designField';
import FormFieldCanvas from './FormFieldCanvas.vue';
import FormDesignCodeView from './FormDesignCodeView.vue';
import FormDesignPalette from './FormDesignPalette.vue';
import FormDesignPropertyPanel from './FormDesignPropertyPanel.vue';
import { RiCodeSSlashFill, RiDownload2Fill, RiShareForwardFill } from '@remixicon/vue';

const props = defineProps<{
  modelValue: PluginDesignConfig | null;
  authentication: PluginDesignAuthentication;
  globalFields: PluginDesignField[];
  /** 当前设计器所属插件 ID，用于加载后端函数和前端扩展列表。 */
  pluginId: string | number | null;
  /** 当前插件唯一标识，用于调试接口定位插件定义。 */
  pluginKey: string;
  /** 当前插件数据版本，用于调试接口定位同一标识下的具体定义。 */
  pluginVersion: number | null;
}>();

const emits = defineEmits<{
  (event: 'update:modelValue', value: PluginDesignConfig): void;
  (event: 'update:authentication', value: PluginDesignAuthentication): void;
  (event: 'update:globalFields', value: PluginDesignField[]): void;
  (event: 'update', value: PluginDesignSaveValue, onComplete: PluginDesignSaveComplete): void;
  (
    event: 'update-function-info',
    value: PluginDesignSaveValue,
    onComplete: PluginDesignFunctionUpdateComplete,
  ): void;
  (
    event: 'sort-functions',
    functions: PluginDesignFunction[],
    onComplete: PluginDesignFunctionSortComplete,
  ): void;
  (event: 'save', value: PluginDesignSaveValue, onComplete: PluginDesignSaveComplete): void;
  (event: 'test', value: PluginDesignSaveValue, onComplete: PluginDesignSaveComplete): void;
}>();

const selectedFieldKey = ref('');
const selectedSubformChildFieldKey = ref('');
const selectedAuthFieldKey = ref('');
// 当前菜单属于设计器临时交互状态，不进入 pluginDesign 持久化数据。
const activeMenu = ref<PluginDesignMenuKey>('common');
const debugVisible = ref(false);
const sidebarVisible = ref(true);
const codeDiagnostics = ref<PluginCodeDiagnostic[]>([]);
const diagnosticFocusKey = ref(0);
// 参数代码视图通过暴露方法向主面板提供保存和导航前校验能力。
const codeViewRef = ref<{
  getSyntaxDiagnostics: () => PluginCodeDiagnostic[];
  refreshEditorContent: () => void;
  validateConfigDraft: () => { valid: boolean; message?: string };
}>();
const {
  createDefaultAuthentication,
  createDefaultDesign,
  createDefaultGlobalFields,
  createField,
  createFunction,
  createSubformChildField,
} = useFormDesignFactory();
const {
  authTemplateList,
  fieldMetadataList,
  loadAuthTemplateList,
  loadPluginFieldMetadata,
  loadPluginTriggerList,
} = useFormDesignMetadata();
const { commonPalette, requestPalette, widgetNameTextMap } =
  useFormDesignPalette(fieldMetadataList);

const rootMenus = computed(() => []);
const functionMenus = computed(() => [
  {
    label: '代码',
    key: 'code' as PluginDesignMenuKey,
    icon: RiCodeSSlashFill,
  },
  {
    label: '请求参数',
    key: 'request' as PluginDesignMenuKey,
    icon: RiDownload2Fill,
  },
  {
    label: '返回参数',
    key: 'response' as PluginDesignMenuKey,
    icon: RiShareForwardFill,
  },
]);

const designData = reactive<PluginDesignConfig>(createDefaultDesign());
const authenticationData = ref<PluginDesignAuthentication>(createDefaultAuthentication());
const globalFieldsData = ref<PluginDesignField[]>(createDefaultGlobalFields());
let skipNextModelSync = false;
let skipNextAuthenticationSync = false;
let skipNextGlobalFieldsSync = false;
let subformChildActionSeed = 0;
// 各模块分别记录最近一次成功保存的数据，用于切换前判断是否需要调用更新接口。
let savedAuthenticationSignature = '';
let savedGlobalFieldsSignature = '';
let savedFunctionScopeKey: string | undefined;
const savedFunctionSignatures = new Map<string, string>();
// 按插件、函数和运行时暂存源码，切换语言时恢复对应草稿，避免覆盖用户代码。
const runtimeCodeDrafts = new Map<string, string>();
// 标记最近一次拖拽排序，旧请求回调不得覆盖更新后的函数顺序。
let functionSortRequestFlag = 0;
// 每个函数独立记录最近一次信息修改，旧请求完成时不得覆盖较新的名称或描述。
const functionInfoUpdateFlags = new Map<string, number>();
let pendingNavigationTarget: PluginDesignNavigationTarget | null = null;
let navigationUpdating = false;

interface PluginDesignNavigationTarget {
  menuKey: PluginDesignMenuKey;
  functionId: string;
}

/**
 * 清理旧版函数中的菜单状态，并统一运行时枚举值及请求字段结构。
 * @param design 插件函数设计数据。
 */
const normalizeDesignData = (design: PluginDesignConfig) => {
  design.functions.forEach((item) => {
    delete (item as PluginDesignFunction & { activeMenu?: PluginDesignMenuKey }).activeMenu;
    // 兼容尚未保存函数描述的旧版 pluginDesign 数据。
    item.functionDescription =
      typeof item.functionDescription === 'string' ? item.functionDescription : '';
    const runtime = normalizeFormRuntime(item.runtime);
    if (formDesignRuntimeOptions.includes(runtime)) item.runtime = runtime;
    item.fields = normalizePluginDesignFields(item.fields, item.id);
  });
  const normalizeFunctionGroup = (isFrontend: boolean) =>
    design.functions
      .filter((item) => (item.functionType === 'frontend') === isFrontend)
      .map((functionData, index) => ({
        functionData,
        // 兼容旧版未保存 seq 的 pluginDesign，缺失时沿用原分组顺序。
        seq: Number.isInteger(functionData.seq) && functionData.seq >= 0 ? functionData.seq : index,
      }))
      .sort((prev, next) => prev.seq - next.seq)
      .map(({ functionData }, seq) => {
        functionData.seq = seq;
        return functionData;
      });
  // 前端扩展和后端函数分别按 seq 回显，组内统一规范为从 0 开始的连续值。
  design.functions.splice(
    0,
    design.functions.length,
    ...normalizeFunctionGroup(true),
    ...normalizeFunctionGroup(false),
  );
};

const currentFunction = computed<PluginDesignFunction>(() => {
  if (!designData.functions.length) {
    const fallbackFunction = createFunction('function_default', '未命名函数');
    designData.functions.push(fallbackFunction);
    designData.activeFunctionId = fallbackFunction.id;
  }
  const target = designData.functions.find((item) => item.id === designData.activeFunctionId);
  return target || designData.functions[0];
});
const currentAuthentication = computed(() => authenticationData.value);
const { createAuthenticationByMethod, createFunctionFromApplet, createFunctionFromTrigger } =
  useFormDesignTemplates({
    authTemplateList,
    createDefaultCode,
    createFunction,
    currentAuthentication,
  });
const isCodePage = computed(
  () => activeMenu.value === 'code' || currentFunction.value.viewMode === 'code',
);
const contentViewClass = computed(() =>
  isCodePage.value
    ? 'plugin-design__content-view plugin-design__content-view--full'
    : 'plugin-design__content-view',
);
const showViewModeSelect = computed(
  () => activeMenu.value !== 'auth' && activeMenu.value !== 'code',
);
const showPalette = computed(() => {
  return (
    currentFunction.value.viewMode === 'form' && ['common', 'request'].includes(activeMenu.value)
  );
});
const showPropertyPanel = computed(() => {
  return !isCodePage.value && !['auth', 'response'].includes(activeMenu.value);
});
const currentTitle = computed(() => {
  if (activeMenu.value === 'common') return '通用参数';
  return `${currentFunction.value.name}-${functionMenus.value.find((item) => item.key === activeMenu.value)?.label || ''}`;
});
const activeFields = computed(() => {
  return activeMenu.value === 'request' ? currentFunction.value.fields : globalFieldsData.value;
});
const paletteFields = computed(() => {
  return activeMenu.value === 'request' ? requestPalette.value : commonPalette.value;
});
const selectedField = computed(() =>
  activeFields.value.find((item) => item.fieldKey === selectedFieldKey.value),
);
const selectedWidgetNameLabel = computed(() => {
  return selectedField.value
    ? widgetNameTextMap.value[selectedField.value.widgetName] || selectedField.value.widgetName
    : '';
});
const authFields = computed(() => authenticationData.value.conf_template.fields);
const selectedAuthField = computed(() =>
  authFields.value.find((item) => item.fieldKey === selectedAuthFieldKey.value),
);

const {
  selectFunction: applySelectFunction,
  sortFunctions,
  switchMenu: applySwitchMenu,
  updateFunction,
} = useFormFunctionActions({
  activeMenu,
  createFunction,
  designData,
  getPluginId: () => props.pluginId,
  selectedFieldKey,
});

const {
  addDragField: addDragRootField,
  addField: addRootField,
  addOption,
  copyField: copyRootField,
  removeField: removeRootField,
  removeOption,
  selectField: selectRootField,
  updateFieldDefaultValue,
  updateOption,
  updateSelectedFieldDefaultValue,
} = useFormFieldActions({
  activeFields,
  createField,
  selectedField,
  selectedFieldKey,
  widgetNameTextMap,
});

const addField = (source: PluginDesignPaletteItem) => {
  selectedSubformChildFieldKey.value = '';
  addRootField(source);
};

const addDragField = (...args: Parameters<typeof addDragRootField>) => {
  selectedSubformChildFieldKey.value = '';
  addDragRootField(...args);
};

const copyField = (...args: Parameters<typeof copyRootField>) => {
  selectedSubformChildFieldKey.value = '';
  copyRootField(...args);
};

const removeField = (...args: Parameters<typeof removeRootField>) => {
  selectedSubformChildFieldKey.value = '';
  removeRootField(...args);
};

const { clearAuthFieldSelection } = useFormAuthFieldActions({
  authFields,
  selectedAuthField,
  selectedAuthFieldKey,
});

const handleContentClick = () => {
  // 身份验证组件只占内容区中部，事件提升到主内容区后，两侧及下方空白也能取消字段选中。
  if (activeMenu.value === 'auth') clearAuthFieldSelection();
};

const updateSelectedFieldKey = (fieldKey: string) => {
  if (!selectedField.value) return;
  // 字段 fieldKey 本身可编辑，更新时同步选中值，避免属性面板因旧 key 找不到字段而消失。
  selectedField.value.fieldKey = fieldKey;
  selectedFieldKey.value = fieldKey;
};

const selectField = (fieldKey: string) => {
  // 切换外层字段时清空子表单内部选中态，避免右侧误展示上一个子字段属性。
  selectedSubformChildFieldKey.value = '';
  selectRootField(fieldKey);
};

const selectSubformChildField = (value: { parentKey: string; childKey: string }) => {
  // 子表单内部字段被点选时，仍保持父子表单为当前外层字段，并让右侧切到该子字段属性。
  selectedFieldKey.value = value.parentKey;
  selectedSubformChildFieldKey.value = value.childKey;
};

const getSubformChildFields = (field?: PluginDesignField, ensureFields = false) => {
  if (ensureFields && field) {
    if (!field.fieldConf) field.fieldConf = {};
    if (!Array.isArray(field.fieldConf.fields)) field.fieldConf.fields = [];
  }
  const fields = field?.fieldConf?.fields;
  return Array.isArray(fields) ? (fields as PluginDesignTemplateField[]) : [];
};

const createSubformChildFieldKey = () => `_widget_${Date.now()}${subformChildActionSeed++}`;

const copySubformChildField = (value: { parentKey: string; childKey: string }) => {
  const parentField = activeFields.value.find((item) => item.fieldKey === value.parentKey);
  const childFields = getSubformChildFields(parentField);
  const index = childFields.findIndex((item) => item.fieldKey === value.childKey);
  if (index === -1) return;
  const fieldKey = createSubformChildFieldKey();
  const nextField = {
    ...cloneFormDesign(childFields[index]),
    // 复制字段按新增数据处理，持久化 id 等待后端返回。
    id: null,
    fieldKey,
    fieldLabel: `${childFields[index].fieldLabel} copy`,
  };
  childFields.splice(index + 1, 0, nextField);
  selectedFieldKey.value = value.parentKey;
  selectedSubformChildFieldKey.value = nextField.fieldKey;
};

const removeSubformChildField = (value: { parentKey: string; childKey: string }) => {
  const parentField = activeFields.value.find((item) => item.fieldKey === value.parentKey);
  const childFields = getSubformChildFields(parentField);
  const index = childFields.findIndex((item) => item.fieldKey === value.childKey);
  if (index === -1) return;
  childFields.splice(index, 1);
  selectedFieldKey.value = value.parentKey;
  selectedSubformChildFieldKey.value =
    childFields[index]?.fieldKey || childFields[index - 1]?.fieldKey || '';
};

const addSubformDragField = (value: PluginDesignDragField & { parentKey: string }) => {
  const parentField = activeFields.value.find((item) => item.fieldKey === value.parentKey);
  const childFields = getSubformChildFields(parentField, true);
  const field = createSubformChildField(value.widgetName, value.dataType);
  if (!field) return;
  const insertIndex = value.index >= 0 ? value.index : childFields.length;
  childFields.splice(insertIndex, 0, field);
  selectedFieldKey.value = value.parentKey;
  selectedSubformChildFieldKey.value = field.fieldKey;
};

const focusCodeDiagnostics = () => {
  diagnosticFocusKey.value += 1;
};

const refreshCurrentCodeDiagnostics = () => {
  codeDiagnostics.value =
    activeMenu.value === 'code' ? validateFormCode(currentFunction.value) : [];
};

const showCodeDiagnosticMessage = (diagnostic: PluginCodeDiagnostic) => {
  ElMessage.error(
    `${'代码语法错误'}：${'代码错误位置'} ${diagnostic.line}:${diagnostic.column}，${diagnostic.message}`,
  );
};

/** 校验当前参数 JSON 草稿；失败时保留代码视图并提示用户修正。 */
const validateCurrentConfigDraft = () => {
  const result = codeViewRef.value?.validateConfigDraft();
  if (!result || result.valid) return true;
  ElMessage.error(result.message || '参数 JSON 的数据结构不正确');
  return false;
};

const validateBeforeSave = () => {
  if (!validateCurrentConfigDraft()) return false;
  const validateResult = validateFormDesignCode(designData.functions);
  if (validateResult) {
    // 保存前统一校验所有函数；若非当前函数出错，先切到该函数代码页再定位错误。
    designData.activeFunctionId = validateResult.functionData.id;
    activeMenu.value = 'code';
    codeDiagnostics.value = validateResult.diagnostics;
    focusCodeDiagnostics();
    showCodeDiagnosticMessage(validateResult.diagnostics[0]);
    return false;
  }

  // Node.js 语法由 Monaco JavaScript 语言服务检查，避免运行时与源码不匹配时提交到后端。
  const editorDiagnostics = codeViewRef.value?.getSyntaxDiagnostics() || [];
  if (!editorDiagnostics.length) return true;
  codeDiagnostics.value = editorDiagnostics;
  focusCodeDiagnostics();
  showCodeDiagnosticMessage(editorDiagnostics[0]);
  return false;
};

const applyPluginFunctionList = (
  triggerList: PluginTrigger[],
  appletList: PluginApplet[],
  force = false,
) => {
  // 仅在没有已保存设计数据时，用接口或 mock 初始化后端函数与前端扩展，避免覆盖用户编辑内容。
  const canUseFunctionList =
    force ||
    (!props.modelValue?.functions?.length &&
      designData.functions.length === 1 &&
      designData.functions[0].id === 'function_default');
  const functionList = [...triggerList, ...appletList];
  if (!canUseFunctionList) {
    // 旧版 pluginDesign 已有函数时仅补齐后端稳定标识，不覆盖尚未保存的代码和参数编辑。
    const functionKeyMap = new Map(functionList.map((item) => [String(item.id), item.functionKey]));
    designData.functions.forEach((functionData) => {
      const functionKey = functionKeyMap.get(functionData.id);
      if (functionKey) functionData.functionKey = functionKey;
    });
    return false;
  }
  if (!functionList.length) return false;
  // 两类函数分别使用后端 seq 排序，再在组内规范为从 0 开始的连续排序号。
  const backendFunctions = triggerList
    .map((item, index) => ({
      seq: typeof item.seq === 'number' ? item.seq : index,
      functionData: createFunctionFromTrigger(item),
    }))
    .sort((prev, next) => prev.seq - next.seq)
    .map(({ functionData }, seq) => ({ ...functionData, seq }));
  const frontendFunctions = appletList
    .map((item, index) => ({
      seq: typeof item.seq === 'number' ? item.seq : index,
      functionData: createFunctionFromApplet(item),
    }))
    .sort((prev, next) => prev.seq - next.seq)
    .map(({ functionData }, seq) => ({ ...functionData, seq }));
  const functions = [...frontendFunctions, ...backendFunctions];
  const previousActiveFunctionId = designData.activeFunctionId;
  designData.functions.splice(0, designData.functions.length, ...functions);
  designData.activeFunctionId = functions.some((item) => item.id === previousActiveFunctionId)
    ? previousActiveFunctionId
    : functions[0].id;
  if (force) {
    resetSavedFunctionSignatures();
    return true;
  }
  // 异步加载函数列表后仍默认停留在通用参数。
  activeMenu.value = 'common';
  selectedFieldKey.value = '';
  selectedSubformChildFieldKey.value = '';
  selectedAuthFieldKey.value = '';
  // 接口加载的初始函数属于已保存数据，首次进入后直接切换不重复更新。
  resetSavedFunctionSignatures();
  return true;
};

const createModuleSignature = (value: unknown) => JSON.stringify(value) ?? '';

/**
 * 函数内容脏检查不包含 seq；拖拽排序通过独立事件立即持久化，避免切换菜单时重复更新。
 * @param functionData 需要比较的函数配置。
 */
const createFunctionSignature = (functionData: PluginDesignFunction) =>
  createModuleSignature({
    id: functionData.id,
    name: functionData.name,
    functionType: functionData.functionType,
    runtime: functionData.runtime,
    fields: functionData.fields,
    responseParams: functionData.responseParams,
    code: functionData.code,
  });

const resetSavedFunctionSignatures = () => {
  savedFunctionSignatures.clear();
  designData.functions.forEach((functionData) => {
    savedFunctionSignatures.set(functionData.id, createFunctionSignature(functionData));
  });
};

/** 判断函数是否仍是尚未由后端创建的本地临时数据。 */
const isLocalFunction = (functionId: string) => /^function_(?:default|\d+)$/.test(functionId);

/**
 * 应用分组内拖拽结果，并将 seq 发生变化的已保存函数交给父抽屉逐条更新。
 * @param payload 当前分组及完整函数排序结果。
 */
const handleSortFunctions = (payload: PluginDesignFunctionSortPayload) => {
  const previousFunctions = cloneFormDesign(designData.functions);
  const previousSeqMap = new Map(
    previousFunctions.map((functionData) => [functionData.id, functionData.seq]),
  );
  const isFrontendGroup = payload.functionGroup === 'frontend';
  const changedFunctions = payload.sortedFunctions.filter(
    (functionData) =>
      (functionData.functionType === 'frontend') === isFrontendGroup &&
      previousSeqMap.get(functionData.id) !== functionData.seq &&
      !isLocalFunction(functionData.id),
  );

  sortFunctions(payload.functions);
  if (!changedFunctions.length) return;

  const requestFlag = ++functionSortRequestFlag;
  const requestPluginId = String(props.pluginId ?? '');
  emits('sort-functions', cloneFormDesign(changedFunctions), async (success) => {
    // 连续拖拽或切换插件后，只允许最后一次请求恢复或确认当前列表。
    if (
      requestFlag !== functionSortRequestFlag ||
      requestPluginId !== String(props.pluginId ?? '')
    ) {
      return;
    }
    if (success) return;

    const { backendList, frontendList } = await loadPluginTriggerList({
      pluginId: props.pluginId,
    });
    if (
      requestFlag !== functionSortRequestFlag ||
      requestPluginId !== String(props.pluginId ?? '')
    ) {
      return;
    }
    // 更新失败优先采用后端真实顺序；列表也不可用时回退到本次拖拽前状态。
    const refreshed = applyPluginFunctionList(backendList, frontendList, true);
    if (!refreshed) sortFunctions(previousFunctions);
  });
};

onMounted(async () => {
  loadAuthTemplateList();
  loadPluginFieldMetadata();
  // 函数列表接口已经按后端和前端分组返回，无需再单独请求前端扩展列表。
  const { backendList: triggerList, frontendList: appletList } = await loadPluginTriggerList({
    pluginId: props.pluginId,
  });
  applyPluginFunctionList(triggerList, appletList);
});

watch(
  () => props.modelValue,
  (value) => {
    if (skipNextModelSync) {
      skipNextModelSync = false;
      return;
    }
    const nextDesign = value?.functions?.length ? cloneFormDesign(value) : createDefaultDesign();
    normalizeDesignData(nextDesign);
    Object.assign(designData, nextDesign);
    // 插件详情首次进入时建立全部函数基线；同一插件的接口回显由保存回调刷新当前函数。
    const nextScopeKey = String(props.pluginId ?? '');
    if (savedFunctionScopeKey !== nextScopeKey) {
      savedFunctionScopeKey = nextScopeKey;
      functionSortRequestFlag += 1;
      functionInfoUpdateFlags.clear();
      runtimeCodeDrafts.clear();
      resetSavedFunctionSignatures();
    }
    selectedFieldKey.value = '';
    selectedSubformChildFieldKey.value = '';
    selectedAuthFieldKey.value = '';
  },
  { immediate: true, deep: true },
);

// 设计器拆成多个子组件后，仍由容器统一向父抽屉同步完整配置。
watch(
  designData,
  () => {
    skipNextModelSync = true;
    emits('update:modelValue', cloneFormDesign(designData));
  },
  { deep: true },
);

// 身份验证与函数设计平级维护，避免再次写回 pluginDesign 内部。
watch(
  () => props.authentication,
  (value) => {
    if (skipNextAuthenticationSync) {
      skipNextAuthenticationSync = false;
      return;
    }
    authenticationData.value = normalizePluginDesignAuthentication(
      cloneFormDesign(value || createDefaultAuthentication()),
    );
    savedAuthenticationSignature = createModuleSignature(authenticationData.value);
    selectedAuthFieldKey.value = '';
  },
  { immediate: true, deep: true },
);

watch(
  authenticationData,
  (value) => {
    skipNextAuthenticationSync = true;
    emits('update:authentication', cloneFormDesign(value));
  },
  { deep: true },
);

// 通用参数为插件级状态，切换、复制或删除函数都不会改变该字段集合。
watch(
  () => props.globalFields,
  (value) => {
    if (skipNextGlobalFieldsSync) {
      skipNextGlobalFieldsSync = false;
      return;
    }
    globalFieldsData.value = normalizePluginDesignFields(
      cloneFormDesign(value || createDefaultGlobalFields()),
      'global',
    );
    savedGlobalFieldsSignature = createModuleSignature(globalFieldsData.value);
    selectedFieldKey.value = '';
    selectedSubformChildFieldKey.value = '';
  },
  { immediate: true, deep: true },
);

watch(
  globalFieldsData,
  (value) => {
    skipNextGlobalFieldsSync = true;
    emits('update:globalFields', cloneFormDesign(value));
  },
  { deep: true },
);

watch(
  () => [
    currentFunction.value.id,
    currentFunction.value.runtime,
    currentFunction.value.code,
    activeMenu.value,
  ],
  () => {
    refreshCurrentCodeDiagnostics();
  },
);

const changeAuthMethod = (authMethod: string) => {
  authenticationData.value = createAuthenticationByMethod(authMethod);
  selectedAuthFieldKey.value = '';
};

const markSaved = () => {
  const now = new Date();
  designData.savedAt = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(
    now.getDate(),
  ).padStart(2, '0')} ${String(now.getHours()).padStart(2, '0')}:${String(
    now.getMinutes(),
  ).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`;
};

// 保存事件携带三个平级模块及当前菜单，父抽屉据此发起对应的局部更新请求。
const createDesignSaveValue = (): PluginDesignSaveValue => ({
  activeMenu: activeMenu.value,
  authentication: cloneFormDesign(authenticationData.value),
  globalFields: cloneFormDesign(globalFieldsData.value),
  pluginDesign: cloneFormDesign(designData),
});

/**
 * 编辑函数名称或描述时立即调用函数更新接口，成功后再将接口回显写入设计器。
 * @param value 编辑浮层提交的函数标识、名称、描述和运行时。
 */
const handleUpdateFunctionInfo = (value: PluginDesignFunctionUpdatePayload) => {
  const sourceFunction = designData.functions.find((item) => item.id === value.id);
  if (
    !sourceFunction ||
    (sourceFunction.name === value.name &&
      sourceFunction.functionDescription === value.functionDescription &&
      sourceFunction.runtime === value.runtime)
  ) {
    return;
  }

  // 新建插件或本地临时函数尚无后端记录，继续保留本地编辑并等待创建/保存流程持久化。
  if (props.pluginId == null || props.pluginId === '' || isLocalFunction(value.id)) {
    updateFunction(value);
    return;
  }

  const saveValue = createDesignSaveValue();
  const savedFunction = saveValue.pluginDesign.functions.find((item) => item.id === value.id);
  if (!savedFunction) return;
  savedFunction.name = value.name;
  savedFunction.functionDescription = value.functionDescription;
  savedFunction.runtime = value.runtime;
  saveValue.pluginDesign.activeFunctionId = value.id;
  saveValue.activeMenu = 'code';

  const requestFlag = (functionInfoUpdateFlags.get(value.id) || 0) + 1;
  const requestPluginId = String(props.pluginId);
  functionInfoUpdateFlags.set(value.id, requestFlag);
  emits('update-function-info', saveValue, (success, updatedFunction) => {
    if (
      !success ||
      !updatedFunction ||
      requestFlag !== functionInfoUpdateFlags.get(value.id) ||
      requestPluginId !== String(props.pluginId ?? '')
    ) {
      return;
    }

    const functionIndex = designData.functions.findIndex((item) => item.id === value.id);
    if (functionIndex === -1) return;
    designData.functions.splice(functionIndex, 1, cloneFormDesign(updatedFunction));
    if (designData.activeFunctionId === value.id) {
      designData.activeFunctionId = updatedFunction.id;
    }
    savedFunctionSignatures.delete(value.id);
    savedFunctionSignatures.set(updatedFunction.id, createFunctionSignature(updatedFunction));
    markSaved();
  });
};

const hasActiveModuleChanges = () => {
  if (activeMenu.value === 'auth') {
    return createModuleSignature(authenticationData.value) !== savedAuthenticationSignature;
  }
  if (activeMenu.value === 'common') {
    return createModuleSignature(globalFieldsData.value) !== savedGlobalFieldsSignature;
  }
  return (
    createFunctionSignature(currentFunction.value) !==
    savedFunctionSignatures.get(currentFunction.value.id)
  );
};

const markActiveModuleSaved = () => {
  if (activeMenu.value === 'auth') {
    savedAuthenticationSignature = createModuleSignature(authenticationData.value);
    return;
  }
  if (activeMenu.value === 'common') {
    savedGlobalFieldsSignature = createModuleSignature(globalFieldsData.value);
    return;
  }
  savedFunctionSignatures.set(
    currentFunction.value.id,
    createFunctionSignature(currentFunction.value),
  );
};

const applyNavigationTarget = (target: PluginDesignNavigationTarget) => {
  if (designData.activeFunctionId !== target.functionId) applySelectFunction(target.functionId);
  if (activeMenu.value !== target.menuKey) applySwitchMenu(target.menuKey);
};

/**
 * 串行处理侧栏导航；请求期间的新点击只保留最后一个目标，避免快速切换产生竞态。
 */
const flushPendingNavigation = async () => {
  if (navigationUpdating) return;
  navigationUpdating = true;
  try {
    while (pendingNavigationTarget) {
      const target = pendingNavigationTarget;
      pendingNavigationTarget = null;
      if (
        target.menuKey === activeMenu.value &&
        target.functionId === designData.activeFunctionId
      ) {
        continue;
      }

      // 参数代码存在错误时禁止离开当前上下文，确保用户草稿不会被下一页覆盖。
      if (!validateCurrentConfigDraft()) {
        pendingNavigationTarget = null;
        break;
      }

      if (hasActiveModuleChanges()) {
        const updated = await new Promise<boolean>((resolve) => {
          // 必须在修改 activeMenu/activeFunctionId 前生成快照，确保保存的是离开模块。
          emits('update', createDesignSaveValue(), resolve);
        });
        if (!updated) {
          pendingNavigationTarget = null;
          break;
        }
        // 等待父组件的接口回显完成，再记录后端确认后的字段 ID 和函数 ID。
        await nextTick();
        markActiveModuleSaved();
        markSaved();
      }
      applyNavigationTarget(target);
    }
  } finally {
    navigationUpdating = false;
  }
};

const requestNavigation = (target: PluginDesignNavigationTarget) => {
  pendingNavigationTarget = target;
  void flushPendingNavigation();
};

const handleSwitchMenu = (menuKey: PluginDesignMenuKey) => {
  requestNavigation({ menuKey, functionId: designData.activeFunctionId });
};

const handleSelectFunction = (functionId: string) => {
  requestNavigation({ menuKey: 'code', functionId });
};

const handleSwitchFunctionMenu = (functionId: string, menuKey: PluginDesignMenuKey) => {
  requestNavigation({ menuKey, functionId });
};

/** 生成函数运行时草稿键，防止不同插件或函数之间错误复用源码。 */
const createRuntimeCodeDraftKey = (functionId: string, runtime: PluginRuntime) =>
  `${String(props.pluginId ?? 'local')}:${functionId}:${runtime}`;

/**
 * 切换运行时时保存当前语言草稿，并恢复目标语言草稿或默认模板。
 * @param runtime 用户选择的目标运行时。
 */
const handleRuntimeChange = (runtime: PluginRuntime) => {
  const functionData = currentFunction.value;
  const previousRuntime = normalizeFormRuntime(functionData.runtime);
  const nextRuntime = normalizeFormRuntime(runtime);
  if (previousRuntime === nextRuntime) return;

  runtimeCodeDrafts.set(
    createRuntimeCodeDraftKey(functionData.id, previousRuntime),
    functionData.code,
  );
  functionData.runtime = nextRuntime;
  functionData.code =
    runtimeCodeDrafts.get(createRuntimeCodeDraftKey(functionData.id, nextRuntime)) ??
    createDefaultCode(nextRuntime);
  codeDiagnostics.value = [];
  // JSON 编辑器 hook 不监听运行时，主动刷新才能立即显示目标语言源码。
  void nextTick().then(() => codeViewRef.value?.refreshEditorContent());
};

/**
 * 切换表单/代码视图前校验参数 JSON，校验失败时保持当前代码视图。
 * @param viewMode 用户选择的目标视图。
 */
const handleChangeViewMode = (viewMode: PluginDesignViewMode) => {
  if (viewMode === currentFunction.value.viewMode) return;
  if (currentFunction.value.viewMode === 'code' && !validateCurrentConfigDraft()) {
    return;
  }
  currentFunction.value.viewMode = viewMode;
};

/**
 * 根据异步保存结果更新保存时间，并在保存测试成功后打开调试抽屉。
 * @param shouldOpenDebug 保存成功后是否需要打开调试抽屉。
 */
const createSaveCompleteHandler = (shouldOpenDebug: boolean): PluginDesignSaveComplete => {
  return (success) => {
    if (!success) return;
    markSaved();
    // 手动保存成功刷新基线和代码视图，使后端生成的字段 ID 及时回显。
    void nextTick().then(() => {
      markActiveModuleSaved();
      codeViewRef.value?.refreshEditorContent();
    });
    if (shouldOpenDebug) debugVisible.value = true;
  };
};

const handleSave = () => {
  if (!validateBeforeSave()) return;
  emits('save', createDesignSaveValue(), createSaveCompleteHandler(false));
};

const handleSaveTest = () => {
  if (!validateBeforeSave()) return;
  // 保存并测试等待父组件确认接口成功，再由完成回调打开调试面板。
  emits('test', createDesignSaveValue(), createSaveCompleteHandler(true));
};
</script>

<style lang="scss" scoped>
.plugin-design {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding-right: var(--el-space-2xl);
  padding-bottom: var(--gp-space-2xl);
  padding-left: var(--gp-space-2xl);
  background-color: var(--el-fill-color-light);

  &__workspace {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    background-color: var(--el-bg-color);
    border-radius: var(--gp-radius-md);
  }

  &__content {
    flex: 1;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    background-color: var(--el-bg-color);
    border-radius: var(--gp-radius-md) var(--gp-radius-md) var(--gp-radius-md) var(--gp-radius-md);

    // 身份验证和返回参数没有控件面板与画布组件，由内容容器补齐完整边框。
    &--bordered {
      box-sizing: border-box;
      border: 1px solid var(--el-border-color);
    }
  }

  &__sidebar-panel {
    display: flex;
    flex-shrink: 0;
    width: 200px;
    min-width: 0;
    overflow: hidden;
  }

  &__content-scrollbar {
    width: 100%;
    height: 100%;
  }

  &__auth-property-panel {
    display: flex;
    flex-shrink: 0;
    width: 260px;
    min-width: 0;
    overflow: hidden;

    :deep(.plugin-design-auth-property) {
      height: 100%;
    }
  }

  :deep(.plugin-design__content-view) {
    display: flex;
    flex-direction: column;
    min-height: 100%;
  }

  :deep(.plugin-design__content-view--full) {
    height: 100%;
  }
}

// 侧栏通过宽度参与工作区布局动画，收起时内容区同步平滑补位。
.plugin-design-sidebar-slide-enter-active,
.plugin-design-sidebar-slide-leave-active {
  transition:
    width 0.22s ease,
    opacity 0.22s ease,
    transform 0.22s ease;
}

.plugin-design-sidebar-slide-enter-from,
.plugin-design-sidebar-slide-leave-to {
  width: 0;
  opacity: 0;
  transform: translateX(calc(-1 * var(--gp-space-2xl)));
}

// 同步过渡面板宽度与位置，避免属性面板出现时内容画布瞬间跳动。
.plugin-design-auth-property-slide-enter-active,
.plugin-design-auth-property-slide-leave-active {
  transition:
    width 0.22s ease,
    opacity 0.22s ease,
    transform 0.22s ease;
}

.plugin-design-auth-property-slide-enter-from,
.plugin-design-auth-property-slide-leave-to {
  width: 0;
  opacity: 0;
  transform: translateX(var(--gp-space-2xl));
}

@media (prefers-reduced-motion: reduce) {
  .plugin-design-sidebar-slide-enter-active,
  .plugin-design-sidebar-slide-leave-active,
  .plugin-design-auth-property-slide-enter-active,
  .plugin-design-auth-property-slide-leave-active {
    transition-duration: 0.01ms;
  }
}
</style>
