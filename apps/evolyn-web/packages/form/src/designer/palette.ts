/** 设计器画布与素材面板共享的拖拽分组名（vuedraggable group）。 */
export const FORM_SCHEMA_DRAG_GROUP = 'form-schema-fields';
/** 子表单内部字段排序组；目标同时接收素材面板组，但拒绝顶层引用字符串。 */
export const FORM_SCHEMA_SUBFORM_DRAG_GROUP = 'form-schema-subform-fields';

/**
 * 素材面板条目：type 为目标协议控件类型；icon 由宿主（页面）注入，
 * 字典层保持纯 TS 不依赖图标库。
 */
export interface FormSchemaPaletteEntry {
  type: string;
  label: string;
  icon: unknown;
}

/**
 * 拖拽入画布时携带的载荷：素材面板 clone 出的临时对象仅含 paletteType 标记，
 * 由画布 add 事件上报页面，页面用 createWidgetItem 替换为真实字段项。
 */
export interface FormSchemaPaletteDrag {
  paletteType: string;
}
