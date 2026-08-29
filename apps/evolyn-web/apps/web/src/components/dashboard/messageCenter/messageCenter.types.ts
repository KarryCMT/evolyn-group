/** 消息中心分类稳定编码（只增不改；展示名/分组由服务端目录下发，
 * approval 审批动态随流程引擎 Phase 6 追增）。 */
export type MessageCategoryId =
  | 'data-reminder'
  | 'app-log'
  | 'approval'
  | 'document-activity'
  | 'usage-reminder'
  | 'contacts-management'
  | 'open-platform'
  | 'system-management'
  | 'operation-notice';

/** 抽屉内在收件箱、通知设置与提醒对象管理之间切换。 */
export type MessageCenterView = 'inbox' | 'settings' | 'recipient-management';

/** 左侧分类：group 用于渲染「灵衍云 / 企业消息」分区标题。 */
export interface MessageCategory {
  id: MessageCategoryId;
  label: string;
  group: 'product' | 'enterprise';
}

/** 消息严重等级（服务端事件目录枚举）。 */
export type MessageSeverity = 'info' | 'success' | 'warning' | 'error';

/** 受控跳转动作：仅稳定动作码与白名单参数，前端 action registry 映射路由。 */
export interface MessageAction {
  type: string;
  [key: string]: string;
}

/** 消息记录（对齐后端收件箱 DTO；readAt 空串表示未读）。 */
export interface MessageRecord {
  id: number;
  categoryId: MessageCategoryId;
  eventCode: string;
  eventLabel: string;
  severity: MessageSeverity;
  title: string;
  content: string;
  createdAt: string;
  read: boolean;
  readAt: string;
  action: MessageAction;
}

export type MessageDeliveryChannel = 'system' | 'email' | 'sms';

/** 接收对象类型：动态规则 + 自定义外部联系人（后者仅邮件/短信渠道）。 */
export type MessageRecipientKind =
  | 'event_actor'
  | 'event_audience'
  | 'tenant_admin'
  | 'custom_recipient';

/** 接收对象视图：动态规则按 kind 出中文标签，自定义联系人追加姓名。 */
export interface MessageRecipient {
  kind: MessageRecipientKind;
  label: string;
  recipientId?: number;
}

/** 设置页中一条事件的有效投递偏好（覆盖行??注册表默认投影）。 */
export interface MessagePreference {
  code: string;
  label: string;
  categoryId: MessageCategoryId;
  severity: MessageSeverity;
  supportedChannels: MessageDeliveryChannel[];
  lockedChannels: MessageDeliveryChannel[];
  channels: Record<MessageDeliveryChannel, boolean>;
  recipients: MessageRecipient[];
}

/** 自定义提醒对象（完整联系方式仅通知设置管理员可达）。 */
export interface ReminderRecipient {
  id: number;
  name: string;
  mobile: string;
  email: string;
  revision: number;
}

/** 新增提醒对象表单的最小输入（手机/邮箱至少一项）。 */
export type ReminderRecipientInput = Omit<ReminderRecipient, 'id' | 'revision'>;
