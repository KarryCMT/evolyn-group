import type { MessageCategory } from './messageCenter.types';

/**
 * 分类码是历史数据与路由的稳定契约，只增不改。展示名与分组以服务端设置
 * 聚合/未读接口下发的目录为准；此处仅作为服务端目录加载失败时的兜底展示，
 * 不再作为业务事实源（消息中心 P1 接入改造）。
 */
export const fallbackMessageCategories: MessageCategory[] = [
  { id: 'data-reminder', label: '数据提醒', group: 'product' },
  { id: 'app-log', label: '应用日志', group: 'product' },
  { id: 'approval', label: '审批动态', group: 'product' },
  { id: 'document-activity', label: '文档动态', group: 'product' },
  { id: 'usage-reminder', label: '用量提醒', group: 'enterprise' },
  { id: 'contacts-management', label: '通讯录管理', group: 'enterprise' },
  { id: 'open-platform', label: '开放平台', group: 'enterprise' },
  { id: 'system-management', label: '系统管理', group: 'enterprise' },
  { id: 'operation-notice', label: '运营通知', group: 'enterprise' },
];
