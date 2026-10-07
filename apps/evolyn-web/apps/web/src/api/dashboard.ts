import type { BusinessDashboardDatasetResult } from '@evolyn.do/dashboard';
import type {
  DashboardDetail,
  DashboardDocumentV1,
  DashboardDraftSaveResult,
  DashboardFormDataSource,
  DashboardFormFieldCatalog,
} from '~/types';
import { http } from '@evolyn.do/utils';

export const DEFAULT_DASHBOARD_NAME = '未命名仪表盘';

export interface PrecreateDashboardPayload {
  requestId: string;
  name: string;
  parentMenuCode?: string;
}

/** 每一次用户发起的新建意图只生成一次 requestId；失败重试须复用原值。 */
export function createDashboardRequestId(): string {
  return globalThis.crypto.randomUUID();
}

export function precreateDashboard(
  appCode: string,
  payload: PrecreateDashboardPayload,
): Promise<DashboardDetail> {
  return http.post(`/apps/code/${encodeURIComponent(appCode)}/dashboards`, payload);
}

export function getDashboard(code: string): Promise<DashboardDetail> {
  return http.get(`/dashboards/${encodeURIComponent(code)}`);
}

export function updateDashboard(
  code: string,
  payload: { name?: string; icon?: string; color?: string; parentMenuCode?: string },
): Promise<DashboardDetail> {
  return http.patch(`/dashboards/${encodeURIComponent(code)}`, payload);
}

export function deleteDashboard(code: string): Promise<null> {
  return http.delete(`/dashboards/${encodeURIComponent(code)}`);
}

export function saveDashboardDraft(
  code: string,
  expectedRevision: number,
  document: DashboardDocumentV1,
): Promise<DashboardDraftSaveResult> {
  return http.put(`/dashboards/${encodeURIComponent(code)}/draft`, {
    expectedRevision,
    protocolVersion: document.version,
    document,
  });
}

export function listDashboardFormDataSources(code: string): Promise<DashboardFormDataSource[]> {
  return http.get(`/dashboards/${encodeURIComponent(code)}/data-sources/forms`);
}

export function getDashboardFormFieldCatalog(
  code: string,
  formCode: string,
): Promise<DashboardFormFieldCatalog> {
  return http.get(
    `/dashboards/${encodeURIComponent(code)}/data-sources/forms/${encodeURIComponent(formCode)}/fields`,
  );
}

export function previewDashboardWidget(
  code: string,
  widgetId: string,
  draftRevision: number,
  page = 1,
): Promise<BusinessDashboardDatasetResult> {
  return http.post(
    `/dashboards/${encodeURIComponent(code)}/widgets/${encodeURIComponent(widgetId)}/preview-query`,
    { draftRevision, page, filterValues: {} },
  );
}
