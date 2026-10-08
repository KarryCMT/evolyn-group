import type { LoginPayload, LoginResult, UserInfoResult } from '~/types/auth';
import { http, request } from '@evolyn.do/utils';

/** 发送登录短信；开发环境可能回显验证码。 */
export function sendLoginSms(phone: string): Promise<{ code?: string }> {
  return http.post('/auth/sms/send', { phone, scene: 'login' });
}

/** 建立 HttpOnly Cookie 会话。 */
export function login(payload: LoginPayload): Promise<LoginResult> {
  return http.post('/auth/token', payload);
}

export function logout(): Promise<null> {
  return http.delete('/auth/token');
}

export function getUserInfo(): Promise<UserInfoResult> {
  // 路由守卫用 401 判断匿名态；此时无需触发全局“会话过期”跳转。
  return request('/auth/userinfo', { skipUnauthorizedHandler: true });
}
