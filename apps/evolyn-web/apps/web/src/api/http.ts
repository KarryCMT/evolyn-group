// 统一 HTTP 客户端：基于原生 fetch 封装，零额外依赖
// - 自动携带 Authorization: Bearer {token}
// - 解析后端统一响应结构 { code, msg, data }，失败抛 ApiError（message 可直接展示）
// - 会话过期（携带 token 收到 401）时清理本地会话并回登录页
import type { ApiResponse } from '~/types'

/** JWT 的存储键名（localStorage / sessionStorage 共用） */
const TOKEN_KEY = 'evolyn.token'

/** 后端 API 基础地址：默认同源 /api/v1（开发期由 Vite 代理转发到 evolyn-core） */
const BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export function getToken(): string | null {
  // 「下次自动登录」决定令牌落点：先查持久存储再查会话存储
  return localStorage.getItem(TOKEN_KEY) ?? sessionStorage.getItem(TOKEN_KEY)
}

/** 写入令牌：remember=true 持久化（跨会话），否则会话级（关闭浏览器失效） */
export function setToken(token: string, remember = true) {
  if (remember) {
    localStorage.setItem(TOKEN_KEY, token)
    sessionStorage.removeItem(TOKEN_KEY)
  } else {
    sessionStorage.setItem(TOKEN_KEY, token)
    localStorage.removeItem(TOKEN_KEY)
  }
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
  sessionStorage.removeItem(TOKEN_KEY)
}

/** 统一业务错误：携带 HTTP 状态码与后端业务码 */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
  query?: Record<string, string | number | boolean | undefined | null>
  signal?: AbortSignal
}

/** 发起请求并解包统一响应，返回 data 字段 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, signal } = options

  // 组装查询串：跳过空值
  const url = new URL(BASE_URL + path, window.location.origin)
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== null && value !== '') {
        url.searchParams.set(key, String(value))
      }
    }
  }

  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`
  const hasBody = body !== undefined
  if (hasBody) headers['Content-Type'] = 'application/json'

  let res: Response
  try {
    res = await fetch(url, {
      method,
      headers,
      body: hasBody ? JSON.stringify(body) : null,
      signal,
    })
  } catch {
    throw new ApiError('网络异常，请检查网络后重试', 0)
  }

  // 会话过期：仅在携带 token 且非登录/换租户接口时清理并回登录页，
  // 避免登录失败（401）被误判为会话过期造成循环跳转
  if (res.status === 401 && token && !path.startsWith('/auth/token')) {
    clearToken()
    const redirect = encodeURIComponent(window.location.pathname + window.location.search)
    window.location.href = `/auth/login?redirect=${redirect}`
    throw new ApiError('登录已过期，请重新登录', 401)
  }

  // 解析统一响应体；非 JSON（如网关错误页）时保持 null，走状态码判断
  let envelope: ApiResponse | null = null
  try {
    envelope = (await res.json()) as ApiResponse
  } catch {
    /* 忽略非 JSON 响应 */
  }

  if (!res.ok) {
    throw new ApiError(envelope?.msg || `请求失败（HTTP ${res.status}）`, res.status, envelope?.code)
  }

  return (envelope?.data ?? null) as T
}

/** 语义化快捷方法 */
export const http = {
  get: <T>(path: string, query?: RequestOptions['query'], signal?: AbortSignal) =>
    request<T>(path, { query, signal }),
  post: <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body }),
  put: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body }),
  delete: <T>(path: string, body?: unknown) => request<T>(path, { method: 'DELETE', body }),
}
