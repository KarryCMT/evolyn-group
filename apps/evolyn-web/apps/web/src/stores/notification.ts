import { defineStore } from 'pinia';
import { computed, ref, shallowRef } from 'vue';
import { getNotificationUnreadSummary, type NotificationUnreadSummary } from '~/api/notifications';

// 消息中心未读摘要 store（消息中心 P1）：会话级未读数的唯一事实源——两个
// 顶栏入口（TopNavigation / ApplicationWorkspaceHeader）与消息中心抽屉都
// 只读这里，单条/批量已读成功后以服务端响应的摘要覆盖本地，不再依赖各抽屉
// 实例各自的 unreadChange 事件。切换租户/退出登录时由 auth store 联动清空
export const useNotificationStore = defineStore('notification', () => {
  const summary = ref<NotificationUnreadSummary | null>(null);
  const loading = shallowRef(false);

  const unreadTotal = computed(() => summary.value?.unreadTotal ?? 0);
  /** 分类 → 未读数（只含大于 0 的分类；侧栏徽标读取） */
  const unreadCountByCategory = computed<Record<string, number>>(() => {
    const counts: Record<string, number> = {};
    for (const category of summary.value?.categories ?? []) {
      counts[category.categoryId] = category.unreadCount;
    }
    return counts;
  });

  /** 拉取/刷新未读摘要（401 等错误静默——顶栏红点不阻断主流程）。 */
  async function load(): Promise<NotificationUnreadSummary | null> {
    loading.value = true;
    try {
      summary.value = await getNotificationUnreadSummary();
      return summary.value;
    } catch {
      return null;
    } finally {
      loading.value = false;
    }
  }

  /** 以服务端已读操作的响应摘要覆盖本地（已读接口响应即最新摘要）。 */
  function applySummary(next: NotificationUnreadSummary): void {
    summary.value = next;
  }

  /** 本地即时递减（点击已读的乐观反馈；下次 load 校正）。 */
  function decrement(categoryId: string): void {
    if (!summary.value) return;
    const categories = summary.value.categories
      .map((category) =>
        category.categoryId === categoryId
          ? { ...category, unreadCount: Math.max(0, category.unreadCount - 1) }
          : category,
      )
      .filter((category) => category.unreadCount > 0);
    const total = Math.max(0, summary.value.unreadTotal - 1);
    summary.value = { ...summary.value, unreadTotal: total, categories };
  }

  /** 会话结束清空（登出/切换租户时由 auth store 调用）。 */
  function clear(): void {
    summary.value = null;
  }

  return {
    summary,
    loading,
    unreadTotal,
    unreadCountByCategory,
    load,
    applySummary,
    decrement,
    clear,
  };
});
