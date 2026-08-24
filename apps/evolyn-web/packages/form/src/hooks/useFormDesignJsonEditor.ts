import { ref, watch } from 'vue';
import type { PluginAuthTemplateSection } from '../api';
import type { PluginDesignField, PluginDesignFunction, PluginDesignMenuKey } from '../types';
import {
  normalizePluginDesignFields,
  normalizePluginDesignResponseFields,
} from '../utils/designField';
import {
  createGlobalFieldsFromTemplate,
  createGlobalTemplateByFields,
} from '../utils/globalTemplate';

type JsonEditorValidationResult = {
  valid: boolean;
  message?: string;
};

interface UseFormDesignJsonEditorOptions {
  /** 获取当前选中的设计器菜单。 */
  getActiveMenu: () => PluginDesignMenuKey;
  /** 获取当前选中的插件函数。 */
  getFunctionData: () => PluginDesignFunction;
  /** 获取插件通用参数字段。 */
  getGlobalFields: () => PluginDesignField[];
  /** 多语言文案转换函数。 */
  t: (key: string) => string;
}

// 只有通用参数、请求参数和返回参数使用可编辑 JSON 配置。
const configMenuList: PluginDesignMenuKey[] = ['common', 'request', 'response'];

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return !!value && typeof value === 'object' && !Array.isArray(value);
};

/** 校验字段列表的基础结构，避免不完整 JSON 覆盖表单配置。 */
const isValidFieldList = (value: unknown): value is Record<string, unknown>[] => {
  if (!Array.isArray(value)) return false;
  return value.every((item) => {
    if (!isRecord(item) || typeof item.fieldKey !== 'string' || !item.fieldKey.trim()) return false;
    if (
      typeof item.fieldLabel !== 'string' ||
      typeof item.widgetName !== 'string' ||
      typeof item.dataType !== 'string'
    ) {
      return false;
    }
    if (item.id != null && !['string', 'number'].includes(typeof item.id)) return false;
    if (item.description != null && typeof item.description !== 'string') return false;
    if (item.placeholder != null && typeof item.placeholder !== 'string') return false;
    if (item.isRequired !== undefined && typeof item.isRequired !== 'boolean') return false;
    if (
      item.options !== undefined &&
      (!Array.isArray(item.options) || item.options.some((option) => typeof option !== 'string'))
    ) {
      return false;
    }
    if (item.fieldConf === undefined) return true;
    if (!isRecord(item.fieldConf)) return false;
    return item.fieldConf.fields === undefined || isValidFieldList(item.fieldConf.fields);
  });
};

/**
 * 管理参数代码视图的 JSON 草稿、结构校验和表单数据回写。
 * @param options 当前菜单、函数、通用参数及多语言函数。
 */
export const useFormDesignJsonEditor = (options: UseFormDesignJsonEditorOptions) => {
  // 编辑器草稿独立于真实配置，保证输入半截 JSON 时不会破坏表单数据。
  const editorContent = ref('');
  // 缓存最近一次校验结果，供保存和导航入口重复确认。
  const validationResult = ref<JsonEditorValidationResult>({ valid: true });

  /** 判断当前菜单是否属于参数 JSON 配置。 */
  const isConfigMenu = (menu = options.getActiveMenu()) => configMenuList.includes(menu);

  /** 根据当前菜单生成与现有保存结构一致的 JSON 内容。 */
  const createEditorContent = () => {
    const activeMenu = options.getActiveMenu();
    const functionData = options.getFunctionData();
    if (activeMenu === 'common') {
      return JSON.stringify(
        { globalTemplate: createGlobalTemplateByFields(options.getGlobalFields()) },
        null,
        2,
      );
    }
    if (activeMenu === 'request') {
      return JSON.stringify({ fields: functionData.fields }, null, 2);
    }
    if (activeMenu === 'response') {
      return JSON.stringify({ fields: functionData.responseParams }, null, 2);
    }
    return functionData.code;
  };

  /** 使用解析成功的数据替换响应式数组，保持父组件和现有保存链路不变。 */
  const applyConfigValue = (value: Record<string, unknown>) => {
    const activeMenu = options.getActiveMenu();
    const functionData = options.getFunctionData();
    if (activeMenu === 'common') {
      const globalTemplate = value.globalTemplate;
      if (!isRecord(globalTemplate) || !isValidFieldList(globalTemplate.fields)) return false;
      const fields = createGlobalFieldsFromTemplate(
        globalTemplate as unknown as PluginAuthTemplateSection,
      );
      const globalFields = options.getGlobalFields();
      globalFields.splice(0, globalFields.length, ...fields);
      return true;
    }
    if (!isValidFieldList(value.fields)) return false;
    if (activeMenu === 'request') {
      const fields = normalizePluginDesignFields(value.fields, functionData.id);
      functionData.fields.splice(0, functionData.fields.length, ...fields);
      return true;
    }
    if (activeMenu === 'response') {
      const fields = normalizePluginDesignResponseFields(value.fields, functionData.id);
      functionData.responseParams.splice(0, functionData.responseParams.length, ...fields);
      return true;
    }
    return false;
  };

  /** 解析并提交当前参数 JSON；失败时只更新错误状态，不触碰真实配置。 */
  const applyConfigDraft = (value: string): JsonEditorValidationResult => {
    try {
      const parsedValue = JSON.parse(value) as unknown;
      if (!isRecord(parsedValue) || !applyConfigValue(parsedValue)) {
        return {
          valid: false,
          message: options.t('参数 JSON 的数据结构不正确'),
        };
      }
      return { valid: true };
    } catch (error) {
      return {
        valid: false,
        message: `${options.t('参数 JSON 格式错误')}：${
          error instanceof Error ? error.message : options.t('未知错误')
        }`,
      };
    }
  };

  /** 接收 Monaco 内容变化；函数源码直接回写，参数 JSON 校验成功后才回写。 */
  const updateEditorContent = (value: string) => {
    editorContent.value = value;
    if (!isConfigMenu()) {
      options.getFunctionData().code = value;
      validationResult.value = { valid: true };
      return;
    }
    validationResult.value = applyConfigDraft(value);
  };

  /** 保存、切换视图或导航前再次校验，防止无效草稿被静默丢弃。 */
  const validateConfigDraft = () => {
    if (!isConfigMenu()) return { valid: true };
    validationResult.value = applyConfigDraft(editorContent.value);
    return validationResult.value;
  };

  /** 使用当前真实配置刷新编辑器，保存接口回填字段 ID 后保持代码视图同步。 */
  const refreshEditorContent = () => {
    editorContent.value = createEditorContent();
    validationResult.value = { valid: true };
  };

  watch(
    () => `${options.getActiveMenu()}:${options.getFunctionData().id}`,
    () => refreshEditorContent(),
    { immediate: true },
  );

  return {
    editorContent,
    refreshEditorContent,
    updateEditorContent,
    validateConfigDraft,
  };
};
