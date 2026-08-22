/** 消息中心左侧的业务分类。 */
export type MessageCategoryId =
  | 'data-reminder'
  | 'app-log'
  | 'document-activity'
  | 'usage-reminder'
  | 'contacts-management'
  | 'open-platform'
  | 'system-management'
  | 'operation-notice';

/** 抽屉内在收件箱、通知设置与提醒对象管理之间切换。 */
export type MessageCenterView = 'inbox' | 'settings' | 'recipient-management';

/** 左侧分类：group 用于渲染「简道云 / 企业消息」分区标题。 */
export interface MessageCategory {
  id: MessageCategoryId;
  label: string;
  group: 'product' | 'enterprise';
}

/** 消息记录。后端接口接入后保持此展示模型即可。 */
export interface MessageRecord {
  id: string;
  categoryId: MessageCategoryId;
  createdAt: string;
  content: string;
  read: boolean;
}

export type MessageDeliveryChannel = 'system' | 'email' | 'sms';

/** 设置页中一条消息类型的投递方式和接收对象。 */
export interface MessagePreference {
  id: string;
  categoryId: MessageCategoryId;
  label: string;
  channels: Record<MessageDeliveryChannel, boolean>;
  recipients: string[];
}

/** 可接收提醒的自定义对象；后续可直接映射至后端 DTO。 */
export interface ReminderRecipient {
  id: string;
  name: string;
  mobile: string;
  email: string;
}

/** 新增提醒对象表单的最小输入。 */
export type ReminderRecipientInput = Omit<ReminderRecipient, 'id'>;
