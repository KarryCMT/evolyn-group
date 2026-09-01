// 产品日志接口：与后端 /api/v1/product-logs 一一对应
// （见 evolyn-core internal/platform/productlog/controller/product_log.go）
import { getToken, http, useGlobSetting } from '@evolyn.do/utils';

/** 产品日志行（受控投影：不含请求 ID/用户代理/原始快照/内部资源 ID） */
export interface ProductLogItem {
  actorName: string;
  /** 操作时间：后端 JSONTime 秒级东八区 yyyy-MM-dd HH:mm:ss */
  operatedAt: string;
  categoryCode: string;
  categoryName: string;
  eventName: string;
  /** 所属应用名称快照；非应用内操作为空串（渲染「—」） */
  applicationName: string;
  /** 操作对象（目标资源名称快照；历史行为空） */
  targetName: string;
  /** 服务端脱敏操作详情；历史行降级「历史操作记录」 */
  summary: string;
  ip: string;
}

export interface ProductLogPage {
  items: ProductLogItem[];
  total: number;
}

/** 操作类型筛选项（稳定事件码 + 中文操作名） */
export interface ProductEventOption {
  code: string;
  name: string;
}

/** 日志范围筛选项（含该范围下可选的操作类型清单） */
export interface ProductCategoryOption {
  code: string;
  name: string;
  events: ProductEventOption[];
}

export interface ProductMemberOption {
  memberId: number;
  name: string;
}

export interface ProductApplicationOption {
  applicationId: number;
  code: string;
  name: string;
}

/** 筛选项聚合（分类/事件码 + 操作人 + 应用，均服务端下发） */
export interface ProductLogOptions {
  categories: ProductCategoryOption[];
  members: ProductMemberOption[];
  applications: ProductApplicationOption[];
}

/** 列表与导出共用的筛选条件（日期为 yyyy-MM-dd 东八区闭区间） */
export interface ProductLogFilterQuery {
  categoryCode?: string;
  eventCode?: string;
  memberId?: number;
  applicationId?: number;
  /** 匹配所属应用/操作对象/操作详情（不查原始快照） */
  keyword?: string;
  startAt?: string;
  endAt?: string;
}

export interface ProductLogListQuery extends ProductLogFilterQuery {
  page?: number;
  pageSize?: number;
}

/** 导出任务状态：ready 就绪 / expired 读时投影（已过有效期） */
export type ProductExportStatus = 'pending' | 'ready' | 'failed' | 'expired';

export interface ProductExportTaskView {
  id: number;
  filters: ProductLogFilterQuery;
  total: number;
  status: ProductExportStatus;
  fileName: string;
  expiresAt: string;
  createdAt: string;
}

/** 产品日志分页查询：操作人/应用筛选按 ID（服务端校验租户归属）。 */
export function listProductLogs(query: ProductLogListQuery = {}): Promise<ProductLogPage> {
  return http.get('/product-logs', {
    categoryCode: query.categoryCode,
    eventCode: query.eventCode,
    memberId: query.memberId,
    applicationId: query.applicationId,
    keyword: query.keyword,
    startAt: query.startAt,
    endAt: query.endAt,
    page: query.page,
    pageSize: query.pageSize,
  });
}

/** 产品日志筛选项：产品分类及事件码、可选操作人、有效应用。 */
export function listProductLogOptions(): Promise<ProductLogOptions> {
  return http.get('/product-logs/options');
}

/** 创建导出任务：携带与列表相同的筛选条件；一期同步生成，响应即就绪状态。 */
export function createProductLogExport(
  payload: ProductLogFilterQuery,
): Promise<ProductExportTaskView> {
  return http.post('/product-logs/exports', payload);
}

/** 查询导出任务状态（ready 且已过期投影为 expired）。 */
export function getProductLogExport(id: number): Promise<ProductExportTaskView> {
  return http.get(`/product-logs/exports/${id}`);
}

/**
 * 下载导出文件（CSV）：下载端点返回文件流而非统一响应信封，携带 Bearer
 * 令牌走原生 fetch；失败时解析信封错误文案抛出（页面统一提示）
 */
export async function downloadProductLogExport(id: number): Promise<Blob> {
  const { apiUrl } = useGlobSetting();
  const token = getToken();
  const response = await fetch(`${apiUrl}/product-logs/exports/${id}/download`, {
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
