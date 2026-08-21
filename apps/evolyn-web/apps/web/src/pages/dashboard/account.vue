<script setup lang="ts">
import { ref } from 'vue';
import { ElMessage } from 'element-plus';
import AccountBasicInfoPanel from '~/components/dashboard/account/AccountBasicInfoPanel.vue';
import AccountSecurityPanel from '~/components/dashboard/account/AccountSecurityPanel.vue';
import AccountSettingsSidebar from '~/components/dashboard/account/AccountSettingsSidebar.vue';
import PasswordEditorDialog from '~/components/dashboard/account/PasswordEditorDialog.vue';
import ProfileEditorDialog from '~/components/dashboard/account/ProfileEditorDialog.vue';
import TopNavigation from '~/components/navigation/TopNavigation.vue';
import { useAccountSettings } from '~/composables/useAccountSettings';
import { useAuth } from '~/composables/auth';
import type { AccountPasswordForm, AccountProfileForm, AccountSettingsTab } from '~/types/account';

defineOptions({ name: 'AccountPage' });

const { userInfo } = useAuth();
const { savingPassword, savingProfile, savePassword, saveProfile } = useAccountSettings();
const activeTab = ref<AccountSettingsTab>('basic');
const profileDialogVisible = ref(false);
const passwordDialogVisible = ref(false);

async function handleProfileSubmit(payload: AccountProfileForm) {
  await saveProfile(payload);
  profileDialogVisible.value = false;
}

async function handlePasswordSubmit(payload: AccountPasswordForm) {
  await savePassword(payload);
  passwordDialogVisible.value = false;
}

// 登录日志尚无服务端查询接口，保留入口并给出明确反馈，避免误导为可用能力。
function handleViewLoginLog() {
  ElMessage.info('登录日志功能正在建设中');
}
</script>

<template>
  <div class="account-page">
    <TopNavigation title="个人设置" back-to="/dashboard" />

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
  </div>
</template>

<style scoped lang="scss">
.account-page {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  background: #f3f3f8;

  /* 与工作台、个人菜单共用的品牌色在本页局部生效。 */
  --el-color-primary: #00b8a9;
  --el-color-primary-light-3: #4dcdc2;
  --el-color-primary-light-7: #b2e9e4;
  --el-color-primary-light-9: #e6f8f6;

  &__main {
    display: flex;
    justify-content: center;
    padding: 0 24px 40px;
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
