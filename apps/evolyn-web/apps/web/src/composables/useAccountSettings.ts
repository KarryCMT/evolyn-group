import { ElMessage } from 'element-plus';
import { ref } from 'vue';
import { changeMyPassword, updateMyProfile } from '~/api/account';
import { encryptPassword } from '~/api/conf';
import { useAuth } from '~/composables/auth';
import type { AccountPasswordForm, AccountProfileForm } from '~/types/account';

/**
 * 个人设置页的写操作收口：资料保存后刷新内存中的登录聚合信息，
 * 密码在此处完成 RSA 加密，避免页面组件接触接口细节。
 */
export function useAccountSettings() {
  const { loadUserInfo } = useAuth();
  const savingProfile = ref(false);
  const savingPassword = ref(false);

  async function saveProfile(payload: AccountProfileForm) {
    savingProfile.value = true;
    try {
      await updateMyProfile(payload);
      await loadUserInfo();
      ElMessage.success('个人资料已保存');
    } finally {
      savingProfile.value = false;
    }
  }

  async function savePassword(payload: AccountPasswordForm) {
    savingPassword.value = true;
    try {
      const [oldPassword, newPassword] = await Promise.all([
        payload.oldPassword ? encryptPassword(payload.oldPassword) : Promise.resolve(undefined),
        encryptPassword(payload.newPassword),
      ]);
      await changeMyPassword({ oldPassword, newPassword });
      await loadUserInfo();
      ElMessage.success('密码已更新');
    } finally {
      savingPassword.value = false;
    }
  }

  return {
    savingProfile,
    savingPassword,
    saveProfile,
    savePassword,
  };
}
