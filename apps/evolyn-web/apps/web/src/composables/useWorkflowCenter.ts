import type { ComputedRef, Ref } from 'vue';
import type {
  WorkflowInstanceSummaryDto,
  WorkflowPendingTaskSummaryDto,
  WorkflowTaskScope,
  WorkflowTaskSummaryDto,
} from '~/types';
import { computed, readonly, shallowRef, watch } from 'vue';
import {
  getWorkflowPendingTaskSummary,
  listStartedWorkflowInstances,
  listWorkflowTasks,
} from '~/api/workflow';

/** 审批中心四个个人视图；与路由 query 的 scope 保持同一套稳定值。 */
export type WorkflowCenterScope = WorkflowTaskScope | 'started';

/** 列表组件只消费统一的展示模型，避免把任务和实例 DTO 分支散落到模板。 */
export interface WorkflowCenterListItem {
  id: number;
  title: string;
  subtitle: string;
  status: string;
  createdAt: string;
  /** task 才能进入审批详情；instance 在首版仅展示进度摘要。 */
  source: 'task' | 'instance';
  taskId?: number;
  instanceId: number;
}

export type WorkflowCenterStatus = 'loading' | 'ready' | 'error';

function taskItem(task: WorkflowTaskSummaryDto): WorkflowCenterListItem {
  const actorNames = task.actors
    .map((actor) => actor.displayName)
    .filter(Boolean)
    .join('、');
  return {
    id: task.id,
    title: `待办 #${task.id}`,
    subtitle: `当前节点：${task.nodeKey}${actorNames ? ` · 审批人：${actorNames}` : ''}`,
    status: task.status,
    createdAt: task.createdAt,
    source: 'task',
    taskId: task.id,
    instanceId: task.instanceId,
  };
}

function instanceItem(instance: WorkflowInstanceSummaryDto): WorkflowCenterListItem {
  return {
    id: instance.id,
    title: instance.definitionCode,
    subtitle: `业务记录：${instance.businessId}`,
    status: instance.status,
    createdAt: instance.createdAt,
    source: 'instance',
    instanceId: instance.id,
  };
}

/**
 * 审批中心数据源：仅管理服务端分页和当前页内关键词筛选。
 * 现有后端列表协议尚未提供关键词条件，不能假装成全量远程搜索；待查询投影
 * 扩展后，只需要替换 load 内的请求参数，展示组件无需调整。
 */
export function useWorkflowCenter(
  scope: Readonly<Ref<WorkflowCenterScope>>,
  formCode: Readonly<Ref<string>>,
) {
  const items = shallowRef<WorkflowCenterListItem[]>([]);
  const keyword = shallowRef('');
  const nextCursor = shallowRef('');
  const status = shallowRef<WorkflowCenterStatus>('loading');
  const errorMessage = shallowRef('审批任务加载失败，请稍后重试');
  const pendingSummary = shallowRef<WorkflowPendingTaskSummaryDto | null>(null);
  let requestVersion = 0;

  const visibleItems: ComputedRef<WorkflowCenterListItem[]> = computed(() => {
    const normalized = keyword.value.trim().toLocaleLowerCase();
    if (!normalized) return items.value;
    return items.value.filter((item) =>
      `${item.title} ${item.subtitle} ${item.status}`.toLocaleLowerCase().includes(normalized),
    );
  });

  async function load(append = false): Promise<void> {
    const version = ++requestVersion;
    const cursor = append ? nextCursor.value : '';
    if (!append) {
      status.value = 'loading';
      items.value = [];
      nextCursor.value = '';
    }

    try {
      if (scope.value === 'started') {
        const page = await listStartedWorkflowInstances({ limit: 30, cursor });
        if (version !== requestVersion) return;
        const incoming = page.items.map(instanceItem);
        items.value = append ? [...items.value, ...incoming] : incoming;
        nextCursor.value = page.nextCursor;
      } else {
        const page = await listWorkflowTasks({
          scope: scope.value,
          limit: 30,
          cursor,
          // 发起/已办/抄送不带流程表单筛选；该筛选只属于「我的待办」展开菜单。
          formCode: scope.value === 'pending' ? formCode.value || undefined : undefined,
        });
        if (version !== requestVersion) return;
        const incoming = page.items.map(taskItem);
        items.value = append ? [...items.value, ...incoming] : incoming;
        nextCursor.value = page.nextCursor;
      }
      status.value = 'ready';
    } catch {
      if (version !== requestVersion) return;
      status.value = 'error';
      errorMessage.value = '审批任务加载失败，请稍后重试';
    }
  }

  function refresh(): Promise<void> {
    return load(false);
  }

  function loadMore(): Promise<void> {
    if (!nextCursor.value || status.value === 'loading') return Promise.resolve();
    return load(true);
  }

  async function refreshPendingSummary(): Promise<void> {
    try {
      pendingSummary.value = await getWorkflowPendingTaskSummary();
    } catch {
      // 摘要失败不能阻断审批列表；菜单暂时隐藏数字，用户仍可正常处理待办。
      pendingSummary.value = null;
    }
  }

  async function refreshAll(): Promise<void> {
    await Promise.all([refresh(), refreshPendingSummary()]);
  }

  watch([scope, formCode], () => void refresh(), { immediate: true });
  void refreshPendingSummary();

  return {
    items: readonly(items),
    visibleItems,
    keyword,
    nextCursor: readonly(nextCursor),
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    pendingSummary: readonly(pendingSummary),
    refresh,
    refreshAll,
    refreshPendingSummary,
    loadMore,
  };
}
