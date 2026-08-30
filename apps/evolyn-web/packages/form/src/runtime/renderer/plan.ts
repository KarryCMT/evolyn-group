import type { FormItem, FormSchemaDocument, FormTabStyle } from '../../schema/types';

/**
 * 预编译渲染计划：FormRenderer 只遍历该静态序列，不读取全部字段值，也不在模板内
 * 做过滤、排序或规则计算。当前为平铺单区块；后续阶段按显隐规则编译多区块延迟挂载。
 */
export interface FormRenderSection {
  key: string;
  nodes: readonly FormRenderNode[];
}

export interface FormRenderFieldNode {
  type: 'field';
  key: string;
  item: FormItem;
}

export interface FormRenderTab {
  key: string;
  title: string;
  fields: readonly FormRenderFieldNode[];
}

export interface FormRenderMultitabNode {
  type: 'multitab';
  key: string;
  tabStyle: FormTabStyle;
  tabs: readonly FormRenderTab[];
}

export type FormRenderNode = FormRenderFieldNode | FormRenderMultitabNode;

export interface FormRenderPlan {
  sections: readonly FormRenderSection[];
}

export function buildRenderPlan(schema: FormSchemaDocument): FormRenderPlan {
  const itemMap = new Map(schema.content.items.map((item) => [item.widget.widgetName, item]));
  const layoutMap = new Map(schema.content.layout_fields.map((layout) => [layout.name, layout]));
  const fieldNode = (reference: string): FormRenderFieldNode | null => {
    const item = itemMap.get(reference);
    return item ? { type: 'field', key: reference, item } : null;
  };
  const nodes = schema.content.field_layout.flatMap<FormRenderNode>((reference) => {
    const field = fieldNode(reference);
    if (field) return [field];
    const layout = layoutMap.get(reference);
    if (!layout) return [];
    return [
      {
        type: 'multitab',
        key: layout.name,
        tabStyle: layout.tabStyle,
        tabs: layout.container.map((tab) => ({
          key: tab.name,
          title: tab.title,
          fields: tab.field_layout.flatMap((fieldReference) => {
            const node = fieldNode(fieldReference);
            return node ? [node] : [];
          }),
        })),
      },
    ];
  });
  return {
    sections: [
      {
        key: 'main',
        nodes,
      },
    ],
  };
}
