import { computed, ref } from 'vue';
import {
  copyWidgetItem,
  createWidgetItem,
  FORM_LAYOUT_LINE_WIDTH,
  generateLayoutName,
  generateTabName,
} from '../schema/dictionary';
import { isLayoutWidgetType } from '../schema/codec';
import type {
  FormItem,
  FormLayoutMode,
  FormMultitabLayout,
  FormSchemaDocument,
  FormWidgetType,
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
    content: { type: 'form', layout: 'normal', items: [], layout_fields: [], field_layout: [] },
  };
}

/**
 * 目标协议设计器编辑状态（P1）：页面唯一持有 content.items，
 * 画布拖拽直接变更 items 数组（vuedraggable :list 就地排序），
 * 增删/复制经本 hook 落盘前动作统一收口；持久化由页面接草稿接口。
 */
export function useFormSchemaEditor(initial?: FormSchemaDocument) {
  const document = ref<FormSchemaDocument>(initial ?? createEmptyFormSchemaDocument());
  const selectedKey = ref('');

  const items = computed(() => document.value.content.items);
  const layouts = computed(() => document.value.content.layout_fields);
  // items 可能瞬时混有素材面板拖入的临时对象（仅 paletteType 标记、无 widget，
  // 会在 add 事件内被真实字段项替换）；所有遍历必须经 widgetOf 收窄，避免
  // undefined.widgetName 在响应式重算路径上抛错。
  const widgetOf = (item: FormItem) => (item as FormItem & Partial<FormSchemaPaletteDrag>).widget;
  const selectedItem = computed(() => {
    const nested = parseSubformSelection(selectedKey.value);
    if (!nested) {
      return items.value.find((item) => widgetOf(item)?.widgetName === selectedKey.value);
    }
    return subformWidgetOf(nested.parentKey)?.items.find(
      (item) => item.widget.widgetName === nested.childKey,
    );
  });
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

  function removeItem(key: string): void {
    const index = items.value.findIndex((item) => widgetOf(item)?.widgetName === key);
    if (index === -1) return;
    items.value.splice(index, 1);
    removeReferenceEverywhere(key);
    if (selectedKey.value === key) {
      // 相邻项也可能是瞬时拖入的临时对象，经 widgetOf 收窄后再取键。
      const neighbor = items.value[index] ?? items.value[index - 1];
      selectedKey.value =
        neighbor && widgetOf(neighbor)?.widgetName ? widgetOf(neighbor)!.widgetName : '';
    }
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
    selectedKey.value = nextKey;
  }

  /** 属性面板提交完整字段副本；替换定义同时保持布局引用与选中键一致。 */
  function updateSelectedItem(next: FormItem): void {
    const current = selectedItem.value;
    if (!current) return;
    const nested = parseSubformSelection(selectedKey.value);
    if (nested) {
      const subform = subformWidgetOf(nested.parentKey);
      const index = subform?.items.findIndex(
        (item) => item.widget.widgetName === current.widget.widgetName,
      );
      if (!subform || index === undefined || index < 0) return;
      subform.items.splice(index, 1, next);
      selectedKey.value = subformSelectionKey(nested.parentKey, next.widget.widgetName);
      return;
    }
    const index = items.value.findIndex(
      (item) => widgetOf(item)?.widgetName === current.widget.widgetName,
    );
    if (index < 0) return;
    const previousKey = current.widget.widgetName;
    // 子表单是横向明细容器，属性回写不能改变其整行宽度约束。
    if (next.widget.type === 'subform') next.lineWidth = 12;
    items.value.splice(index, 1, next);
    if (next.widget.widgetName !== previousKey) {
      replaceReferenceEverywhere(previousKey, next.widget.widgetName);
      selectedKey.value = next.widget.widgetName;
    }
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

  /** 删除标签页只解散容器：其中字段移动到整个标签页组之后，绝不删除字段定义。 */
  function removeTab(layoutName: string, tabName: string): void {
    const layout = layoutByName(layoutName);
    if (!layout) return;
    const tabIndex = layout.container.findIndex((tab) => tab.name === tabName);
    if (tabIndex < 0) return;
    const [tab] = layout.container.splice(tabIndex, 1);
    moveReferencesAfterLayout(layoutName, tab.field_layout);
    if (layout.container.length === 0) removeMultitab(layoutName);
  }

  /** 删除标签页组时按标签页顺序展开字段，保证用户字段与填写数据均不丢失。 */
  function removeMultitab(layoutName: string): void {
    const layoutIndex = layouts.value.findIndex((layout) => layout.name === layoutName);
    if (layoutIndex < 0) return;
    const [layout] = layouts.value.splice(layoutIndex, 1);
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
      const widget = widgetOf(item);
      if (widget) item.lineWidth = defaultLineWidth(widget.type, lineWidth);
    }
  }

  /** 复制标签页时同步复制其字段定义，避免同一字段引用出现在两个标签页。 */
  function duplicateTab(layoutName: string, tabName: string): void {
    const layout = layoutByName(layoutName);
    if (!layout || layout.container.length >= 20) return;
    const index = layout.container.findIndex((tab) => tab.name === tabName);
    if (index < 0) return;
    const source = layout.container[index];
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
    document.value = next;
    selectedKey.value = '';
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
  };
}
