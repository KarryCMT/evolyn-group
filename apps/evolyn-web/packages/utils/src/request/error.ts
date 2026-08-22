/**
 * 统一业务错误（与后端 ADR-008 对齐）。
 *
 * 由请求层的响应错误归一逻辑抛出，携带 HTTP 状态码与稳定业务码：
 * 消费方按 errCode 分支（禁止 message 文本匹配，文案会随运营调整）。
 */
export class ApiError extends Error {
  constructor(
    message: string,
    /** HTTP 状态码；网络异常/超时等未到达服务端的错误为 0 */
    readonly status: number,
    /** 稳定业务码，如 AUTH_COOLDOWN；成功响应不携带 */
    readonly errCode?: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
