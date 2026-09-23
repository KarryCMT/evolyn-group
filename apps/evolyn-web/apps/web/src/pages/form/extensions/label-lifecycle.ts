import type { LabelSchema } from '@evolyn.do/label';
import type { LabelTemplateDetailDto } from '~/api/label';
import { ApiError } from '@evolyn.do/utils';

export type LabelLifecycleAction = 'load' | 'preview' | 'publish' | 'render' | 'save';

export interface LabelLifecycleFeedback {
  message: string;
  reload: boolean;
  tone: 'error' | 'warning';
}

/** 草稿保存只推进修订号；旧 previewedDraftRevision 原样保留即自然失效。 */
export function withSavedLabelDraft(
  target: LabelTemplateDetailDto,
  schema: LabelSchema,
  draftRevision: number,
): LabelTemplateDetailDto {
  return { ...target, draft: schema, draftRevision };
}

export function withPreviewedLabelDraft(
  target: LabelTemplateDetailDto,
): LabelTemplateDetailDto {
  return { ...target, previewedDraftRevision: target.draftRevision };
}

export function withPublishedLabelDraft(
  target: LabelTemplateDetailDto,
  versionNo: number,
): LabelTemplateDetailDto {
  return {
    ...target,
    status: 'published',
    publishedVersion: versionNo,
    publishedDraftRevision: target.draftRevision,
  };
}

export function isCurrentLabelDraftPreviewed(target: LabelTemplateDetailDto): boolean {
  return target.previewedDraftRevision === target.draftRevision;
}

/**
 * 将稳定业务码收敛为不泄露资源存在性的页面反馈。调用方不得按后端中文文案
 * 判断分支；未知错误才透出已经过请求层脱敏的 message。
 */
export function labelLifecycleFeedback(
  error: unknown,
  action: LabelLifecycleAction,
): LabelLifecycleFeedback {
  if (!(error instanceof ApiError)) {
    return {
      message: error instanceof Error ? error.message : '二维码标签操作失败',
      reload: false,
      tone: 'error',
    };
  }

  switch (error.errCode) {
    case 'LABEL_REVISION_CONFLICT':
      return { message: '标签配置已被他人更新，正在重新加载', reload: true, tone: 'warning' };
    case 'LABEL_REAL_PREVIEW_REQUIRED':
      return {
        message: '请先使用一条真实表单记录完成预览校验',
        reload: false,
        tone: 'warning',
      };
    case 'LABEL_FORM_ALREADY_BOUND':
      return { message: '该表单已经绑定标签模板', reload: true, tone: 'warning' };
    case 'LABEL_SCHEMA_INVALID':
      return { message: '标签配置校验失败，请检查字段和样式', reload: false, tone: 'warning' };
    case 'LABEL_FIELD_NOT_FOUND':
    case 'LABEL_FIELD_NO_PERMISSION':
      return { message: '标签包含不存在或无权访问的表单字段', reload: false, tone: 'warning' };
    case 'LABEL_RECORD_NOT_FOUND':
    case 'LABEL_RECORD_NO_PERMISSION':
      return { message: '记录不存在或无权访问', reload: false, tone: 'warning' };
    case 'FORBIDDEN':
    case 'NOT_FOUND':
    case 'LABEL_TEMPLATE_NOT_FOUND':
      return {
        message: action === 'load' ? '没有权限访问二维码标签配置' : '标签模板不存在或无权访问',
        reload: false,
        tone: 'error',
      };
    default:
      return { message: error.message || '二维码标签操作失败', reload: false, tone: 'error' };
  }
}
