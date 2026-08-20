import { storeToRefs } from 'pinia';
import { readonly } from 'vue';
import { useAuthStore } from '~/stores/auth';

// 会话域消费入口：auth store（stores/auth.ts）之上的只读适配层——
// token/userInfo 以 readonly 暴露，变更一律走 store 提供的动作（login/logout/
// switchTenant 等），消费方（路由守卫/登录页/注册页）签名与 store 化之前保持一致
export function useAuth() {
  const store = useAuthStore();
  // storeToRefs 保持响应式：state/getter 提取为 ref，action 本身已绑定 store 可直接取
  const { token, userInfo, isAuthenticated, displayName, isTenantOwner } = storeToRefs(store);

  return {
    token: readonly(token),
    userInfo: readonly(userInfo),
    isAuthenticated,
    displayName,
    isTenantOwner,
    login: store.login,
    applyJwt: store.applyJwt,
    loadUserInfo: store.loadUserInfo,
    logout: store.logout,
    switchTenant: store.switchTenant,
    loadTenants: store.loadTenants,
  };
}
