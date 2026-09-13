import type { ComputedRef } from 'vue';
import type { FormRuntimeBootstrap } from '~/types';
import { migrateFormSchema } from '@evolyn.do/form/schema';
import { ApiError } from '@evolyn.do/utils';
import { computed, onScopeDispose, readonly, shallowRef } from 'vue';
import { getFormRuntime } from '~/api/form';

export type FormRecordCreateStatus =
  | 'idle'
  | 'loading'
  | 'ready'
  | 'not-published'
  | 'forbidden'
  | 'error';

interface UseFormRecordCreateOptions {
  appCode: ComputedRef<string>;
  formCode: ComputedRef<string>;
}

/**
 * 新增记录的运行时会话只服务于「数据管理」弹窗：每次打开都读取最新发布快照，
 * 以发布双口令、字段权限和动态控件配置为唯一事实源，绝不复用列表的旧快照提交。
 */
export function useFormRecordCreate(options: UseFormRecordCreateOptions) {
  const bootstrap = shallowRef<FormRuntimeBootstrap | null>(null);
  // readonly(shallowRef) 会把递归 JSON Schema 推导成 DeepReadonly，和运行时
  // 组件消费的可变协议类型不兼容；computed 保持值类型并只暴露读取通道。
  const bootstrapSnapshot = computed<FormRuntimeBootstrap | null>(() => bootstrap.value);
  const status = shallowRef<FormRecordCreateStatus>('idle');
  const errorMessage = shallowRef('');
  let controller: AbortController | null = null;
  let requestVersion = 0;

  async function load(): Promise<void> {
    controller?.abort();
    const requestController = new AbortController();
    controller = requestController;
    const version = ++requestVersion;
    const appCode = options.appCode.value;
    const formCode = options.formCode.value;

    bootstrap.value = null;
    errorMessage.value = '';
    if (!appCode || !formCode.startsWith('form_')) {
      status.value = 'error';
      errorMessage.value = '表单上下文无效，无法添加数据';
      return;
    }

    status.value = 'loading';
    try {
      const next = await getFormRuntime(appCode, formCode, requestController.signal);
      // 请求实现即便没有及时响应 AbortSignal，也不能让旧响应覆盖刚打开的新会话。
      if (requestController.signal.aborted || version !== requestVersion) return;

      if (next.permissions?.operations && !next.permissions.operations.includes('add')) {
        status.value = 'forbidden';
        errorMessage.value = '你没有添加此表单数据的权限';
        return;
      }

      const migrated = migrateFormSchema(next.content, next.protocolVersion);
      if (!migrated.document) {
        status.value = 'error';
        errorMessage.value = `表单配置无效：${migrated.issues[0]?.message ?? '未知错误'}`;
        return;
      }

      bootstrap.value = {
        ...next,
        protocolVersion: migrated.protocolVersion,
        content: migrated.document,
      };
      status.value = 'ready';
    } catch (error) {
      if (requestController.signal.aborted || version !== requestVersion) return;
      if (error instanceof ApiError && error.errCode === 'FORM_NOT_PUBLISHED') {
        status.value = 'not-published';
        errorMessage.value = '表单尚未发布，暂时不能添加数据';
        return;
      }
      status.value = 'error';
      errorMessage.value = '表单加载失败，请稍后重试';
    }
  }

  function reset(): void {
    controller?.abort();
    controller = null;
    requestVersion += 1;
    bootstrap.value = null;
    status.value = 'idle';
    errorMessage.value = '';
  }

  onScopeDispose(reset);

  return {
    bootstrap: bootstrapSnapshot,
    status: readonly(status),
    errorMessage: readonly(errorMessage),
    load,
    reset,
  };
}
