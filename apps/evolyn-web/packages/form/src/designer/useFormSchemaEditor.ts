import { computed, ref } from 'vue';
import { copyWidgetItem, createWidgetItem } from '../schema/dictionary';
import type { FormItem, FormSchemaDocument, FormWidgetType } from '../schema/types';
import type { FormSchemaPaletteDrag } from './palette';

/** 空表单协议文档（新建/草稿初始化用）。 */
export function createEmptyFormSchemaDocument(): FormSchemaDocument {
  return { content: { type: 'form', items: [] } };
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
  // items 可能瞬时混有素材面板拖入的临时对象（仅 paletteType 标记、无 widget，
  // 会在 add 事件内被真实字段项替换）；所有遍历必须经 widgetOf 收窄，避免
  // undefined.widgetName 在响应式重算路径上抛错。
  const widgetOf = (item: FormItem) => (item as FormItem & Partial<FormSchemaPaletteDrag>).widget;
  const selectedItem = computed(() =>
    items.value.find((item) => widgetOf(item)?.widgetName === selectedKey.value),
  );

  /** 新增字段（素材点击或拖拽落点替换临时对象）；index<0 追加到末尾。 */
  function addItem(type: FormWidgetType, index = -1): FormItem {
    const item = createWidgetItem(type);
    if (index >= 0 && index <= items.value.length) {
      items.value.splice(index, 1, item);
    } else {
      items.value.push(item);
    }
    selectedKey.value = item.widget.widgetName;
    return item;
  }

  /** 复制字段：深拷贝并换新 widgetName，插入原字段之后。 */
  function copyItem(item: FormItem): void {
    const next = copyWidgetItem(item);
    const index = items.value.findIndex(
      (entry) => widgetOf(entry)?.widgetName === item.widget.widgetName,
    );
    if (index === -1) return;
    items.value.splice(index + 1, 0, next);
    selectedKey.value = next.widget.widgetName;
  }

  function removeItem(key: string): void {
    const index = items.value.findIndex((item) => widgetOf(item)?.widgetName === key);
    if (index === -1) return;
    items.value.splice(index, 1);
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

  /** widgetName 就地重命名（属性面板）：同步选中键，唯一性由保存校验兜底。 */
  function renameItemKey(nextKey: string): void {
    const item = selectedItem.value;
    if (!item) return;
    item.widget.widgetName = nextKey;
    selectedKey.value = nextKey;
  }

  /** 整体替换文档（草稿加载/保存回读）。 */
  function replaceDocument(next: FormSchemaDocument): void {
    document.value = next;
    selectedKey.value = '';
  }

  return {
    document,
    items,
    selectedKey,
    selectedItem,
    addItem,
    copyItem,
    removeItem,
    selectItem,
    renameItemKey,
    replaceDocument,
  };
}
