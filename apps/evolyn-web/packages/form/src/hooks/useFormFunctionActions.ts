import { type Ref } from 'vue';
import { type PluginFunctionCreateResult, type PluginFunctionDetail } from '../api';
import type {
  PluginDesignConfig,
  PluginDesignFunction,
  PluginDesignFunctionUpdatePayload,
  PluginDesignMenuKey,
  PluginRuntime,
} from '../types';

interface UseFormFunctionActionsOptions {
  /** 设计器当前菜单，仅用于页面交互，不写入函数配置。 */
  activeMenu: Ref<PluginDesignMenuKey>;
  createFunction: (
    id: string,
    name: string,
    runtime?: PluginRuntime,
    functionType?: PluginDesignFunction['functionType'],
  ) => PluginDesignFunction;
  designData: PluginDesignConfig;
  /** 获取设计器所属插件 ID，确保新增请求使用最新的插件信息。 */
  getPluginId: () => string | number | null;
  selectedFieldKey: Ref<string>;
}

/** 判断接口响应节点是否为可继续读取的对象。 */
const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value);

/**
 * 从 Axios 响应和业务 data 包装中提取新增函数结果。
 * @param payload 新增函数接口的原始响应。
 */
const getCreatedFormFunction = (payload: unknown): PluginFunctionCreateResult | null => {
  if (!isRecord(payload)) return null;
  if (typeof payload.id === 'string' || typeof payload.id === 'number') {
    return payload as PluginFunctionCreateResult;
  }
  return getCreatedFormFunction(payload.data);
};

/**
 * 从 Axios 响应和业务 data 包装中提取复制后的完整函数。
 * @param payload 复制函数接口的原始响应。
 */
const getCopiedPluginFunction = (payload: unknown): PluginFunctionDetail | null => {
  if (!isRecord(payload)) return null;
  if (typeof payload.id === 'string' || typeof payload.id === 'number') {
    return payload as unknown as PluginFunctionDetail;
  }
  return getCopiedPluginFunction(payload.data);
};

export const useFormFunctionActions = ({
  activeMenu,
  designData,
  selectedFieldKey,
}: UseFormFunctionActionsOptions) => {
  const switchMenu = (key: PluginDesignMenuKey) => {
    activeMenu.value = key;
    selectedFieldKey.value = '';
  };

  const selectFunction = (id: string) => {
    if (!designData.functions.some((item) => item.id === id)) return;
    designData.activeFunctionId = id;
    activeMenu.value = 'code';
    selectedFieldKey.value = '';
  };

  const sortFunctions = (functions: PluginDesignFunction[]) => {
    // 拖拽排序同步重排后的分组 seq，函数内容和当前选中状态保持不变。
    designData.functions.splice(0, designData.functions.length, ...functions);
  };

  const updateFunction = (value: PluginDesignFunctionUpdatePayload) => {
    const target = designData.functions.find((item) => item.id === value.id);
    if (!target) return;
    target.name = value.name;
    target.functionDescription = value.functionDescription;
    target.runtime = value.runtime;
  };

  return {
    selectFunction,
    sortFunctions,
    switchMenu,
    updateFunction,
  };
};
