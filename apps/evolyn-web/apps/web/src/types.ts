import type { App } from 'vue'

// 模块安装函数：不依赖 vite-ssg 上下文，直接接收 Vue 应用实例
export type UserModule = (app: App) => void

// ---------- 认证域 API 契约（与 evolyn-core internal/platform 对齐） ----------

/** 后端统一响应结构（httpx.Response） */
export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

/** 登录成功签发的 JWT（model.JWTToken） */
export interface JwtToken {
  token: string
  describe: string
}

/** 登录请求（model.AuthUser）：name/phone + 密码，或 phone + smsCode（验证码登录） */
export interface LoginPayload {
  name?: string
  phone?: string
  /** 密码登录必填；验证码登录（smsCode）时留空 */
  password?: string
  /** 短信验证码登录（与 password 互斥） */
  smsCode?: string
  /** 指定登录目标租户编码；缺省取账号第一个成员关系（默认租户体验） */
  tenantCode?: string
}

/** 注册请求（model.CreatedAccount）：创建账号并在默认租户建立成员关系 */
export interface RegisterPayload {
  name: string
  phone: string
  email?: string
  password: string
}

/** 账号的租户成员关系（service.TenantMembership） */
export interface TenantMembership {
  tenantId: number
  code: string
  name: string
  memberId: number
  isOwner: boolean
}

/** 租户（tenantmodel.Tenant，前端用到的字段子集） */
export interface Tenant {
  id: number
  code: string
  name: string
  plan: string
  status: string
  ownerAccountId: number | null
}

/** 自助开通租户请求：name 必填，其余为注册向导企业画像（选填） */
export interface OpenTenantPayload {
  name: string
  /** 你的需求（单选） */
  demand?: string
  /** 所属行业（单选） */
  industry?: string
  /** 企业内部管理需求（多选） */
  managementNeeds?: string[]
}

/** 账号注册引导画像（model.AccountOnboarding）：注册向导第 3 步「完善信息」采集 */
export interface AccountOnboarding {
  /** 你的角色 */
  role?: string
  /** 你从哪里了解到我们 */
  channel?: string
}

/** 账号自助资料更新（PUT /accounts/me）：昵称非空时后端同步当前成员的租户内称呼 */
export interface AccountProfilePayload {
  nickname?: string
  onboarding?: AccountOnboarding
}
