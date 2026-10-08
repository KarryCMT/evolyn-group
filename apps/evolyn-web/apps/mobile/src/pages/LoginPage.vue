<script setup lang="ts">
import {
  showFailToast,
  showSuccessToast,
  showToast,
  Checkbox as VanCheckbox,
  Icon as VanIcon,
  Loading as VanLoading,
} from 'vant';
import { onBeforeUnmount, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { sendLoginSms } from '~/api/auth';
import { encryptPassword } from '~/api/conf';
import { useAuthStore } from '~/stores/auth';

type LoginMode = 'sms' | 'password';

const REMEMBER_IDENTITY_KEY = 'evolyn.mobile.login.identity';
const PHONE_PATTERN = /^1[3-9]\d{9}$/;

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const mode = ref<LoginMode>('sms');
const loading = ref(false);
const sending = ref(false);
const countdown = ref(0);
let countdownTimer: ReturnType<typeof setInterval> | undefined;

const form = reactive({
  identity: localStorage.getItem(REMEMBER_IDENTITY_KEY) ?? '',
  code: '',
  password: '',
  remember: localStorage.getItem(REMEMBER_IDENTITY_KEY) !== null,
  passwordVisible: false,
});

function validatePhone(): string | null {
  const phone = form.identity.trim();
  if (!phone) return '请输入手机号';
  if (!PHONE_PATTERN.test(phone)) return '手机号格式不正确';
  return null;
}

function startCountdown(): void {
  countdown.value = 60;
  if (countdownTimer) clearInterval(countdownTimer);
  countdownTimer = setInterval(() => {
    countdown.value -= 1;
    if (countdown.value <= 0 && countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = undefined;
    }
  }, 1000);
}

async function handleSendCode(): Promise<void> {
  if (sending.value || countdown.value > 0) return;
  const error = validatePhone();
  if (error) {
    showFailToast(error);
    return;
  }

  sending.value = true;
  try {
    const result = await sendLoginSms(form.identity.trim());
    startCountdown();
    if (result.code) showToast({ message: `本地验证码：${result.code}`, duration: 8000 });
    else showSuccessToast('验证码已发送');
  } catch (error) {
    showFailToast(error instanceof Error ? error.message : '验证码发送失败');
  } finally {
    sending.value = false;
  }
}

function rememberIdentity(): void {
  if (form.remember) localStorage.setItem(REMEMBER_IDENTITY_KEY, form.identity.trim());
  else localStorage.removeItem(REMEMBER_IDENTITY_KEY);
}

async function handleSubmit(): Promise<void> {
  if (loading.value) return;
  const identity = form.identity.trim();
  if (!identity) {
    showFailToast(mode.value === 'sms' ? '请输入手机号' : '请输入手机号或账号');
    return;
  }
  if (mode.value === 'sms') {
    const phoneError = validatePhone();
    if (phoneError) {
      showFailToast(phoneError);
      return;
    }
    if (!/^\d{6}$/.test(form.code)) {
      showFailToast('请输入 6 位验证码');
      return;
    }
  } else if (!form.password) {
    showFailToast('请输入密码');
    return;
  }

  loading.value = true;
  try {
    const result = mode.value === 'sms'
      ? await auth.login({ phone: identity, smsCode: form.code }, form.remember)
      : await auth.login(
          {
            ...(PHONE_PATTERN.test(identity) ? { phone: identity } : { name: identity }),
            password: await encryptPassword(form.password),
          },
          form.remember,
        );

    if ('mfaRequired' in result && result.mfaRequired) {
      showFailToast('此账号已启用二次验证，请先使用桌面端登录');
      return;
    }

    rememberIdentity();
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
    await router.replace(redirect);
  } catch (error) {
    showFailToast(error instanceof Error ? error.message : '登录失败，请稍后重试');
  } finally {
    loading.value = false;
  }
}

function switchMode(): void {
  mode.value = mode.value === 'sms' ? 'password' : 'sms';
  form.code = '';
  form.password = '';
}

onBeforeUnmount(() => {
  if (countdownTimer) clearInterval(countdownTimer);
});
</script>

<template>
  <main class="login-page">
    <section class="login-card" aria-labelledby="login-title">
      <header class="login-card__header">
        <h1 id="login-title">
          账号登录
        </h1>
      </header>

      <form class="login-form" @submit.prevent="handleSubmit">
        <div v-if="mode === 'sms'" class="login-input login-input--phone">
          <button class="login-input__prefix" type="button" aria-label="当前区号中国大陆">
            +86 <VanIcon name="arrow-down" />
          </button>
          <input
            v-model="form.identity"
            name="phone"
            inputmode="tel"
            autocomplete="tel"
            maxlength="11"
            placeholder="手机号"
            aria-label="手机号"
          >
        </div>

        <div v-else class="login-input">
          <input
            v-model="form.identity"
            name="identity"
            autocomplete="username"
            placeholder="手机号/账号"
            aria-label="手机号或账号"
          >
        </div>

        <div v-if="mode === 'sms'" class="login-input login-input--code">
          <input
            v-model="form.code"
            name="code"
            inputmode="numeric"
            autocomplete="one-time-code"
            maxlength="6"
            placeholder="验证码"
            aria-label="验证码"
          >
          <button
            class="login-input__action"
            type="button"
            :disabled="sending || countdown > 0"
            @click="handleSendCode"
          >
            {{ countdown > 0 ? `重新获取(${countdown})` : sending ? '发送中…' : '获取验证码' }}
          </button>
        </div>

        <div v-else class="login-input login-input--password">
          <input
            v-model="form.password"
            name="password"
            :type="form.passwordVisible ? 'text' : 'password'"
            autocomplete="current-password"
            placeholder="密码"
            aria-label="密码"
          >
          <button
            class="login-input__eye"
            type="button"
            :aria-label="form.passwordVisible ? '隐藏密码' : '显示密码'"
            @click="form.passwordVisible = !form.passwordVisible"
          >
            <VanIcon :name="form.passwordVisible ? 'eye-o' : 'closed-eye'" />
          </button>
        </div>

        <div class="login-form__options">
          <VanCheckbox v-model="form.remember" icon-size="20px" shape="square">
            下次自动登录
          </VanCheckbox>
          <button
            v-if="mode === 'password'"
            type="button"
            class="login-form__forgot"
            @click="showToast('请联系管理员重置密码')"
          >
            忘记密码？
          </button>
        </div>

        <button class="login-form__submit" type="submit" :disabled="loading">
          <VanLoading v-if="loading" color="#fff" size="22px" />
          <span v-else>登录</span>
        </button>

        <button class="login-form__switch" type="button" @click="switchMode">
          {{ mode === 'sms' ? '密码登录' : '验证码登录' }}
        </button>
      </form>
    </section>

    <button class="login-locale" type="button" @click="showToast('当前语言：简体中文')">
      <VanIcon name="location-o" />
      <span>简体中文</span>
      <VanIcon name="arrow-down" />
    </button>
  </main>
</template>

<style scoped>
.login-page {
  box-sizing: border-box;
  display: flex;
  min-height: 100dvh;
  padding: max(26vh, 150px) 20px 48px;
  color: #182235;
  background: #f5f6f8;
  flex-direction: column;
  align-items: center;
}

.login-card {
  width: min(100%, 494px);
  overflow: hidden;
  background: #fff;
  border-radius: 10px;
  box-shadow: 0 1px 0 rgb(16 24 40 / 2%);
}

.login-card__header {
  display: grid;
  height: 74px;
  border-bottom: 1px solid #ebedf0;
  place-items: center;
}

.login-card__header h1 {
  margin: 0;
  font-size: 25px;
  font-weight: 600;
  letter-spacing: 1px;
}

.login-form {
  padding: 47px 31px 39px;
}

.login-input {
  box-sizing: border-box;
  display: flex;
  height: 62px;
  overflow: hidden;
  border: 1px solid #d8dde5;
  border-radius: 9px;
  align-items: stretch;
}

.login-input + .login-input {
  margin-top: 25px;
}

.login-input:focus-within {
  border-color: var(--van-primary-color);
  box-shadow: 0 0 0 1px var(--van-primary-color);
}

.login-input input {
  min-width: 0;
  padding: 0 14px;
  font-size: 20px;
  color: #202a3c;
  background: transparent;
  border: 0;
  outline: 0;
  flex: 1;
}

.login-input input::placeholder {
  color: #a8afb9;
}

.login-input__prefix {
  display: flex;
  width: 126px;
  padding: 0 14px;
  font-size: 19px;
  color: #202a3c;
  background: transparent;
  border: 0;
  border-right: 1px solid #d8dde5;
  align-items: center;
  justify-content: space-between;
}

.login-input__action {
  width: 186px;
  padding: 0 10px;
  font-size: 18px;
  color: var(--van-primary-color);
  background: color-mix(in srgb, var(--van-primary-color) 5%, #f8f9fb);
  border: 0;
  border-left: 1px solid #d8dde5;
}

.login-input__action:disabled {
  color: #9aa1ac;
}

.login-input__eye {
  width: 56px;
  padding: 0;
  font-size: 24px;
  color: #536071;
  background: transparent;
  border: 0;
}

.login-form__options {
  display: flex;
  min-height: 52px;
  font-size: 18px;
  color: #505b6a;
  align-items: center;
  justify-content: space-between;
}

.login-form__options :deep(.van-checkbox__label) {
  color: #505b6a;
}

.login-form__forgot,
.login-form__switch {
  padding: 0;
  color: var(--van-primary-color);
  background: none;
  border: 0;
}

.login-form__forgot {
  margin-left: auto;
  font-size: 17px;
}

.login-form__submit {
  display: grid;
  width: 100%;
  height: 62px;
  margin-top: 20px;
  font-size: 19px;
  font-weight: 600;
  color: #fff;
  background: var(--van-primary-color);
  border: 0;
  border-radius: 9px;
  place-items: center;
}

.login-form__submit:disabled {
  opacity: 0.72;
}

.login-form__switch {
  margin-top: 25px;
  font-size: 17px;
}

.login-locale {
  display: flex;
  margin-top: 28px;
  margin-left: min(56%, 278px);
  padding: 4px;
  gap: 10px;
  font-size: 17px;
  color: #667080;
  background: none;
  border: 0;
  align-items: center;
  white-space: nowrap;
}

@media (max-width: 420px) {
  .login-page {
    padding-top: max(18vh, 112px);
  }

  .login-form {
    padding: 32px 24px 30px;
  }

  .login-input,
  .login-form__submit {
    height: 54px;
  }

  .login-input input,
  .login-input__prefix {
    font-size: 16px;
  }

  .login-input__prefix {
    width: 104px;
  }

  .login-input__action {
    width: 138px;
    font-size: 14px;
  }

  .login-form__options {
    font-size: 15px;
  }
}
</style>
