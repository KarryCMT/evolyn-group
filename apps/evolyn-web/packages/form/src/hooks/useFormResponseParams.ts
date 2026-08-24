import { type ComputedRef } from 'vue';
import type { PluginDesignFunction, PluginDesignResponseField } from '../types';

/**
 * 管理当前函数返回参数的新增与删除操作。
 * @param currentFunction 当前侧栏选中的函数。
 * @param t 全局多语言翻译函数。
 */
export const useFormResponseParams = (currentFunction: ComputedRef<PluginDesignFunction>) => {
  /** 创建尚未持久化的普通返回字段，字段 ID 保存后由后端生成。 */
  const createResponseField = (fieldKey: string): PluginDesignResponseField => ({
    id: null,
    fieldKey,
    fieldLabel: '返回参数',
    widgetName: '',
    dataType: 'any',
    fieldConf: {},
  });

  const addResponseParam = () => {
    currentFunction.value.responseParams.push(createResponseField(`field_${Date.now()}`));
  };

  const addChildParam = (param: PluginDesignResponseField) => {
    // 子级入口仅对 vector 显示；此处再次校验，避免调用方误操作改变参数类型。
    if (param.dataType !== 'vector') return;
    if (!Array.isArray(param.fieldConf.fields)) param.fieldConf.fields = [];
    param.fieldConf.fields.push(createResponseField(`child_${Date.now()}`));
  };

  const removeParam = (param: PluginDesignResponseField) => {
    const index = currentFunction.value.responseParams.indexOf(param);
    if (index !== -1) currentFunction.value.responseParams.splice(index, 1);
  };

  const removeChildParam = (param: PluginDesignResponseField, child: PluginDesignResponseField) => {
    const fields = param.fieldConf.fields;
    if (!Array.isArray(fields)) return;
    const index = fields.indexOf(child);
    if (index !== -1) fields.splice(index, 1);
  };

  return {
    addChildParam,
    addResponseParam,
    removeChildParam,
    removeParam,
  };
};
