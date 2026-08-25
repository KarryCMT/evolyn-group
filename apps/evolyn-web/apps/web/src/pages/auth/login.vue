<script setup lang="ts">
// 登录页：认证域薄壳视图，组合骨架与两种登录表单，负责提交分发与登录后去向；
// 多租户账号在弹窗中选择进入的团队（单租户直接进入，与后端默认租户体验一致）
import { shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import MfaVerifyDialog from '~/components/auth/MfaVerifyDialog.vue';
import { sendSmsCode } from '~/api/auth';
import { encryptPassword } from '~/api/conf';
import { useAuth } from '~/composables';
import type { LoginMfaChallenge, TenantMembership } from '~/types';
import { ApiError } from '@evolyn.do/utils';
import { ERROR_CODES } from '@evolyn.do/utils';

type LoginMode = 'password' | 'sms';

const route = useRoute();
const router = useRouter();
const { login, completeMfaLogin, loadTenants, switchTenant } = useAuth();

// 对齐主流登录页首屏：优先展示短信验证码，密码方式保留为可切换的备选。
const mode = shallowRef<LoginMode>('sms');
const loading = shallowRef(false);
// 仅在短信接口成功后递增，作为子表单启动倒计时的确认信号。
const smsSentVersion = shallowRef(0);

// 多租户选择弹窗
const tenantDialogVisible = shallowRef(false);
const memberships = shallowRef<TenantMembership[]>([]);
const switching = shallowRef(false);
const mfaChallenge = shallowRef<LoginMfaChallenge | null>(null);
const mfaDialogVisible = shallowRef(false);

/** 密码登录：密码先经平台公钥 RSA 加密再上送；remember 决定令牌存储范围 */
async function handlePasswordSubmit(payload: {
  phone: string;
  password: string;
  remember: boolean;
}) {
  loading.value = true;
  try {
    const password = await encryptPassword(payload.password);
    const result = await login({ phone: payload.phone, password }, payload.remember);
    if ('mfaRequired' in result && result.mfaRequired) {
      mfaChallenge.value = result;
      mfaDialogVisible.value = true;
      return;
    }
    await afterLogin();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '登录失败，请稍后重试');
  } finally {
    loading.value = false;
  }
}

/** 验证码登录：免密，走 phone + smsCode；remember 决定令牌存储范围 */
async function handleSmsSubmit(payload: { phone: string; code: string; remember: boolean }) {
  loading.value = true;
  try {
    const result = await login({ phone: payload.phone, smsCode: payload.code }, payload.remember);
    if ('mfaRequired' in result && result.mfaRequired) {
      mfaChallenge.value = result;
      mfaDialogVisible.value = true;
      return;
    }
    await afterLogin();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '登录失败，请稍后重试');
  } finally {
    loading.value = false;
  }
}

async function handleMfaVerify(payload: { method: 'totp' | 'recovery'; code: string }) {
  if (!mfaChallenge.value) return;
  loading.value = true;
  try {
    await completeMfaLogin({ ...payload, mfaChallenge: mfaChallenge.value.mfaChallenge });
    mfaDialogVisible.value = false;
    mfaChallenge.value = null;
    await afterLogin();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '验证失败，请重试');
  } finally {
    loading.value = false;
  }
}

/** 发送短信验证码：本地联调（devEcho）时后端回显验证码，弹长时提示便于取码 */
async function handleSendCode(phone: string) {
  try {
    const result = await sendSmsCode(phone, 'login');
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
    // 冷却中给更友好的中文提示
    if (err instanceof ApiError && err.errCode === ERROR_CODES.AUTH_COOLDOWN) {
      ElMessage.warning('发送太频繁，请稍后再试');
    } else {
      ElMessage.error(err instanceof Error ? err.message : '验证码发送失败');
    }
  }
}

/** 登录成功后的去向：多租户弹窗选择，单租户/拉取失败直接进入 */
async function afterLogin() {
  let tenants: TenantMembership[] = [];
  try {
    tenants = await loadTenants();
  } catch {
    // 拉取租户失败不阻断登录，走后端默认租户
  }

  if (tenants.length > 1) {
    memberships.value = tenants;
    tenantDialogVisible.value = true;
    return;
  }
  goNext();
}

/** 选定租户：后端重新签发令牌后跳转 */
async function handleChooseTenant(tenant: TenantMembership) {
  switching.value = true;
  try {
    await switchTenant(tenant.tenantId);
    tenantDialogVisible.value = false;
    goNext();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '切换团队失败，请重试');
  } finally {
    switching.value = false;
  }
}

/** 优先回跳登录前想访问的页面 */
function goNext() {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
  router.replace(redirect);
}
</script>

<template>
  <AuthLayout title="账号登录" variant="login">
    <PasswordLoginForm v-if="mode === 'password'" :loading="loading" @submit="handlePasswordSubmit">
      <template #footer>
        <div class="login-page__switch-row">
          <button class="login-page__switch" type="button" @click="mode = 'sms'">验证码登录</button>
        </div>
      </template>
    </PasswordLoginForm>

    <SmsLoginForm
      v-else
      :loading="loading"
      :sent-version="smsSentVersion"
      @submit="handleSmsSubmit"
      @send-code="handleSendCode"
    >
      <template #footer>
        <div class="login-page__switch-row">
          <button class="login-page__switch" type="button" @click="mode = 'password'">
            密码登录
          </button>
        </div>
      </template>
    </SmsLoginForm>

    <OAuthLoginPanel />

    <template #footer>
      <span class="login-page__register-tip">
        没有账号？
        <router-link to="/auth/register">免费注册</router-link>
      </span>
    </template>
  </AuthLayout>

  <!-- 多租户选择：后端登录默认取第一个成员关系，多团队时显式选择进入哪个 -->
  <el-dialog
    v-model="tenantDialogVisible"
    title="选择进入的团队"
    width="420px"
    :close-on-click-modal="false"
  >
    <div class="login-page__tenant-list">
      <button
        v-for="membership in memberships"
        :key="membership.tenantId"
        class="login-page__tenant-item"
        type="button"
        :disabled="switching"
        @click="handleChooseTenant(membership)"
      >
        <span class="login-page__tenant-name">{{ membership.name }}</span>
        <span class="login-page__tenant-code">{{ membership.code }}</span>
        <el-tag v-if="membership.isOwner" size="small">所有者</el-tag>
      </button>
    </div>
  </el-dialog>
  <MfaVerifyDialog v-model="mfaDialogVisible" :loading="loading" @submit="handleMfaVerify" />
</template>

<style lang="scss" scoped>
.login-page__switch-row {
  margin-top: 16px;
  text-align: center;
}

.login-page__switch {
  padding: 0;
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-secondary);
  background: none;
  border: none;
  cursor: pointer;

  &:hover {
    color: var(--el-color-primary);
  }
}

.login-page__register-tip a {
  font-weight: 500;
  color: var(--el-color-primary);
}

.login-page__tenant-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.login-page__tenant-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 12px 16px;
  text-align: left;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  cursor: pointer;
  transition: border-color 0.2s;

  &:hover:not(:disabled) {
    border-color: var(--el-color-primary);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.login-page__tenant-name {
  font-size: var(--el-font-size-medium);
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.login-page__tenant-code {
  flex: 1;
  font-size: var(--el-font-size-base);
  color: var(--el-text-color-secondary);
}
</style>
