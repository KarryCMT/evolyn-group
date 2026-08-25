import type { AccountProfilePayload } from '~/types';

/** 个人设置页的内容分区。 */
export type AccountSettingsTab = 'basic' | 'security';

/** 资料弹窗仅暴露当前后端可安全保存的账号字段。 */
export type AccountProfileForm = Required<Pick<AccountProfilePayload, 'nickname'>> &
  Pick<AccountProfilePayload, 'email' | 'avatar'>;

/** 改密弹窗提交明文；组合式函数统一加密后再调用接口。 */
export interface AccountPasswordForm {
  oldPassword?: string;
  newPassword: string;
  /** 首次设置密码经短信验证后的验证码，由找回密码接口在最终保存时原子校验。 */
  smsCode?: string;
}
