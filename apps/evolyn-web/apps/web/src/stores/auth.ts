import type { LoginPayload, TenantMembership, UserInfoResult } from '~/types';
import { defineStore } from 'pinia';
import { computed, shallowRef } from 'vue';
import {
  login as apiLogin,
  logout as apiLogout,
  switchTenant as apiSwitchTenant,
  getUserInfo,
  listTenants,
} from '~/api/auth';
import { clearToken, getToken, setToken } from '~/api/http';

// 会话域 store（pinia setup store）：token 单例状态 + 登录/登出/切换租户动作 +
// 登录聚合信息（账号/成员/租户/配额，对齐简道云 login_user_info 引导形态），
// 路由守卫、登录页与后续业务布局共用；消费方统一走 composables/auth.ts 的
// useAuth() 只读适配入口，不直接引本 store，避免绕过只读约束直写状态
export const useAuthStore = defineStore('auth', () => {
  // token 全局唯一，刷新页面时从本地存储恢复（范围由「下次自动登录」决定）
  const token = shallowRef<string | null>(getToken());

  // 登录聚合信息（账号+成员+租户+配额）：整体替换的拉取型数据，用 shallowRef；
  // 内存态不持久化，刷新页面后由路由守卫兜底重拉
  const userInfo = shallowRef<UserInfoResult | null>(null);

  // 「下次自动登录」的存储范围记忆：切换租户重签令牌时沿用登录时的选择，
  // 避免会话级令牌被升级为持久令牌
  let rememberLogin = true;

  const isAuthenticated = computed(() => token.value !== null);

  // 展示名：租户内称呼优先，回落账号昵称（成员昵称为空是合法状态）
  const displayName = computed(() => {
    const info = userInfo.value;
    if (!info) return '';
    return info.member.nickname || info.account.nickname;
  });

  // 当前成员是否为租户所有者（开通者）：控制团队管理入口等 owner 专属能力
  const isTenantOwner = computed(() => {
    const info = userInfo.value;
    return !!info && info.tenant.ownerAccountId === info.account.id;
  });

  /** 登录：成功后持有 JWT 并拉取登录聚合信息；remember 控制令牌存储范围（持久/会话级） */
  async function login(payload: LoginPayload, remember = true) {
    applyJwt(await apiLogin(payload), remember);
    await loadUserInfo();
  }

  /**
   * 直接持有已签发的 JWT（注册即登录等后端已返回令牌的场景），登录的底层复用；
   * 仅落令牌不拉聚合信息，调用方按场景自行补 loadUserInfo
   */
  function applyJwt(jwt: { token: string }, remember = true) {
    rememberLogin = remember;
    setToken(jwt.token, remember);
    token.value = jwt.token;
  }

  /**
   * 拉取登录聚合信息（/auth/userinfo）并存入 store：属引导增强数据，失败只告警
   * 不抛出，不阻断登录/注册主流程（保留旧值，消费方按空值自行降级展示）
   */
  async function loadUserInfo() {
    try {
      userInfo.value = await getUserInfo();
    } catch (err) {
      console.warn('[auth] 登录聚合信息拉取失败', err);
    }
    return userInfo.value;
  }

  /** 登出：无论接口成败都清理本地会话与聚合信息 */
  async function logout() {
    try {
      await apiLogout();
    } finally {
      clearToken();
      token.value = null;
      userInfo.value = null;
    }
  }

  /**
   * 多租户账号切换当前租户：后端重新签发令牌，前端原位替换（沿用存储范围），
   * 并重拉聚合信息——成员/租户身份已随切换变化
   */
  async function switchTenant(tenantId: TenantMembership['tenantId']) {
    const jwt = await apiSwitchTenant(tenantId);
    setToken(jwt.token, rememberLogin);
    token.value = jwt.token;
    await loadUserInfo();
  }

  /** 拉取账号的租户成员关系（登录后判断单/多租户走向） */
  function loadTenants() {
    return listTenants();
  }

  return {
    token,
    userInfo,
    isAuthenticated,
    displayName,
    isTenantOwner,
    login,
    applyJwt,
    loadUserInfo,
    logout,
    switchTenant,
    loadTenants,
  };
});
