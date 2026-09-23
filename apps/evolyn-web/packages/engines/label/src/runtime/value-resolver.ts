import type { LabelRenderData, ValueSource } from '../schema/label-schema.js';

export interface ValueResolverOptions {
  evaluateExpression?: (expression: string, data: LabelRenderData) => unknown | Promise<unknown>;
}

/** 只解析 LabelSchema 中的绑定关系；字段权限与真实记录由服务端 Data/Permission Engine 提供。 */
export async function resolveValueSource(
  source: ValueSource,
  data: LabelRenderData,
  options: ValueResolverOptions = {},
): Promise<string> {
  let value: unknown;
  switch (source.type) {
    case 'static':
      value = source.value;
      break;
    case 'field':
      value = data.fields[source.fieldId];
      break;
    case 'system':
      value = data.system[source.key];
      break;
    case 'expression':
      if (!options.evaluateExpression) return '';
      value = await options.evaluateExpression(source.expression, data);
      break;
  }
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value);
  }
  return JSON.stringify(value);
}
