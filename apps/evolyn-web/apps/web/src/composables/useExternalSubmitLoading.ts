import type { MaybeRefOrGetter } from 'vue';
import { computed, nextTick, shallowRef, toValue, watch } from 'vue';

/**
 * 衔接由父级持有的异步提交状态。
 *
 * 子组件发出 submit 事件后，父级才会开始请求并回传 loading；本组合式函数先在
 * 子组件内展示加载态，并在父级接管后持续跟随其状态，避免点击提交到下一次渲染之间
 * 出现无反馈窗口。
 */
export function useExternalSubmitLoading(loading: MaybeRefOrGetter<boolean | undefined>) {
  const pending = shallowRef(false);
  const isLoading = computed(() => pending.value || Boolean(toValue(loading)));

  watch(
    () => Boolean(toValue(loading)),
    (current, previous) => {
      // 父级完成已接管的请求后，结束本地桥接状态。
      if (previous && !current) pending.value = false;
    },
  );

  /** 开始本地提交；返回 false 表示已有请求进行中。 */
  function begin(): boolean {
    if (isLoading.value) return false;
    pending.value = true;
    return true;
  }

  /**
   * 在 emit 后调用，让父级有一个更新周期接管 loading。若监听器没有启动请求，
   * 则回退本地状态，避免按钮永久显示加载中。
   */
  async function handoff(): Promise<void> {
    await nextTick();
    if (!toValue(loading)) pending.value = false;
  }

  /** 本地准备阶段失败（如图片处理失败）时，主动取消加载态。 */
  function cancel(): void {
    pending.value = false;
  }

  return { isLoading, begin, handoff, cancel };
}
