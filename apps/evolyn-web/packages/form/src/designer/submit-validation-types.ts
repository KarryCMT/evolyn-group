/**
 * 提交校验设计态专用的前端交互模型。
 *
 * 此文件仅承载设计器编辑辅助；持久化事实类型位于 schema/types.ts 的 v7 协议，
 * 不再维护与 content.validators / preSubmitConfirm 脱节的第二份模型。
 */
import {
  DEFAULT_PRE_SUBMIT_CONFIRM,
  type PreSubmitConfirm,
  type SubmitValidator,
  type SubmitValidatorFailAction,
} from '../schema';

export type SubmitValidatorDraft = SubmitValidator;
export type PreSubmitConfirmDraft = PreSubmitConfirm;
export type { SubmitValidatorFailAction };

export function createSubmitValidatorDraft(): SubmitValidatorDraft {
  return {
    formula: '',
    remind: '',
    remark: '',
    realtime: false,
    failAction: 0,
  };
}

/**
 * 协议对象可能来自 Vue 的响应式 props，不能直接交给 structuredClone。
 * 明确投影持久化字段既隔离编辑草稿，也避免把 Proxy 带入浏览器克隆算法。
 */
export function cloneSubmitValidatorDraft(
  source: Readonly<SubmitValidator>,
): SubmitValidatorDraft {
  return {
    formula: source.formula,
    remind: source.remind,
    remark: source.remark,
    realtime: source.realtime,
    failAction: source.failAction,
  };
}

export function createPreSubmitConfirmDraft(): PreSubmitConfirmDraft {
  return {
    enable: DEFAULT_PRE_SUBMIT_CONFIRM.enable,
    title: DEFAULT_PRE_SUBMIT_CONFIRM.title,
    content: DEFAULT_PRE_SUBMIT_CONFIRM.content,
  };
}
