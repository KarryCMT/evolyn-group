<script setup lang="ts">
import { ApiError, ERROR_CODES } from '@evolyn.do/utils';
import { ElAlert, ElMessage } from 'element-plus';
// 受邀注册页：公开链接仅让受邀人完成本人信息与短信验证，注册成功后会话直接
// 绑定目标企业成员身份，不展示或提交创建企业向导。
import { computed, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { acceptPublicInvitation, registerPublicInvitation, sendSmsCode } from '~/api/auth';
import AuthLayout from '~/components/auth/AuthLayout.vue';
import PublicInvitationRegisterForm from '~/components/auth/PublicInvitationRegisterForm.vue';
import { useAuth } from '~/composables';

const route = useRoute();
const router = useRouter();
const { applyJwt, isAuthenticated, loadUserInfo, switchTenant } = useAuth();

const submitting = shallowRef(false);
const smsSentVersion = shallowRef(0);
const inviteToken = computed(() => (typeof route.query.token === 'string' ? route.query.token.trim() : ''));
const invalidInvitation = computed(() => !inviteToken.value);

async function handleSendCode(phone: string) {
  try {
    const result = await sendSmsCode(phone, 'register');
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
    if (err instanceof ApiError && err.errCode === ERROR_CODES.AUTH_COOLDOWN) {
      ElMessage.warning('发送太频繁，请稍后再试');
    } else {
      ElMessage.error(err instanceof Error ? err.message : '验证码发送失败');
    }
  }
}

async function handleSubmit(payload: { phone: string; smsCode: string; nickname: string }) {
  if (invalidInvitation.value) {
    ElMessage.error('邀请链接无效，请向企业管理员重新获取');
    return;
  }

  submitting.value = true;
  try {
    const result = await registerPublicInvitation({ ...payload, inviteToken: inviteToken.value });
    applyJwt(result);
    await loadUserInfo();
    ElMessage.success(result.created ? '注册完成，欢迎加入企业！' : '已加入企业，欢迎回来！');
    await router.replace('/');
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      ElMessage.warning('验证码已过期或错误，请重新获取');
    } else {
      ElMessage.error(err instanceof Error ? err.message : '加入企业失败，请联系企业管理员');
    }
  } finally {
    submitting.value = false;
  }
}

async function handleExistingAccountJoin() {
  if (invalidInvitation.value) {
    ElMessage.error('邀请链接无效，请向企业管理员重新获取');
    return;
  }

  submitting.value = true;
  try {
    const member = await acceptPublicInvitation(inviteToken.value);
    await switchTenant(member.tenantId);
    ElMessage.success('已加入企业，欢迎回来！');
    await router.replace('/');
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '加入企业失败，请联系企业管理员');
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <AuthLayout variant="login" title="邀请你加入企业">
    <template #after-title>
      <p class="public-invitation-page__login-tip">
        已有账号？
        <router-link :to="{ name: 'login', query: { redirect: route.fullPath } }">
          直接登录
        </router-link>
      </p>
    </template>

    <ElAlert
      v-if="invalidInvitation"
      class="public-invitation-page__invalid"
      title="邀请链接无效"
      description="请向企业管理员重新获取有效邀请链接。"
      type="error"
      :closable="false"
      show-icon
    />
    <section v-else-if="isAuthenticated" class="public-invitation-page__returning-account">
      <p>你已登录灵衍云账号，确认后即可加入该企业。</p>
      <el-button type="primary" :loading="submitting" @click="handleExistingAccountJoin">
        加入企业
      </el-button>
    </section>
    <PublicInvitationRegisterForm
      v-else
      :loading="submitting"
      :sent-version="smsSentVersion"
      @send-code="handleSendCode"
      @submit="handleSubmit"
    />
  </AuthLayout>
</template>

<style lang="scss" scoped>
.public-invitation-page__login-tip {
  margin: calc(-1 * var(--el-space-xl)) 0 var(--el-space-3xl);
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-regular);
  text-align: center;

  a {
    color: var(--el-color-primary);
  }
}

.public-invitation-page__invalid {
  margin-top: var(--el-space-xl);
}

.public-invitation-page__returning-account {
  padding: var(--el-space-xl) 0;
  text-align: center;

  p {
    margin: 0 0 var(--el-space-3xl);
    color: var(--el-text-color-regular);
  }
}
</style>
