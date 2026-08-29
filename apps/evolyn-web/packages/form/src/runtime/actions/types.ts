/** 运行时操作的执行通道；业务扩展统一走 custom 并交由宿主处理。 */
export type FormRuntimeActionBehavior = 'submit' | 'save-draft' | 'reset' | 'custom';

/** 与具体组件库解耦的视觉语义，由操作栏映射到 Element Plus。 */
export type FormRuntimeActionIntent = 'primary' | 'secondary' | 'danger' | 'plain';

/** 移动端动作呈现策略；低频动作进入“更多”菜单。 */
export type FormRuntimeMobilePresentation = 'button' | 'compact' | 'overflow';

export interface FormRuntimeActionDefinition {
  /** 单个操作栏内唯一，用于事件分派、加载态与测试定位。 */
  key: string;
  label: string;
  behavior: FormRuntimeActionBehavior;
  intent?: FormRuntimeActionIntent;
  order?: number;
  disabled?: boolean;
  loading?: boolean;
  visible?: boolean;
  mobilePresentation?: FormRuntimeMobilePresentation;
  /** 存在时由 Surface 统一确认，操作栏本身不产生业务副作用。 */
  confirmText?: string;
}

export type FormRuntimeLayout = 'auto' | 'desktop' | 'mobile';
