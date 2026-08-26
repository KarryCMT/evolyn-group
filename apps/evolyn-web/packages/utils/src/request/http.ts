// defHttp 的路径式门面：request(path, { method, body, query, signal }) 形态的便捷调用层，
// 业务侧（应用 api 模块）统一从这里发起请求，不必感知 AxiosRequestConfig 细节。
import { ApiError } from './error';
import { defHttp } from './instance';
import { getToken } from '../auth';

/** 门面层调用参数 */
export interface HttpRequestOptions {
  method?: string;
  body?: unknown;
  query?: Record<string, string | number | boolean | undefined | null>;
  signal?: AbortSignal;
  /**
   * 当前会话仍有效、但本次身份凭据不正确时使用（如安全设置的密码二次验证）。
   * 此类 401 只应反馈给当前表单，不能按「登录态失效」清令牌并跳转登录页。
   */
  skipUnauthorizedHandler?: boolean;
}

/**
 * 会话过期处理器（应用行为注入）：清会话、跳登录页等由应用注册，
 * 请求层本身不感知路由与应用页面结构。
 */
type UnauthorizedHandler = (error: ApiError) => void;

let unauthorizedHandler: UnauthorizedHandler | null = null;

/** 注册会话过期处理，应用启动时调用（未注册时仅抛出 ApiError） */
export function setUnauthorizedHandler(handler: UnauthorizedHandler): void {
  unauthorizedHandler = handler;
}

/** 查询参数过滤空值：undefined/null/'' 不上送 */
function cleanQuery(
  query: HttpRequestOptions['query'],
): Record<string, string | number | boolean> | undefined {
  if (!query) return undefined;
  const cleaned: Record<string, string | number | boolean> = {};
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== '') {
      cleaned[key] = value;
    }
  }
  return cleaned;
}

/**
 * 发起请求并解包统一响应，返回 data 字段（失败抛 ApiError）。
 * 会话过期（携带令牌收到 401 且非认证接口）时触发注入的处理器，
 * 排除 /auth/token 前缀避免登录失败（401）被误判为会话过期造成循环跳转。
 */
export async function request<T>(path: string, options: HttpRequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, signal, skipUnauthorizedHandler = false } = options;
  try {
    return await defHttp.request<T>({
      url: path,
      method,
      data: body as Record<string, unknown>,
      params: cleanQuery(query),
      signal,
    });
  } catch (err) {
    if (
      err instanceof ApiError &&
      err.status === 401 &&
      getToken() &&
      !path.startsWith('/auth/token') &&
      !skipUnauthorizedHandler
    ) {
      unauthorizedHandler?.(err);
      throw new ApiError('登录已过期，请重新登录', 401);
    }
    throw err;
  }
}

/** 语义化快捷方法 */
export const http = {
  get: <T>(path: string, query?: HttpRequestOptions['query'], signal?: AbortSignal) =>
    request<T>(path, { query, signal }),
  post: <T>(
    path: string,
    body?: unknown,
    options?: Pick<HttpRequestOptions, 'signal' | 'skipUnauthorizedHandler'>,
  ) => request<T>(path, { method: 'POST', body, ...options }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body }),
  patch: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body }),
  delete: <T>(path: string, body?: unknown) => request<T>(path, { method: 'DELETE', body }),
};
