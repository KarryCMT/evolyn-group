import { ElMessage } from 'element-plus';
import { shallowRef } from 'vue';
import { changeMyPassword, updateMyProfile } from '~/api/account';
import { resetPassword } from '~/api/auth';
import { encryptPassword } from '~/api/conf';
import { useAuth } from '~/composables/auth';
import type { AccountPasswordForm, AccountProfileForm } from '~/types/account';

/**
 * 个人设置页的写操作收口：资料保存后刷新内存中的登录聚合信息，
 * 密码在此处完成 RSA 加密，避免页面组件接触接口细节。
 */
export function useAccountSettings() {
  const { loadUserInfo, logout, userInfo } = useAuth();
  const savingProfile = shallowRef(false);
  const savingPassword = shallowRef(false);

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
      if (payload.smsCode && userInfo.value?.account.phone) {
        await resetPassword({
          phone: userInfo.value.account.phone,
          smsCode: payload.smsCode,
          newPassword,
        });
      } else {
        await changeMyPassword({ oldPassword, newPassword });
      }
      // 后端递增账号会话版本，当前会话也会失效；立即清本地令牌并引导重新登录。
      // 当前 token 已被后端拒绝，登出接口可能返回 401；store 的 finally 仍会
      // 清理本地会话，这里吞掉该预期响应，保证后续提示和跳转能够执行。
      await logout().catch(() => {});
      ElMessage.success('密码已更新，请使用新密码重新登录');
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
