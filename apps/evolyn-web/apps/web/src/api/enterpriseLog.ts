// 企业日志接口：与后端 /api/v1/enterprise-logs 一一对应
// （见 evolyn-core internal/platform/enterpriselog/controller/enterprise_log.go）
import { getToken, http, useGlobSetting } from '@evolyn.do/utils';

/** 登录平台形态（后端 UA 解析枚举），展示文案由前端映射 */
export type EnterpriseLogClient = 'web' | 'wap' | 'unknown';

/** 登录日志行（受控投影：不含登录方式/UA/请求 ID） */
export interface EnterpriseLoginLogItem {
  actorName: string;
  /** 登录时间：后端 JSONTime 秒级东八区 yyyy-MM-dd HH:mm:ss */
  loggedAt: string;
  location: string;
  client: EnterpriseLogClient;
  ip: string;
}

export interface EnterpriseLoginLogPage {
  items: EnterpriseLoginLogItem[];
  total: number;
}

/** 操作日志行（受控投影：操作详情为服务端脱敏摘要；历史行降级「历史操作记录」） */
export interface EnterpriseOperationLogItem {
  actorName: string;
  operatedAt: string;
  categoryName: string;
  eventName: string;
  summary: string;
  ip: string;
}

export interface EnterpriseOperationLogPage {
  items: EnterpriseOperationLogItem[];
  total: number;
}

/** 操作类型筛选项（稳定事件码 + 中文操作名） */
export interface OperationEventOption {
  code: string;
  name: string;
}

/** 日志范围筛选项（含该范围下可选的操作类型清单） */
export interface OperationCategoryOption {
  code: string;
  name: string;
  events: OperationEventOption[];
}

/** 列表与导出共用的筛选条件（日期为 yyyy-MM-dd 东八区闭区间） */
export interface EnterpriseLogFilterQuery {
  memberId?: number;
  categoryCode?: string;
  eventCode?: string;
  startAt?: string;
  endAt?: string;
}

export interface EnterpriseLogListQuery extends EnterpriseLogFilterQuery {
  page?: number;
  pageSize?: number;
}

export type EnterpriseLogKind = 'login' | 'operation';

/** 导出任务状态：ready 就绪 / expired 读时投影（已过有效期） */
export type EnterpriseExportStatus = 'pending' | 'ready' | 'failed' | 'expired';

export interface EnterpriseExportTaskView {
  id: number;
  kind: EnterpriseLogKind;
  kindLabel: string;
  filters: EnterpriseLogFilterQuery;
  total: number;
  status: EnterpriseExportStatus;
  fileName: string;
  expiresAt: string;
  createdAt: string;
}

export interface EnterpriseExportPayload extends EnterpriseLogFilterQuery {
  kind: EnterpriseLogKind;
}

/** 企业登录日志分页查询：登录人筛选按成员 ID（服务端校验租户归属）。 */
export function listEnterpriseLoginLogs(
  query: EnterpriseLogListQuery = {},
): Promise<EnterpriseLoginLogPage> {
  return http.get('/enterprise-logs/login', {
    memberId: query.memberId,
    startAt: query.startAt,
    endAt: query.endAt,
    page: query.page,
    pageSize: query.pageSize,
  });
}

/** 企业操作日志分页查询：支持日志范围/操作类型/操作人/时间筛选。 */
export function listEnterpriseOperationLogs(
  query: EnterpriseLogListQuery = {},
): Promise<EnterpriseOperationLogPage> {
  return http.get('/enterprise-logs/operations', {
    memberId: query.memberId,
    categoryCode: query.categoryCode,
    eventCode: query.eventCode,
    startAt: query.startAt,
    endAt: query.endAt,
    page: query.page,
    pageSize: query.pageSize,
  });
}

/** 操作日志筛选项：日志范围与各自可选的操作类型事件码清单。 */
export function listOperationCategories(): Promise<OperationCategoryOption[]> {
  return http.get('/enterprise-logs/operation-categories');
}

/** 创建导出任务：携带与列表相同的筛选条件；一期同步生成，响应即就绪状态。 */
export function createEnterpriseLogExport(
  payload: EnterpriseExportPayload,
): Promise<EnterpriseExportTaskView> {
  return http.post('/enterprise-logs/exports', payload);
}

/** 查询导出任务状态（ready 且已过期投影为 expired）。 */
export function getEnterpriseLogExport(id: number): Promise<EnterpriseExportTaskView> {
  return http.get(`/enterprise-logs/exports/${id}`);
}

/**
 * 下载导出文件（CSV）：下载端点返回文件流而非统一响应信封，携带 Bearer
 * 令牌走原生 fetch；失败时解析信封错误文案抛出（页面统一提示）
 */
export async function downloadEnterpriseLogExport(id: number): Promise<Blob> {
  const { apiUrl } = useGlobSetting();
  const token = getToken();
  const response = await fetch(`${apiUrl}/enterprise-logs/exports/${id}/download`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!response.ok) {
    // 失败响应为统一 {code, errCode, msg} 信封：尽量透出后端文案
    let message = '导出文件下载失败';
    try {
      const body = (await response.json()) as { msg?: string };
      if (body?.msg) message = body.msg;
    } catch {
      /* 非 JSON 响应按兜底文案 */
    }
    throw new Error(message);
  }
  return response.blob();
}
