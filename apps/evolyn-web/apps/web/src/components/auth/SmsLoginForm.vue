<script setup lang="ts">
// 验证码登录表单：手机号 + 短信验证码 +「下次自动登录」。「获取验证码」通过
// 手机号校验后上抛父级发送，父级成功后才启动 60s 重发倒计时；提交校验通过后上抛，
// 登录调用在父级。remember 决定令牌存储范围（持久/会话级，见 composables/auth）
import type { FormInstance, FormRules } from 'element-plus';
import { RiSmartphoneLine } from '@remixicon/vue';
import { computed, nextTick, reactive, shallowRef, useTemplateRef, watch } from 'vue';
import { useSmsCountdown } from '~/composables/useSmsCountdown';

const props = defineProps<{
  /** 提交中：按钮显示 loading 并防重复提交 */
  loading?: boolean;
  /** 父级仅在短信发送成功后递增，用于启动重发倒计时 */
  sentVersion?: number;
}>();

const emit = defineEmits<{
  /** 校验通过后上抛手机号 + 验证码 + 是否下次自动登录 */
  submit: [payload: { phone: string; code: string; remember: boolean }];
  /** 请求发送短信验证码（先通过手机号校验） */
  sendCode: [phone: string];
}>();

/** 重发倒计时秒数：与后端发送冷却窗口一致 */
const RESEND_SECONDS = 60;

const formRef = useTemplateRef<FormInstance>('formRef');

const form = reactive({
  phone: '',
  code: '',
  remember: true,
});

const rules: FormRules = {
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' },
  ],
  code: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { pattern: /^\d{6}$/, message: '验证码为 6 位数字', trigger: 'blur' },
  ],
};

const { countdown, start: startCountdown } = useSmsCountdown(RESEND_SECONDS);

// 父级 loading 在 emit 后才会下传。这里先在本地置位，确保点击提交后的首个渲染帧
// 就显示按钮加载态；父级接管后再由其请求生命周期负责保持和结束该状态。
const submitting = shallowRef(false);
const isSubmitting = computed(() => submitting.value || Boolean(props.loading));

// 父级异步发送成功后才启动倒计时；发送失败不改变 sentVersion，用户可立即重试。
watch(
  () => props.sentVersion,
  (version, previous) => {
    if (version && version !== previous) startCountdown();
  },
);

watch(
  () => props.loading,
  (loading, wasLoading) => {
    // 仅在父级结束一次已接管的提交后复位，避免初始 false 覆盖本地的即时加载态。
    if (wasLoading && !loading) submitting.value = false;
  },
);

async function handleSubmit() {
  if (isSubmitting.value) return;
  const valid = await formRef.value?.validate().then(
    () => true,
    () => false,
  );
  if (!valid) return;

  submitting.value = true;
  emit('submit', { phone: form.phone.trim(), code: form.code, remember: form.remember });

  // 让父级有一个更新周期接管 loading。若父级没有启动请求（例如外部监听器提前返回），
  // 则回退本地状态，避免按钮永久处于加载中。
  await nextTick();
  if (!props.loading) submitting.value = false;
}

function handleSendCode() {
  // 仅校验手机号字段，通过后上抛发送请求；倒计时由父级成功响应驱动。
  formRef.value
    ?.validateField('phone')
    .then(() => {
      emit('sendCode', form.phone.trim());
    })
    .catch(() => {});
}
</script>

<template>
  <el-form
    ref="formRef"
    class="sms-login-form"
    :model="form"
    :rules="rules"
    size="large"
    @submit.prevent="handleSubmit"
  >
    <el-form-item prop="phone">
      <el-input
        v-model="form.phone"
        name="phone"
        placeholder="你的手机号"
        autocomplete="tel"
        clearable
        :prefix-icon="RiSmartphoneLine"
      >
        <template #prepend>
          <span class="auth-phone-prefix">+86</span>
        </template>
      </el-input>
    </el-form-item>

    <el-form-item prop="code">
      <el-input
        v-model="form.code"
        name="code"
        placeholder="收到的验证码"
        autocomplete="one-time-code"
        maxlength="6"
        @keyup.enter="handleSubmit"
      >
        <!-- 获取验证码：输入框内右侧文字按钮（设计稿口径），倒计时中禁用 -->
        <template #suffix>
          <button
            class="sms-login-form__send"
            type="button"
            :disabled="countdown > 0 || isSubmitting"
            @click="handleSendCode"
          >
            {{ countdown > 0 ? `${countdown}s 后重发` : '获取验证码' }}
          </button>
        </template>
      </el-input>
    </el-form-item>

    <el-form-item class="sms-login-form__remember">
      <el-checkbox v-model="form.remember">
        下次自动登录
      </el-checkbox>
    </el-form-item>

    <el-button
      class="sms-login-form__submit"
      type="primary"
      size="large"
      native-type="submit"
      :loading="isSubmitting"
      :disabled="isSubmitting"
    >
      登录
    </el-button>

    <!-- 登录方式切换（如「密码登录」）由父级通过 footer 插槽注入 -->
    <slot name="footer" />
  </el-form>
</template>

<style lang="scss" scoped>
// 获取验证码：输入框内右侧无边框文字按钮，与后端冷却窗口联动禁用
.sms-login-form__send {
  padding: 0;
  font-size: var(--el-font-size-base);
  color: var(--el-color-primary);
  background: none;
  border: none;
  cursor: pointer;

  &:hover:not(:disabled) {
    color: var(--el-color-primary-light-3);
  }

  &:disabled {
    color: var(--el-text-color-placeholder);
    cursor: not-allowed;
  }
}

// 勾选行收紧下边距，贴近主按钮
.sms-login-form__remember {
  margin-bottom: var(--el-space-lg);
}

.sms-login-form__submit {
  width: 100%;
}

.auth-phone-prefix {
  color: var(--el-color-primary);
}
</style>
