import type { App } from 'vue';

// 模块安装函数：不依赖 vite-ssg 上下文，直接接收 Vue 应用实例
export type UserModule = (app: App) => void;

// ---------- 认证域 API 契约（与 evolyn-core internal/platform 对齐） ----------
// 后端统一响应结构由 @evolyn.do/utils 的 Result 类型承载（请求层已统一解包）

/** 登录完成后签发的 JWT。 */
export interface JwtToken {
  token: string;
  describe: string;
}

/** 登录第一步结果：启用 MFA 时不返回令牌，只给出五分钟一次性 challenge。 */
export interface LoginMfaChallenge {
  mfaRequired: true;
  mfaChallenge: string;
  token?: never;
}

export type LoginResult = JwtToken | LoginMfaChallenge;

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
  /** 公开成员邀请链接携带的租户 token；存在时注册后加入目标租户。 */
  tenantInvite?: string;
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

// ---------- 登录聚合信息（GET /auth/userinfo，对齐灵衍云 login_user_info 引导形态） ----------

/** 平台账号（model.Account 前端用到的字段子集）：登录身份，「人」的属性挂账号 */
export interface AccountInfo {
  id: number;
  /** 登录名（免密注册时服务端生成的脱敏名） */
  name: string;
  /** 平台级展示昵称 */
  nickname: string;
  phone: string;
  email: string;
  /** 外部头像 URL 或个人设置裁剪后保存的 data URL */
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

// ---------- 账号安全概览（GET /accounts/me/security） ----------

/** 账号安全策略与当前设备会话摘要；仅当前登录账号可读取。 */
export interface AccountSecurityOverview {
  /** 是否已启用 TOTP 登录二次验证。 */
  mfaEnabled: boolean;
  /** 是否仅保留一个活跃设备会话。 */
  singleSessionEnabled: boolean;
  /** 是否存在已验证且未停用的 TOTP 因子。 */
  totpEnrolled: boolean;
  /** 尚未使用的 MFA 恢复码数量。 */
  recoveryCodesRemaining: number;
  /** 当前未撤销且未过期的设备会话数量。 */
  activeSessions: number;
}

/** 设备级账号会话；仅返回当前账号未撤销且未过期的会话。 */
export interface AccountSession {
  /** JWT 内携带的随机设备会话标识。 */
  sid: string;
  /** 第一阶段认证方式：password、sms、oauth 或 register。 */
  authMethod: string;
  /** 第二阶段 MFA 方式；未启用时为 null。 */
  mfaMethod: string | null;
  createdAt: string;
  lastSeenAt: string;
  expiresAt: string;
  ip: string;
  location: string;
  userAgent: string;
}

/** TOTP 绑定向导：验证器导入地址仅在内存中保留至本次确认完成。 */
export interface TOTPEnrollment {
  enrollmentId: string;
  otpauthUrl: string;
}

// ---------- 应用配置（GET /app/conf，形态对齐灵衍云 conf 接口） ----------

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

// ---------- 应用管理域 API 契约（M2-A，与 evolyn-core application 域对齐） ----------

/** 历史图标键，仅用于旧页面的回退渲染。 */
export type ApplicationIconKey = 'bookmark' | 'briefcase' | 'contacts' | 'chart' | 'check';

/** 应用图标：系统 Remix 图标或自定义文件地址。 */
export type ApplicationIcon = EvolynIconPickerValue;

export const DEFAULT_APPLICATION_ICON: ApplicationIcon = {
  type: 'remix',
  name: 'bookmark',
  background: '#f7be54,#eda426',
};

/** 旧图标展示入口使用该键回退，完整图标渲染由 ApplicationIconPicker 负责。 */
export function getApplicationIconName(icon: ApplicationIcon | undefined): string {
  return icon?.type === 'remix' ? icon.name : 'bookmark';
}

/** 应用稳定颜色键（服务端枚举，渲染时映射主题色变量） */
export type ApplicationColor = 'primary';

/** 应用入口形态：构建引导与运行时首页由应用自身状态决定，不依赖当前成员菜单数量。 */
export type ApplicationHomeMode = 'builder' | 'application';

/** 创建空白应用请求（POST /applications）：名称必填，图标/颜色可省略取服务端默认 */
export interface CreateBlankApplicationPayload {
  name: string;
  icon?: ApplicationIcon;
  color?: ApplicationColor;
}

/** 更新应用请求（PATCH /applications/:id）：白名单字段，指针语义（未传不改） */
export interface UpdateApplicationPayload {
  name?: string;
  icon?: ApplicationIcon;
  color?: ApplicationColor;
  sortOrder?: number;
  /** 仅 active↔archived 互转（承载归档/恢复） */
  status?: 'active' | 'archived';
}

/** 应用来源摘要：type=blank 空白创建 / template 模板安装（M2-B） */
export interface ApplicationSource {
  type: 'blank' | 'template';
  channel: 'self' | 'template_center' | 'admin' | 'api';
}

/** 当前成员的运行时能力（后端读取时派生，不落库） */
export interface ApplicationCapabilities {
  view: boolean;
  edit: boolean;
  delete: boolean;
}

/** 应用详情（创建/详情/列表条目共用，model.ApplicationDetail 字段子集） */
export interface ApplicationItem {
  id: number;
  code: string;
  name: string;
  icon: ApplicationIcon;
  color: ApplicationColor;
  source: ApplicationSource;
  status: 'active' | 'archived';
  provisionStatus: 'ready' | 'pending' | 'running' | 'failed';
  homeMode: ApplicationHomeMode;
  ownerMemberId: number;
  creatorMemberId: number;
  sortOrder: number;
  capabilities: ApplicationCapabilities;
  /** 后端 JSONTime 秒级东八区 yyyy-MM-dd HH:mm:ss */
  createdAt: string;
  updatedAt: string;
}

/** 应用列表游标分页结果：nextCursor 为空且 hasMore=false 即末页，游标只原样回传 */
export interface ApplicationPage {
  items: ApplicationItem[];
  nextCursor: string;
  hasMore: boolean;
}

/** 应用列表查询参数 */
export interface ApplicationListQuery {
  keyword?: string;
  status?: 'active' | 'archived';
  limit?: number;
  cursor?: string;
}

// ---------- 应用菜单 API 契约（M2-菜单-1，与后端 MenuSnapshot 出网对齐） ----------

/** 应用菜单节点类型：group 分组无资产引用，其余为可打开的资产节点 */
export type ApplicationMenuEntryType = 'group' | 'form' | 'dashboard' | 'page';

/** 菜单节点资产引用：code 为稳定公开编码；formType 仅属于表单目标。 */
export type ApplicationMenuTarget =
  | {
      type: 'form';
      code: string;
      formType: FormType;
    }
  | {
      type: 'dashboard';
      code: string;
    }
  | {
      type: 'page';
      code: string;
    };

/** 菜单节点运行时能力（后端按当前成员权限与应用状态读取时派生，不落库） */
export interface ApplicationMenuCapabilities {
  view: boolean;
  manage: boolean;
  move: boolean;
  delete: boolean;
  favorite: boolean;
}

/** 应用菜单节点（entryMap 的值；parentEntryId 为 null 即根节点） */
export interface ApplicationMenuEntry {
  entryId: string;
  parentEntryId: string | null;
  type: ApplicationMenuEntryType;
  name: string;
  /** 后端稳定图标键，可为 null；前端受控映射表转换为图标组件 */
  icon: string | null;
  color: string | null;
  sortOrder: number;
  target: ApplicationMenuTarget | null;
  capabilities: ApplicationMenuCapabilities;
}

/** 应用菜单快照（GET /applications/code/:code/menu）：entryMap 仅含当前
 * 成员可见节点，无可见后代的分组已被服务端裁剪；menuRevision 供后续
 * 管理接口做乐观并发；空菜单（rootEntryIds 为空数组）是合法结果 */
export interface ApplicationMenu {
  applicationCode: string;
  menuRevision: number;
  rootEntryIds: string[];
  entryMap: Record<string, ApplicationMenuEntry>;
  /** 只表达已注册的后端能力，流程引擎未接入前 workflow 恒 false */
  features: { workflow: boolean };
}

/** 创建应用菜单分组请求：parentEntryId 为空时创建根分组。 */
export interface CreateApplicationMenuGroupPayload {
  name: string;
  parentEntryId?: string;
  baseMenuRevision: number;
}

/** 分组创建后的增量结果；完整树仍以重新读取菜单快照为准。 */
export interface ApplicationMenuGroupMutation {
  entryId: string;
  parentEntryId: string | null;
  name: string;
  menuRevision: number;
}

// ---- 版本信息（管理后台「版本信息」页，一期：真实订阅与资源概览）----

/** 订阅状态：active 活动 / expired 已到期（读时投影）/ legacy_pending_review 有效期待确认 */
export type EditionSubscriptionStatus = 'active' | 'expired' | 'legacy_pending_review';

/** 计量状态：ready 已接入真实用量 / pending 待计量（不展示已用值） */
export type EditionMeteringStatus = 'ready' | 'pending';

/** 上限解析来源：套餐版本 / 租户覆盖 / 旧配额迁移 / 到期回退免费版 */
export type EditionLimitSource =
  | 'plan_version'
  | 'tenant_override'
  | 'legacy_quota'
  | 'expiry_fallback';

/** 当前订阅投影（到期未降级窗口即返回 expired，权益按免费版解析） */
export interface EditionSubscription {
  planCode: string;
  planName: string;
  status: EditionSubscriptionStatus;
  grantType: string;
  startsAt: string;
  endsAt?: string;
  expiresAction: 'none' | 'downgrade_to_free';
}

/** 资源容量项：pending 时省略 usage/usagePercent/asOf，不返回伪零值 */
export interface EditionQuota {
  key: string;
  category: string;
  name: string;
  unit: string;
  limit: number;
  usage?: number;
  usagePercent?: number;
  meteringStatus: EditionMeteringStatus;
  limitSource: EditionLimitSource;
  asOf?: string;
  resetCycle?: string;
}

/** 功能权益项：仅展示与前端入口裁剪，不替代 RBAC */
export interface EditionFeature {
  group: string;
  key: string;
  name: string;
  available: boolean;
  parameters?: Record<string, unknown>;
  description?: string;
}

/** GET /editions/current 响应：订阅 + 容量配额 + 功能权益 */
export interface CurrentEdition {
  subscription: EditionSubscription;
  quotas: EditionQuota[];
  features: EditionFeature[];
  asOf: string;
}

// ---- 表单资产域（ADR-010，见 docs/低代码平台/表单设计器/表单资产域后端契约.md） ----

/** 目标保存协议文档（唯一事实结构，与 packages/form schema 同源） */
export type FormSchemaDocument = import('@evolyn.do/form/schema').FormSchemaDocument;

/** 表单资产类型：由后端持久化，创建后不可变 */
export type FormType = 'standard' | 'workflow';

/** 表单详情（GET /forms/:code）：含草稿全文与修订口令 */
export interface FormDetail {
  applicationId: number;
  code: string;
  name: string;
  formType: FormType;
  draftRevision: number;
  publishedVersion: number;
  draft: FormSchemaDocument;
  createdAt: string;
  updatedAt: string;
}

/** 表单列表条目（不含草稿全文） */
export interface FormSummary {
  applicationId: number;
  code: string;
  name: string;
  formType: FormType;
  publishedVersion: number;
  updatedAt: string;
}

/** GET /forms 游标分页结果 */
export interface FormPage {
  items: FormSummary[];
  nextCursor: string;
  hasMore: boolean;
}

/** PUT /forms/:code/draft 结果：新口令供下次保存回传 */
export interface FormDraftSaveResult {
  draftRevision: number;
}

/** POST /forms/:code/publish 结果：发布双口令 */
export interface FormPublishResult {
  publishedVersion: number;
  schemaRevision: string;
}

/** GET /applications/code/:appCode/forms/:formCode/runtime 响应（运行时引导） */
export interface FormRuntimeBootstrap {
  formCode: string;
  name: string;
  publishedVersion: number;
  schemaRevision: string;
  content: FormSchemaDocument;
}

/** POST /form-records 受理结果 */
export interface FormRecordSubmitResult {
  recordId: number;
}
import type { EvolynIconPickerValue } from '@evolyn.do/ui';
