import {
  PluginRuntime,
  type PluginDesignAuthentication,
  type PluginDesignConfig,
  type PluginDesignField,
  type PluginDesignFunction,
  type PluginDesignResponseField,
  type PluginDesignTemplateField,
} from '../types';
import { isFormNodeRuntime, normalizeFormRuntime, formRuntimeOptions } from './useFormRuntime';
import {
  createPluginSelectorDefaultValue,
  getPluginSelectorValueKey,
  isPluginSelectWidget,
  isFormSubformWidget,
  isPluginTextWidget,
  pluginWidgetNames,
} from '../utils/widgetName';

export type FormDesignTranslate = (key: string, ...args: unknown[]) => string;
/** @deprecated 使用 FormDesignTranslate。 */
export type PluginDesignTranslate = FormDesignTranslate;

export const formDesignRuntimeOptions = formRuntimeOptions;
/** @deprecated 使用 formDesignRuntimeOptions。 */
export const runtimeOptions = formDesignRuntimeOptions;
// 返回参数直接使用后端 dataType，普通值为 any，对象数组为 vector。
export const paramTypes = ['any', 'vector'];

/** 表单设计数据仅包含可序列化值，深拷贝用于字段复制和外部变更隔离。 */
export const cloneFormDesign = <T>(value: T): T => JSON.parse(JSON.stringify(value));
/** @deprecated 旧插件模块迁移期间的兼容别名。 */
export const cloneDesign = cloneFormDesign;

let subformChildFieldSeed = 0;

export const createDefaultCode = (runtime: PluginRuntime) => {
  if (isFormNodeRuntime(runtime)) {
    return [
      '// 可以通过获取预定义的全局变量中的属性来获取你定义的参数',
      'const tenantId = triggerConf.tenantId',
      'const apiKey = String(agentConf.apiKey || "")',
      '',
      'return {',
      '  tenantId,',
      '  apiKey',
      '}',
    ].join('\n');
  }
  return [
    '# 可以引用一些第三方库.',
    'import json',
    'import requests',
    '',
    "api_key = str(agentConf.get('apiKey'))",
    "tenantId = triggerConf.get('tenantId')",
    '',
    "url = 'https://gpdev.msic.com.cn/openapi/employment/v1/businessList'",
    "payload = {'tenantId': tenantId}",
    "headers = {'Authorization': 'Bearer ' + api_key}",
    '',
    'response = requests.post(url, json=payload, headers=headers)',
    'body = response.json()',
    '',
    'if response.status_code >= 300:',
    "    message = body.get('message', '未知错误')",
    "    code = body.get('code', body.get('state', -1))",
    "    raise ValueError('请求错误(%s): %s' % (code, message))",
    '',
    "if body.get('state') != 200:",
    "    message = body.get('message', '未知错误')",
    "    raise ValueError('业务错误(%s): %s' % (body.get('state'), message))",
    '',
    "business_list = body.get('data', [])",
    '',
    'items = [',
    '    {',
    "        'id': item.get('id'),",
    "        'businessLineName': item.get('businessLineName'),",
    "        'businessLineNo': item.get('businessLineNo'),",
    "        'remark': item.get('remark'),",
    '    }',
    '    for item in business_list',
    ']',
    '',
    'return {',
    "    'items': items,",
    "    'count': len(items)",
    '}',
  ].join('\n');
};

export const useFormDesignFactory = () => {
  const normalizeSubformChildWidgetName = (widgetName: string) => {
    const supportedWidgetNames: string[] = [
      pluginWidgetNames.text,
      pluginWidgetNames.number,
      pluginWidgetNames.datetime,
      pluginWidgetNames.select,
      pluginWidgetNames.member,
      pluginWidgetNames.department,
    ];
    return supportedWidgetNames.includes(widgetName) ? widgetName : null;
  };

  const getSubformChildFieldLabel = (widgetName: string) => {
    const labelMap: Record<string, string> = {
      [pluginWidgetNames.text]: '文本',
      [pluginWidgetNames.number]: '数字',
      [pluginWidgetNames.datetime]: '日期时间',
      [pluginWidgetNames.select]: '下拉框',
      [pluginWidgetNames.member]: '成员选择',
      [pluginWidgetNames.department]: '部门选择',
    };
    return labelMap[widgetName] || widgetName;
  };

  const getDefaultSubformSelectItems = () => [
    { text: '选项1', value: '选项1' },
    { text: '选项2', value: '选项2' },
  ];

  // 子表单子字段和外层字段结构不同，统一在工厂里生成，避免拖入和右侧添加出现结构差异。
  const createSubformChildField = (
    widgetName: string,
    dataType: string,
    label?: string,
  ): PluginDesignTemplateField | null => {
    const childWidgetName = normalizeSubformChildWidgetName(widgetName);
    if (!childWidgetName) return null;
    const fieldKey = `_widget_${Date.now()}${subformChildFieldSeed++}`;
    const field: PluginDesignTemplateField = {
      id: null,
      fieldKey,
      fieldLabel: label || getSubformChildFieldLabel(childWidgetName),
      description: '',
      widgetName: childWidgetName,
      dataType,
      isHidden: false,
      isEnabled: true,
      isRequired: false,
      fieldConf: {},
    };
    if (isPluginTextWidget(childWidgetName)) {
      field.fieldConf = { isMultiLine: false };
      field.defaultValue = '';
    }
    if (childWidgetName === pluginWidgetNames.datetime) field.defaultValue = '';
    if (isPluginSelectWidget(childWidgetName)) {
      field.fieldConf = { items: getDefaultSubformSelectItems() };
      field.defaultValue = '';
    }
    if (getPluginSelectorValueKey(childWidgetName)) {
      field.defaultValue = createPluginSelectorDefaultValue(childWidgetName);
    }
    return field;
  };

  const createDefaultAuthentication = (): PluginDesignAuthentication => ({
    type: 'none',
    conf_template: {
      fields: [],
    },
  });

  // 插件级通用参数默认提供 apiKey，调试时统一进入 agent_conf.apiKey。
  const createDefaultGlobalFields = (): PluginDesignField[] => [
    createField('令牌', pluginWidgetNames.text, 'String', 'apiKey'),
  ];

  const getDefaultFieldValue = (widgetName: string) => {
    if (isFormSubformWidget(widgetName)) return undefined;
    if (getPluginSelectorValueKey(widgetName)) return createPluginSelectorDefaultValue(widgetName);
    return '';
  };

  const getDefaultFieldConf = (widgetName: string): Record<string, unknown> | undefined => {
    if (isFormSubformWidget(widgetName)) return { fields: [] };
    return undefined;
  };

  const createField = (
    fieldLabel: string,
    widgetName: string,
    dataType: string,
    fieldKey?: string,
  ): PluginDesignField => {
    const nextFieldKey = fieldKey || `_widget_${Date.now()}`;
    return {
      id: null,
      fieldKey: nextFieldKey,
      fieldLabel,
      // 组件属性由字段面板直接传入，字段不再维护额外的 fieldType。
      widgetName,
      dataType,
      placeholder:
        isFormSubformWidget(widgetName) || isPluginSelectWidget(widgetName)
          ? ''
          : getPluginSelectorValueKey(widgetName)
            ? '请选择'
            : widgetName === pluginWidgetNames.datetime
              ? '请选择'
              : '请输入',
      isRequired: false,
      defaultValue: getDefaultFieldValue(widgetName),
      options: isPluginSelectWidget(widgetName) ? ['选项1', '选项2'] : undefined,
      fieldConf: getDefaultFieldConf(widgetName),
    };
  };

  /**
   * 创建与后端返回参数字段一致的普通值字段。
   * @param fieldKey 返回值中的业务字段标识。
   */
  const createResponseField = (fieldKey: string): PluginDesignResponseField => ({
    id: null,
    fieldKey,
    fieldLabel: '返回参数',
    widgetName: '',
    dataType: 'any',
    fieldConf: {},
  });

  const createFunction = (
    id: string,
    name: string,
    runtime: PluginRuntime = PluginRuntime.Python3,
    functionType: PluginDesignFunction['functionType'] = 'backend',
  ): PluginDesignFunction => {
    const normalizedRuntime = normalizeFormRuntime(runtime);
    return {
      id,
      name,
      // 新函数默认没有描述，后续可通过函数操作浮层单独编辑。
      functionDescription: '',
      functionType,
      // 新建函数加入分组时会校验并补充最终排序号。
      seq: 0,
      runtime: normalizedRuntime,
      viewMode: 'form',
      fields: [createField('tenantId', pluginWidgetNames.text, 'String', 'tenantId')],
      responseParams: [
        {
          id: null,
          fieldKey: 'items',
          fieldLabel: '返回参数',
          widgetName: '',
          dataType: 'vector',
          fieldConf: {
            fields: [
              createResponseField('id'),
              createResponseField('businessLineName'),
              createResponseField('businessLineNo'),
              createResponseField('remark'),
            ],
          },
        },
        createResponseField('count'),
      ],
      code: createDefaultCode(normalizedRuntime),
    };
  };

  const createDefaultDesign = (): PluginDesignConfig => ({
    activeFunctionId: 'function_default',
    functions: [createFunction('function_default', '未命名函数')],
  });

  return {
    createDefaultAuthentication,
    createDefaultGlobalFields,
    createDefaultDesign,
    createField,
    createFunction,
    createSubformChildField,
  };
};

/** @deprecated 使用 useFormDesignFactory。 */
export const usePluginDesignFactory = useFormDesignFactory;
