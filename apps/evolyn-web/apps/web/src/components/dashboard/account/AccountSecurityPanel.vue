<script setup lang="ts">
import type { DeepReadonly } from 'vue';
import type { UserInfoResult } from '~/types';
import { ApiError, ERROR_CODES } from '@evolyn.do/utils';
import { ElMessage } from 'element-plus';
import { RiErrorWarningFill, RiInformationFill, RiLinksFill } from '@remixicon/vue';
import { onMounted, shallowRef } from 'vue';
import AccountCancellationDialog from '~/components/dashboard/account/AccountCancellationDialog.vue';
import AccountSessionDrawer from '~/components/dashboard/account/AccountSessionDrawer.vue';
import SecurityReauthDialog from '~/components/dashboard/account/SecurityReauthDialog.vue';
import TotpEnrollmentDialog from '~/components/dashboard/account/TotpEnrollmentDialog.vue';
import { cancelMyAccount, disableMyTOTP, updateMySingleSession } from '~/api/account';
import { useAccountSecurityOverview } from '~/composables/useAccountSecurityOverview';

defineOptions({ name: 'AccountSecurityPanel' });

const props = defineProps<{
  userInfo: DeepReadonly<UserInfoResult> | null;
}>();
const emit = defineEmits<{
  accountCancelled: [];
}>();

const { overview, loading, loadError, loadOverview } = useAccountSecurityOverview();
const sessionDrawerVisible = shallowRef(false);
const totpEnrollmentVisible = shallowRef(false);
const accountCancellationVisible = shallowRef(false);
const reauthDialogVisible = shallowRef(false);
const pendingAction = shallowRef<'cancel-account' | 'disable-totp' | 'single-session' | null>(null);
const securityActionLoading = shallowRef(false);
const requestedSingleSession = shallowRef(false);

// 该组件仅在「账号安全」页签激活时挂载，因此首次挂载即代表用户进入该页签。
onMounted(loadOverview);

function handleSessionRevoked() {
  void loadOverview();
}

function handleTotpCompleted() {
  void loadOverview();
}

function requestDisableTOTP() {
  pendingAction.value = 'disable-totp';
  reauthDialogVisible.value = true;
}

function requestSingleSessionUpdate(value: string | number | boolean) {
  if (typeof value !== 'boolean' || !overview.value || securityActionLoading.value) return;
  pendingAction.value = 'single-session';
  reauthDialogVisible.value = true;
  // 将用户本次开关意图保存在同一个受控状态中，二次验证成功后才真正提交。
  requestedSingleSession.value = value;
}

function requestAccountCancellation() {
  accountCancellationVisible.value = true;
}

function confirmAccountCancellation() {
  accountCancellationVisible.value = false;
  pendingAction.value = 'cancel-account';
  reauthDialogVisible.value = true;
}

function handleReauthDialogVisibility(visible: boolean) {
  reauthDialogVisible.value = visible;
  if (!visible && !securityActionLoading.value) pendingAction.value = null;
}

async function applySecurityAction(reauthToken: string) {
  const action = pendingAction.value;
  if (!action) return;

  securityActionLoading.value = true;
  try {
    if (action === 'disable-totp') {
      await disableMyTOTP(reauthToken);
      ElMessage.success('登录二次验证已关闭，其他设备已退出');
    } else if (action === 'single-session') {
      await updateMySingleSession({ reauthToken, enabled: requestedSingleSession.value });
      ElMessage.success(requestedSingleSession.value ? '已开启禁止同时登录' : '已允许多设备登录');
    } else {
      await cancelMyAccount(reauthToken);
      ElMessage.success('账号已注销');
      emit('accountCancelled');
      return;
    }
    reauthDialogVisible.value = false;
    pendingAction.value = null;
    await loadOverview();
  } catch (error) {
    if (
      action === 'cancel-account' &&
      error instanceof ApiError &&
      error.errCode === ERROR_CODES.ACCOUNT_OWNS_TENANT
    ) {
      ElMessage.error('注销失败。若你仍是团队创建人，请先转移创建人或注销团队。');
    } else if (action === 'cancel-account') {
      ElMessage.error('账号注销失败，请稍后重试。');
    } else {
      ElMessage.error('安全设置更新失败，请稍后重试');
    }
  } finally {
    securityActionLoading.value = false;
  }
}
</script>

<template>
  <section class="account-security" aria-label="账号安全">
    <div class="account-security__row">
      <div>
        <strong>登录二次验证</strong>
        <el-tooltip content="绑定验证器后，登录时需额外输入动态码" placement="top">
          <el-icon class="account-security__help"><RiInformationFill /></el-icon>
        </el-tooltip>
      </div>
      <div class="account-security__status">
        <el-switch
          :model-value="overview?.mfaEnabled ?? false"
          disabled
          aria-label="登录二次验证状态"
        />
        <el-tag v-if="overview?.totpEnrolled" size="small" type="success" effect="plain"
          >已绑定验证器</el-tag
        >
        <span v-else class="account-security__muted">未绑定验证器</span>
        <el-button
          v-if="!overview?.totpEnrolled"
          link
          type="primary"
          :disabled="!props.userInfo?.account.passwordInitialized"
          @click="totpEnrollmentVisible = true"
        >
          绑定验证器
        </el-button>
        <el-button v-else link type="danger" @click="requestDisableTOTP">关闭验证器</el-button>
      </div>
    </div>
    <div class="account-security__row">
      <div>
        <strong>禁止同时登录</strong>
        <el-tooltip content="开启后，新设备登录会自动退出其他设备" placement="top">
          <el-icon class="account-security__help"><RiInformationFill /></el-icon>
        </el-tooltip>
      </div>
      <div class="account-security__status">
        <el-switch
          :model-value="overview?.singleSessionEnabled ?? false"
          :loading="securityActionLoading && pendingAction === 'single-session'"
          :disabled="!overview || !props.userInfo?.account.passwordInitialized"
          aria-label="禁止同时登录状态"
          @update:model-value="requestSingleSessionUpdate"
        />
        <span class="account-security__muted">
          {{ overview?.singleSessionEnabled ? '仅保留当前设备会话' : '允许多设备登录' }}
        </span>
      </div>
    </div>

    <div
      class="account-security__summary"
      :class="{ 'account-security__summary--loading': loading }"
    >
      <span>活跃设备 {{ overview?.activeSessions ?? '-' }}</span>
      <span>剩余恢复码 {{ overview?.recoveryCodesRemaining ?? '-' }}</span>
      <el-button link type="primary" @click="sessionDrawerVisible = true">管理设备</el-button>
      <el-button link type="primary" :loading="loading" @click="loadOverview">刷新</el-button>
    </div>
    <p v-if="loadError" class="account-security__error" role="alert">{{ loadError }}</p>

    <div class="account-security__binding">
      <strong>账号绑定</strong>
      <div class="account-security__binding-content">
        <template v-if="userInfo?.account.authInfos.length">
          <div
            v-for="authInfo in userInfo.account.authInfos"
            :key="authInfo.id"
            class="account-security__provider"
          >
            <el-icon><RiLinksFill /></el-icon>
            <span>{{ authInfo.authType }}</span>
            <el-tag size="small" type="success" effect="plain">已绑定</el-tag>
          </div>
        </template>
        <span v-else class="account-security__muted">暂未绑定第三方账号</span>
        <el-tooltip content="第三方账号绑定能力正在建设中" placement="top">
          <el-button link type="primary" disabled>管理绑定</el-button>
        </el-tooltip>
      </div>
    </div>

    <el-divider />

    <div class="account-security__danger">
      <strong>
        <el-icon><RiErrorWarningFill /></el-icon>
        账号注销
      </strong>
      <span>
        <el-button link type="danger" @click="requestAccountCancellation">注销</el-button>
      </span>
    </div>

    <AccountCancellationDialog
      v-model="accountCancellationVisible"
      @confirm="confirmAccountCancellation"
    />
    <AccountSessionDrawer v-model="sessionDrawerVisible" @session-revoked="handleSessionRevoked" />
    <TotpEnrollmentDialog
      v-model="totpEnrollmentVisible"
      :password-initialized="props.userInfo?.account.passwordInitialized ?? false"
      @completed="handleTotpCompleted"
    />
    <SecurityReauthDialog
      :model-value="reauthDialogVisible"
      :title="
        pendingAction === 'cancel-account'
          ? '验证身份后注销账号'
          : pendingAction === 'disable-totp'
            ? '关闭登录二次验证'
            : '验证身份'
      "
      :description="
        pendingAction === 'cancel-account'
          ? '为保护账号安全，请输入当前密码。验证成功后会立即注销且无法恢复。'
          : pendingAction === 'disable-totp'
            ? '关闭后，登录将不再要求验证器动态码；其他设备会自动退出。'
            : '修改禁止同时登录设置前，需要再次确认是你本人操作。'
      "
      :confirm-text="
        pendingAction === 'cancel-account'
          ? '确认注销'
          : pendingAction === 'disable-totp'
            ? '确认关闭'
            : '继续'
      "
      :password-initialized="props.userInfo?.account.passwordInitialized ?? false"
      :action-loading="securityActionLoading"
      @update:model-value="handleReauthDialogVisibility"
      @verified="applySecurityAction"
    />
  </section>
</template>

<style scoped lang="scss">
.account-security {
  &__row,
  &__binding,
  &__danger {
    display: grid;
    grid-template-columns: 162px minmax(0, 1fr);
    align-items: center;
    min-height: 54px;
  }

  &__status,
  &__summary {
    display: flex;
    align-items: center;
    gap: var(--el-space-md);
  }

  &__summary {
    min-height: 42px;
    padding: 0 0 var(--el-space-md) 162px;
    color: var(--el-text-color-secondary);
  }

  &__summary--loading {
    opacity: 0.72;
  }

  &__error {
    margin: -4px 0 var(--el-space-md)
      162px;
    color: var(--el-color-danger);
    font-size: var(--el-font-size-small);
  }

  &__row > div,
  &__binding > strong,
  &__danger > strong {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  &__help {
    margin-left: var(--el-space-xs);
    vertical-align: -2px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }

  &__binding {
    align-items: start;
    padding: var(--el-space-lg) 0;
  }

  &__binding-content {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    min-height: 30px;
    gap: var(--el-space-lg);
  }

  &__provider {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-sm);
    color: var(--el-text-color-regular);
  }

  &__muted,
  &__danger > span {
    color: var(--el-text-color-secondary);
  }

  &__danger > strong {
    display: inline-flex;
    align-items: center;
    gap: var(--el-space-sm);
  }
}
</style>
