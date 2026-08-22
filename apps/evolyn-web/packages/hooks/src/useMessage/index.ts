import { ElMessage, ElMessageBox } from 'element-plus';
import { setRequestMessage } from '@evolyn.do/utils';
import type { RequestMessageContent } from '@evolyn.do/utils';

/**
 * 统一消息提示（Element Plus 适配）。
 *
 * 两类消费场景：
 * 1. 业务组件直接调用 useMessage() 拿 createMessage / createConfirm 等方法；
 * 2. 应用启动时调用 setupRequestMessage()，把本实现注入 utils 请求层
 *    （defHttp / checkStatus 的错误提示），utils 自身不依赖任何 UI 库。
 */

/** ElMessage 参数归一：请求层传文案或 { content, key }，key 语义用 grouping 合并近似 */
function toElMessageArgs(message: RequestMessageContent) {
  return typeof message === 'string' ? message : { message: message.content, grouping: true };
}

/** 应用启动时调用：将 Element Plus 实现注册到 utils 请求层 */
export function setupRequestMessage(): void {
  setRequestMessage({
    createMessage: {
      error: (message) => ElMessage.error(toElMessageArgs(message)),
    },
    createErrorModal: (options) => {
      // 用户关闭弹窗会 reject，静默处理避免未捕获的 Promise 异常
      ElMessageBox.alert(options.content ?? '', options.title ?? '错误提示', {
        type: 'error',
        // MessageBox 默认 alert 仅一个确认按钮，无需区分行为
        confirmButtonText: '知道了',
      }).catch(() => {});
    },
  });
}

/** 轻量消息提示：直接透传 ElMessage（success/error/warning/info/close 等） */
function createMessage() {
  return ElMessage;
}

export function useMessage() {
  return {
    createMessage,
    /**
     * 确认弹窗：resolve(true) 确认、resolve(false) 取消/关闭，
     * 业务侧用 const ok = await createConfirm({...}) 即可，无需 catch。
     */
    createConfirm: async (options: {
      title?: string;
      content?: string;
      type?: 'success' | 'warning' | 'info' | 'error';
    }) => {
      try {
        await ElMessageBox.confirm(options.content ?? '', options.title ?? '提示', {
          type: options.type ?? 'warning',
        });
        return true;
      } catch {
        return false;
      }
    },
    /** 重要错误弹窗（与注入请求层的实现保持一致的行为） */
    createErrorModal: (options: { title?: string; content?: string }) => {
      ElMessageBox.alert(options.content ?? '', options.title ?? '错误提示', {
        type: 'error',
        confirmButtonText: '知道了',
      }).catch(() => {});
    },
  };
}
