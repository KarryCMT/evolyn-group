import type {
  IntelligentFieldOption,
  IntelligentFieldValueKind,
} from '@evolyn.do/intelligent';
import type { FormItem, FormWidgetType } from '@evolyn.do/form/schema';
import type { FormSchemaDocument } from '~/types';

const VALUE_KINDS: Readonly<Partial<Record<FormWidgetType, IntelligentFieldValueKind>>> = {
  text: 'text',
  textarea: 'text',
  phone: 'text',
  number: 'number',
  decimal: 'number',
  money: 'number',
  percent: 'number',
  datetime: 'date',
  radiogroup: 'choice',
  combo: 'choice',
  checkboxgroup: 'multi-choice',
  combocheck: 'multi-choice',
  user: 'member',
  usergroup: 'members',
  dept: 'department',
  deptgroup: 'departments',
  address: 'address',
};

function projectField(item: FormItem): IntelligentFieldOption | null {
  const valueKind = VALUE_KINDS[item.widget.type];
  if (!valueKind) return null;
  const choices =
    'options' in item.widget && Array.isArray(item.widget.options)
      ? item.widget.options.map((option) => ({ label: option.label, value: option.value }))
      : undefined;
  return {
    fieldId: item.widget.fieldId || item.widget.widgetName,
    widgetName: item.widget.widgetName,
    label: item.label || item.widget.widgetName,
    widgetType: item.widget.type,
    valueKind,
    required: item.widget.allowBlank === false,
    ...(choices ? { choices } : {}),
  };
}

/** 将表单协议投影为智能助手可写字段，派生字段由其运行时自行计算。 */
export function projectIntelligentFormFields(
  document: FormSchemaDocument,
): IntelligentFieldOption[] {
  const formulaTargets = new Set(
    document.content.fieldFormulas.flatMap((formula) =>
      formula.enabled ? [formula.targetFieldId] : [],
    ),
  );
  return document.content.items.flatMap((item) => {
    if (
      formulaTargets.has(item.widget.widgetName) ||
      (item.widget.fieldId && formulaTargets.has(item.widget.fieldId))
    ) {
      return [];
    }
    const field = projectField(item);
    return field ? [field] : [];
  });
}

/** 记录信封中的系统字段可以作为下游节点来源，但不能作为目标表单字段。 */
export function intelligentSystemFields(): IntelligentFieldOption[] {
  return [
    {
      fieldId: 'sys_submitted_at',
      widgetName: 'sys.submittedAt',
      label: '提交时间',
      widgetType: 'system-datetime',
      valueKind: 'date',
      required: false,
    },
    {
      fieldId: 'sys_updated_at',
      widgetName: 'sys.updatedAt',
      label: '更新时间',
      widgetType: 'system-datetime',
      valueKind: 'date',
      required: false,
    },
    {
      fieldId: 'sys_submitted_by',
      widgetName: 'sys.submittedBy',
      label: '提交人',
      widgetType: 'system-member',
      valueKind: 'member',
      required: false,
    },
    {
      fieldId: 'sys_updated_by',
      widgetName: 'sys.updatedBy',
      label: '最后更新人',
      widgetType: 'system-member',
      valueKind: 'member',
      required: false,
    },
  ];
}
