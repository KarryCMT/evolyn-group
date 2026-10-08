/** 移动端认证请求：密码和短信验证码二选一。 */
export interface LoginPayload {
  name?: string;
  phone?: string;
  password?: string;
  smsCode?: string;
  setCookie?: boolean;
}

export interface LoginMfaChallenge {
  mfaRequired: true;
  mfaChallenge: string;
}

export interface SessionEstablished {
  mfaRequired?: false;
}

export type LoginResult = LoginMfaChallenge | SessionEstablished;

export interface AccountInfo {
  id: number;
  name: string;
  nickname: string;
  phone: string;
  email: string;
  avatar: string;
  passwordInitialized: boolean;
}

export interface MemberInfo {
  id: number;
  memberCode: string;
  accountId: number;
  nickname: string;
  tenantId: number;
  roles: { id: number; name: string; scope: string }[];
}

export interface TenantInfo {
  id: number;
  code: string;
  name: string;
  plan: string;
  status: string;
  ownerAccountId: number | null;
  config: { locale: string };
}

export interface UserInfoResult {
  account: AccountInfo;
  member: MemberInfo;
  tenant: TenantInfo;
  effectiveQuotas: Record<string, number>;
}

export interface AppConf {
  pki: {
    algorithm: string;
    keys: { public_key: string };
  };
}
