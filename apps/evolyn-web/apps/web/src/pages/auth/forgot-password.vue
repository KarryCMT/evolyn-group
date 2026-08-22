<script setup lang="ts">
// 找回密码路由页：编排 reset 场景验证码发送、RSA 加密和后端重设接口；
// 字段收集与本地校验留在 ResetPasswordForm，避免页面承担表单细节。
import { shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { resetPassword, sendSmsCode } from '~/api/auth';
import { encryptPassword } from '~/api/conf';
import ResetPasswordForm from '~/components/auth/ResetPasswordForm.vue';

const router = useRouter();
const submitting = shallowRef(false);
const smsSentVersion = shallowRef(0);

async function handleSendCode(phone: string) {
  try {
    const result = await sendSmsCode(phone, 'reset');
    if (result.code) {
      ElMessage({
        message: `【本地联调】验证码：${result.code}（5 分钟内有效）`,
        type: 'info',
        duration: 10000,
        showClose: true,
      });
    } else {
      ElMessage.success('验证码已发送，请注意查收短信');
    }
    smsSentVersion.value += 1;
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '验证码发送失败');
  }
}

async function handleSubmit(payload: { phone: string; smsCode: string; newPassword: string }) {
  submitting.value = true;
  try {
    await resetPassword({
      phone: payload.phone,
      smsCode: payload.smsCode,
      newPassword: await encryptPassword(payload.newPassword),
    });
    ElMessage.success('密码已重设，请使用新密码登录');
    await router.replace('/auth/login');
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '密码重设失败，请稍后重试');
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <AuthLayout title="找回密码" subtitle="验证手机号后重设登录密码">
    <ResetPasswordForm
      :loading="submitting"
      :sent-version="smsSentVersion"
      @send-code="handleSendCode"
      @submit="handleSubmit"
    />
  </AuthLayout>
</template>
