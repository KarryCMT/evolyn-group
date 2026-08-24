import { computed, type ComputedRef, type Ref } from 'vue';
import type {
  PluginAuthTemplate,
  PluginAuthTemplateField,
  PluginApplet,
  PluginFunctionResponseField,
  PluginTrigger,
  PluginTriggerTemplateField,
} from '../api';
import {
  PluginRuntime,
  type PluginDesignAuthentication,
  type PluginDesignField,
  type PluginDesignFunction,
  type PluginDesignOption,
  type PluginDesignOAuthVectorRow,
  type PluginDesignResponseField,
  type PluginDesignTemplateField,
  type PluginDesignTemplateSection,
} from '../types';
import { normalizeFormRuntime } from './useFormRuntime';
import {
  createPluginSelectorDefaultValue,
  getPluginSelectorValueKey,
  isFormSubformWidget,
} from '../utils/widgetName';

type PluginTemplateField = PluginAuthTemplateField | PluginTriggerTemplateField;

interface UseFormDesignTemplatesOptions {
  authTemplateList: Ref<PluginAuthTemplate[]>;
  createDefaultCode: (runtime: PluginRuntime) => string;
  createFunction: (
    id: string,
    name: string,
    runtime?: PluginRuntime,
    functionType?: PluginDesignFunction['functionType'],
  ) => PluginDesignFunction;
  currentAuthentication: ComputedRef<PluginDesignAuthentication>;
}

const authTypeTemplateAliasMap: Record<string, string> = {
  wecom: 'wxwork',
  wechat: 'wechat_service',
  basic: 'basic_auth',
  oauth_code: 'oauth2_authorization_code',
  oauth_client: 'oauth2_client_credentials',
};

const getTemplateFieldOptions = (field: PluginTemplateField) => {
  const items = field.fieldConf?.items;
  if (!Array.isArray(items)) return undefined;
  return items.map((item) => {
    if (item && typeof item === 'object') {
      const option = item as { text?: unknown; value?: unknown };
      return String(option.text ?? option.value ?? '');
    }
    return String(item);
  });
};

const normalizeTemplateDefaultValue = (field: PluginAuthTemplateField) => {
  if (Array.isArray(field.defaultValue)) {
    return field.defaultValue.map((item) => {
      if (item && typeof item === 'object') {
        const row = item as { key?: unknown; name?: unknown; value?: unknown; val?: unknown };
        return {
          key: String(row.key ?? row.name ?? ''),
          value: String(row.value ?? row.val ?? ''),
        };
      }
      return { key: '', value: String(item ?? '') };
    }) as PluginDesignOAuthVectorRow[];
  }
  if (isFormSubformWidget(field.widgetName)) return [];
  return field.defaultValue === undefined || field.defaultValue === null
    ? ''
    : String(field.defaultValue);
};

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return !!value && typeof value === 'object' && !Array.isArray(value);
};

const normalizeTriggerDefaultValue = (field: PluginTriggerTemplateField) => {
  if (isFormSubformWidget(field.widgetName)) return undefined;
  if (getPluginSelectorValueKey(field.widgetName)) {
    return isRecord(field.defaultValue)
      ? field.defaultValue
      : createPluginSelectorDefaultValue(field.widgetName);
  }
  const defaultValue = field.defaultValue;
  if (defaultValue === undefined || defaultValue === null) return '';
  // 保留接口返回的基础类型，避免 Number、Boolean 等默认值在调试请求中被统一转成字符串。
  if (
    typeof defaultValue === 'string' ||
    typeof defaultValue === 'number' ||
    typeof defaultValue === 'boolean'
  ) {
    return defaultValue;
  }
  return String(defaultValue);
};

const createAuthFieldFromTemplate = (
  field: PluginAuthTemplateField,
): PluginDesignTemplateField => ({
  // 从认证模板新建字段时尚未持久化，id 由保存后的详情接口返回。
  id: null,
  fieldKey: field.fieldKey,
  fieldLabel: field.fieldLabel || field.fieldKey,
  description: field.description || '',
  widgetName: field.widgetName,
  dataType: field.dataType,
  isHidden: Boolean(field.isHidden),
  isEnabled: field.isEnabled !== false,
  isRequired: Boolean(field.isRequired),
  fieldConf: field.fieldConf || {},
  defaultValue: normalizeTemplateDefaultValue(field),
});

const createAuthSectionFromTemplate = (section?: {
  fields?: PluginAuthTemplateField[];
}): PluginDesignTemplateSection => ({
  fields: (section?.fields || [])
    .filter((field) => field.isHidden !== true && field.isEnabled !== false)
    .map(createAuthFieldFromTemplate),
});

const createRedirectUriField = (): PluginDesignTemplateField => ({
  id: null,
  fieldKey: 'redirect_uri',
  fieldLabel: 'redirect_uri',
  description: '',
  widgetName: 'input',
  dataType: 'String',
  isHidden: false,
  isEnabled: true,
  isRequired: true,
  fieldConf: {
    isMultiLine: false,
  },
  defaultValue: '',
});

const ensureRedirectUriField = (section: PluginDesignTemplateSection) => {
  if (section.fields.some((field) => field.fieldKey === 'redirect_uri')) return section;
  return {
    ...section,
    fields: [...section.fields, createRedirectUriField()],
  };
};

const getTriggerTemplateFieldKey = (field: PluginTriggerTemplateField) => {
  return field.fieldKey || String(field.id ?? '');
};

const getTriggerTemplateFieldLabel = (field: PluginTriggerTemplateField, fieldKey: string) => {
  return field.fieldLabel || field.label || fieldKey;
};

/**
 * 将接口子表单字段递归转换为设计器内部模板字段。
 * @param field 函数接口返回的请求参数字段。
 */
function createSubformFieldFromTriggerTemplate(
  field: PluginTriggerTemplateField,
): PluginDesignTemplateField {
  const fieldKey = getTriggerTemplateFieldKey(field);
  const fieldConf = field.fieldConf ? { ...field.fieldConf } : {};
  const childFields = fieldConf.fields;
  if (Array.isArray(childFields)) {
    fieldConf.fields = (childFields as PluginTriggerTemplateField[]).map(
      createSubformFieldFromTriggerTemplate,
    );
  }
  const nextField: PluginDesignTemplateField = {
    id: field.id == null ? null : String(field.id),
    fieldKey,
    fieldLabel: getTriggerTemplateFieldLabel(field, fieldKey),
    description: field.description || '',
    widgetName: field.widgetName,
    dataType: field.dataType,
    isHidden: Boolean(field.isHidden),
    isEnabled: field.isEnabled !== false,
    isRequired: Boolean(field.isRequired),
    fieldConf,
  };
  if (field.defaultValue !== undefined && field.defaultValue !== null) {
    nextField.defaultValue = field.defaultValue as PluginDesignTemplateField['defaultValue'];
  }
  return nextField;
}

const createFieldFromTriggerTemplate = (field: PluginTriggerTemplateField): PluginDesignField => {
  // 新接口使用 fieldKey/fieldLabel；旧模板缺失时才回退到 id/label。
  const fieldKey = getTriggerTemplateFieldKey(field);
  const fieldLabel = getTriggerTemplateFieldLabel(field, fieldKey);
  const fieldConf = field.fieldConf ? { ...field.fieldConf } : {};
  if (Array.isArray(fieldConf.fields)) {
    fieldConf.fields = (fieldConf.fields as PluginTriggerTemplateField[]).map(
      createSubformFieldFromTriggerTemplate,
    );
  } else if (isFormSubformWidget(field.widgetName)) {
    fieldConf.fields = [];
  }
  return {
    // 保存接口会回传字段持久化 id，保留后可用于下一次函数更新。
    id: field.id == null ? null : String(field.id),
    fieldKey,
    fieldLabel,
    // 已保存字段直接使用函数接口返回的组件属性。
    widgetName: field.widgetName,
    dataType: field.dataType,
    placeholder: field.description || '',
    isRequired: Boolean(field.isRequired),
    defaultValue: normalizeTriggerDefaultValue(field),
    options: getTemplateFieldOptions(field),
    fieldConf:
      Object.keys(fieldConf).length || isFormSubformWidget(field.widgetName)
        ? fieldConf
        : undefined,
  };
};

export const createFieldsFromTriggerTemplate = (
  fields: PluginTriggerTemplateField[] = [],
): PluginDesignField[] => {
  // 后端函数入参模板只同步启用且未隐藏字段，避免把平台内部字段展示给用户。
  return fields
    .filter((field) => field.isHidden !== true && field.isEnabled !== false)
    .map(createFieldFromTriggerTemplate);
};

/**
 * 将接口返回字段复制为设计器返回参数字段，并递归保留 fieldConf.fields。
 * @param field 后端 responseParameter.fields 中的字段。
 */
const createResponseFieldFromTrigger = (
  field: PluginFunctionResponseField,
): PluginDesignResponseField => {
  const fieldConf = { ...field.fieldConf };
  if (Array.isArray(field.fieldConf.fields)) {
    fieldConf.fields = field.fieldConf.fields.map(createResponseFieldFromTrigger);
  }
  return {
    id: field.id == null ? null : String(field.id),
    fieldKey: field.fieldKey,
    fieldLabel: field.fieldLabel,
    widgetName: field.widgetName,
    dataType: field.dataType,
    fieldConf,
  };
};

export const createResponseFieldsFromTrigger = (
  fields: PluginFunctionResponseField[] = [],
): PluginDesignResponseField[] => fields.map(createResponseFieldFromTrigger);

export const useFormDesignTemplates = ({
  authTemplateList,
  createDefaultCode,
  createFunction,
  currentAuthentication,
}: UseFormDesignTemplatesOptions) => {
  const authTypeLabelMap = computed<Record<string, string>>(() => ({
    basic_auth: 'Basic Auth',
    basic: 'Basic Auth',
    oauth2_authorization_code: 'OAuth 2.0 - Authorization Code',
    oauth_code: 'OAuth 2.0 - Authorization Code',
    oauth2_client_credentials: 'OAuth 2.0 - Client Credentials',
    oauth_client: 'OAuth 2.0 - Client Credentials',
  }));

  const getTemplateAuthType = (authMethod: string) =>
    authTypeTemplateAliasMap[authMethod] || authMethod;

  const getAuthTemplate = (authMethod: string) => {
    const templateAuthType = getTemplateAuthType(authMethod);
    return authTemplateList.value.find((item) => item.auth_type === templateAuthType);
  };

  const createAuthenticationByMethod = (authMethod: string): PluginDesignAuthentication => {
    if (authMethod === 'none') {
      return {
        type: 'none',
        conf_template: {
          fields: [],
        },
      };
    }
    const template = getAuthTemplate(authMethod);
    // 身份验证按灵衍云提交结构保存：type + conf_template.fields。
    const authentication: PluginDesignAuthentication = {
      type: getTemplateAuthType(authMethod),
      conf_template: createAuthSectionFromTemplate(template?.conf_template),
    };
    if (template?.credential_template)
      authentication.credential_template = createAuthSectionFromTemplate(
        template.credential_template,
      );
    if (template?.oauth2_request_template) {
      authentication.oauth2_request_template = createAuthSectionFromTemplate(
        template.oauth2_request_template,
      );
    }
    if (template?.refresh_token_request_template) {
      authentication.refresh_token_request_template = {
        ...createAuthSectionFromTemplate(template.refresh_token_request_template),
        enabled: false,
      };
    }
    if (template?.app_auth_template)
      authentication.app_auth_template = createAuthSectionFromTemplate(template.app_auth_template);
    if (authentication.type === 'oauth2_authorization_code') {
      authentication.app_auth_template = ensureRedirectUriField(
        authentication.app_auth_template || { fields: [] },
      );
    }
    return authentication;
  };

  const authMethods = computed<PluginDesignOption[]>(() => {
    const methods = authTemplateList.value.map((item) => ({
      label: authTypeLabelMap.value[item.auth_type] || item.auth_type,
      value: item.auth_type,
    }));
    const result = [{ label: '无', value: 'none' }, ...methods];
    const selectedAuthMethod = currentAuthentication.value.type;
    if (selectedAuthMethod && !result.some((item) => item.value === selectedAuthMethod)) {
      // 接口模板未返回当前认证类型时，补充回显选项，避免选择框显示为空。
      result.push({
        label: authTypeLabelMap.value[selectedAuthMethod] || selectedAuthMethod,
        value: selectedAuthMethod,
      });
    }
    return result;
  });

  const createFunctionFromTrigger = (trigger: PluginTrigger): PluginDesignFunction => {
    const runtime = normalizeFormRuntime(trigger.runtime);
    // 新接口以 id 替代旧 _id，缺失时使用 functionKey 保证设计器函数标识稳定。
    const functionId = String(trigger.id || trigger.functionKey);
    const nextFunction = createFunction(functionId, trigger.functionName, runtime, 'backend');
    // 保留后端函数唯一标识，调试接口会使用该字段定位待执行函数。
    nextFunction.functionKey = trigger.functionKey;
    nextFunction.functionDescription = trigger.functionDescription || '';
    nextFunction.seq = typeof trigger.seq === 'number' ? trigger.seq : 0;
    nextFunction.fields = createFieldsFromTriggerTemplate(trigger.requestParameter?.fields || []);
    nextFunction.responseParams = createResponseFieldsFromTrigger(
      trigger.responseParameter?.fields || [],
    );
    nextFunction.code = trigger.sourceCode || createDefaultCode(runtime);
    return nextFunction;
  };

  const createFunctionFromApplet = (applet: PluginApplet): PluginDesignFunction => {
    const runtime = PluginRuntime.NodeJS;
    // applet 是前端扩展函数，接口暂不返回运行时，沿用新增前端函数的默认 nodejs 运行时。
    // 新接口以 id 替代旧 _id，缺失时使用 functionKey 保证设计器函数标识稳定。
    const functionId = String(applet.id || applet.functionKey);
    const nextFunction = createFunction(functionId, applet.functionName, runtime, 'frontend');
    // 前端扩展同样使用接口返回的 functionKey，避免误用数据库 id。
    nextFunction.functionKey = applet.functionKey;
    nextFunction.functionDescription = applet.functionDescription || '';
    nextFunction.seq = typeof applet.seq === 'number' ? applet.seq : 0;
    nextFunction.fields = createFieldsFromTriggerTemplate(applet.requestParameter?.fields || []);
    nextFunction.responseParams = createResponseFieldsFromTrigger(
      applet.responseParameter?.fields || [],
    );
    nextFunction.code = applet.sourceCode || createDefaultCode(runtime);
    return nextFunction;
  };

  return {
    authMethods,
    createAuthenticationByMethod,
    createFunctionFromApplet,
    createFunctionFromTrigger,
  };
};
