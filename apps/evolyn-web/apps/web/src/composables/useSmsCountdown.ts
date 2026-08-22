import { onUnmounted, shallowRef } from 'vue';

/**
 * 短信验证码重发倒计时：仅由调用方在“短信已发送成功”后触发 start，
 * 避免请求失败时错误锁定用户。多个认证表单共用这一行为。
 */
export function useSmsCountdown(seconds = 60) {
  const countdown = shallowRef(0);
  let timer: ReturnType<typeof setInterval> | undefined;

  function stop() {
    if (timer !== undefined) {
      clearInterval(timer);
      timer = undefined;
    }
  }

  function start() {
    stop();
    countdown.value = seconds;
    timer = setInterval(() => {
      countdown.value -= 1;
      if (countdown.value <= 0) {
        stop();
      }
    }, 1000);
  }

  onUnmounted(stop);

  return { countdown, start };
}
