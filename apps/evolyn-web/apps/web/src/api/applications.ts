import type {
  ApplicationItem,
  ApplicationListQuery,
  ApplicationPage,
  CreateBlankApplicationPayload,
  UpdateApplicationPayload,
} from '~/types';
// 应用管理域接口：与后端 /api/v1/applications* 一一对应
// （见 evolyn-core internal/platform/application/controller/application.go）
import { http } from '@evolyn.do/utils';

/**
 * 创建空白应用（POST /applications）：后端单事务完成配额校验与应用/安装
 * 记录写入，返回 provisionStatus=ready 的应用详情；超限抛 errCode=QUOTA_EXCEEDED
 */
export function createBlankApplication(
  payload: CreateBlankApplicationPayload,
): Promise<ApplicationItem> {
  return http.post('/applications', payload);
}

/** 当前租户应用列表（游标分页）：keyword 按名称模糊，cursor 原样回传 */
export function listApplications(query: ApplicationListQuery = {}): Promise<ApplicationPage> {
  return http.get('/applications', {
    keyword: query.keyword,
    status: query.status,
    limit: query.limit,
    cursor: query.cursor,
  });
}

/** 应用详情（含当前成员运行时 capabilities） */
export function getApplication(id: number): Promise<ApplicationItem> {
  return http.get(`/applications/${id}`);
}

/**
 * 按编码查询应用详情（GET /applications/code/:code）：code 租户内唯一，
 * 响应结构与按 ID 查询一致；工作区等以 code 定位应用的入口使用
 */
export function getApplicationByCode(code: string): Promise<ApplicationItem> {
  return http.get(`/applications/code/${code}`);
}

/**
 * 更新应用（PATCH /applications/:id）：白名单字段 name/icon/color/sortOrder/status；
 * status 仅 active↔archived 互转（归档/恢复）
 */
export function updateApplication(
  id: number,
  payload: UpdateApplicationPayload,
): Promise<ApplicationItem> {
  return http.patch(`/applications/${id}`, payload);
}

/** 删除应用（软删）：初始化进行中的应用会返回 errCode=APP_PROVISIONING */
export function deleteApplication(id: number): Promise<null> {
  return http.delete(`/applications/${id}`);
}
