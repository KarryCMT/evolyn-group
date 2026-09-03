import { computed, ref, type Ref } from 'vue';
import {
  copyWidgetItem,
  createFieldShowRule,
  createWidgetItem,
  FORM_LAYOUT_LINE_WIDTH,
  generateFieldShowRuleId,
  generateLayoutName,
  generateTabName,
} from '../schema/dictionary';
import { isLayoutWidgetType } from '../schema/codec';
import {
  isSubmitRuleEligibleType,
  normalizeWidgetSubmitRules,
} from '../schema/invisible-value-policy';
import type {
  FieldShowRule,
  FormItem,
  FormLayoutMode,
  FormMultitabLayout,
  FormSchemaDocument,
  FormWidgetType,
  SubmitRule,
  SubformWidget,
} from '../schema/types';
import { SUBFORM_ALLOWED_WIDGET_TYPES } from '../schema/types';
import type { FormSchemaPaletteDrag } from './palette';

export type FormLayoutTarget =
  | { type: 'top' }
  | { type: 'tab'; layoutName: string; tabName: string };

const SUBFORM_SELECTION_PREFIX = 'subform:';

/** 子字段允许与顶层重名，选中态必须携带父子两级键，不能只保存 child widgetName。 */
export function subformSelectionKey(parentKey: string, childKey: string): string {
  return `${SUBFORM_SELECTION_PREFIX}${parentKey}:${childKey}`;
}

export function parseSubformSelection(
  selection: string,
): { parentKey: string; childKey: string } | null {
  if (!selection.startsWith(SUBFORM_SELECTION_PREFIX)) return null;
  const [parentKey, childKey] = selection.slice(SUBFORM_SELECTION_PREFIX.length).split(':');
  return parentKey && childKey ? { parentKey, childKey } : null;
}

/** 空表单协议文档（新建/草稿初始化用）。 */
export function createEmptyFormSchemaDocument(): FormSchemaDocument {
  return {
    content: {
      type: 'form',
      layout: 'normal',
      items: [],
      layout_fields: [],
      field_layout: [],
      fieldShowRules: [],
      submitRule: 2,
      widget_submit_rules: {},
    },
  };
}

/**
 * 目标协议设计器编辑状态（P1）：页面唯一持有 content.items，
 * 画布拖拽直接变更 items 数组（vuedraggable :list 就地排序），
 * 增删/复制经本 hook 落盘前动作统一收口；持久化由页面接草稿接口。
 */
export function useFormSchemaEditor(initial?: FormSchemaDocument) {
  // as 收窄 ref 的 UnwrapRef 展开结果（递归协议类型会过度实例化 TS2589）；
  // 文档只含 JSON 安全值，不存在嵌套 ref，运行时行为与深解包完全一致。
  const document = ref(initial ?? createEmptyFormSchemaDocument()) as Ref<FormSchemaDocument>;
  const selectedKey = ref('');

  const items = computed(() => document.value.content.items);
  const layouts = computed(() => document.value.content.layout_fields);
  // v5 显隐规则 + v6 不可见字段赋值：防御性兜底旧文档缺键（正规读取路径经迁移器补齐）。
  normalizeContentKeys(document.value.content);
  const fieldShowRules = computed(() => document.value.content.fieldShowRules);
  const submitRule = computed(() => document.value.content.submitRule);
  const widgetSubmitRules = computed(() => document.value.content.widget_submit_rules);
  // items 可能瞬时混有素材面板拖入的临时对象（仅 paletteType 标记、无 widget，
  // 会在 add 事件内被真实字段项替换）；所有遍历必须经 widgetOf 收窄，避免
  // undefined.widgetName 在响应式重算路径上抛错。
  // 轻量结构收窄（避免 FormItem 与拖拽载荷的交叉联合在推断中过度展开）：
  // items 可能瞬时混有素材面板拖入的临时对象（仅 paletteType 标记、无 widget）。
  type ItemWithWidget = { widget?: { widgetName?: string; type?: string } };
  const widgetOf = (item: ItemWithWidget | null | undefined) => item?.widget;
  // 以独立函数承载查找并显式标注返回类型，避免与素材拖拽临时结构相交的
  // 联合类型在 computed 泛型推断中过度展开（TS2589）。
  function findSelectedItem(key: string): FormItem | undefined {
    const nested = parseSubformSelection(key);
    if (!nested) {
      for (const item of items.value) {
        if (widgetOf(item)?.widgetName === key) return item;
      }
      return undefined;
    }
    const children = subformWidgetOf(nested.parentKey)?.items ?? [];
    for (const item of children) {
      if (item.widget.widgetName === nested.childKey) return item;
    }
    return undefined;
  }
  const selectedItem = computed((): FormItem | undefined => findSelectedItem(selectedKey.value));
  const selectedLayout = computed(() =>
    layouts.value.find((layout) => layout.name === selectedKey.value),
  );
  const pendingPaletteFieldKeys = new Set<string>();
  let paletteCleanupQueued = false;

  /** 新增字段（素材点击或拖拽落点替换临时对象）；index<0 追加到末尾。 */
  function addItem(
    type: FormWidgetType,
    index = -1,
    target: FormLayoutTarget = { type: 'top' },
  ): FormItem {
    const item = createWidgetItem(type);
    item.lineWidth = defaultLineWidth(type);
    items.value.push(item);
    const references = referencesOf(target);
    if (references)
      references.splice(normalizeInsertIndex(index, references.length), 0, item.widget.widgetName);
    selectedKey.value = item.widget.widgetName;
    return item;
  }

  /** 复制字段：深拷贝并换新 widgetName，插入原字段之后。 */
  function copyItem(item: FormItem): void {
    const next = copyWidgetItem(item);
    if (next.widget.type === 'subform') next.lineWidth = 12;
    const itemIndex = items.value.findIndex(
      (entry) => widgetOf(entry)?.widgetName === item.widget.widgetName,
    );
    if (itemIndex === -1) return;
    items.value.splice(itemIndex + 1, 0, next);
    const placement = findPlacement(item.widget.widgetName);
    if (placement) placement.references.splice(placement.index + 1, 0, next.widget.widgetName);
    selectedKey.value = next.widget.widgetName;
  }

  /** 子表单新增字段：只接受协议白名单，子字段始终占子表格的一列。 */
  function addSubformItem(parentKey: string, type: FormWidgetType, index = -1): FormItem | null {
    const subform = subformWidgetOf(parentKey);
    if (!subform || !SUBFORM_ALLOWED_WIDGET_TYPES.includes(type) || subform.items.length >= 200) {
      return null;
    }
    const item = createWidgetItem(type);
    item.lineWidth = 12;
    subform.items.splice(normalizeInsertIndex(index, subform.items.length), 0, item);
    selectedKey.value = subformSelectionKey(parentKey, item.widget.widgetName);
    return item;
  }

  function copySubformItem(parentKey: string, childKey: string): void {
    const subform = subformWidgetOf(parentKey);
    if (!subform || subform.items.length >= 200) return;
    const index = subform.items.findIndex((item) => item.widget.widgetName === childKey);
    if (index < 0) return;
    const copied = copyWidgetItem(subform.items[index]!);
    copied.lineWidth = 12;
    subform.items.splice(index + 1, 0, copied);
    selectedKey.value = subformSelectionKey(parentKey, copied.widget.widgetName);
  }

  function removeSubformItem(parentKey: string, childKey: string): void {
    const subform = subformWidgetOf(parentKey);
    if (!subform) return;
    const index = subform.items.findIndex((item) => item.widget.widgetName === childKey);
    if (index < 0) return;
    subform.items.splice(index, 1);
    if (selectedKey.value !== subformSelectionKey(parentKey, childKey)) return;
    const neighbor = subform.items[index] ?? subform.items[index - 1];
    selectedKey.value = neighbor
      ? subformSelectionKey(parentKey, neighbor.widget.widgetName)
      : parentKey;
  }

  /**
   * 子表单拖拽数组可瞬时混入素材载荷；统一在状态层转换并过滤禁用类型，
   * 保证协议对象从不落入 paletteType 临时结构。
   */
  function replaceSubformItems(parentKey: string, entries: unknown[]): void {
    const subform = subformWidgetOf(parentKey);
    if (!subform) return;
    const next = entries.flatMap<FormItem>((entry) => {
      if (isFormItem(entry)) return [entry];
      const type = (entry as Partial<FormSchemaPaletteDrag> | null)?.paletteType as
        | FormWidgetType
        | undefined;
      if (!type || !SUBFORM_ALLOWED_WIDGET_TYPES.includes(type)) return [];
      const item = createWidgetItem(type);
      item.lineWidth = 12;
      selectedKey.value = subformSelectionKey(parentKey, item.widget.widgetName);
      return [item];
    });
    subform.items.splice(0, subform.items.length, ...next.slice(0, 200));
  }

  /** 删除顶层字段：被显隐规则或特殊赋值规则引用时阻断，直至规则被处理（§5.2/v6 §3.2）。 */
  function removeItem(key: string): boolean {
    if (fieldShowRulesReferencing(key).length > 0) return false;
    if (widgetSubmitRulesOf(key) !== undefined) return false;
    const index = items.value.findIndex((item) => widgetOf(item)?.widgetName === key);
    if (index === -1) return false;
    items.value.splice(index, 1);
    removeReferenceEverywhere(key);
    if (selectedKey.value === key) {
      // 相邻项也可能是瞬时拖入的临时对象，经 widgetOf 收窄后再取键。
      const neighbor = items.value[index] ?? items.value[index - 1];
      const neighborKey = widgetOf(neighbor)?.widgetName;
      selectedKey.value = neighborKey ?? '';
    }
    return true;
  }

  function selectItem(key: string): void {
    selectedKey.value = key;
  }

  function selectSubformItem(parentKey: string, childKey: string): void {
    selectedKey.value = subformSelectionKey(parentKey, childKey);
  }

  function selectLayout(name: string): void {
    selectedKey.value = name;
  }

  /** widgetName 就地重命名（属性面板）：同步选中键，唯一性由保存校验兜底。 */
  function renameItemKey(nextKey: string): void {
    const item = selectedItem.value;
    if (!item) return;
    const previousKey = item.widget.widgetName;
    item.widget.widgetName = nextKey;
    const nested = parseSubformSelection(selectedKey.value);
    if (nested) {
      selectedKey.value = subformSelectionKey(nested.parentKey, nextKey);
      return;
    }
    replaceReferenceEverywhere(previousKey, nextKey);
    // 显隐规则以 widgetName 引用条件源与目标：改名必须原子同步（设计方案 §5.2）。
    replaceFieldShowRuleReferences(previousKey, nextKey);
    // 特殊字段赋值规则同样以 widgetName 为键：改名原子同步（v6 §3.2）。
    replaceWidgetSubmitRuleReferences(previousKey, nextKey);
    selectedKey.value = nextKey;
  }

  /** 属性面板提交完整字段副本；替换定义同时保持布局引用与选中键一致。
   * 类型变更或转为静态隐藏的字段被显隐规则引用时阻断（§5.2：条件指纹
   * 失配与不可达语义必须在规则侧先行处理）。 */
  function updateSelectedItem(next: FormItem): boolean {
    const current = selectedItem.value;
    if (!current) return false;
    const nested = parseSubformSelection(selectedKey.value);
    if (nested) {
      const subform = subformWidgetOf(nested.parentKey);
      const index = subform?.items.findIndex(
        (item) => item.widget.widgetName === current.widget.widgetName,
      );
      if (!subform || index === undefined || index < 0) return false;
      subform.items.splice(index, 1, next);
      selectedKey.value = subformSelectionKey(nested.parentKey, next.widget.widgetName);
      return true;
    }
    const index = items.value.findIndex(
      (item) => widgetOf(item)?.widgetName === current.widget.widgetName,
    );
    if (index < 0) return false;
    const previousKey = current.widget.widgetName;
    const typeChanged = next.widget.type !== current.widget.type;
    const hiddenTurn = current.widget.visible && !next.widget.visible;
    if ((typeChanged || hiddenTurn) && fieldShowRulesReferencing(previousKey).length > 0) {
      return false;
    }
    // 特殊赋值规则依赖字段的值语义：类型变更为不可处理控件时阻断（v6 §3.2）。
    if (
      typeChanged &&
      widgetSubmitRulesOf(previousKey) !== undefined &&
      !isSubmitRuleEligibleType(next.widget.type)
    ) {
      return false;
    }
    // 子表单是横向明细容器，属性回写不能改变其整行宽度约束。
    if (next.widget.type === 'subform') next.lineWidth = 12;
    items.value.splice(index, 1, next);
    if (next.widget.widgetName !== previousKey) {
      replaceReferenceEverywhere(previousKey, next.widget.widgetName);
      replaceFieldShowRuleReferences(previousKey, next.widget.widgetName);
      replaceWidgetSubmitRuleReferences(previousKey, next.widget.widgetName);
      selectedKey.value = next.widget.widgetName;
    }
    return true;
  }

  /** 新增标签页布局；布局是引用容器，不进入 content.items。 */
  function addMultitab(): FormMultitabLayout {
    const layout: FormMultitabLayout = {
      name: generateLayoutName(),
      type: 'multitab',
      tabStyle: 'style2',
      container: [createTab('标签页1'), createTab('标签页2')],
    };
    layouts.value.push(layout);
    document.value.content.field_layout.push(layout.name);
    selectedKey.value = layout.name;
    return layout;
  }

  function addTab(layoutName: string): void {
    const layout = layoutByName(layoutName);
    if (!layout || layout.container.length >= 20) return;
    layout.container.push(createTab(`标签页${layout.container.length + 1}`));
  }

  /** 引用指定字段（条件源或目标）的显隐规则清单：删除/变更前的依赖提示。 */
  function fieldShowRulesReferencing(key: string): FieldShowRule[] {
    return document.value.content.fieldShowRules.filter(
      (rule) =>
        rule.fields.includes(key) || rule.filter.cond.some((condition) => condition.field === key),
    );
  }

  /** 就地替换规则中的字段引用（条件源与目标），与布局引用同步维护。 */
  function replaceFieldShowRuleReferences(previousKey: string, nextKey: string): void {
    for (const rule of document.value.content.fieldShowRules) {
      for (const condition of rule.filter.cond) {
        if (condition.field === previousKey) condition.field = nextKey;
      }
      rule.fields = rule.fields.map((field) => (field === previousKey ? nextKey : field));
    }
  }

  /** 删除标签页只解散容器：其中字段移动到整个标签页组之后，绝不删除字段定义。 */
  function removeTab(layoutName: string, tabName: string): void {
    const layout = layoutByName(layoutName);
    if (!layout) return;
    const tabIndex = layout.container.findIndex((tab) => tab.name === tabName);
    if (tabIndex < 0) return;
    const removed = layout.container.splice(tabIndex, 1);
    const tab = removed[0];
    if (!tab) return;
    moveReferencesAfterLayout(layoutName, tab.field_layout);
    if (layout.container.length === 0) removeMultitab(layoutName);
  }

  /** 删除标签页组时按标签页顺序展开字段，保证用户字段与填写数据均不丢失。 */
  function removeMultitab(layoutName: string): void {
    const layoutIndex = layouts.value.findIndex((layout) => layout.name === layoutName);
    if (layoutIndex < 0) return;
    const removed = layouts.value.splice(layoutIndex, 1);
    const layout = removed[0];
    if (!layout) return;
    const flattened = layout.container.flatMap((tab) => tab.field_layout);
    const topIndex = document.value.content.field_layout.indexOf(layoutName);
    if (topIndex >= 0) {
      document.value.content.field_layout.splice(topIndex, 1, ...flattened);
    } else {
      document.value.content.field_layout.push(...flattened);
    }
    if (selectedKey.value === layoutName) selectedKey.value = flattened[0] ?? '';
  }

  function renameTab(layoutName: string, tabName: string, title: string): void {
    const tab = layoutByName(layoutName)?.container.find((entry) => entry.name === tabName);
    if (tab) tab.title = title;
  }

  function setTabStyle(layoutName: string, style: FormMultitabLayout['tabStyle']): void {
    const layout = layoutByName(layoutName);
    if (layout) layout.tabStyle = style;
  }

  /** 切换表单默认列布局时同步普通字段；布局组件与子表单固定占满 12 栅格。 */
  function setFormLayout(layout: FormLayoutMode): void {
    document.value.content.layout = layout;
    const lineWidth = FORM_LAYOUT_LINE_WIDTH[layout];
    for (const item of items.value) {
      const widgetType = widgetOf(item)?.type;
      // 瞬时拖拽载荷已被判空过滤；未知类型走 defaultLineWidth 的兜底分支。
      if (widgetType) item.lineWidth = defaultLineWidth(widgetType as FormWidgetType, lineWidth);
    }
  }

  /** 复制标签页时同步复制其字段定义，避免同一字段引用出现在两个标签页。 */
  function duplicateTab(layoutName: string, tabName: string): void {
    const layout = layoutByName(layoutName);
    if (!layout || layout.container.length >= 20) return;
    const index = layout.container.findIndex((tab) => tab.name === tabName);
    if (index < 0) return;
    const source = layout.container[index];
    if (!source) return;
    const fieldLayout = source.field_layout.flatMap((fieldKey) => {
      const item = items.value.find((entry) => widgetOf(entry)?.widgetName === fieldKey);
      if (!item) return [];
      const copied = copyWidgetItem(item);
      items.value.push(copied);
      return [copied.widget.widgetName];
    });
    layout.container.splice(index + 1, 0, {
      name: generateTabName(),
      title: `${source.title} copy`,
      type: 'tab',
      field_layout: fieldLayout,
    });
  }

  /** 标签页顺序只由稳定键列表驱动，拒绝缺失、重复或外部键。 */
  function reorderTabs(layoutName: string, tabNames: string[]): void {
    const layout = layoutByName(layoutName);
    if (!layout || tabNames.length !== layout.container.length) return;
    const tabMap = new Map(layout.container.map((tab) => [tab.name, tab]));
    if (new Set(tabNames).size !== tabNames.length || tabNames.some((name) => !tabMap.has(name))) {
      return;
    }
    layout.container.splice(
      0,
      layout.container.length,
      ...tabNames.map((name) => tabMap.get(name)!),
    );
  }

  /**
   * 拖拽列表只传递引用；素材面板临时标记在这里转换为真实字段并写入 items，
   * 防止设计组件直接维护协议的双事实源。
   */
  function replaceReferences(target: FormLayoutTarget, entries: unknown[]): void {
    const references = referencesOf(target);
    if (!references) return;
    const next = entries.flatMap<string>((entry) => {
      if (typeof entry === 'string') return [entry];
      const type = (entry as Partial<FormSchemaPaletteDrag> | null)?.paletteType;
      if (typeof type !== 'string') return [];
      if (type === 'multitab') {
        return target.type === 'top' ? [addMultitab().name] : [];
      }
      const item = createWidgetItem(type as FormWidgetType);
      item.lineWidth = defaultLineWidth(item.widget.type);
      items.value.push(item);
      // 嵌套 Sortable 会让外层列表短暂收到同一次素材克隆；记录本轮新字段，
      // 待所有同步拖拽事件结束后清理没有最终布局引用的瞬时顶层定义。
      pendingPaletteFieldKeys.add(item.widget.widgetName);
      schedulePaletteOrphanCleanup();
      selectedKey.value = item.widget.widgetName;
      return [item.widget.widgetName];
    });
    references.splice(0, references.length, ...next);
  }

  /** 整体替换文档（草稿加载/保存回读）。 */
  function replaceDocument(next: FormSchemaDocument): void {
    normalizeContentKeys(next.content);
    document.value = next;
    selectedKey.value = '';
  }

  // ---- v5 字段显隐规则管理（列表顺序仅影响展示，不参与运行结果） ----

  /** 新建空白规则（id 已生成，条件与目标由编辑器补全后落盘）。 */
  function createFieldShowRuleDraft(): FieldShowRule {
    return createFieldShowRule();
  }

  /** 保存（新增或按 id 整体替换）单条规则；深拷贝切断编辑器草稿引用。 */
  function saveFieldShowRule(rule: FieldShowRule): void {
    const snapshot = JSON.parse(JSON.stringify(rule)) as FieldShowRule;
    const rules = document.value.content.fieldShowRules;
    const index = rules.findIndex((entry) => entry.id === rule.id);
    if (index >= 0) {
      rules.splice(index, 1, snapshot);
    } else {
      rules.push(snapshot);
    }
  }

  /** 复制规则：换新 id，其余内容原样复制并紧随其后插入。 */
  function duplicateFieldShowRule(ruleId: string): FieldShowRule | null {
    const rules = document.value.content.fieldShowRules;
    const index = rules.findIndex((entry) => entry.id === ruleId);
    if (index < 0) return null;
    const copied = JSON.parse(JSON.stringify(rules[index])) as FieldShowRule;
    copied.id = generateFieldShowRuleId();
    rules.splice(index + 1, 0, copied);
    return copied;
  }

  function removeFieldShowRule(ruleId: string): void {
    const rules = document.value.content.fieldShowRules;
    const index = rules.findIndex((entry) => entry.id === ruleId);
    if (index >= 0) rules.splice(index, 1);
  }

  /** 列表重排：仅改善阅读与审计 diff，拒绝缺失或外部键。 */
  function reorderFieldShowRules(ruleIds: string[]): void {
    const rules = document.value.content.fieldShowRules;
    if (ruleIds.length !== rules.length) return;
    const ruleMap = new Map(rules.map((rule) => [rule.id, rule]));
    if (new Set(ruleIds).size !== ruleIds.length || ruleIds.some((id) => !ruleMap.has(id))) {
      return;
    }
    rules.splice(0, rules.length, ...ruleIds.map((id) => ruleMap.get(id)!));
  }

  // ---- v6 不可见字段赋值管理（§5.1：默认策略与特殊规则同一属性页配置） ----

  /** 读取字段的特殊赋值规则（未配置返回 undefined）。 */
  function widgetSubmitRulesOf(key: string): SubmitRule | undefined {
    return document.value.content.widget_submit_rules[key];
  }

  /**
   * 切换表单默认策略：自动移除与新默认相同的冗余映射（§3.1——校验器对
   * 冗余项拒绝保存，设计器在切换侧先行归一化）。
   */
  function setSubmitRule(rule: SubmitRule): void {
    const content = document.value.content;
    content.submitRule = rule;
    content.widget_submit_rules = normalizeWidgetSubmitRules(content.widget_submit_rules, rule);
  }

  /** 整体替换特殊规则映射（对话框「确定」提交归一化结果）。 */
  function applyWidgetSubmitRules(rules: Record<string, SubmitRule>): void {
    document.value.content.widget_submit_rules = normalizeWidgetSubmitRules(
      rules,
      document.value.content.submitRule,
    );
  }

  /** 就地替换特殊规则键（字段改名时原子同步）。 */
  function replaceWidgetSubmitRuleReferences(previousKey: string, nextKey: string): void {
    const rules = document.value.content.widget_submit_rules;
    if (!(previousKey in rules)) return;
    const value = rules[previousKey]!;
    delete rules[previousKey];
    rules[nextKey] = value;
  }

  function layoutByName(name: string): FormMultitabLayout | undefined {
    return layouts.value.find((layout) => layout.name === name);
  }

  function subformWidgetOf(parentKey: string): SubformWidget | undefined {
    const item = items.value.find((entry) => widgetOf(entry)?.widgetName === parentKey);
    return item?.widget.type === 'subform' ? item.widget : undefined;
  }

  function isFormItem(entry: unknown): entry is FormItem {
    return Boolean(
      entry &&
      typeof entry === 'object' &&
      'widget' in entry &&
      (entry as FormItem).widget?.widgetName,
    );
  }

  function referencesOf(target: FormLayoutTarget): string[] | null {
    if (target.type === 'top') return document.value.content.field_layout;
    return (
      layoutByName(target.layoutName)?.container.find((tab) => tab.name === target.tabName)
        ?.field_layout ?? null
    );
  }

  function findPlacement(key: string): { references: string[]; index: number } | null {
    const topIndex = document.value.content.field_layout.indexOf(key);
    if (topIndex >= 0) return { references: document.value.content.field_layout, index: topIndex };
    for (const layout of layouts.value) {
      for (const tab of layout.container) {
        const index = tab.field_layout.indexOf(key);
        if (index >= 0) return { references: tab.field_layout, index };
      }
    }
    return null;
  }

  function removeReferenceEverywhere(key: string): void {
    const lists = [
      document.value.content.field_layout,
      ...layouts.value.flatMap((layout) => layout.container.map((tab) => tab.field_layout)),
    ];
    for (const references of lists) {
      let index = references.indexOf(key);
      while (index >= 0) {
        references.splice(index, 1);
        index = references.indexOf(key);
      }
    }
  }

  function replaceReferenceEverywhere(previousKey: string, nextKey: string): void {
    const lists = [
      document.value.content.field_layout,
      ...layouts.value.flatMap((layout) => layout.container.map((tab) => tab.field_layout)),
    ];
    for (const references of lists) {
      references.forEach((reference, index) => {
        if (reference === previousKey) references[index] = nextKey;
      });
    }
  }

  function moveReferencesAfterLayout(layoutName: string, references: string[]): void {
    if (references.length === 0) return;
    const index = document.value.content.field_layout.indexOf(layoutName);
    document.value.content.field_layout.splice(
      index >= 0 ? index + 1 : document.value.content.field_layout.length,
      0,
      ...references,
    );
  }

  /** 只回收本轮素材拖拽产生的瞬时孤儿，不影响跨顶层/标签页移动的既有字段。 */
  function schedulePaletteOrphanCleanup(): void {
    if (paletteCleanupQueued) return;
    paletteCleanupQueued = true;
    queueMicrotask(() => {
      paletteCleanupQueued = false;
      const referenced = new Set([
        ...document.value.content.field_layout,
        ...layouts.value.flatMap((layout) => layout.container.flatMap((tab) => tab.field_layout)),
      ]);
      const removedKeys = new Set<string>();
      const retained = items.value.filter((item) => {
        const key = widgetOf(item)?.widgetName;
        const keep = !key || !pendingPaletteFieldKeys.has(key) || referenced.has(key);
        if (!keep && key) removedKeys.add(key);
        return keep;
      });
      items.value.splice(0, items.value.length, ...retained);
      if (removedKeys.has(selectedKey.value)) {
        selectedKey.value = document.value.content.field_layout[0] ?? '';
      }
      pendingPaletteFieldKeys.clear();
    });
  }

  function createTab(title: string) {
    return { name: generateTabName(), title, type: 'tab' as const, field_layout: [] };
  }

  function normalizeInsertIndex(index: number, length: number): number {
    return index >= 0 && index <= length ? index : length;
  }

  function defaultLineWidth(
    type: FormWidgetType,
    layoutWidth = FORM_LAYOUT_LINE_WIDTH[document.value.content.layout],
  ): number {
    return isLayoutWidgetType(type) || type === 'subform' ? 12 : layoutWidth;
  }

  return {
    document,
    items,
    layouts,
    fieldShowRules,
    submitRule,
    widgetSubmitRules,
    selectedKey,
    selectedItem,
    selectedLayout,
    addItem,
    copyItem,
    addSubformItem,
    copySubformItem,
    removeSubformItem,
    replaceSubformItems,
    removeItem,
    selectItem,
    selectSubformItem,
    selectLayout,
    renameItemKey,
    updateSelectedItem,
    fieldShowRulesReferencing,
    widgetSubmitRulesOf,
    setSubmitRule,
    applyWidgetSubmitRules,
    addMultitab,
    addTab,
    removeTab,
    removeMultitab,
    renameTab,
    setTabStyle,
    setFormLayout,
    duplicateTab,
    reorderTabs,
    replaceReferences,
    replaceDocument,
    createFieldShowRuleDraft,
    saveFieldShowRule,
    duplicateFieldShowRule,
    removeFieldShowRule,
    reorderFieldShowRules,
  };
}

/** 防御性补齐 v5/v6 表单级键（正规读取路径经迁移器补齐，这里兜底直载旧文档）。 */
function normalizeContentKeys(content: FormSchemaDocument['content']): void {
  if (!Array.isArray(content.fieldShowRules)) content.fieldShowRules = [];
  if (!isSubmitRuleValue(content.submitRule)) content.submitRule = 2;
  if (!isPlainRecord(content.widget_submit_rules)) content.widget_submit_rules = {};
}

function isSubmitRuleValue(value: unknown): value is SubmitRule {
  return typeof value === 'number' && Number.isInteger(value) && value >= 1 && value <= 3;
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
