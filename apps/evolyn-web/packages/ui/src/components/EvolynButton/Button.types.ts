import type { Component } from 'vue';

/** 与 Element Plus 按钮保持一致的语义类型。 */
export type ButtonType = 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info';
export type ButtonSize = 'large' | 'default' | 'small' | 'medium';
export type ButtonNativeType = 'button' | 'submit' | 'reset';

export interface ButtonProps {
  /** 按钮视觉类型，未传入时使用默认按钮。 */
  type?: ButtonType;
  /** `medium` 为兼容旧 API，视觉效果等同于 Element Plus 的 `default`。 */
  size?: ButtonSize;
  plain?: boolean;
  text?: boolean;
  bg?: boolean;
  link?: boolean;
  disabled?: boolean;
  round?: boolean;
  circle?: boolean;
  loading?: boolean;
  loadingIcon?: Component;
  icon?: Component;
  autofocus?: boolean;
  nativeType?: ButtonNativeType;
  autoInsertSpace?: boolean;
}

export interface ButtonEmits {
  (e: 'click', event: MouseEvent): void;
}
