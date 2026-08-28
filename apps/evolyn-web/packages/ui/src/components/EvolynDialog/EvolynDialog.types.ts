import type { Component, TransitionProps } from 'vue';

export type EvolynDialogBeforeClose = (done: (cancel?: boolean) => void) => void;
export type EvolynDialogTransition = string | TransitionProps;
export type EvolynDialogClass = string | string[] | Record<string, boolean>;

/** 参考 el-dialog 的能力定义，并增加统一内容滚动和操作区。 */
export interface EvolynDialogProps {
  alignCenter?: boolean;
  appendTo?: string | HTMLElement;
  appendToBody?: boolean;
  beforeClose?: EvolynDialogBeforeClose;
  bodyClass?: EvolynDialogClass;
  bodyHeight?: number | string;
  cancelButtonText?: string;
  center?: boolean;
  closeDelay?: number;
  closeIcon?: string | Component;
  closeOnClickModal?: boolean;
  closeOnPressEscape?: boolean;
  confirmButtonText?: string;
  confirmDisabled?: boolean;
  confirmLoading?: boolean;
  destroyOnClose?: boolean;
  draggable?: boolean;
  footerClass?: EvolynDialogClass;
  fullscreen?: boolean;
  headerAriaLevel?: string;
  headerClass?: EvolynDialogClass;
  lockScroll?: boolean;
  modal?: boolean;
  modalClass?: EvolynDialogClass;
  modalPenetrable?: boolean;
  modelValue: boolean;
  openDelay?: number;
  overflow?: boolean;
  showCancelButton?: boolean;
  showClose?: boolean;
  showFooter?: boolean;
  title?: string;
  top?: string;
  transition?: EvolynDialogTransition;
  trapFocus?: boolean;
  width?: number | string;
  zIndex?: number;
}

export interface EvolynDialogEmits {
  (event: 'cancel'): void;
  (event: 'close'): void;
  (event: 'close-auto-focus'): void;
  (event: 'closed'): void;
  (event: 'confirm'): void;
  (event: 'open'): void;
  (event: 'open-auto-focus'): void;
  (event: 'opened'): void;
  (event: 'update:modelValue', value: boolean): void;
}

export interface EvolynDialogInstance {
  handleClose: () => void;
  resetPosition: () => void;
}
