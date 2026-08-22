import type { App } from 'vue';

// 模块安装函数：不依赖 vite-ssg 上下文，直接接收 Vue 应用实例
export type UserModule = (app: App) => void;

// ---------- 认证域 API 契约（与 evolyn-core internal/platform 对齐） ----------
// 后端统一响应结构由 @evolyn.do/utils 的 Result 类型承载（请求层已统一解包）

/** 登录成功签发的 JWT（model.JWTToken） */
export interface JwtToken {
  token: string;
  describe: string;
}

/** 登录请求（model.AuthUser）：name/phone + 密码，或 phone + smsCode（验证码登录） */
export interface LoginPayload {
  name?: string;
  phone?: string;
  /** 密码登录必填；验证码登录（smsCode）时留空 */
  password?: string;
  /** 短信验证码登录（与 password 互斥） */
  smsCode?: string;
  /** 指定登录目标租户编码；缺省取账号第一个成员关系（默认租户体验） */
  tenantCode?: string;
}

/**
 * 注册向导最终提交请求（POST /auth/register）：三步纯前端采集的全量数据，
 *  「进入产品」时一次性上送，此前向导各步不产生任何服务端写副作用
 */
export interface RegisterCompletePayload {
  phone: string;
  /** 短信验证码（scene=register，随本请求一次性校验，5 分钟有效期） */
  smsCode: string;
  /** 怎么称呼你；空串保留后端默认昵称（脱敏手机号） */
  nickname: string;
  /** 账号画像：角色/了解渠道（「人」的属性挂账号） */
  onboarding: AccountOnboarding;
  /** 企业画像：注册向导第 2 步采集，随租户开通写入 Config */
  tenant: {
    name: string;
    demand?: string;
    industry?: string;
  };
}

/**
 * 注册结果：单事务完成注册（账号+画像+租户+owner 绑定）后直接返回绑定
 *  新租户的会话令牌；created=false 表示手机号已注册（等价短信登录）
 */
export interface RegisterResult extends JwtToken {
  created: boolean;
}

/** 账号的租户成员关系（service.TenantMembership） */
export interface TenantMembership {
  tenantId: number;
  code: string;
  name: string;
  memberId: number;
  isOwner: boolean;
}

/** 租户（tenantmodel.Tenant，前端用到的字段子集） */
export interface Tenant {
  id: number;
  code: string;
  name: string;
  plan: string;
  status: string;
  ownerAccountId: number | null;
}

/** 自助开通租户请求：name 必填，其余为注册向导企业画像（选填） */
export interface OpenTenantPayload {
  name: string;
  /** 你的需求（单选） */
  demand?: string;
  /** 所属行业（单选） */
  industry?: string;
  /** 企业内部管理需求（多选） */
  managementNeeds?: string[];
}

/** 账号注册引导画像（model.AccountOnboarding）：注册向导第 3 步「完善信息」采集 */
export interface AccountOnboarding {
  /** 你的角色 */
  role?: string;
  /** 你从哪里了解到我们 */
  channel?: string;
}

// ---------- 登录聚合信息（GET /auth/userinfo，对齐简道云 login_user_info 引导形态） ----------

/** 平台账号（model.Account 前端用到的字段子集）：登录身份，「人」的属性挂账号 */
export interface AccountInfo {
  id: number;
  /** 登录名（免密注册时服务端生成的脱敏名） */
  name: string;
  /** 平台级展示昵称 */
  nickname: string;
  phone: string;
  email: string;
  avatar: string;
  /** 密码是否已由本人设置：免密注册账号为 false，个人中心应引导首次设置 */
  passwordInitialized: boolean;
  onboarding: AccountOnboarding;
  /** 第三方登录凭证（github/wechat 等，token 类字段不出网） */
  authInfos: { id: number; authType: string; authId: string; url: string }[];
}

/** 租户成员角色摘要（model.Role 字段子集） */
export interface RoleBrief {
  id: number;
  name: string;
  /** 角色作用域（如 tenant-admin 的判定依据） */
  scope: string;
}

/** 租户成员（model.User 字段子集）：租户内身份，昵称/部门/分组/角色挂成员 */
export interface MemberInfo {
  id: number;
  accountId: number;
  /** 租户内展示名，空则回落账号昵称 */
  nickname: string;
  tenantId: number;
  departments: { id: number; name: string }[];
  groups: { id: number; name: string }[];
  roles: RoleBrief[];
}

/** 租户配置（tenantmodel.TenantConfig）：水印/品牌主题前端暂未消费，按需扩展 */
export interface TenantConfigInfo {
  timezone: string;
  locale: string;
  onboarding: { demand?: string; industry?: string; managementNeeds?: string[] };
  watermark?: unknown;
  theme?: unknown;
}

/** 租户（含配置与配额覆盖，model.Tenant 字段子集） */
export interface TenantInfo extends Tenant {
  config: TenantConfigInfo;
  /** 套餐配额覆盖值，空则用套餐默认（生效值见 UserInfoResult.effectiveQuotas） */
  quotas: Record<string, number>;
}

/**
 * 登录聚合信息（service.UserInfoResult）：账号资料 + 当前成员身份 + 租户配置/套餐，
 *  登录或注册完成后一次拉齐，作为主框架（顶栏/菜单/配额裁剪）的引导数据源
 */
export interface UserInfoResult {
  account: AccountInfo;
  member: MemberInfo;
  tenant: TenantInfo;
  /** 生效配额（覆盖值优先，缺键回落套餐默认）：apps/forms/members/storage_gb/workflow_runs_month */
  effectiveQuotas: Record<string, number>;
}

/** 账号自助资料更新（PUT /accounts/me）：昵称非空时后端同步当前成员的租户内称呼 */
export interface AccountProfilePayload {
  nickname?: string;
  phone?: string;
  email?: string;
  avatar?: string;
  onboarding?: AccountOnboarding;
}

// ---------- 登录日志（GET /accounts/me/login-logs，个人中心「账号设置-登录日志」） ----------

/** 客户端形态（后端 UA 解析枚举），展示文案由前端映射 */
export type LoginLogClient = 'web' | 'wap' | 'unknown';

/** 单条登录日志（controller.loginLogItem）：会话建立流水，仅本人可见 */
export interface LoginLogItem {
  /** 登录时间：后端 JSONTime 秒级东八区 yyyy-MM-dd HH:mm:ss */
  loggedAt: string;
  ip: string;
  /** IP 归属地：后端写时离线解析，内网/解析失败为「内网地址/未知」兜底文案 */
  location: string;
  client: LoginLogClient;
  /** 登录方式：password 密码 / sms 短信验证码 / oauth_github、oauth_wechat 第三方 / register 注册即登录 */
  method: string;
}

/** 登录日志分页结果（controller.loginLogPage） */
export interface LoginLogPage {
  items: LoginLogItem[];
  total: number;
}

/** 登录日志查询参数：日期为 yyyy-MM-dd 闭区间（东八区自然日） */
export interface LoginLogQuery {
  page?: number;
  pageSize?: number;
  startDate?: string;
  endDate?: string;
}

// ---------- 应用配置（GET /app/conf，形态对齐简道云 conf 接口） ----------

/** 手机区号项：三语文案 + E.164 前缀值 */
export interface CallingCode {
  text: { zh_cn: string; en_us: string; zh_tw: string };
  value: string;
}

/** 区号分组（一期单组） */
export interface CallingCodeGroup {
  label: string;
  children: CallingCode[];
}

/** 登录口令加密公钥：密码字段以该公钥 RSA 加密后上送（jsencrypt PKCS#1 v1.5） */
export interface PkiConf {
  algorithm: string;
  keys: { public_key: string };
}

/** 应用配置：版本 / 区号列表 / 口令加密公钥 / 平台能力开关 */
export interface AppConf {
  version: string;
  calling_code_list: CallingCodeGroup[];
  pki: PkiConf;
  tenant_register: boolean;
  platform_sms: boolean;
  register_persona: boolean;
}
