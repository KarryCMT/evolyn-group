import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import FormRecordLabelDownloadDialog from '../FormRecordLabelDownloadDialog.vue';

const api = vi.hoisted(() => ({
  create: vi.fn(),
  download: vi.fn(),
  profile: vi.fn(),
  task: vi.fn(),
  preview: vi.fn(),
}));
const router = vi.hoisted(() => ({ push: vi.fn() }));
const route = vi.hoisted(() => ({ params: { appCode: 'app_demo' } }));

vi.mock('~/api/label', () => ({
  createLabelBatchRender: api.create,
  downloadLabelRenderTask: api.download,
  getFormLabelProfile: api.profile,
  getLabelRenderTask: api.task,
  previewFormLabel: api.preview,
}));
vi.mock('vue-router', () => ({ useRoute: () => route, useRouter: () => router }));

const profile = {
  available: true,
  templateCode: 'label_user',
  templateName: '用户标签',
  publishedVersion: 3,
  canManageTemplate: true,
  outputPresets: [
    {
      id: 'large',
      name: '大尺寸',
      width: 150,
      height: 100,
      unit: 'mm',
      dpi: 300,
      pixelWidth: 1772,
      pixelHeight: 1181,
    },
    {
      id: 'small',
      name: '小尺寸',
      width: 60,
      height: 40,
      unit: 'mm',
      dpi: 300,
      pixelWidth: 709,
      pixelHeight: 472,
    },
  ],
};

const stubs = {
  ElDialog: {
    props: ['modelValue'],
    template: '<div><slot name="header"/><slot/><slot name="footer"/></div>',
  },
  ElButton: { template: '<button v-bind="$attrs" @click="$emit(\'click\')"><slot/></button>' },
  ElEmpty: {
    props: ['description', 'imageSize'],
    template: '<div :data-image-size="imageSize">{{ description }}<slot/></div>',
  },
  ElSkeleton: { template: '<div>loading</div>' },
  ElAlert: { props: ['title'], template: '<div>{{ title }}</div>' },
  ElProgress: { props: ['percentage'], template: '<div>{{ percentage }}</div>' },
};

function render(recordIds = [11, 12]) {
  return mount(FormRecordLabelDownloadDialog, {
    props: {
      formCode: 'form_users',
      recordIds,
      modelValue: true,
      'onUpdate:modelValue': () => undefined,
    },
    global: { stubs },
  });
}

describe('formRecordLabelDownloadDialog', () => {
  beforeEach(() => {
    api.profile.mockResolvedValue(profile);
    api.preview.mockResolvedValue(new Blob(['<svg/>'], { type: 'image/svg+xml' }));
    api.download.mockResolvedValue(new Blob(['pdf'], { type: 'application/pdf' }));
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:label');
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('按记录和尺寸请求预览，并复用已生成缓存', async () => {
    const wrapper = render();
    await flushPromises();
    expect(api.preview).toHaveBeenCalledWith(
      'form_users',
      { recordId: '11', outputPresetId: 'large' },
      expect.any(AbortSignal),
    );

    await wrapper.get('[aria-label="下一条"]').trigger('click');
    await flushPromises();
    expect(api.preview).toHaveBeenLastCalledWith(
      'form_users',
      { recordId: '12', outputPresetId: 'large' },
      expect.any(AbortSignal),
    );
    await wrapper.get('[aria-label="上一条"]').trigger('click');
    await flushPromises();
    expect(api.preview).toHaveBeenCalledTimes(2);
    wrapper.unmount();
    expect(URL.revokeObjectURL).toHaveBeenCalled();
  });

  it('轮询异步任务并在成功后下载 PDF', async () => {
    vi.useFakeTimers();
    api.create.mockResolvedValue({ taskId: 'lrt_123456789012345678901234', status: 'pending' });
    api.task
      .mockResolvedValueOnce({
        taskId: 'lrt_123456789012345678901234',
        status: 'pending',
        progress: 30,
      })
      .mockResolvedValueOnce({
        taskId: 'lrt_123456789012345678901234',
        status: 'success',
        progress: 100,
      });
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined);
    const wrapper = render([11]);
    await flushPromises();

    const downloadButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('下载文件'));
    expect(downloadButton).toBeDefined();
    await downloadButton!.trigger('click');
    await flushPromises();
    expect(api.create).toHaveBeenCalledWith({
      formCode: 'form_users',
      recordIds: ['11'],
      outputPresetId: 'large',
    });
    await vi.advanceTimersByTimeAsync(1000);
    await flushPromises();
    expect(api.download).toHaveBeenCalledWith('lrt_123456789012345678901234');
    expect(click).toHaveBeenCalled();
  });

  it('无发布模板时展示空状态且不请求预览', async () => {
    api.profile.mockResolvedValue({ available: false, outputPresets: [], canManageTemplate: true });
    const wrapper = render();
    await flushPromises();
    expect(wrapper.text()).toContain('当前表单尚未保存二维码标签配置');
    expect(wrapper.text()).toContain('配置二维码标签');
    expect(wrapper.get('.label-download-dialog__empty').attributes('data-image-size')).toBe('96');
    expect(api.preview).not.toHaveBeenCalled();
  });

  it('已有未生效草稿时引导保存，并携带当前工作区参数', async () => {
    api.profile.mockResolvedValue({
      available: false,
      templateCode: 'label_draft',
      templateName: '用户标签',
      publishedVersion: 0,
      outputPresets: [],
      canManageTemplate: true,
    });
    const wrapper = render();
    await flushPromises();

    expect(wrapper.text()).toContain('二维码标签配置尚未生效，请前往设置页保存后再预览');
    const configureButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('前往保存配置'));
    expect(configureButton).toBeDefined();
    await configureButton!.trigger('click');
    expect(router.push).toHaveBeenCalledWith({
      name: 'form-extension-qrcode',
      params: { appCode: 'app_demo', formCode: 'form_users' },
    });
  });

  it('切换记录会中止旧预览，关闭弹窗释放缓存资源', async () => {
    let resolveFirst!: (value: Blob) => void;
    api.preview
      .mockImplementationOnce(
        () =>
          new Promise<Blob>((resolve) => {
            resolveFirst = resolve;
          }),
      )
      .mockResolvedValueOnce(new Blob(['second'], { type: 'image/svg+xml' }));
    vi.mocked(URL.createObjectURL).mockReset().mockReturnValue('blob:second');
    const wrapper = render();
    await flushPromises();
    const firstSignal = api.preview.mock.calls[0][2] as AbortSignal;
    await wrapper.get('[aria-label="下一条"]').trigger('click');
    await flushPromises();
    expect(firstSignal.aborted).toBe(true);
    expect(wrapper.get('img').attributes('src')).toBe('blob:second');
    resolveFirst(new Blob(['first'], { type: 'image/svg+xml' }));
    await flushPromises();
    expect(wrapper.get('img').attributes('src')).toBe('blob:second');

    await wrapper.setProps({ modelValue: false });
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:second');
  });
});
