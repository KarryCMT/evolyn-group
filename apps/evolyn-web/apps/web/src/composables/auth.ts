// 会话域 composable：token 单例状态 + 登录/登出/切换租户动作，
// 路由守卫、登录页与后续业务布局共用；引入 pinia 后可平移为 store，消费方无感
import { computed, readonly, shallowRef } from 'vue'
import {
  listTenants,
  login as apiLogin,
  logout as apiLogout,
  switchTenant as apiSwitchTenant,
} from '~/api/auth'
import { clearToken, getToken, setToken } from '~/api/http'
import type { LoginPayload, TenantMembership } from '~/types'

// 模块级单例：token 全局唯一，刷新页面时从本地存储恢复（范围由「下次自动登录」决定）
const token = shallowRef<string | null>(getToken())

// 「下次自动登录」的存储范围记忆：切换租户重签令牌时沿用登录时的选择，
// 避免会话级令牌被升级为持久令牌
let rememberLogin = true

export function useAuth() {
  const isAuthenticated = computed(() => token.value !== null)

  /** 登录：成功后持有 JWT；remember 控制令牌存储范围（持久/会话级） */
  async function login(payload: LoginPayload, remember = true) {
    applyJwt(await apiLogin(payload), remember)
  }

  /** 直接持有已签发的 JWT（注册即登录等后端已返回令牌的场景），登录的底层复用 */
  function applyJwt(jwt: { token: string }, remember = true) {
    rememberLogin = remember
    setToken(jwt.token, remember)
    token.value = jwt.token
  }

  /** 登出：无论接口成败都清理本地会话 */
  async function logout() {
    try {
      await apiLogout()
    } finally {
      clearToken()
      token.value = null
    }
  }

  /** 多租户账号切换当前租户：后端重新签发令牌，前端原位替换（沿用存储范围） */
  async function switchTenant(tenantId: TenantMembership['tenantId']) {
    const jwt = await apiSwitchTenant(tenantId)
    setToken(jwt.token, rememberLogin)
    token.value = jwt.token
  }

  /** 拉取账号的租户成员关系（登录后判断单/多租户走向） */
  function loadTenants() {
    return listTenants()
  }

  return { token: readonly(token), isAuthenticated, login, applyJwt, logout, switchTenant, loadTenants }
}
