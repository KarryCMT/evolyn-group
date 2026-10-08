import type { LoginPayload, LoginResult, UserInfoResult } from '~/types/auth';
import { clearLegacyToken } from '@evolyn.do/utils';
import { defineStore } from 'pinia';
import { computed, shallowRef } from 'vue';
import { login as apiLogin, logout as apiLogout, getUserInfo } from '~/api/auth';

/** 移动宿主只保存会话镜像，真实凭据由 HttpOnly Cookie 承载。 */
export const useAuthStore = defineStore('mobile-auth', () => {
  const sessionActive = shallowRef(false);
  const userInfo = shallowRef<UserInfoResult | null>(null);

  const isAuthenticated = computed(() => sessionActive.value);
  const displayName = computed(() => {
    const info = userInfo.value;
    return info?.member.nickname || info?.account.nickname || info?.account.name || '灵衍云用户';
  });
  const isTenantOwner = computed(() => {
    const info = userInfo.value;
    return Boolean(info && info.tenant.ownerAccountId === info.account.id);
  });

  async function login(payload: LoginPayload, remember: boolean): Promise<LoginResult> {
    const result = await apiLogin({ ...payload, setCookie: remember });
    if ('mfaRequired' in result && result.mfaRequired) return result;
    clearLegacyToken();
    sessionActive.value = true;
    await loadUserInfo();
    return result;
  }

  async function loadUserInfo(): Promise<UserInfoResult | null> {
    try {
      userInfo.value = await getUserInfo();
      sessionActive.value = true;
    } catch {
      sessionActive.value = false;
      userInfo.value = null;
    }
    return userInfo.value;
  }

  async function restoreSession(): Promise<boolean> {
    if (sessionActive.value && userInfo.value) return true;
    await loadUserInfo();
    return sessionActive.value;
  }

  function clearSession(): void {
    clearLegacyToken();
    sessionActive.value = false;
    userInfo.value = null;
  }

  async function logout(): Promise<void> {
    try {
      await apiLogout();
    } finally {
      clearSession();
    }
  }

  return {
    sessionActive,
    userInfo,
    isAuthenticated,
    displayName,
    isTenantOwner,
    login,
    loadUserInfo,
    restoreSession,
    clearSession,
    logout,
  };
});
