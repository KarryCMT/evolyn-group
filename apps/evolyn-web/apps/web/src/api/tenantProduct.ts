// 产品中心域接口：与后端 /api/v1/tenant-products* 一一对应
// （见 evolyn-core internal/platform/tenantproduct/controller/tenant_product.go）。
// 卡片版本信息来自版本信息域事实源，可用范围与成员数由服务端计算。
import { http } from '@evolyn.do/utils';

/** 可用范围模式：all 全部有效成员 / partial 仅选中部门（含子部门）与成员 */
export type TenantProductScopeMode = 'all' | 'partial';

/** 卡片「当前版本」投影：订阅套餐与状态（读时投影，同 /editions/current） */
export interface TenantProductEdition {
  planCode: string;
  planName: string;
  status: string;
}

/** 范围选择项：type 取 department / member，只回传稳定 ID 与当前名称 */
export interface TenantProductScopeSelection {
  type: 'department' | 'member';
  id: number;
  label: string;
}

/** 可用范围视图：悬挂引用（离职/停用等）已被服务端过滤 */
export interface TenantProductAccessScope {
  mode: TenantProductScopeMode;
  eligibleMemberCount: number;
  departmentIds: number[];
  memberIds: number[];
  selections: TenantProductScopeSelection[];
}

/** 产品卡片：revision 为配置乐观锁版本，写操作提交时携带 */
export interface TenantProductCard {
  code: string;
  name: string;
  icon: string;
  enabled: boolean;
  revision: number;
  edition: TenantProductEdition;
  accessScope: TenantProductAccessScope;
  entryPath: string;
}

/** GET /tenant-products 响应 */
export interface TenantProductCenter {
  items: TenantProductCard[];
}

/** 启停请求体 */
export interface UpdateTenantProductEnabledInput {
  enabled: boolean;
  revision: number;
}

/** 范围全量替换请求体：mode=all 时不携带 ID 清单 */
export interface UpdateTenantProductAccessScopeInput {
  mode: TenantProductScopeMode;
  departmentIds?: number[];
  memberIds?: number[];
  revision: number;
}

/** 查询产品中心卡片列表 */
export function getTenantProducts(): Promise<TenantProductCenter> {
  return http.get('/tenant-products');
}

/** 启用或停用产品，成功返回最新卡片 */
export function setTenantProductEnabled(
  code: string,
  data: UpdateTenantProductEnabledInput,
): Promise<TenantProductCard> {
  return http.put(`/tenant-products/${code}/enabled`, data);
}

/** 全量替换产品可用范围，成功返回最新卡片 */
export function updateTenantProductAccessScope(
  code: string,
  data: UpdateTenantProductAccessScopeInput,
): Promise<TenantProductCard> {
  return http.put(`/tenant-products/${code}/access-scope`, data);
}
