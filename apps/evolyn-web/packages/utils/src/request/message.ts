/**
 * 请求错误提示注入。
 *
 * utils 层不直接依赖具体 UI 库（Element Plus 等），请求模块默认静默处理错误；
 * 应用启动时调用 setRequestMessage 注册实现（如 ElMessage / ElMessageBox），
 * 保持与原 useMessage({ createMessage, createErrorModal }) 相同的调用形状。
 */

/** 消息提示参数：直接传文案，或携带 key 供去重更新 */
export type RequestMessageContent = string | { content?: string; key?: string };

export interface HttpRequestMessage {
  /** 轻量消息提示（对应 ElMessage.error 等） */
  createMessage: {
    error: (message: RequestMessageContent) => void;
  };
  /** 重要错误弹窗（对应 ElMessageBox.alert 等） */
  createErrorModal: (options: { title?: string; content?: string }) => void;
}

// 默认空实现：未注册时仅静默，不阻塞请求流程
const noopMessage: HttpRequestMessage = {
  createMessage: {
    error: () => {},
  },
  createErrorModal: () => {},
};

let messageImpl: HttpRequestMessage = noopMessage;

/** 注册错误提示实现，应用启动时调用 */
export function setRequestMessage(message: HttpRequestMessage): void {
  messageImpl = message;
}

/** 获取当前错误提示实现（未注册时为空实现） */
export function getRequestMessage(): HttpRequestMessage {
  return messageImpl;
}
