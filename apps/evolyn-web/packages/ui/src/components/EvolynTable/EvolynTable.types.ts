import type { ListTableConstructorOptions, MousePointerCellEvent, TYPES } from '@visactor/vtable';
import type { EvolynTableEventName, EvolynTablePointerEventName } from './events';
/**
 * 行记录的宽松类型。interface 声明没有隐式索引签名（Record<string, unknown>
 * 会以「缺少索引签名」拒绝赋值），值放宽到 any 让业务接口类型可直接作为
 * records 传入；组件自身以泛型 Row 保留精确类型。
 */
export type EvolynTableRow = Record<string, any>;

/** 声明式富单元格：text/image/circle 等元素的绝对定位组合，透传 VTable customRender */
export type EvolynTableCustomRender = TYPES.ICustomRender;

/**
 * 富单元格的单个元素与整体对象类型。重导出的意义：customRender 是函数时
 * 内部字面量拿不到上下文类型，type 字段会被拓宽为 string 而报错，使用方
 * 需要这两个类型给中间变量做标注（应用侧不直接依赖 @visactor/vtable）。
 */
export type EvolynTableCustomRenderElement = TYPES.ICustomRenderElement;
export type EvolynTableCustomRenderObj = TYPES.ICustomRenderObj;

/**
 * 收窄后的列定义。未显式声明的 VTable 原生列配置（style/headerIcon/
 * customLayout 等）经索引签名原样透传，作为逃生舱。
 */
export interface EvolynTableColumn {
  /** 取值字段名 */
  field: string;
  /** 列标题 */
  title: string;
  /** 列宽：数字为像素，字符串可为百分比 */
  width?: number | string;
  /** 最小列宽（像素），widthMode 为 adaptive 时参与剩余宽度分配 */
  minWidth?: number;
  /** 内容对齐方式，表头与表体一致 */
  align?: 'left' | 'center' | 'right';
  /** 开启列排序（表头出现排序图标） */
  sortable?: boolean;
  /** 单元格类型，默认 text */
  cellType?:
    | 'text'
    | 'link'
    | 'image'
    | 'video'
    | 'checkbox'
    | 'radio'
    | 'switch'
    | 'progressbar'
    | 'sparkline'
    | 'button';
  /** 文本格式化：入参为行数据与列、行下标 */
  format?: (record: EvolynTableRow, col: number, row: number) => string;
  /** 声明式富单元格，优先级高于 cellType/format；画布无法消费 CSS 变量，色值需传具体值 */
  customRender?: EvolynTableCustomRender;
  /** 逃生舱：VTable ColumnDefine 的其余字段原样透传 */
  [key: string]: any;
}

/**
 * 逃生舱：ListTable 完整配置。数据入口（columns/records）与主题由组件
 * props 统一管理，避免出现双通道数据来源。
 */
export type EvolynTableOptions = Partial<
  Omit<ListTableConstructorOptions, 'columns' | 'records' | 'theme'>
>;

export interface EvolynTableProps {
  /** 列定义 */
  columns: EvolynTableColumn[];
  /** 行数据；替换数组引用即触发数据增量刷新（内部走 setRecords，不整表重建） */
  records?: EvolynTableRow[];
  /** 逃生舱：其余 ListTable 配置，优先级高于组件内置默认值 */
  options?: EvolynTableOptions;
  /** 容器宽度，默认 100% */
  width?: number | string;
  /** 容器高度，默认 100%，父容器需要有确定高度 */
  height?: number | string;
  /**
   * 视觉模式：变化时从 --el-* CSS 变量重读主题（应用切换暗色后更新此值即可），
   * 默认 light。
   */
  theme?: 'light' | 'dark';
  /** 空数据提示文案 */
  emptyText?: string;
}

/**
 * 事件契约（调用签名重载）：
 * - 单元格指针事件的负载为 VTable 的 MousePointerCellEvent（精确类型）；
 * - 其余事件的负载未知（原始参数原样转发）。
 * 注意不要改写成映射类型 `[K in ...]`——SFC 编译器提取运行时 emits 时
 * 无法把跨文件映射类型解析为有限键，会直接编译失败。
 */
export interface EvolynTableEmits {
  (event: EvolynTablePointerEventName, args: MousePointerCellEvent): void;
  (event: Exclude<EvolynTableEventName, EvolynTablePointerEventName>, args: unknown): void;
}
