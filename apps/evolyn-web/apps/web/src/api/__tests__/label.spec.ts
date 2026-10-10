import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  createLabelBatchRender,
  downloadLabelRenderTask,
  getFormLabelProfile,
  previewFormLabel,
} from '../label';

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}));

vi.mock('@evolyn.do/utils', () => ({
  ApiError: class ApiError extends Error {},
  http: { get: mocks.get, post: mocks.post, put: vi.fn(), delete: vi.fn() },
  useGlobSetting: () => ({ apiUrl: '/api/v1' }),
}));

describe('二维码标签运行态 API', () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    vi.unstubAllGlobals();
  });

  it('按表单编码读取 profile 并创建固定预设的批量任务', async () => {
    mocks.get.mockResolvedValue({ available: true });
    mocks.post.mockResolvedValue({ taskId: 'lrt_test', status: 'pending' });
    await getFormLabelProfile('form users');
    expect(mocks.get).toHaveBeenCalledWith('/forms/form%20users/labels/profile');

    await createLabelBatchRender({
      formCode: 'form users', recordIds: ['7', '3'], outputPresetId: 'small',
    });
    expect(mocks.post).toHaveBeenCalledWith('/forms/form%20users/labels/batch-render', {
      recordIds: ['7', '3'], outputPresetId: 'small', format: 'pdf',
    });
  });

  it('预览请求返回 SVG Blob，并透传取消信号', async () => {
    const signal = new AbortController().signal;
    const fetchMock = vi.fn().mockResolvedValue(new Response('<svg/>', {
      status: 200, headers: { 'Content-Type': 'image/svg+xml' },
    }));
    vi.stubGlobal('fetch', fetchMock);
    const blob = await previewFormLabel('form_users', { recordId: '8', outputPresetId: 'large' }, signal);
    expect(blob.type).toContain('image/svg+xml');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/forms/form_users/labels/preview', expect.objectContaining({
      method: 'POST', signal, body: JSON.stringify({ recordId: '8', outputPresetId: 'large' }),
    }));
  });

  it('先取得临时地址再下载完成任务文件', async () => {
    mocks.get.mockResolvedValue({ method: 'GET', url: 'https://files.lingyanyun.com/label.pdf', headers: { 'x-test': '1' } });
    const fetchMock = vi.fn().mockResolvedValue(new Response('pdf', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    await downloadLabelRenderTask('lrt_test');
    expect(mocks.get).toHaveBeenCalledWith('/labels/render-tasks/lrt_test/download');
    expect(fetchMock).toHaveBeenCalledWith('https://files.lingyanyun.com/label.pdf', {
      method: 'GET', headers: { 'x-test': '1' },
    });
  });
});
