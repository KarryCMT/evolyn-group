/**
 * 提交校验设计态专用的前端交互模型。
 *
 * 当前专项只交付设计器 UI，尚未改动服务端保存协议；因此类型与状态刻意限定在
 * designer 层，不写入 FormSchemaDocument，也不改变既有草稿/发布请求。
 */
export type SubmitValidatorFailAction = 0 | 1;

export interface SubmitValidatorDraft {
  formula: string;
  remind: string;
  remark: string;
  realtime: boolean;
  failAction: SubmitValidatorFailAction;
}

export interface PreSubmitConfirmDraft {
  enable: boolean;
  title: string;
  content: string;
}

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
  return {
    enable: false,
    title: '确认继续提交吗？',
    content: '请确认填写内容无误后继续提交。',
  };
}
