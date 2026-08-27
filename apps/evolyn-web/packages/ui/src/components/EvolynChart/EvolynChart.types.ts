import type { IInitOption, ISpec, IVChart } from '@visactor/vchart';

/** EvolynChart 接受原生 VChart Spec，不在 UI 层收窄业务图表类型。 */
export type EvolynChartSpec = ISpec;

/**
 * 图表初始化选项的受控子集。
 *
 * 挂载节点、主题与自适应由组件统一管理，避免调用侧覆盖后造成实例无法销毁或
 * 明暗主题不同步；其余 VChart 初始化能力仍可按需透传。
 */
export type EvolynChartOptions = Omit<IInitOption, 'autoFit' | 'dom' | 'renderCanvas' | 'theme'>;

export interface EvolynChartProps {
  /** VChart 图表声明。调用方以替换 Spec 对象的方式触发更新。 */
  spec: EvolynChartSpec;
  /** 跟随宿主 Element Plus 主题的展示模式。 */
  theme?: 'light' | 'dark';
  /** 容器宽度；数字按像素处理。 */
  width?: number | string;
  /** 容器高度；数字按像素处理。 */
  height?: number | string;
  /** 是否由 VChart 随容器尺寸自适应。 */
  autoFit?: boolean;
  /** 透传受控之外的 VChart 初始化配置。 */
  options?: EvolynChartOptions;
}

export interface EvolynChartEmits {
  /** 图表首次完成同步渲染后触发。 */
  ready: [chart: IVChart];
  /** 图表初始化或更新失败时触发。 */
  error: [error: Error];
}
