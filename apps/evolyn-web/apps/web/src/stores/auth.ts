import type {
  LoginPayload,
  LoginResult,
  SessionEstablished,
  TenantMembership,
  UserInfoResult,
} from '~/types';
import { ApiError, clearLegacyToken, getRequestMessage } from '@evolyn.do/utils';
import { defineStore } from 'pinia';
import { computed, shallowRef } from 'vue';
import {
  login as apiLogin,
  logout as apiLogout,
  switchTenant as apiSwitchTenant,
  getUserInfo,
  listTenants,
  verifyMfaLogin,
} from '~/api/auth';
import { useNotificationStore } from '~/stores/notification';

// 会话域 store（pinia setup store）：认证状态 + 登录/登出/切换租户动作 +
// 登录聚合信息（账号/成员/租户/配额，对齐灵衍云 login_user_info 引导形态），
// 路由守卫、登录页与后续业务布局共用；消费方统一走 composables/auth.ts 的
// useAuth() 只读适配入口，不直接引本 store，避免绕过只读约束直写状态
export const useAuthStore = defineStore('auth', () => {
  // 凭据由 HttpOnly Cookie 承载，前端只保存当前页的认证结论，刷新后必须向
  // /auth/userinfo 重新确认，绝不能从 localStorage/sessionStorage 恢复 JWT。
  const sessionActive = shallowRef(false);

  // 登录聚合信息（账号+成员+租户+配额）：整体替换的拉取型数据，用 shallowRef；
  // 内存态不持久化，刷新页面后由路由守卫兜底重拉
  const userInfo = shallowRef<UserInfoResult | null>(null);

  const isAuthenticated = computed(() => sessionActive.value);

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

  /** 登录：成功后建立 Cookie 会话并拉取聚合信息；remember 决定 Cookie 的生命周期。 */
  async function login(payload: LoginPayload, remember = true): Promise<LoginResult> {
    const result = await apiLogin({ ...payload, setCookie: remember });
    if ('mfaRequired' in result && result.mfaRequired) {
      return result;
    }
    establishSession();
    await loadUserInfo();
    return result;
  }

  /** 完成登录第二步：仅服务端验证 challenge 成功后才写入本地会话状态。 */
  async function completeMfaLogin(payload: {
    mfaChallenge: string;
    method: 'totp' | 'recovery';
    code: string;
  }): Promise<SessionEstablished> {
    const session = await verifyMfaLogin(payload);
    establishSession();
    await loadUserInfo();
    return session;
  }

  /**
   * 接收服务端已写入的 HttpOnly Cookie（注册即登录等场景）。仅更新内存态，
   * 聚合信息由调用方按场景自行拉取；同时删除升级前遗留的浏览器可读 JWT。
   */
  function establishSession() {
    clearLegacyToken();
    sessionActive.value = true;
  }

  /**
   * 拉取登录聚合信息（/auth/userinfo）并存入 store：属引导增强数据，失败不抛出，
   * 不阻断登录/注册主流程（保留旧值，消费方按空值自行降级展示）。
   */
  async function loadUserInfo() {
    try {
      userInfo.value = await getUserInfo();
      sessionActive.value = userInfo.value !== null;
    } catch (err) {
      console.warn('[auth] 登录聚合信息拉取失败', err);
      // 401 表示 Cookie 会话不存在或已失效；5xx 不能误清仍可能有效的会话结论。
      if (err instanceof ApiError && err.status === 401) {
        sessionActive.value = false;
      }
      // 该请求会在登录成功后由 store 内部发起并吞掉异常；5xx 若只写控制台，
      // 用户会停留在登录页却不知道服务端发生故障，因此在此补充一次可见提示。
      if (err instanceof ApiError && err.status >= 500) {
        getRequestMessage().createMessage.error({
          content: '服务暂时不可用，请稍后重试',
          key: 'auth_userinfo_server_error',
        });
      }
    }
    return userInfo.value;
  }

  /** 刷新页面或进入受保护路由时，以 Cookie 请求确认会话而不是读取本地令牌。 */
  async function restoreSession(): Promise<boolean> {
    if (sessionActive.value && userInfo.value) return true;
    await loadUserInfo();
    return sessionActive.value;
  }

  /** 登出：无论接口成败都清理本地会话与聚合信息 */
  async function logout() {
    try {
      await apiLogout();
    } finally {
      clearSession();
    }
  }

  /**
   * 清理本地会话镜像：除持久化令牌外，还必须同步清空 Pinia 内存态。
   * 被其他设备挤下线时由全局 401 处理器调用，避免受保护界面继续按旧状态渲染。
   * 消息中心未读摘要属租户 × 成员维度数据，随会话一并清空
   */
  function clearSession() {
    clearLegacyToken();
    sessionActive.value = false;
    userInfo.value = null;
    useNotificationStore().clear();
  }

  /**
   * 多租户账号切换当前租户：后端原位更新 HttpOnly 会话 Cookie，
   * 并重拉聚合信息——成员/租户身份已随切换变化；消息中心未读摘要按新租户清零重拉
   */
  async function switchTenant(tenantId: TenantMembership['tenantId']) {
    await apiSwitchTenant(tenantId);
    establishSession();
    useNotificationStore().clear();
    await loadUserInfo();
    void useNotificationStore().load();
  }

  /** 拉取账号的租户成员关系（登录后判断单/多租户走向） */
  function loadTenants() {
    return listTenants();
  }

  return {
    sessionActive,
    userInfo,
    isAuthenticated,
    displayName,
    isTenantOwner,
    login,
    completeMfaLogin,
    establishSession,
    loadUserInfo,
    restoreSession,
    clearSession,
    logout,
    switchTenant,
    loadTenants,
  };
});
