// 账号自助接口：与后端 /api/v1/accounts/me 一一对应
// （见 evolyn-core internal/platform/iam/controller/account.go）
import { http } from './http'
import type { AccountProfilePayload } from '~/types'

/** 更新我的账号资料：昵称/邮箱/头像与注册引导画像（角色/了解渠道） */
export function updateMyProfile(payload: AccountProfilePayload): Promise<null> {
  return http.put('/accounts/me', payload)
}
