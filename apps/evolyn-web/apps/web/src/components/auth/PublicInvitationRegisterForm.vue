<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus';
import { Iphone, User } from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
// 公开邀请注册表单只承担收集与本地校验；发送短信和最终注册由页面编排，
// 使接口副作用和成功后的会话跳转保持在路由层。
import { reactive, useTemplateRef, watch } from 'vue';
import { useSmsCountdown } from '~/composables/useSmsCountdown';

const props = defineProps<{
  loading?: boolean;
  sentVersion?: number;
}>();

const emit = defineEmits<{
  submit: [payload: { phone: string; smsCode: string; nickname: string }];
  sendCode: [phone: string];
}>();

const RESEND_SECONDS = 60;

const formRef = useTemplateRef<FormInstance>('formRef');
const form = reactive({ phone: '', smsCode: '', nickname: '' });
const rules: FormRules = {
  phone: [
    { required: true, message: '请输入手机号', trigger: 'blur' },
    { pattern: /^1[3-9]\d{9}$/, message: '手机号格式不正确', trigger: 'blur' },
  ],
  smsCode: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { pattern: /^\d{6}$/, message: '验证码为 6 位数字', trigger: 'blur' },
  ],
  nickname: [
    { required: true, message: '请输入你的姓名', trigger: 'blur' },
    { min: 2, max: 20, message: '姓名长度为 2 到 20 个字符', trigger: 'blur' },
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
    nickname: form.nickname.trim(),
  });
}

function handleSendCode() {
  formRef.value
    ?.validateField('phone')
    .then(() => emit('sendCode', form.phone.trim()))
    .catch(() => {});
}
</script>

<template>
  <el-form
    ref="formRef"
    class="public-invitation-register-form"
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
        :prefix-icon="Iphone"
      >
        <template #prepend>
          <span class="public-invitation-register-form__prefix">+86</span>
        </template>
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
            class="public-invitation-register-form__send"
            type="button"
            :disabled="countdown > 0 || loading"
            @click="handleSendCode"
          >
            {{ countdown > 0 ? `${countdown}s 后重发` : '获取验证码' }}
          </button>
        </template>
      </el-input>
    </el-form-item>

    <el-form-item prop="nickname">
      <el-input
        v-model="form.nickname"
        name="nickname"
        placeholder="你的姓名"
        autocomplete="name"
        maxlength="20"
        clearable
        :prefix-icon="User"
        @keyup.enter="handleSubmit"
      />
    </el-form-item>

    <p class="public-invitation-register-form__agreement">
      点击注册表明你已阅读并同意
      <a @click.prevent="ElMessage.info('服务条款文档即将上线')">《服务条款》</a>
      和
      <a @click.prevent="ElMessage.info('隐私声明文档即将上线')">《隐私声明》</a>
    </p>

    <el-button
      class="public-invitation-register-form__submit"
      type="primary"
      native-type="submit"
      :loading="loading"
    >
      注册并加入
    </el-button>
  </el-form>
</template>

<style lang="scss" scoped>
.public-invitation-register-form__prefix {
  color: var(--el-color-primary);
}

.public-invitation-register-form__send {
  padding: 0;
  font-size: var(--el-font-size-base);
  color: var(--el-color-primary);
  cursor: pointer;
  background: transparent;
  border: 0;

  &:hover:not(:disabled) {
    color: var(--el-color-primary-light-3);
  }

  &:disabled {
    color: var(--el-text-color-placeholder);
    cursor: not-allowed;
  }
}

.public-invitation-register-form__agreement {
  margin: var(--el-space-3xl) 0 var(--el-space-lg);
  font-size: var(--el-font-size-small);
  line-height: 1.7;
  color: var(--el-text-color-secondary);

  a {
    color: var(--el-color-primary);
    cursor: pointer;
  }
}

.public-invitation-register-form__submit {
  width: 100%;
  height: 40px;
  margin-top: var(--el-space-md);
}
</style>
