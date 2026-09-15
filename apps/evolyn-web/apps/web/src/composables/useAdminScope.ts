import { shallowRef } from 'vue';
import { getMyAdminScopes } from '~/api/adminGroup';

/**
 * 当前成员是否企业管理员（内置系统管理员组身份，即租户管理员）。
 * 经 /auth/admin-scopes 身份自查（管理后台入口控制的既有数据源），
 * 会话级单次缓存：多个消费入口（工作台设置入口/页面直访守卫）共读同一
 * 次请求；自查失败按非管理员回落（服务端 RBAC 仍是最终授权边界）。
 */
let systemAdminPromise: Promise<boolean> | null = null;

function fetchSystemAdmin(): Promise<boolean> {
  return getMyAdminScopes()
    .then((scopes) => scopes.systemAdmin)
    .catch(() => false);
}

export function useAdminScope() {
  // null = 自查未返回（加载中）；入口渲染以 === true 放行
  const isSystemAdmin = shallowRef<boolean | null>(null);

  systemAdminPromise ??= fetchSystemAdmin();
  void systemAdminPromise.then((value) => {
    isSystemAdmin.value = value;
  });

  return { isSystemAdmin };
}
