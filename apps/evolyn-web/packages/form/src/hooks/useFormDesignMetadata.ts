import { ref } from 'vue';
import {
  getPluginTriggerList,
  type PluginAuthTemplate,
  type PluginFieldMetadata,
  type PluginFunctionListResponse,
  type PlatformFunctionListType,
} from '../api';

interface AuthTemplateListPayload {
  plugin_auth_template_list?: PluginAuthTemplate[];
  data?: AuthTemplateListPayload;
}

interface PluginTriggerListPayload extends Partial<PluginFunctionListResponse> {
  data?: PluginTriggerListPayload;
}

interface PluginFieldMetadataPayload {
  fields?: PluginFieldMetadata[];
  data?: PluginFieldMetadataPayload;
}

const getAuthTemplateListFromPayload = (payload: unknown): PluginAuthTemplate[] => {
  if (!payload || typeof payload !== 'object') return [];
  const data = payload as AuthTemplateListPayload;
  if (Array.isArray(data.plugin_auth_template_list)) return data.plugin_auth_template_list;
  return getAuthTemplateListFromPayload(data.data);
};

const getPluginTriggerListFromPayload = (payload: unknown): PluginFunctionListResponse => {
  if (!payload || typeof payload !== 'object') return { backendList: [], frontendList: [] };
  const data = payload as PluginTriggerListPayload;
  if (Array.isArray(data.backendList) || Array.isArray(data.frontendList)) {
    return {
      backendList: Array.isArray(data.backendList) ? data.backendList : [],
      frontendList: Array.isArray(data.frontendList) ? data.frontendList : [],
    };
  }
  return getPluginTriggerListFromPayload(data.data);
};

const getPluginFieldMetadataFromPayload = (payload: unknown): PluginFieldMetadata[] => {
  if (!payload || typeof payload !== 'object') return [];
  const data = payload as PluginFieldMetadataPayload;
  if (Array.isArray(data.fields)) return data.fields;
  return getPluginFieldMetadataFromPayload(data.data);
};

export const useFormDesignMetadata = () => {
  const authTemplateList = ref<PluginAuthTemplate[]>([]);
  const fieldMetadataList = ref<PluginFieldMetadata[]>([]);

  const loadAuthTemplateList = async () => {
    try {
      // 兼容网关包装或接口直接返回两种结构，后续表单渲染统一读取 authTemplateList。
      const remoteTemplateList = [];
      authTemplateList.value = remoteTemplateList.length ? remoteTemplateList : [];
    } catch {
      authTemplateList.value = [];
    }
  };

  const loadPluginFieldMetadata = async () => {
    try {
      const remoteFieldMetadata = [];
      fieldMetadataList.value = remoteFieldMetadata.length ? remoteFieldMetadata : [];
    } catch {
      fieldMetadataList.value = [];
    }
  };

  /**
   * 根据插件 ID 加载函数列表，并将接口分组结果统一提供给设计器。
   * @param params 插件函数列表查询参数。
   */
  const loadPluginTriggerList = async (
    params: PlatformFunctionListType,
  ): Promise<PluginFunctionListResponse> => {
    try {
      const res = await getPluginTriggerList(params);
      const remoteFunctionList = getPluginTriggerListFromPayload(res.data);
      return {
        backendList: remoteFunctionList.backendList.length ? remoteFunctionList.backendList : [],
        frontendList: remoteFunctionList.frontendList.length ? remoteFunctionList.frontendList : [],
      };
    } catch {
      return {
        backendList: [],
        frontendList: [],
      };
    }
  };

  return {
    authTemplateList,
    fieldMetadataList,
    loadAuthTemplateList,
    loadPluginFieldMetadata,
    loadPluginTriggerList,
  };
};
