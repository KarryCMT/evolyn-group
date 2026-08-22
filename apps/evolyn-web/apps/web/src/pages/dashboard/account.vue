<script setup lang="ts">
import type { AccountPasswordForm, AccountProfileForm, AccountSettingsTab } from '~/types/account';
import { shallowRef } from 'vue';
import AccountBasicInfoPanel from '~/components/dashboard/account/AccountBasicInfoPanel.vue';
import AccountSecurityPanel from '~/components/dashboard/account/AccountSecurityPanel.vue';
import AccountSettingsSidebar from '~/components/dashboard/account/AccountSettingsSidebar.vue';
import LoginLogDrawer from '~/components/dashboard/account/LoginLogDrawer.vue';
import PasswordEditorDialog from '~/components/dashboard/account/PasswordEditorDialog.vue';
import ProfileEditorDialog from '~/components/dashboard/account/ProfileEditorDialog.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useAuth } from '~/composables/auth';
import { useAccountSettings } from '~/composables/useAccountSettings';

defineOptions({ name: 'AccountPage' });

const { userInfo } = useAuth();
const { savingPassword, savingProfile, savePassword, saveProfile } = useAccountSettings();
const activeTab = shallowRef<AccountSettingsTab>('basic');
const profileDialogVisible = shallowRef(false);
const passwordDialogVisible = shallowRef(false);
const loginLogVisible = shallowRef(false);

async function handleProfileSubmit(payload: AccountProfileForm) {
  await saveProfile(payload);
  profileDialogVisible.value = false;
}

async function handlePasswordSubmit(payload: AccountPasswordForm) {
  await savePassword(payload);
  passwordDialogVisible.value = false;
}

// 登录日志查询接口待后端落地，抽屉内暂以演示数据展示（见 LoginLogDrawer）。
function handleViewLoginLog() {
  loginLogVisible.value = true;
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
            @edit-profile="profileDialogVisible = true"
            @change-password="passwordDialogVisible = true"
            @view-login-log="handleViewLoginLog"
          />
          <AccountSecurityPanel v-else :user-info="userInfo" />
        </div>
      </section>
    </main>

    <ProfileEditorDialog
      v-model="profileDialogVisible"
      :account="userInfo?.account"
      :loading="savingProfile"
      @submit="handleProfileSubmit"
    />
    <PasswordEditorDialog
      v-model="passwordDialogVisible"
      :password-initialized="userInfo?.account.passwordInitialized ?? true"
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
  background: #f3f3f8;

  &__main {
    display: flex;
    min-height: 0;
    flex: 1;
    justify-content: center;
    padding: 22px 24px 36px;
  }

  &__card {
    display: flex;
    width: min(900px, 100%);
    min-height: 680px;
    overflow: hidden;
    border-radius: 8px;
    background: var(--el-bg-color);
    box-shadow: 0 1px 2px rgb(0 0 0 / 2%);
  }

  &__content {
    min-width: 0;
    flex: 1;
    padding: 26px 28px 40px;
  }
}

@media (max-width: 640px) {
  .account-page {
    height: auto;
    min-height: 100vh;

    &__main {
      padding: 0 12px 24px;
    }

    &__card {
      min-height: 0;
      flex-direction: column;
    }

    &__content {
      padding: 20px;
    }
  }
}
</style>
