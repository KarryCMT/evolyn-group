<script setup lang="ts">
// 找回密码表单：只负责手机号、验证码和两次新密码的采集校验；发送短信与重设请求
// 分别通过事件交由路由页编排，保持认证接口调用和表单展示职责分离。
import { reactive, useTemplateRef, watch } from 'vue';
import type { FormInstance, FormRules } from 'element-plus';
import { RiLockPasswordFill, RiSmartphoneFill } from '@remixicon/vue';
import { useSmsCountdown } from '~/composables/useSmsCountdown';

const RESEND_SECONDS = 60;

const props = defineProps<{
  loading?: boolean;
  /** 父级仅在 reset 场景短信发送成功后递增，用于启动重发倒计时 */
  sentVersion?: number;
}>();

const emit = defineEmits<{
  'send-code': [phone: string];
  submit: [payload: { phone: string; smsCode: string; newPassword: string }];
}>();

const formRef = useTemplateRef<FormInstance>('formRef');
const form = reactive({
  phone: '',
  smsCode: '',
  newPassword: '',
  confirmPassword: '',
});

const rules: FormRules = {
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' },
  ],
  smsCode: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { pattern: /^\d{6}$/, message: '验证码为 6 位数字', trigger: 'blur' },
  ],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, max: 64, message: '密码长度为 8-64 位', trigger: 'blur' },
    {
      // 与后端口径一致（8-64 位且同时包含字母和数字，弱口令由后端黑名单拦截）
      pattern: /^(?=.*[A-Za-z])(?=.*\d).{8,64}$/,
      message: '密码需同时包含字母和数字',
      trigger: 'blur',
    },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_rule, value: string, callback) => {
        callback(value === form.newPassword ? undefined : new Error('两次输入的密码不一致'));
      },
      trigger: 'blur',
    },
  ],
};

const { countdown, start: startCountdown } = useSmsCountdown(RESEND_SECONDS);

watch(
  () => props.sentVersion,
  (version, previous) => {
    if (version && version !== previous) startCountdown();
  },
);

async function handleSubmit() {
  const valid = await formRef.value?.validate().then(
    () => true,
    () => false,
  );
  if (!valid) return;

  emit('submit', {
    phone: form.phone.trim(),
    smsCode: form.smsCode,
    newPassword: form.newPassword,
  });
}

function handleSendCode() {
  formRef.value
    ?.validateField('phone')
    .then(() => emit('send-code', form.phone.trim()))
    .catch(() => {});
}
</script>

<template>
  <el-form
    ref="formRef"
    class="reset-password-form"
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
        :prefix-icon="RiSmartphoneFill"
      >
        <template #prepend><span class="auth-phone-prefix">+86</span></template>
      </el-input>
    </el-form-item>

    <el-form-item prop="smsCode">
      <el-input
        v-model="form.smsCode"
        name="sms-code"
        placeholder="收到的验证码"
        autocomplete="one-time-code"
        maxlength="6"
      >
        <template #suffix>
          <button
            class="reset-password-form__send"
            type="button"
            :disabled="countdown > 0 || loading"
            @click="handleSendCode"
          >
            {{ countdown > 0 ? `${countdown}s 后重发` : '获取验证码' }}
          </button>
        </template>
      </el-input>
    </el-form-item>

    <el-form-item prop="newPassword">
      <el-input
        v-model="form.newPassword"
        name="new-password"
        type="password"
        placeholder="设置新密码（8-64 位，含字母和数字）"
        autocomplete="new-password"
        show-password
        :prefix-icon="RiLockPasswordFill"
      />
    </el-form-item>

    <el-form-item prop="confirmPassword">
      <el-input
        v-model="form.confirmPassword"
        name="confirm-password"
        type="password"
        placeholder="再次输入新密码"
        autocomplete="new-password"
        show-password
        :prefix-icon="RiLockPasswordFill"
        @keyup.enter="handleSubmit"
      />
    </el-form-item>

    <el-button
      class="reset-password-form__submit"
      type="primary"
      native-type="submit"
      :loading="loading"
    >
      重设密码
    </el-button>

    <div class="reset-password-form__login">
      想起密码了？<router-link to="/auth/login">返回登录</router-link>
    </div>
  </el-form>
</template>

<style lang="scss" scoped>
.reset-password-form__send {
  padding: 4px 0 4px 12px;
  border: 0;
  border-left: 1px solid var(--el-border-color);
  background: transparent;
  color: var(--el-color-primary);
  cursor: pointer;

  &:hover:not(:disabled) {
    color: var(--el-color-primary-light-3);
  }

  &:disabled {
    color: var(--el-text-color-placeholder);
    cursor: not-allowed;
  }
}

.reset-password-form__submit {
  width: 100%;
}

.reset-password-form__login {
  margin-top: 16px;
  text-align: center;
  color: var(--el-text-color-regular);

  a {
    color: var(--el-color-primary);

    &:hover {
      color: var(--el-color-primary-light-3);
    }
  }
}

.auth-phone-prefix {
  color: var(--el-color-primary);
}
</style>
