// 企业自定义工作台接口：与后端 /api/v1/workbench 一一对应
// （见 evolyn-core internal/platform/workbench/controller/workbench.go）。
// 工作台是租户级资产（000078）：企业管理员配置、全员共用；租户开通时
// 服务端已种子默认布局，读取行恒存在（缺失时 data 为 null，前端回退本地默认）
import { http } from '@evolyn.do/utils';
import type { DashboardSchema } from '~/types/dashboard';

/** 工作台读取/保存响应：document 为 DashboardSchema 原文 */
export interface WorkbenchView {
  revision: number;
  document: DashboardSchema;
}

/** 保存请求：revision 为乐观锁口令（首次保存传 0，之后携带上次返回值） */
export interface SaveWorkbenchInput {
  revision: number;
  document: DashboardSchema;
}

/**
 * 查询企业工作台；行缺失时后端 data 为 null（前端回退默认布局）。
 * 返回 null 而不是抛错，供持久化适配器按「无记录」语义消费。
 */
export function getWorkbench(): Promise<WorkbenchView | null> {
  return http.get<WorkbenchView | null>('/workbench');
}

/**
 * 全量保存企业工作台（仅企业管理员，无权限返回 403 FORBIDDEN），
 * 返回服务端确认后的文档与最新 revision
 */
export function saveWorkbench(data: SaveWorkbenchInput): Promise<WorkbenchView> {
  return http.put<WorkbenchView>('/workbench', data);
}
