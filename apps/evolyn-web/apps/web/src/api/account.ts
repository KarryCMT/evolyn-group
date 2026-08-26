// 账号自助接口：与后端 /api/v1/accounts/me 一一对应
// （见 evolyn-core internal/platform/iam/controller/account.go）
import { http } from '@evolyn.do/utils';
import type {
  AccountInfo,
  AccountProfilePayload,
  AccountSession,
  AccountSecurityOverview,
  LoginLogPage,
  LoginLogQuery,
  TOTPEnrollment,
} from '~/types';

/** 更新我的账号资料：昵称/邮箱/头像与注册引导画像（角色/了解渠道） */
export function updateMyProfile(payload: AccountProfilePayload): Promise<AccountInfo> {
  return http.put('/accounts/me', payload);
}

/** 绑定邮箱第一步：消费当前手机号的 rebind 验证码，获取短时一次性身份凭证。 */
export function verifyMyEmailIdentity(smsCode: string): Promise<{ verificationToken: string }> {
  return http.post('/accounts/me/email/identity', { smsCode });
}

/** 绑定邮箱第二步：向已通过身份验证的新邮箱发送验证码；开发环境可回显固定码。 */
export function sendMyEmailCode(payload: {
  email: string;
  verificationToken: string;
}): Promise<{ code?: string }> {
  return http.post('/accounts/me/email/code', payload);
}

/** 绑定邮箱最终提交：服务端原子消费身份凭证与邮箱验证码后才更新资料。 */
export function bindMyEmail(payload: {
  email: string;
  emailCode: string;
  verificationToken: string;
}): Promise<AccountInfo> {
  return http.put('/accounts/me/email', payload);
}

/** 修改登录密码：短信免密注册的账号首次设置可免旧密码（oldPassword 留空） */
export function changeMyPassword(payload: {
  oldPassword?: string;
  newPassword: string;
}): Promise<null> {
  return http.put('/accounts/me/password', payload);
}

/** 我的登录日志分页查询（仅本人）：日期为 yyyy-MM-dd 闭区间，按东八区自然日过滤 */
export function listMyLoginLogs(query: LoginLogQuery = {}): Promise<LoginLogPage> {
  // 显式展开为字面量：LoginLogQuery 接口无索引签名，直传不满足 query 参数类型
  return http.get('/accounts/me/login-logs', {
    page: query.page,
    pageSize: query.pageSize,
    startDate: query.startDate,
    endDate: query.endDate,
  });
}

/** 我的账号安全概览：MFA、单会话、恢复码和活跃设备会话摘要。 */
export function getMySecurityOverview(): Promise<AccountSecurityOverview> {
  return http.get('/accounts/me/security');
}

/** 获取当前账号的活跃设备会话，按最近活跃时间倒序。 */
export function listMySessions(): Promise<AccountSession[]> {
  return http.get('/accounts/me/sessions');
}

/** 撤销当前账号指定设备会话；后续携带该 sid 的令牌会立即失效。 */
export function revokeMySession(sid: string): Promise<null> {
  return http.delete(`/accounts/me/sessions/${encodeURIComponent(sid)}`);
}

/** 高风险安全操作前重新验证当前身份，成功后返回五分钟一次性凭证。 */
export function reauthAccountSecurity(payload: {
  password: string;
}): Promise<{ reauthToken: string }> {
  // 密码输错返回 401 仅代表本次二次验证失败；不能清掉仍有效的登录会话。
  return http.post('/accounts/me/security/reauth', payload, { skipUnauthorizedHandler: true });
}

/** 创建 TOTP 绑定向导；返回短时有效的验证器导入地址。 */
export function enrollMyTOTP(reauthToken: string): Promise<TOTPEnrollment> {
  return http.post('/accounts/me/security/mfa/totp/enroll', { reauthToken });
}

/** 确认首个 TOTP 动态码并启用 MFA；恢复码只在本次响应中返回。 */
export function confirmMyTOTP(
  enrollmentId: string,
  code: string,
): Promise<{ recoveryCodes: string[] }> {
  return http.post('/accounts/me/security/mfa/totp/confirm', { enrollmentId, code });
}

/** 关闭当前账号的 TOTP 登录二次验证；后端会同时撤销其他设备会话。 */
export function disableMyTOTP(reauthToken: string): Promise<null> {
  return http.delete('/accounts/me/security/mfa/totp', { reauthToken });
}

/** 更新禁止同时登录开关；开启时仅保留当前设备会话。 */
export function updateMySingleSession(payload: {
  reauthToken: string;
  enabled: boolean;
}): Promise<null> {
  return http.put('/accounts/me/security/single-session', payload);
}

/** 注销当前账号；账号仍是任一团队创建人时，后端会拒绝并要求先处理团队归属。 */
export function cancelMyAccount(reauthToken: string): Promise<null> {
  return http.delete('/accounts/me', { reauthToken });
}
