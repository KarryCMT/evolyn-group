import type { MessageCategory, MessagePreference, MessageRecord } from './messageCenter.types';

/** 分类顺序与产品信息架构一致，供侧栏和通知设置 Tab 共同使用。 */
export const messageCategories: MessageCategory[] = [
  { id: 'data-reminder', label: '数据提醒', group: 'product' },
  { id: 'app-log', label: '应用日志', group: 'product' },
  { id: 'document-activity', label: '文档动态', group: 'product' },
  { id: 'usage-reminder', label: '用量提醒', group: 'enterprise' },
  { id: 'contacts-management', label: '通讯录管理', group: 'enterprise' },
  { id: 'open-platform', label: '开放平台', group: 'enterprise' },
  { id: 'system-management', label: '系统管理', group: 'enterprise' },
  { id: 'operation-notice', label: '运营通知', group: 'enterprise' },
];

/** 接口未落地前的视觉与交互样本；后续由查询接口替换即可。 */
export const initialMessageRecords: MessageRecord[] = [
  {
    id: 'message-app-log-1',
    categoryId: 'app-log',
    createdAt: '2026-08-21 13:46',
    content: '李同学 创建了应用「CRM_精斗云」',
    read: false,
  },
  {
    id: 'message-app-log-2',
    categoryId: 'app-log',
    createdAt: '2026-08-21 11:25',
    content: '李同学 创建了应用「CRM_云星辰多账套」',
    read: false,
  },
  {
    id: 'message-app-log-3',
    categoryId: 'app-log',
    createdAt: '2026-08-21 11:20',
    content: '李同学 创建了应用「CRM_云星辰」',
    read: true,
  },
  {
    id: 'message-app-log-4',
    categoryId: 'app-log',
    createdAt: '2026-08-14 23:21',
    content: '李同学 创建了应用「灵衍云高级功能介绍」',
    read: true,
  },
  {
    id: 'message-system-1',
    categoryId: 'system-management',
    createdAt: '2026-08-19 09:12',
    content: '系统已完成本次功能更新，相关服务运行正常。',
    read: false,
  },
];

/** 通知设置的本地初始状态；服务端接入后以租户偏好设置覆盖。 */
export const initialMessagePreferences: MessagePreference[] = [
  {
    id: 'data-push',
    categoryId: 'app-log',
    label: '数据推送提醒',
    channels: { system: true, email: false, sms: true },
    recipients: ['创建者', '系统管理员'],
  },
  {
    id: 'assistant-failed',
    categoryId: 'app-log',
    label: '智能助手执行失败',
    channels: { system: true, email: false, sms: false },
    recipients: ['创建者', '系统管理员'],
  },
  {
    id: 'flow-failed',
    categoryId: 'app-log',
    label: '数据流执行失败',
    channels: { system: true, email: false, sms: false },
    recipients: ['创建者', '系统管理员'],
  },
  {
    id: 'export-failed',
    categoryId: 'app-log',
    label: '输出表同步失败',
    channels: { system: true, email: false, sms: false },
    recipients: ['创建者', '系统管理员'],
  },
];
