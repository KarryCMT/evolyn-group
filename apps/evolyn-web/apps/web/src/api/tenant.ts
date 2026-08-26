import { http } from '@evolyn.do/utils';
import type { Tenant } from '~/types';

/** 更新当前登录租户的组织根节点名称，不暴露套餐或配额等运营字段。 */
export function updateMyTenantProfile(payload: { name: string }): Promise<Tenant> {
  return http.put('/tenant/profile', payload);
}
