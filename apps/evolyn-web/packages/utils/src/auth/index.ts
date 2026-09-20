import { TOKEN_KEY } from '../enums/cacheEnum';

/**
 * 清理历史版本遗留的浏览器可读 JWT。
 *
 * 当前认证凭据仅由服务端写入 HttpOnly Cookie，前端不能也不应读取令牌。本函数
 * 仅用于升级时移除旧键，保留导出以避免其他工作区包继续持有旧凭据。
 */
export function clearLegacyToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  sessionStorage.removeItem(TOKEN_KEY);
}
