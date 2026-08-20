<script setup lang="ts">
import { ElMessage } from 'element-plus';
// 注册向导：三步（注册账号 → 选择团队 → 完善信息），全程纯前端采集——
// 第 1/2 步只做本地校验与暂存，不产生任何服务端写副作用；直到第 3 步
// 「进入产品」才把三步采集的全量数据一次性提交（POST /auth/register），
// 服务端单事务完成：注册账号（已注册手机号等价短信登录）→ 落账号画像 →
// 开通租户并绑定 owner/tenant-admin → 签发绑定新租户的会话令牌。
// - 验证码随最终提交一次性校验：向导停留超 5 分钟有效期将返回 401，
//   引导回第 1 步重新获取（手机号带回免重填）
// - 注册全程不设密码：账号为免密状态，密码由用户后续在个人中心首设
import { computed, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { registerComplete, sendSmsCode } from '~/api/auth';
import { ERROR_CODES } from '~/api/errorCodes';
import { ApiError } from '~/api/http';
import { useAuth } from '~/composables';

const route = useRoute();
const router = useRouter();
const { applyJwt, loadUserInfo } = useAuth();

const step = shallowRef(0);
const submitting = shallowRef(false);

// 向导采集的全量数据：第 1/2 步暂存于此，第 3 步随「进入产品」汇总提交
const phone = shallowRef('');
const smsCode = shallowRef('');
const tenantProfile = shallowRef<{ tenantName: string; demand?: string; industry: string }>();

/** 手机号脱敏（138****1234）：第 3 步昵称默认值，与后端免密注册默认昵称同口径 */
const maskedPhone = computed(() => {
  const value = phone.value;
  return value.length === 11 ? `${value.slice(0, 3)}****${value.slice(7)}` : value;
});

// 标题随步骤切换；首步标题下直接放登录入口，与登录页的轻量表单层级保持一致。
const titles = ['注册账号', '选择团队', '欢迎使用'];
const subtitles = ['', '', '完善信息，即刻开启高效协同之旅'];
const title = computed(() => titles[step.value]);
const subtitle = computed(() => subtitles[step.value]);

/** 第 1 步「注册」：本地校验通过即暂存手机号+验证码并推进（验证码在最终提交时校验） */
function handleAccountSubmit(payload: { phone: string; smsCode: string }) {
  phone.value = payload.phone;
  smsCode.value = payload.smsCode;
  step.value = 1;
}

/** 发送注册验证码：本地联调（devEcho）时后端回显验证码，弹长时提示便于取码 */
async function handleSendCode(target: string) {
  try {
    const result = await sendSmsCode(target, 'register');
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
  } catch (err) {
    // 冷却中给更友好的中文提示
    if (err instanceof ApiError && err.errCode === ERROR_CODES.AUTH_COOLDOWN) {
      ElMessage.warning('发送太频繁，请稍后再试');
    } else {
      ElMessage.error(err instanceof Error ? err.message : '验证码发送失败');
    }
  }
}

/** 第 2 步「下一步」：暂存企业画像并推进（租户开通合并进最终提交） */
function handleTenantSubmit(profile: { tenantName: string; demand?: string; industry: string }) {
  tenantProfile.value = profile;
  step.value = 2;
}

/**
 * 第 3 步「进入产品」：一次性提交三步采集的全量数据——服务端单事务完成
 *  注册账号、落账号画像（昵称同步 owner 成员称呼）、开通/复用租户并绑定
 *  owner；返回的令牌直接绑定新租户，已注册手机号等价短信登录（created=false）
 */
async function handleProfileSubmit(profile: { nickname: string; role: string; channel: string }) {
  const tenant = tenantProfile.value;
  if (!tenant) return;

  submitting.value = true;
  try {
    const result = await registerComplete({
      phone: phone.value,
      smsCode: smsCode.value,
      nickname: profile.nickname,
      onboarding: { role: profile.role, channel: profile.channel },
      tenant: {
        name: tenant.tenantName,
        demand: tenant.demand,
        industry: tenant.industry,
      },
    });
    applyJwt(result);
    // 注册即登录：跳首页前拉齐聚合信息（账号/成员/租户/配额），失败不阻断跳转
    await loadUserInfo();
    if (!result.created) {
      ElMessage.info('该手机号已注册，已为你直接登录');
    } else {
      ElMessage.success('注册完成，欢迎加入！');
    }
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
    router.replace(redirect);
  } catch (err) {
    // 验证码失效/错误（401）：引导回第 1 步重新获取，手机号带回免重填
    if (err instanceof ApiError && err.status === 401) {
      ElMessage.warning('验证码已过期或错误，请重新获取');
      step.value = 0;
    } else {
      ElMessage.error(err instanceof Error ? err.message : '提交失败，请稍后重试');
    }
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <AuthLayout class="register-page" :title="title" :subtitle="subtitle" variant="login">
    <!-- 步骤条置于标题前：与双栏注册首屏的视觉顺序一致。 -->
    <template #before-title>
      <el-steps class="register-page__steps" :active="step" align-center>
        <el-step />
        <el-step />
        <el-step />
      </el-steps>
    </template>

    <template #after-title>
      <span v-if="step === 0" class="register-page__login-tip">
        已有账号？
        <router-link to="/auth/login">直接登录</router-link>
      </span>
    </template>

    <RegisterAccountStep
      v-if="step === 0"
      :loading="submitting"
      :default-phone="phone"
      @submit="handleAccountSubmit"
      @send-code="handleSendCode"
    />

    <TenantChoiceStep v-else-if="step === 1" :loading="submitting" @submit="handleTenantSubmit" />

    <RegisterProfileStep
      v-else
      :default-nickname="maskedPhone"
      :loading="submitting"
      @submit="handleProfileSubmit"
    />
  </AuthLayout>
</template>

<style lang="scss" scoped>
.register-page__steps {
  margin-bottom: 30px;
}

.register-page__login-tip {
  display: block;
  margin: -18px 0 30px;
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-regular);
}

.register-page__login-tip a {
  font-weight: 500;
  color: var(--el-color-primary);
}

// 注册首步采用窄幅、左对齐的信息列，匹配双栏右侧表单的留白与阅读顺序。
.register-page {
  :deep(.auth-layout__card) {
    width: min(288px, 100%);
  }

  :deep(.auth-layout__title) {
    margin-bottom: 14px;
    text-align: left;
  }

  :deep(.el-step__head.is-process) {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary);
  }

  :deep(.el-step__head.is-process .el-step__icon) {
    color: var(--el-color-white);
    background-color: var(--el-color-primary);
  }

  :deep(.el-step__head.is-wait) {
    color: var(--el-text-color-placeholder);
    border-color: var(--el-fill-color-dark);
  }

  :deep(.el-step__head.is-wait .el-step__icon) {
    background-color: var(--el-fill-color-dark);
  }

  :deep(.el-step__line-inner) {
    border-color: var(--el-border-color-lighter);
  }
}
</style>
