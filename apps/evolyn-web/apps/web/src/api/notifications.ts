// 消息中心收件箱接口：与后端 /api/v1/notifications 一一对应
// （见 evolyn-core internal/platform/notification/controller/notification.go）
import { http } from '@evolyn.do/utils';

/** 消息严重等级（服务端事件目录枚举） */
export type NotificationSeverity = 'info' | 'success' | 'warning' | 'error';

/** 受控跳转动作：仅稳定动作码与白名单参数（前端 action registry 映射路由） */
export interface NotificationAction {
  type: string;
  [key: string]: string;
}

/** 消息列表项（纯文本展示快照；readAt 为空串表示未读） */
export interface NotificationItem {
  id: number;
  categoryId: string;
  eventCode: string;
  eventLabel: string;
  severity: NotificationSeverity;
  title: string;
  content: string;
  createdAt: string;
  read: boolean;
  readAt: string;
  action: NotificationAction;
}

/** 游标分页响应：nextCursor 原样回传取下一页；serverTime 供批量已读 through 口令 */
export interface NotificationPage {
  items: NotificationItem[];
  nextCursor: string;
  hasMore: boolean;
  retentionMonths: number;
  serverTime: string;
}

/** 未读摘要：只含未读数大于 0 的分类（顶栏红点单一事实源） */
export interface NotificationUnreadSummary {
  unreadTotal: number;
  categories: { categoryId: string; unreadCount: number }[];
  generatedAt: string;
}

/** 收件箱列表查询：categoryId 必填；eventCode 必须属于当前分类 */
export interface NotificationListQuery {
  categoryId: string;
  eventCode?: string;
  unreadOnly?: boolean;
  cursor?: string;
  limit?: number;
}

/** 批量已读请求：through 回传本次列表响应的 serverTime（不误伤新到达消息） */
export interface NotificationReadAllPayload {
  categoryId: string;
  eventCode?: string;
  through?: string;
}

/** 查询当前成员未读摘要（顶栏红点数据源）。 */
export function getNotificationUnreadSummary(): Promise<NotificationUnreadSummary> {
  return http.get('/notifications/unread-summary');
}

/** 查询当前成员分类消息列表（游标分页，按事件时间倒序）。 */
export function listNotifications(query: NotificationListQuery): Promise<NotificationPage> {
  return http.get('/notifications', {
    categoryId: query.categoryId,
    eventCode: query.eventCode,
    unreadOnly: query.unreadOnly,
    cursor: query.cursor,
    limit: query.limit,
  });
}

/** 幂等标记单条已读：响应携带最新未读摘要，避免二次请求。 */
export function markNotificationRead(id: number): Promise<NotificationUnreadSummary> {
  return http.put(`/notifications/${id}/read`);
}

/** 标记当前分类（可选事件）through 之前的未读消息为已读。 */
export function markNotificationsAllRead(
  payload: NotificationReadAllPayload,
): Promise<NotificationUnreadSummary> {
  return http.put('/notifications/read-all', payload);
}
