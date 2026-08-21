// 账号自助接口：与后端 /api/v1/accounts/me 一一对应
// （见 evolyn-core internal/platform/iam/controller/account.go）
import { http } from './http';
import type { AccountInfo, AccountProfilePayload } from '~/types';

/** 更新我的账号资料：昵称/邮箱/头像与注册引导画像（角色/了解渠道） */
export function updateMyProfile(payload: AccountProfilePayload): Promise<AccountInfo> {
  return http.put('/accounts/me', payload);
}

/** 修改登录密码：短信免密注册的账号首次设置可免旧密码（oldPassword 留空） */
export function changeMyPassword(payload: {
  oldPassword?: string;
  newPassword: string;
}): Promise<null> {
  return http.put('/accounts/me/password', payload);
}
