import type {
  JwtToken,
  LoginResult,
  LoginPayload,
  OpenTenantPayload,
  RegisterCompletePayload,
  RegisterResult,
  Tenant,
  TenantMembership,
  UserInfoResult,
} from '~/types';
// 认证域接口：与后端 /api/v1/auth/* 一一对应
// （见 evolyn-core internal/platform/auth/controller/auth.go）
import { http } from '@evolyn.do/utils';

/** 短信验证码场景：login=登录 / register=注册 / reset=找回密码 / rebind=安全凭证换绑 */
export type SmsScene = 'login' | 'register' | 'reset' | 'rebind';

/** 发送短信验证码（60s 冷却/5min 有效由后端控制）；本地联调 devEcho 时回显验证码 */
export function sendSmsCode(
  phone: string,
  scene: SmsScene,
  purpose?: 'old' | 'new',
): Promise<{ code?: string }> {
  return http.post('/auth/sms/send', { phone, scene, purpose });
}

/** 密码登录（用户名/手机号 + 密码），成功返回 JWT */
export function login(payload: LoginPayload): Promise<LoginResult> {
  return http.post('/auth/token', payload);
}

/** 消费登录第一步返回的 MFA challenge，成功后才得到设备会话 JWT。 */
export function verifyMfaLogin(payload: {
  mfaChallenge: string;
  method: 'totp' | 'recovery';
  code: string;
}): Promise<JwtToken> {
  return http.post('/auth/mfa/verify', payload);
}

/** OAuth 登录（github/wechat 授权码换取平台会话） */
export function oauthLogin(authType: string, authCode: string): Promise<JwtToken> {
  return http.post('/auth/token', { authType, authCode });
}

/** 退出登录（后端清理会话 Cookie） */
export function logout(): Promise<null> {
  return http.delete('/auth/token');
}

/**
 * 注册（注册向导最终提交「进入产品」）：三步采集的全量数据一次性上送，
 *  服务端单事务完成免密注册账号（已注册手机号等价短信登录，created=false）、
 *  落账号画像、开通租户并绑定 tenant-admin，返回绑定新租户的会话令牌。
 *  验证码随本请求一次性校验，超有效期返回 401 需回第 1 步重新获取
 */
export function registerComplete(payload: RegisterCompletePayload): Promise<RegisterResult> {
  return http.post('/auth/register', payload);
}

/** 找回密码：验证码使用 reset 场景，新密码必须先由调用方经 RSA 公钥加密。 */
export function resetPassword(payload: {
  phone: string;
  smsCode: string;
  newPassword: string;
}): Promise<null> {
  return http.post('/auth/password/reset', payload);
}

/** 自助开通租户：当前账号成为所有者并绑定 tenant-admin（注册向导「创建团队」），企业画像随请求写入租户配置 */
export function openMyTenant(payload: OpenTenantPayload): Promise<Tenant> {
  return http.post('/auth/tenant', payload);
}

/** 当前账号加入的租户列表（含所有者标记） */
export function listTenants(): Promise<TenantMembership[]> {
  return http.get('/auth/tenants');
}

/** 切换当前租户成员身份：后端重新签发令牌，前端原位替换 */
export function switchTenant(tenantId: number): Promise<JwtToken> {
  return http.post('/auth/token/switch', { tenantId });
}

/**
 * 登录聚合信息（对齐灵衍云 login_user_info）：账号资料 + 当前成员身份 +
 *  租户配置/套餐/生效配额，登录或注册完成后拉取，作为主框架引导数据源
 */
export function getUserInfo(): Promise<UserInfoResult> {
  return http.get('/auth/userinfo');
}
