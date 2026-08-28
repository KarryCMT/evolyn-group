// 通知设置接口：与后端 /api/v1/notification-settings 一一对应
// （见 evolyn-core internal/platform/notification/controller/setting.go）
import { http } from '@evolyn.do/utils';

/** 投递渠道（system 站内信 / email 邮件 / sms 短信） */
export type NotificationChannel = 'system' | 'email' | 'sms';

/** 接收对象类型：动态规则 + 自定义外部联系人（后者仅邮件/短信） */
export type NotificationRecipientKind =
  | 'event_actor'
  | 'event_audience'
  | 'tenant_admin'
  | 'custom_recipient';

export interface NotificationRecipientView {
  kind: NotificationRecipientKind;
  label: string;
  recipientId?: number;
}

/** 事件有效偏好：channels 为 system/email/sms 有效开关（覆盖行??注册表默认） */
export interface NotificationEventSetting {
  code: string;
  label: string;
  severity: string;
  supportedChannels: NotificationChannel[];
  lockedChannels: NotificationChannel[];
  channels: Record<NotificationChannel, boolean>;
  recipients: NotificationRecipientView[];
}

/** 设置页分类目录：configurable=false 表示尚无可配置事件（仍出现在收件箱目录） */
export interface NotificationCategorySetting {
  id: string;
  label: string;
  group: 'product' | 'enterprise';
  configurable: boolean;
  events: NotificationEventSetting[];
}

/** 渠道能力：available=false 时禁用勾选并展示 reason（P3 前 email/sms 恒不可用） */
export interface NotificationChannelCapability {
  available: boolean;
  reason: string;
}

/** 设置聚合：目录 + 有效偏好 + 渠道能力 + 聚合 revision（乐观锁口令） */
export interface NotificationSettingAggregate {
  revision: number;
  categories: NotificationCategorySetting[];
  channelCapabilities: Record<NotificationChannel, NotificationChannelCapability>;
  /** 云币/短信额度：计费事实源未接入时为 null，前端隐藏数值摘要 */
  smsBudget: null | { coinBalance: number; smsUnitCost: number; remainingCount: number };
}

/** 偏好更新：channels 部分更新（缺省键不变）；recipients 出现即全量替换 */
export interface NotificationPreferencePatch {
  revision: number;
  channels?: Partial<Record<NotificationChannel, boolean>>;
  recipients?: { kind: NotificationRecipientKind; recipientId?: number }[];
}

/** 偏好更新响应：该事件的新有效偏好 + 新聚合 revision */
export interface NotificationPreferencePatchResult {
  revision: number;
  event: NotificationEventSetting;
}

/** 自定义提醒对象（完整联系方式仅设置管理员可达） */
export interface NotificationCustomRecipient {
  id: number;
  name: string;
  mobile: string;
  email: string;
  revision: number;
}

export interface CreateNotificationRecipientPayload {
  revision: number;
  name: string;
  mobile?: string;
  email?: string;
}

/** 查询通知设置聚合（分类/事件目录 + 租户有效偏好 + 渠道能力）。 */
export function getNotificationSettings(): Promise<NotificationSettingAggregate> {
  return http.get('/notification-settings');
}

/** 更新事件偏好：revision 过期返回 409（NOTIFICATION_SETTINGS_CONFLICT）。 */
export function patchNotificationPreference(
  eventCode: string,
  payload: NotificationPreferencePatch,
): Promise<NotificationPreferencePatchResult> {
  return http.patch(`/notification-settings/preferences/${eventCode}`, payload);
}

/** 查询自定义提醒对象列表。 */
export function listNotificationRecipients(): Promise<NotificationCustomRecipient[]> {
  return http.get('/notification-settings/recipients');
}

/** 新增自定义提醒对象（手机/邮箱至少一项，服务端校验）。 */
export function createNotificationRecipient(
  payload: CreateNotificationRecipientPayload,
): Promise<NotificationCustomRecipient> {
  return http.post('/notification-settings/recipients', payload);
}

/** 删除未被偏好引用的提醒对象（在用返回 409 与 usedByEventCodes）。 */
export function deleteNotificationRecipient(id: number, revision: number): Promise<null> {
  return http.delete(`/notification-settings/recipients/${id}?revision=${revision}`);
}
