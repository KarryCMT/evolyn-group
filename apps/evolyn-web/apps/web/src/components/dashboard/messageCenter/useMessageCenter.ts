import type {
  MessageAction,
  MessageCategory,
  MessageCategoryId,
  MessageCenterView,
  MessageDeliveryChannel,
  MessagePreference,
  MessageRecord,
  ReminderRecipient,
  ReminderRecipientInput,
} from './messageCenter.types';
import { ERROR_CODES } from '@evolyn.do/utils';
import { ElMessage } from 'element-plus';
import { computed, ref, shallowRef, watch } from 'vue';
import { useRouter } from 'vue-router';
import {
  listNotifications,
  markNotificationRead,
  markNotificationsAllRead,
  type NotificationUnreadSummary,
} from '~/api/notifications';
import {
  createNotificationRecipient,
  deleteNotificationRecipient,
  getNotificationSettings,
  listNotificationRecipients,
  patchNotificationPreference,
  type NotificationChannel,
  type NotificationChannelCapability,
  type NotificationRecipientKind,
  type NotificationSettingAggregate,
} from '~/api/notificationSettings';
import { useNotificationStore } from '~/stores/notification';
import { fallbackMessageCategories } from './messageCenter.constants';

/** 列表分页默认值（与服务端一致：默认 20，上限 100） */
const PAGE_LIMIT = 20;

/**
 * 受控跳转动作注册表（消息中心 P1）：action.type 仅白名单内动作允许导航，
 * 参数按动作白名单透传；到达目标页后仍执行目标资源的正常鉴权，消息本身
 * 不是访问凭证。未登记动作码不导航。
 */
const ACTION_ROUTES: Record<
  string,
  (action: MessageAction) => { name: string; params?: Record<string, string> }
> = {
  open_application: (action) => ({ name: 'App', params: { appCode: action.appCode } }),
};

/**
 * 消息中心的功能组装 composable（消息中心 P1/P2 接入改造）：
 * - 收件箱为服务端游标分页数据（分类/筛选变化重置游标，请求序号防竞态）；
 * - 未读摘要收敛到 Pinia notification store（两个顶栏共读单一事实源）；
 * - 通知设置/提醒对象为服务端数据，写操作采用等待服务端成功的保守更新。
 */
export function useMessageCenter() {
  const router = useRouter();
  const notificationStore = useNotificationStore();

  // ---- 视图与筛选状态 ----
  const activeCategoryId = shallowRef<MessageCategoryId>('app-log');
  const activeView = shallowRef<MessageCenterView>('inbox');
  const showUnreadOnly = shallowRef(false);
  const activeEventCode = shallowRef('');

  // ---- 收件箱服务端数据 ----
  const records = ref<MessageRecord[]>([]);
  const inboxLoading = shallowRef(false);
  const inboxError = shallowRef(false);
  const hasMore = shallowRef(false);
  const nextCursor = shallowRef('');
  const retentionMonths = shallowRef(6);
  const serverTime = shallowRef('');
  /** 请求序号：分类/筛选快速切换时丢弃旧响应，防止过期数据覆盖新状态 */
  let requestSeq = 0;

  // ---- 通知设置服务端数据 ----
  const settingsAggregate = shallowRef<NotificationSettingAggregate | null>(null);
  const settingsLoading = shallowRef(false);
  /** 普通成员无 notification-settings 权限：403 时隐藏设置入口（后端仍是最终边界） */
  const settingsDenied = shallowRef(false);
  const recipients = ref<ReminderRecipient[]>([]);
  const recipientSaving = shallowRef(false);

  // ---- 分类目录：服务端设置聚合下发，失败回落兜底常量 ----
  const categories = computed<MessageCategory[]>(() => {
    if (settingsAggregate.value) {
      return settingsAggregate.value.categories.map((category) => ({
        id: category.id as MessageCategoryId,
        label: category.label,
        group: category.group,
      }));
    }
    return fallbackMessageCategories;
  });

  const activeCategory = computed(
    () =>
      categories.value.find((item) => item.id === activeCategoryId.value) ?? categories.value[0],
  );
  const activeCategoryLabel = computed(() => activeCategory.value?.label ?? '');
  const unreadCount = computed(() => notificationStore.unreadTotal);
  const unreadCountByCategory = computed(() => notificationStore.unreadCountByCategory);

  /** 当前分类的事件筛选选项（服务端目录；空则隐藏下拉） */
  const activeEventOptions = computed(() => {
    const category = settingsAggregate.value?.categories.find(
      (item) => item.id === activeCategoryId.value,
    );
    return category?.events.map((event) => ({ code: event.code, label: event.label })) ?? [];
  });

  /** 当前分类的有效偏好（设置页表格数据源） */
  const activePreferences = computed<MessagePreference[]>(() => {
    const category = settingsAggregate.value?.categories.find(
      (item) => item.id === activeCategoryId.value,
    );
    if (!category) return [];
    return category.events.map((event) => ({
      code: event.code,
      label: event.label,
      categoryId: category.id as MessageCategoryId,
      severity: event.severity as MessagePreference['severity'],
      supportedChannels: event.supportedChannels,
      lockedChannels: event.lockedChannels,
      channels: { ...event.channels },
      recipients: event.recipients.map((recipient) => ({ ...recipient })),
    }));
  });

  /** 渠道能力（email/sms 能力未就绪时禁用勾选并提示原因） */
  const channelCapabilities = computed<Record<string, NotificationChannelCapability>>(
    () =>
      settingsAggregate.value?.channelCapabilities ?? { system: { available: true, reason: '' } },
  );
  const smsBudget = computed(() => settingsAggregate.value?.smsBudget ?? null);

  // ---- 收件箱加载 ----

  /** 加载当前分类首页（重置游标）；loadingMore 追加下一页。 */
  async function loadInbox(loadingMore = false): Promise<void> {
    const seq = ++requestSeq;
    if (!loadingMore) {
      inboxLoading.value = true;
      inboxError.value = false;
    }
    try {
      const page = await listNotifications({
        categoryId: activeCategoryId.value,
        eventCode: activeEventCode.value || undefined,
        unreadOnly: showUnreadOnly.value,
        cursor: loadingMore ? nextCursor.value : undefined,
        limit: PAGE_LIMIT,
      });
      if (seq !== requestSeq) return; // 过期响应丢弃
      // 服务端 categoryId 为稳定分类码（string），落库到视图层强类型前收敛
      const items = page.items as unknown as MessageRecord[];
      if (loadingMore) {
        records.value = [...records.value, ...items];
      } else {
        records.value = items;
      }
      hasMore.value = page.hasMore;
      nextCursor.value = page.nextCursor;
      retentionMonths.value = page.retentionMonths;
      serverTime.value = page.serverTime;
    } catch {
      if (seq === requestSeq) inboxError.value = true;
    } finally {
      if (seq === requestSeq && !loadingMore) inboxLoading.value = false;
    }
  }

  /** 分类/事件筛选/只看未读变化：重置游标重新加载首页。 */
  watch([activeCategoryId, activeEventCode, showUnreadOnly], () => {
    if (activeView.value !== 'inbox') return;
    void loadInbox();
  });

  function selectCategory(categoryId: MessageCategoryId) {
    activeCategoryId.value = categoryId;
    activeView.value = 'inbox';
  }

  /** 设置页切换 Tab 时保留当前设置视图，不返回收件箱。 */
  function selectSettingsCategory(categoryId: MessageCategoryId) {
    activeCategoryId.value = categoryId;
  }

  // ---- 通知设置 ----

  /** 加载设置聚合：分类/事件目录 + 有效偏好 + 渠道能力 + revision。 */
  async function loadSettings(force = false): Promise<void> {
    if (settingsLoading.value && !force) return;
    if (settingsDenied.value && !force) return;
    settingsLoading.value = true;
    try {
      settingsAggregate.value = await getNotificationSettings();
      settingsDenied.value = false;
      await loadRecipients();
    } catch (error) {
      // 403：普通成员隐藏「通知设置」入口（后端权限仍是最终边界）
      if ((error as { status?: number })?.status === 403) {
        settingsDenied.value = true;
        settingsAggregate.value = null;
      }
    } finally {
      settingsLoading.value = false;
    }
  }

  function openSettings() {
    activeView.value = 'settings';
    if (!settingsAggregate.value && !settingsDenied.value) void loadSettings();
  }

  async function loadRecipients(): Promise<void> {
    try {
      recipients.value = await listNotificationRecipients();
    } catch {
      recipients.value = [];
    }
  }

  /**
   * 渠道开关保守更新：等待服务端成功后以响应整行覆盖；409 冲突时重载最新
   * 配置并提示已被其他管理员修改（不伪造持久成功）。
   */
  async function updatePreference(
    eventCode: string,
    channel: MessageDeliveryChannel,
    checked: boolean,
  ): Promise<void> {
    const aggregate = settingsAggregate.value;
    if (!aggregate) return;
    try {
      const result = await patchNotificationPreference(eventCode, {
        revision: aggregate.revision,
        channels: { [channel]: checked } as Partial<Record<NotificationChannel, boolean>>,
      });
      applyPreferencePatch(result.revision, result.event);
    } catch (error) {
      await handleSettingsError(error);
    }
  }

  /**
   * 接收规则全量替换（事件「修改」入口）：动态规则与自定义联系人一次提交；
   * 成功后以响应投影覆盖并刷新联系人列表（引用关系可能变化）。
   */
  async function replacePreferenceRecipients(
    eventCode: string,
    next: { kind: string; recipientId?: number }[],
  ): Promise<boolean> {
    const aggregate = settingsAggregate.value;
    if (!aggregate) return false;
    try {
      const result = await patchNotificationPreference(eventCode, {
        revision: aggregate.revision,
        // kind 为服务端稳定枚举，经选择器产出（未知类型由服务端终审拒绝）
        recipients: next as { kind: NotificationRecipientKind; recipientId?: number }[],
      });
      applyPreferencePatch(result.revision, result.event);
      return true;
    } catch (error) {
      await handleSettingsError(error);
      return false;
    }
  }

  /** 以补丁响应就地更新聚合内对应事件行（含新 revision）。 */
  function applyPreferencePatch(
    revision: number,
    event: NotificationSettingAggregate['categories'][number]['events'][number],
  ): void {
    const aggregate = settingsAggregate.value;
    if (!aggregate) return;
    const category = aggregate.categories.find((item) =>
      item.events.some((entry) => entry.code === event.code),
    );
    if (!category) return;
    const index = category.events.findIndex((entry) => entry.code === event.code);
    if (index >= 0) category.events[index] = event;
    aggregate.revision = revision;
  }

  /** 设置写入失败统一处理：409 冲突重载最新配置，其余透出服务端文案。 */
  async function handleSettingsError(error: unknown): Promise<void> {
    const errCode = (error as { errCode?: string })?.errCode;
    if (errCode === ERROR_CODES.NOTIFICATION_SETTINGS_CONFLICT) {
      ElMessage.warning('通知设置已被其他管理员修改，已为您加载最新配置');
      await loadSettings(true);
      return;
    }
    if (errCode === ERROR_CODES.NOTIFICATION_RECIPIENT_IN_USE) {
      const usedBy = (error as { data?: { usedByEventCodes?: string[] } })?.data?.usedByEventCodes;
      ElMessage.warning(
        usedBy?.length
          ? `该提醒对象仍被「${usedBy.join('、')}」事件引用，请先在对应事件中移除`
          : '该提醒对象仍被事件偏好引用，请先移除引用',
      );
      return;
    }
    const message = (error as { message?: string })?.message;
    ElMessage.error(message || '操作失败，请稍后重试');
  }

  // ---- 提醒对象（联系人池） ----

  function openRecipientManagement() {
    activeView.value = 'recipient-management';
    void loadRecipients();
  }

  /** 新增提醒对象：携带聚合 revision；成功后刷新列表并同步 revision。 */
  async function addRecipient(payload: ReminderRecipientInput): Promise<boolean> {
    const aggregate = settingsAggregate.value;
    if (!aggregate || recipientSaving.value) return false;
    recipientSaving.value = true;
    try {
      await createNotificationRecipient({ ...payload, revision: aggregate.revision });
      await loadSettings(true);
      return true;
    } catch (error) {
      await handleSettingsError(error);
      return false;
    } finally {
      recipientSaving.value = false;
    }
  }

  /** 删除未被偏好引用的提醒对象（在用时服务端返回 409 + usedByEventCodes）。 */
  async function removeRecipient(recipientId: number): Promise<void> {
    const target = recipients.value.find((item) => item.id === recipientId);
    if (!target) return;
    try {
      await deleteNotificationRecipient(recipientId, settingsAggregate.value?.revision ?? 0);
      await loadSettings(true);
    } catch (error) {
      await handleSettingsError(error);
    }
  }

  // ---- 已读与跳转 ----

  /** 单击消息：先幂等已读（成功后同步 store），再按 action 白名单导航。 */
  async function markAsRead(messageId: number): Promise<void> {
    const record = records.value.find((item) => item.id === messageId);
    if (record && !record.read) {
      try {
        const summary = await markNotificationRead(messageId);
        notificationStore.applySummary(summary as NotificationUnreadSummary);
        record.read = true;
      } catch {
        // 已读失败不伪造成功、不阻断跳转提示
        ElMessage.error('标记已读失败，请稍后重试');
        return;
      }
    }
    navigateByAction(record?.action);
  }

  /** 当前分类全部已读：through 回传本次列表 serverTime，不误伤新到达消息。 */
  async function markAllAsRead(): Promise<void> {
    try {
      const summary = await markNotificationsAllRead({
        categoryId: activeCategoryId.value,
        eventCode: activeEventCode.value || undefined,
        through: serverTime.value || undefined,
      });
      notificationStore.applySummary(summary);
      await loadInbox();
    } catch {
      ElMessage.error('操作失败，请稍后重试');
    }
  }

  /** 按动作注册表导航：未登记动作码不导航（非白名单 action 一律忽略）。 */
  function navigateByAction(action?: MessageAction): void {
    if (!action?.type) return;
    const routeFor = ACTION_ROUTES[action.type];
    if (!routeFor) return;
    void router.push(routeFor(action));
  }

  /** 抽屉打开时刷新未读摘要与首页数据（消息量少时的轻量同步口径）。 */
  function onDrawerOpen(): void {
    void notificationStore.load();
    if (activeView.value === 'inbox') void loadInbox();
  }

  return {
    activeCategoryId,
    activeCategoryLabel,
    activeEventCode,
    activeEventOptions,
    activePreferences,
    activeView,
    categories,
    channelCapabilities,
    hasMore,
    inboxError,
    inboxLoading,
    loadInbox,
    openRecipientManagement,
    openSettings,
    recipientSaving,
    recipients,
    records,
    retentionMonths,
    removeRecipient,
    addRecipient,
    replacePreferenceRecipients,
    selectCategory,
    selectSettingsCategory,
    settingsAggregate,
    settingsDenied,
    settingsLoading,
    showUnreadOnly,
    smsBudget,
    unreadCount,
    unreadCountByCategory,
    markAllAsRead,
    markAsRead,
    onDrawerOpen,
    updatePreference,
  };
}
