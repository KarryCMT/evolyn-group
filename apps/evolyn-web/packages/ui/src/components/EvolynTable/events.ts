/**
 * VTable 蛇形事件名与 EvolynTable 烤串事件名的机械映射表。
 * 新增事件时两侧同步维护：本表 + EvolynTable.types.ts 的负载类型分区。
 */
const EVOLYN_TABLE_EVENT_MAP = {
  // 单元格指针事件（负载为 MousePointerCellEvent，见 types.ts 的精确签名）
  click_cell: 'click-cell',
  dblclick_cell: 'dblclick-cell',
  mousedown_cell: 'mousedown-cell',
  mouseup_cell: 'mouseup-cell',
  mouseenter_cell: 'mouseenter-cell',
  mouseleave_cell: 'mouseleave-cell',
  mousemove_cell: 'mousemove-cell',
  contextmenu_cell: 'contextmenu-cell',
  selected_cell: 'selected-cell',
  selected_clear: 'selected-clear',
  // 表格级指针/键盘事件
  mouseenter_table: 'mouseenter-table',
  mouseleave_table: 'mouseleave-table',
  mousedown_table: 'mousedown-table',
  contextmenu_canvas: 'contextmenu-canvas',
  keydown: 'keydown',
  // 排序与冻结
  sort_click: 'sort-click',
  after_sort: 'after-sort',
  freeze_click: 'freeze-click',
  // 列宽调整与表头拖拽换位
  resize_column: 'resize-column',
  resize_column_end: 'resize-column-end',
  change_header_position: 'change-header-position',
  change_header_position_start: 'change-header-position-start',
  change_header_position_fail: 'change-header-position-fail',
  // 滚动
  scroll: 'scroll',
  scroll_horizontal_end: 'scroll-horizontal-end',
  scroll_vertical_end: 'scroll-vertical-end',
  // 菜单与图标
  dropdown_menu_click: 'dropdown-menu-click',
  dropdown_menu_clear: 'dropdown-menu-clear',
  dropdown_icon_click: 'dropdown-icon-click',
  show_menu: 'show-menu',
  hide_menu: 'hide-menu',
  icon_click: 'icon-click',
  // 选择与框选
  drag_select_end: 'drag-select-end',
  // 单元格交互控件状态
  checkbox_state_change: 'checkbox-state-change',
  radio_state_change: 'radio-state-change',
  switch_state_change: 'switch-state-change',
  change_cell_value: 'change-cell-value',
  button_click: 'button-click',
  // 树形展开收起
  tree_hierarchy_state_change: 'tree-hierarchy-state-change',
  // 空态与生命周期
  empty_tip_click: 'empty-tip-click',
  empty_tip_dblclick: 'empty-tip-dblclick',
  after_render: 'after-render',
  initialized: 'initialized',
  updated: 'updated',
} as const;

/** VTable 侧事件名（蛇形） */
export type EvolynTableVTableEvent = keyof typeof EVOLYN_TABLE_EVENT_MAP;

/** EvolynTable 侧事件名（烤串，模板中 @click-cell 使用） */
export type EvolynTableEventName = (typeof EVOLYN_TABLE_EVENT_MAP)[EvolynTableVTableEvent];

/** 负载为 MousePointerCellEvent 的单元格指针事件子集 */
export type EvolynTablePointerEventName =
  | 'click-cell'
  | 'dblclick-cell'
  | 'mousedown-cell'
  | 'mouseup-cell'
  | 'mouseenter-cell'
  | 'mouseleave-cell'
  | 'mousemove-cell'
  | 'contextmenu-cell'
  | 'selected-cell';

/** 供实例绑事件用的只读映射（key 为 VTable 事件名） */
export const EVOLYN_TABLE_EVENTS: Readonly<Record<EvolynTableVTableEvent, EvolynTableEventName>> =
  EVOLYN_TABLE_EVENT_MAP;
