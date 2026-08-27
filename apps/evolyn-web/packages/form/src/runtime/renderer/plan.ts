import type { FormItem, FormSchemaDocument } from '../../schema/types';

/**
 * 预编译渲染计划：FormRenderer 只遍历该静态序列，不读取全部字段值，也不在模板内
 * 做过滤、排序或规则计算。当前为平铺单区块；后续阶段按显隐规则编译多区块延迟挂载。
 */
export interface FormRenderSection {
  key: string;
  items: readonly FormItem[];
}

export interface FormRenderPlan {
  sections: readonly FormRenderSection[];
}

export function buildRenderPlan(schema: FormSchemaDocument): FormRenderPlan {
  return {
    sections: [
      {
        key: 'main',
        items: schema.content.items,
      },
    ],
  };
}
