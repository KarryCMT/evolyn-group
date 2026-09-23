import type { LabelSchema } from '@evolyn.do/label';
import type { LabelTemplateDetailDto } from '~/api/label';
import { flushPromises, shallowMount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, shallowRef } from 'vue';
import QrCodeSettingsPage from '../qrcode.vue';
import { formWorkspaceContextKey } from '../../workspace-context';

const api = vi.hoisted(() => ({
  createLabelTemplate: vi.fn(),
  deleteLabelTemplate: vi.fn(),
  getLabelTemplate: vi.fn(),
  listLabelTemplates: vi.fn(),
  previewLabelTemplate: vi.fn(),
  publishLabelTemplate: vi.fn(),
  renderPublishedLabel: vi.fn(),
  saveLabelTemplateDraft: vi.fn(),
}));

vi.mock('~/api/label', () => api);

const ButtonStub = defineComponent({
  name: 'ElButton',
  inheritAttrs: false,
  setup(_props, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.());
  },
});

const emptySchema: LabelSchema = {
  schemaVersion: '1.0',
  name: '资产标签',
  page: { width: 90, height: 60, unit: 'mm', dpi: 300, background: '#ffffff' },
  source: { type: 'form', formId: 'form_assets' },
  elements: [],
  settings: { snapToGrid: true, gridSize: 1, showGrid: false },
};

function templateDetail(): LabelTemplateDetailDto {
  return {
    code: 'label_test',
    name: '资产标签',
    description: '',
    appId: 1,
    formCode: 'form_assets',
    status: 'draft',
    publishedVersion: 0,
    draftRevision: 1,
    previewedDraftRevision: 0,
    publishedDraftRevision: 0,
    width: 90,
    height: 60,
    unit: 'mm',
    dpi: 300,
    creatorMemberId: 1,
    createdAt: '2026-09-23 10:00:00',
    updatedAt: '2026-09-23 10:00:00',
    draft: emptySchema,
  };
}

describe('qrcode label settings page', () => {
  it('按表单懒创建模板，保存草稿不会隐式发布', async () => {
    api.listLabelTemplates.mockResolvedValue({ items: [], nextCursor: '' });
    api.createLabelTemplate.mockResolvedValue(templateDetail());
    api.saveLabelTemplateDraft.mockResolvedValue({ draftRevision: 2 });

    const detail = shallowRef({
      code: 'form_assets',
      name: '资产登记',
      draft: { content: { items: [] } },
    });
    const wrapper = shallowMount(QrCodeSettingsPage, {
      global: {
        provide: {
          [formWorkspaceContextKey as symbol]: {
            detail,
            loading: shallowRef(false),
            loadFailed: shallowRef(false),
            renaming: shallowRef(false),
            setDetail: vi.fn(),
            patchDetail: vi.fn(),
            rename: vi.fn(),
            reload: vi.fn(),
          },
        },
        stubs: {
          'el-button': ButtonStub,
          'el-input': true,
          'el-option': true,
          'el-select': true,
          'el-switch': true,
        },
      },
    });
    await flushPromises();

    expect(api.listLabelTemplates).toHaveBeenCalledWith({ formCode: 'form_assets', limit: 1 });
    const saveButton = wrapper.findAll('button').find((button) => button.text().includes('保存草稿'));
    expect(saveButton).toBeDefined();

    await saveButton!.trigger('click');
    await flushPromises();

    expect(api.createLabelTemplate).toHaveBeenCalledWith(
      expect.objectContaining({ formCode: 'form_assets' }),
    );
    expect(api.saveLabelTemplateDraft).toHaveBeenCalledWith(
      'label_test',
      expect.objectContaining({ draftRevision: 1 }),
    );
    expect(api.publishLabelTemplate).not.toHaveBeenCalled();
  });
});
