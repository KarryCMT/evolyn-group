import { TOKEN_KEY } from '../enums/cacheEnum';

// 令牌存取：「下次自动登录」决定落点——remember=true 持久化 localStorage（跨会话），
// 否则会话级 sessionStorage（关闭浏览器失效）；读取时先查持久再查会话。
// 请求层（requestInterceptors）每次请求动态读取，登录/切换租户后立即生效。

/** 读取令牌：持久存储优先，回落会话存储 */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY) ?? sessionStorage.getItem(TOKEN_KEY);
}

/** 写入令牌：remember 控制存储范围，写入前清空另一侧避免双份残留 */
export function setToken(token: string, remember = true): void {
  if (remember) {
    localStorage.setItem(TOKEN_KEY, token);
    sessionStorage.removeItem(TOKEN_KEY);
  } else {
    sessionStorage.setItem(TOKEN_KEY, token);
    localStorage.removeItem(TOKEN_KEY);
  }
}

/** 清空令牌（两处存储一并清理） */
export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(TOKEN_KEY);
}
