import type {
  AppItem,
  AppListQuery,
  AppMenu,
  AppMenuNodeMutation,
  AppMenuGroupMutation,
  AppPage,
  CreateAppMenuGroupPayload,
  CreateBlankAppPayload,
  UpdateAppMenuNodePayload,
  UpdateAppPayload,
} from '~/types';
// 应用管理域接口：与后端 /api/v1/apps* 一一对应
// （见 evolyn-core internal/platform/app/controller/app.go）
import { http } from '@evolyn.do/utils';

/**
 * 创建空白应用（POST /apps）：后端单事务完成配额校验与应用/安装
 * 记录写入，返回 provisionStatus=ready 的应用详情；超限抛 errCode=QUOTA_EXCEEDED
 */
export function createBlankApp(payload: CreateBlankAppPayload): Promise<AppItem> {
  return http.post('/apps', payload);
}

/** 当前租户应用列表（游标分页）：keyword 按名称模糊，cursor 原样回传 */
export function listApps(query: AppListQuery = {}): Promise<AppPage> {
  return http.get('/apps', {
    keyword: query.keyword,
    status: query.status,
    limit: query.limit,
    cursor: query.cursor,
  });
}

/** 应用详情（含当前成员运行时 capabilities） */
export function getApp(id: number): Promise<AppItem> {
  return http.get(`/apps/${id}`);
}

/**
 * 按编码查询应用详情（GET /apps/code/:code）：code 租户内唯一，
 * 响应结构与按 ID 查询一致；工作区等以 code 定位应用的入口使用
 */
export function getAppByCode(code: string): Promise<AppItem> {
  return http.get(`/apps/code/${code}`);
}

/**
 * 按编码读取应用菜单（GET /apps/code/:code/menu）：返回当前成员
 * 可见的菜单树（rootMenuIds + entryMap）；资产域落地前菜单为空数组
 * （空树是合法结果）；应用不存在抛 errCode=APP_NOT_FOUND
 */
export function getAppMenuByCode(code: string): Promise<AppMenu> {
  return http.get(`/apps/code/${code}/menu`);
}

/** 创建根分组或二级子分组；baseMenuRevision 用于拒绝陈旧菜单写入。 */
export function createAppMenuGroup(
  code: string,
  payload: CreateAppMenuGroupPayload,
): Promise<AppMenuGroupMutation> {
  return http.post(`/apps/code/${code}/menu/groups`, payload);
}

/** 分组改名、资产隐藏或移动菜单节点；服务端会校验节点类型、层级和菜单修订号。 */
export function updateAppMenuNode(
  code: string,
  menuCode: string,
  payload: UpdateAppMenuNodePayload,
): Promise<AppMenuNodeMutation> {
  return http.patch(`/apps/code/${code}/menu/nodes/${menuCode}`, payload);
}

/**
 * 更新应用（PATCH /apps/:id）：白名单字段 name/icon/color/sortOrder/status；
 * status 仅 active↔archived 互转（归档/恢复）
 */
export function updateApp(id: number, payload: UpdateAppPayload): Promise<AppItem> {
  return http.patch(`/apps/${id}`, payload);
}

/** 删除应用（软删）：初始化进行中的应用会返回 errCode=APP_PROVISIONING */
export function deleteApp(id: number): Promise<null> {
  return http.delete(`/apps/${id}`);
}
