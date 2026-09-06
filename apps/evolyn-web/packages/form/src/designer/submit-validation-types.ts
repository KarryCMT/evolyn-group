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

export function createPreSubmitConfirmDraft(): PreSubmitConfirmDraft {
  return structuredClone(DEFAULT_PRE_SUBMIT_CONFIRM);
}
