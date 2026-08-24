import { pluginFunctionUpdate, type PluginFunctionUpdateType } from '../api';
import type { PluginDesignFunction, PluginDesignFunctionSortComplete } from '../types';

interface UseFormFunctionUpdateQueueOptions {
  /** 将设计器函数转换为后端更新接口完整参数。 */
  createUpdatePayload: (functionData: PluginDesignFunction) => PluginFunctionUpdateType | null;
  /** 获取当前抽屉会话标识，用于丢弃旧插件的排序任务。 */
  getSession: () => number;
}

interface FormFunctionSortTask {
  /** 抽屉会话标识，防止旧插件的排序请求影响新会话。 */
  session: number;
  /** 本次拖拽中 seq 发生变化且已由后端保存的函数。 */
  functions: PluginDesignFunction[];
  onComplete: PluginDesignFunctionSortComplete;
}

/**
 * 统一调度函数内容保存和拖拽排序请求，避免完整更新参数之间发生竞态覆盖。
 * @param options 更新参数转换函数及当前抽屉会话读取函数。
 */
export const useFormFunctionUpdateQueue = ({
  createUpdatePayload,
  getSession,
}: UseFormFunctionUpdateQueueOptions) => {
  // 同一函数按触发顺序串行提交，确保较新的名称、代码或 seq 最后写入后端。
  const functionUpdateQueues = new Map<string, Promise<unknown>>();
  // 排序请求串行执行，并只保留等待中的最新顺序。
  let pendingSortTask: FormFunctionSortTask | null = null;
  let sortSaving = false;

  /** 将同一函数的请求工厂加入串行队列。 */
  const enqueueFunctionRequest = (
    functionId: string,
    requestFactory: () => Promise<unknown>,
  ): Promise<unknown> => {
    const previousRequest = functionUpdateQueues.get(functionId) || Promise.resolve();
    const currentRequest = previousRequest.catch(() => undefined).then(requestFactory);
    functionUpdateQueues.set(functionId, currentRequest);
    const clearQueue = () => {
      if (functionUpdateQueues.get(functionId) === currentRequest) {
        functionUpdateQueues.delete(functionId);
      }
    };
    void currentRequest.then(clearQueue, clearQueue);
    return currentRequest;
  };

  /** 将已经生成完整参数的函数更新请求加入串行队列。 */
  const enqueueFunctionUpdate = (payload: PluginFunctionUpdateType): Promise<unknown> =>
    enqueueFunctionRequest(String(payload.id), () => pluginFunctionUpdate(payload));

  /** 排序请求真正执行前读取最新函数内容，避免完整 payload 覆盖刚保存的名称或代码。 */
  const enqueueFunctionSortUpdate = (functionData: PluginDesignFunction): Promise<unknown> =>
    enqueueFunctionRequest(functionData.id, () => {
      const payload = createUpdatePayload(functionData);
      return payload
        ? pluginFunctionUpdate(payload)
        : Promise.reject(new Error('Invalid plugin function sort payload'));
    });

  /** 串行提交拖拽排序，保证后端最终接收最新一次分组顺序。 */
  const flushSortQueue = async () => {
    if (sortSaving) return;
    sortSaving = true;
    try {
      while (pendingSortTask) {
        const task = pendingSortTask;
        pendingSortTask = null;
        if (task.session !== getSession()) {
          task.onComplete(false);
          continue;
        }

        try {
          // 后端没有批量排序接口，同一次拖拽中仅逐条更新 seq 发生变化的函数。
          const results = await Promise.allSettled(task.functions.map(enqueueFunctionSortUpdate));
          const failedResult = results.find(
            (result): result is PromiseRejectedResult => result.status === 'rejected',
          );
          if (failedResult) throw failedResult.reason;
          task.onComplete(task.session === getSession());
        } catch (error) {
          task.onComplete(false);
          // 当前批次可能部分成功，取消等待任务并让最新拖拽重新拉取后端顺序。
          if (pendingSortTask) {
            pendingSortTask.onComplete(false);
            pendingSortTask = null;
          }
          console.error(error);
        }
      }
    } finally {
      sortSaving = false;
    }
  };

  /**
   * 将分组排序加入队列；尚未发送的旧顺序会被最新一次拖拽替换。
   * @param functions 本次 seq 发生变化的已保存函数。
   * @param onComplete 排序保存结果回调。
   */
  const enqueueFunctionSort = (
    functions: PluginDesignFunction[],
    onComplete: PluginDesignFunctionSortComplete,
  ) => {
    if (pendingSortTask) {
      // 被替换回调的请求标识已经过期，不会触发旧列表恢复。
      pendingSortTask.onComplete(false);
    }
    pendingSortTask = {
      session: getSession(),
      functions,
      onComplete,
    };
    void flushSortQueue();
  };

  /** 清理尚未发送的排序任务，已发请求通过会话标识忽略回显。 */
  const resetFunctionSortQueue = () => {
    if (!pendingSortTask) return;
    pendingSortTask.onComplete(false);
    pendingSortTask = null;
  };

  return {
    enqueueFunctionSort,
    enqueueFunctionUpdate,
    resetFunctionSortQueue,
  };
};
