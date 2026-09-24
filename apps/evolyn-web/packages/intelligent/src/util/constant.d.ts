/** 旧画布常量的类型声明，供 Vue 3 组件与 LogicFlow 适配层共享。 */
export const NODE_SPACE_X: number;
export const NODE_SPACE_Y: number;
export const PATH_SPACE_Y: number;
export const GRAPH_START_X: number;
export const GRAPH_START_Y: number;
export const NODE_WIDTH: number;
export const NODE_HEIGHT: number;
export const LINE_OFFSET: number;

export const EDITOR_EVENT: Readonly<{
  ONLOAD: string;
  ERROR: string;
  RESET: string;
  PREVIEW: string;
  CANVAS_MODEL_ACTIVATED: string;
  CANVAS_MODEL_DELETED: string;
  CANVAS_MODEL_HOVERED: string;
  CANVAS_MODEL_CLICKED: string;
  LOGIC_NODE_ADD: string;
  LOGIC_NODE_HOVER: string;
  LOGIC_NODE_DYN_DATA: string;
  LOGIC_HISTORY_CHANGE: string;
}>;
