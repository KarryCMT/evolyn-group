// 认证域接口：与后端 /api/v1/auth/* 一一对应
// （见 evolyn-core internal/platform/auth/controller/auth.go）
import { http } from './http'
import type { JwtToken, LoginPayload, OpenTenantPayload, RegisterPayload, Tenant, TenantMembership } from '~/types'

/** 短信验证码场景：一期仅登录 */
export type SmsScene = 'login'

/** 发送短信验证码（60s 冷却/5min 有效由后端控制）；本地联调 devEcho 时回显验证码 */
export function sendSmsCode(phone: string, scene: SmsScene): Promise<{ code?: string }> {
  return http.post('/auth/sms/send', { phone, scene })
}

/** 密码登录（用户名/手机号 + 密码），成功返回 JWT */
export function login(payload: LoginPayload): Promise<JwtToken> {
  return http.post('/auth/token', payload)
}

/** OAuth 登录（github/wechat 授权码换取平台会话） */
export function oauthLogin(authType: string, authCode: string): Promise<JwtToken> {
  return http.post('/auth/token', { authType, authCode })
}

/** 退出登录（后端清理会话 Cookie） */
export function logout(): Promise<null> {
  return http.delete('/auth/token')
}

/** 注册平台账号（后端同时在默认租户建立成员身份） */
export function register(payload: RegisterPayload): Promise<null> {
  return http.post('/auth/user', payload)
}

/** 自助开通租户：当前账号成为所有者并绑定 tenant-admin（注册向导「创建团队」），企业画像随请求写入租户配置 */
export function openMyTenant(payload: OpenTenantPayload): Promise<Tenant> {
  return http.post('/auth/tenant', payload)
}

/** 当前账号加入的租户列表（含所有者标记） */
export function listTenants(): Promise<TenantMembership[]> {
  return http.get('/auth/tenants')
}

/** 切换当前租户成员身份：后端重新签发令牌，前端原位替换 */
export function switchTenant(tenantId: number): Promise<JwtToken> {
  return http.post('/auth/token/switch', { tenantId })
}
