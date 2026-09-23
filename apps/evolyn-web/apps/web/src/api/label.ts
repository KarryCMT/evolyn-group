import type { LabelSchema } from '@evolyn.do/label';
import { ApiError, http, useGlobSetting } from '@evolyn.do/utils';

export interface LabelTemplateSummaryDto {
  code: string;
  name: string;
  description: string;
  appId: number;
  formCode: string;
  status: 'draft' | 'published' | 'disabled';
  publishedVersion: number;
  draftRevision: number;
  previewedDraftRevision: number;
  publishedDraftRevision: number;
  width: number;
  height: number;
  unit: 'mm' | 'px';
  dpi: 96 | 203 | 300 | 600;
  creatorMemberId: number;
  createdAt: string;
  updatedAt: string;
}

export interface LabelTemplateDetailDto extends LabelTemplateSummaryDto {
  draft: LabelSchema;
}

export interface LabelTemplatePageDto {
  items: LabelTemplateSummaryDto[];
  nextCursor: string;
}

export function listLabelTemplates(query: {
  formCode?: string;
  keyword?: string;
  status?: 'draft' | 'published' | 'disabled';
  limit?: number;
  cursor?: string;
}): Promise<LabelTemplatePageDto> {
  return http.get('/label-templates', query);
}

export function getLabelTemplate(code: string): Promise<LabelTemplateDetailDto> {
  return http.get(`/label-templates/${code}`);
}

export function createLabelTemplate(payload: {
  name: string;
  description?: string;
  formCode: string;
  width: number;
  height: number;
  unit: 'mm' | 'px';
  dpi: 96 | 203 | 300 | 600;
}): Promise<LabelTemplateDetailDto> {
  return http.post('/label-templates', payload);
}

export function saveLabelTemplateDraft(
  code: string,
  payload: { draftRevision: number; schema: LabelSchema },
): Promise<{ draftRevision: number }> {
  return http.put(`/label-templates/${code}/draft`, payload);
}

export function publishLabelTemplate(
  code: string,
  draftRevision: number,
): Promise<{ versionNo: number }> {
  return http.post(`/label-templates/${code}/publish`, { draftRevision });
}

export function deleteLabelTemplate(code: string): Promise<void> {
  return http.delete(`/label-templates/${code}`);
}

async function requestLabelFile(path: string, payload: Record<string, unknown>): Promise<Blob> {
  const { apiUrl } = useGlobSetting();
  const response = await fetch(`${apiUrl}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    let message = '标签生成失败';
    let errCode: string | undefined;
    let data: unknown;
    try {
      const body = (await response.json()) as { data?: unknown; errCode?: string; msg?: string };
      if (body.msg) message = body.msg;
      errCode = body.errCode;
      data = body.data;
    } catch {
      // 非 JSON 错误响应使用稳定兜底文案。
    }
    // 文件流接口也必须沿用统一业务错误类型，页面才能按稳定 errCode 分支，
    // 禁止退化为文案匹配或泄露服务端内部错误。
    throw new ApiError(message, response.status, errCode, data);
  }
  return response.blob();
}

/** 使用当前草稿与一条真实记录预览，服务端仍执行记录和字段权限校验。 */
export function previewLabelTemplate(
  code: string,
  payload: { recordId: string; format: 'svg' },
): Promise<Blob> {
  return requestLabelFile(`/label-templates/${code}/preview`, payload);
}

/** 正式渲染端点返回文件流，不经过统一 JSON data 信封。 */
export async function renderPublishedLabel(payload: {
  templateCode: string;
  recordId: string;
  format: 'svg';
}): Promise<Blob> {
  return requestLabelFile('/labels/render', payload);
}
