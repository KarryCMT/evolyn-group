import { previewLabelTemplate } from '~/api/label';
import { ApiError } from '@evolyn.do/utils';
import { afterEach, describe, expect, it, vi } from 'vitest';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('label file API', () => {
  it('文件流错误保留稳定业务码和安全数据', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 409,
            errCode: 'LABEL_REVISION_CONFLICT',
            msg: '标签模板已被他人更新，请刷新后重试',
            data: { draftRevision: 4 },
          }),
          { status: 409, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    );

    const error = await previewLabelTemplate('label_test', { recordId: '1', format: 'svg' }).catch(
      (reason: unknown) => reason,
    );

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 409,
      errCode: 'LABEL_REVISION_CONFLICT',
      data: { draftRevision: 4 },
    });
  });
});
