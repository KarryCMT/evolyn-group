import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import QRScanPage from '../QRScanPage.vue';

const mocks = vi.hoisted(() => ({
  resolve: vi.fn(),
  replace: vi.fn(),
}));

vi.mock('~/api/label', () => ({ resolveLabelQRToken: mocks.resolve }));
vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>();
  return {
    ...actual,
    useRoute: () => ({ params: { token: 'scan_token' } }),
    useRouter: () => ({ replace: mocks.replace }),
  };
});

describe('qr scan page', () => {
  beforeEach(() => {
    mocks.resolve.mockReset();
    mocks.replace.mockReset();
  });

  it('权限校验成功后定位到对应表单记录', async () => {
    mocks.resolve.mockResolvedValue({
      targetType: 'form_record',
      appCode: 'app_assets',
      formCode: 'form_assets',
      recordId: '42',
    });
    mount(QRScanPage, { global: { stubs: { ElButton: true } } });
    await flushPromises();

    expect(mocks.resolve).toHaveBeenCalledWith('scan_token');
    expect(mocks.replace).toHaveBeenCalledWith({
      name: 'form-data',
      params: { appCode: 'app_assets', formCode: 'form_assets' },
      query: { recordId: '42', source: 'qrcode' },
    });
  });

  it('无权限或失效时不泄露记录存在性', async () => {
    mocks.resolve.mockRejectedValue(new Error('not found'));
    const wrapper = mount(QRScanPage, { global: { stubs: { ElButton: true } } });
    await flushPromises();

    expect(wrapper.text()).toContain('二维码已失效，或当前账号没有查看这条数据的权限');
    expect(mocks.replace).not.toHaveBeenCalled();
  });
});
