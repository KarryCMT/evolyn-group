import type { LabelSchema } from '@evolyn.do/label';
import type { LabelTemplateDetailDto } from '~/api/label';
import { ApiError } from '@evolyn.do/utils';
import { describe, expect, it } from 'vitest';
import {
  isCurrentLabelDraftPreviewed,
  labelLifecycleFeedback,
  withPreviewedLabelDraft,
  withPublishedLabelDraft,
  withSavedLabelDraft,
} from '../label-lifecycle';

const schema: LabelSchema = {
  schemaVersion: '1.0',
  name: '资产标签',
  page: { width: 80, height: 50, unit: 'mm', dpi: 300, background: '#ffffff' },
  source: { type: 'form', formId: 'form_assets' },
  elements: [],
  settings: { snapToGrid: true, gridSize: 1, showGrid: false },
};

function detail(): LabelTemplateDetailDto {
  return {
    code: 'label_test',
    name: '资产标签',
    description: '',
    appId: 1,
    formCode: 'form_assets',
    status: 'published',
    publishedVersion: 1,
    draftRevision: 3,
    previewedDraftRevision: 3,
    publishedDraftRevision: 3,
    width: 80,
    height: 50,
    unit: 'mm',
    dpi: 300,
    creatorMemberId: 1,
    createdAt: '2026-09-23 10:00:00',
    updatedAt: '2026-09-23 10:00:00',
    draft: schema,
  };
}

describe('label lifecycle state', () => {
  it('保存新草稿推进修订并使旧真实预览自然失效', () => {
    const saved = withSavedLabelDraft(detail(), { ...schema, name: '资产标签 V2' }, 4);

    expect(saved.draftRevision).toBe(4);
    expect(saved.previewedDraftRevision).toBe(3);
    expect(saved.publishedDraftRevision).toBe(3);
    expect(isCurrentLabelDraftPreviewed(saved)).toBe(false);
  });

  it('真实预览和发布只推进各自精确修订标记', () => {
    const saved = withSavedLabelDraft(detail(), schema, 4);
    const previewed = withPreviewedLabelDraft(saved);
    const published = withPublishedLabelDraft(previewed, 2);

    expect(isCurrentLabelDraftPreviewed(previewed)).toBe(true);
    expect(published).toMatchObject({
      status: 'published',
      publishedVersion: 2,
      publishedDraftRevision: 4,
      previewedDraftRevision: 4,
    });
  });
});

describe('label lifecycle error feedback', () => {
  it('修订冲突要求重新加载且不执行静默覆盖', () => {
    expect(
      labelLifecycleFeedback(
        new ApiError('conflict', 409, 'LABEL_REVISION_CONFLICT'),
        'save',
      ),
    ).toEqual({ message: '标签配置已被他人更新，正在重新加载', reload: true, tone: 'warning' });
  });

  it.each(['LABEL_RECORD_NOT_FOUND', 'LABEL_RECORD_NO_PERMISSION'])(
    '%s 使用相同安全文案，不泄露记录是否存在',
    (errCode) => {
      const feedback = labelLifecycleFeedback(new ApiError('server detail', 403, errCode), 'preview');
      expect(feedback.message).toBe('记录不存在或无权访问');
      expect(feedback.message).not.toContain('server detail');
    },
  );

  it.each(['LABEL_FIELD_NOT_FOUND', 'LABEL_FIELD_NO_PERMISSION'])(
    '%s 使用相同安全文案，不泄露字段是否存在',
    (errCode) => {
      expect(labelLifecycleFeedback(new ApiError('field_x', 403, errCode), 'preview').message).toBe(
        '标签包含不存在或无权访问的表单字段',
      );
    },
  );
});
