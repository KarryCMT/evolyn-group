<script setup lang="ts">
import type { AccountPasswordForm, AccountSettingsTab } from '~/types/account';
import { shallowRef } from 'vue';
import { useRouter } from 'vue-router';
import AccountBasicInfoPanel from '~/components/dashboard/account/AccountBasicInfoPanel.vue';
import AccountSecurityPanel from '~/components/dashboard/account/AccountSecurityPanel.vue';
import AccountSettingsSidebar from '~/components/dashboard/account/AccountSettingsSidebar.vue';
import AvatarEditorDialog from '~/components/dashboard/account/AvatarEditorDialog.vue';
import EmailBindingDialog from '~/components/dashboard/account/EmailBindingDialog.vue';
import LoginLogDrawer from '~/components/dashboard/account/LoginLogDrawer.vue';
import PasswordEditorDialog from '~/components/dashboard/account/PasswordEditorDialog.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { uploadAvatar } from '~/api/file';
import { useAuth } from '~/composables/auth';
import { useAccountSettings } from '~/composables/useAccountSettings';

defineOptions({ name: 'AccountPage' });

const { userInfo, loadUserInfo, logout } = useAuth();
const { savingPassword, savingProfile, savePassword, saveProfile } = useAccountSettings();
const router = useRouter();
const activeTab = shallowRef<AccountSettingsTab>('basic');
const emailBindingVisible = shallowRef(false);
const avatarDialogVisible = shallowRef(false);
const avatarDialogSource = shallowRef<File | null>(null);
const passwordDialogVisible = shallowRef(false);
const loginLogVisible = shallowRef(false);

async function handleEmailBound() {
  // 绑定接口独立完成双重校验；仅在成功后刷新顶栏与当前页共享的登录聚合资料。
  await loadUserInfo();
  emailBindingVisible.value = false;
}

async function handleAvatarSubmit(avatar: File) {
  await saveProfile({
    avatar: await uploadAvatar(avatar),
    nickname: userInfo.value?.account.nickname || '',
  });
  avatarDialogVisible.value = false;
}

function handleAvatarEdit(file: File) {
  avatarDialogSource.value = file;
  avatarDialogVisible.value = true;
}

// 通讯录姓名使用与灵衍云一致的行内编辑；后端会在同一事务中同步账号与当前成员昵称。
async function handleContactNameUpdate(nickname: string, onSuccess: () => void) {
  await saveProfile({ nickname });
  onSuccess();
}

async function handlePasswordSubmit(payload: AccountPasswordForm) {
  await savePassword(payload);
  await router.replace('/auth/login');
}

// 打开登录日志抽屉：抽屉打开时自动拉取最新流水（见 LoginLogDrawer）。
function handleViewLoginLog() {
  loginLogVisible.value = true;
}

async function handleAccountCancelled() {
  // 注销接口已删除服务端账号；logout 会在接口返回 401 时仍清掉本地会话。
  try {
    await logout();
  } finally {
    await router.replace('/auth/login');
  }
}
</script>

<template>
  <div class="account-page">
    <!-- 个人设置遵循详情页顶栏：返回、标题、通知和用户菜单。 -->
    <TopNavigation title="个人设置" back-to="/dashboard" :show-help="false" />

    <main class="account-page__main">
      <section class="account-page__card">
        <AccountSettingsSidebar v-model="activeTab" />
        <div class="account-page__content">
          <AccountBasicInfoPanel
            v-if="activeTab === 'basic'"
            :user-info="userInfo"
            :saving-contact-name="savingProfile"
            @bind-email="emailBindingVisible = true"
            @edit-avatar="handleAvatarEdit"
            @update-contact-name="handleContactNameUpdate"
            @change-password="passwordDialogVisible = true"
            @view-login-log="handleViewLoginLog"
          />
          <AccountSecurityPanel
            v-else
            :user-info="userInfo"
            @account-cancelled="handleAccountCancelled"
          />
        </div>
      </section>
    </main>

    <EmailBindingDialog
      v-model="emailBindingVisible"
      :account="userInfo?.account"
      @bound="handleEmailBound"
    />
    <AvatarEditorDialog
      v-model="avatarDialogVisible"
      :source-file="avatarDialogSource"
      :avatar="userInfo?.account.avatar || ''"
      :loading="savingProfile"
      @submit="handleAvatarSubmit"
    />
    <PasswordEditorDialog
      v-model="passwordDialogVisible"
      :password-initialized="userInfo?.account.passwordInitialized ?? true"
      :phone="userInfo?.account.phone || ''"
      :loading="savingPassword"
      @submit="handlePasswordSubmit"
    />
    <LoginLogDrawer
      v-model="loginLogVisible"
      :nickname="userInfo?.member.nickname || userInfo?.account.nickname || ''"
      :avatar="userInfo?.account.avatar || ''"
    />
  </div>
</template>

<style scoped lang="scss">
.account-page {
  display: flex;
  height: 100vh;
  min-height: 720px;
  flex-direction: column;
  overflow: auto;
  background: var(--el-bg-color-page);

  &__main {
    display: flex;
    min-height: 0;
    flex: 1;
    justify-content: center;
    padding: var(--el-space-2xl) var(--el-space-3xl) var(--el-space-4xl);
  }

  &__card {
    display: flex;
    width: min(900px, 100%);
    min-height: 680px;
    overflow: hidden;
    border-radius: var(--el-border-radius-medium);
    background: var(--el-bg-color);
    box-shadow: var(--el-box-shadow-light);
  }

  &__content {
    min-width: 0;
    flex: 1;
    padding: var(--el-space-3xl) var(--el-space-3xl) var(--el-space-5xl);
  }
}

@media (max-width: 640px) {
  .account-page {
    height: auto;
    min-height: 100vh;

    &__main {
      padding: 0 var(--el-space-lg) var(--el-space-3xl);
    }

    &__card {
      min-height: 0;
      flex-direction: column;
    }

    &__content {
      padding: var(--el-space-2xl);
    }
  }
}
</style>
