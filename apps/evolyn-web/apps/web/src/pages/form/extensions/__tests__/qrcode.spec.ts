import type { LabelSchema, TextStyle } from '@evolyn.do/label';
import type { LabelTemplateDetailDto } from '~/api/label';
import { flushPromises, shallowMount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, shallowRef } from 'vue';
import { formWorkspaceContextKey } from '../../workspace-context';
import QrCodeSettingsPage from '../qrcode.vue';

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

function templateDetail(draft: LabelSchema = emptySchema): LabelTemplateDetailDto {
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
    draft,
  };
}

function elementBase(
  id: string,
  x: number,
  y: number,
  width: number,
  height: number,
  zIndex: number,
) {
  return {
    id,
    x,
    y,
    width,
    height,
    rotation: 0,
    zIndex,
    visible: true,
    locked: false,
  };
}

function textStyle(fontSize: number, color: string): TextStyle {
  return {
    fontFamily: 'Noto Sans CJK SC',
    fontSize,
    fontWeight: 600,
    color,
    lineHeight: 1.25,
    textAlign: 'left',
    verticalAlign: 'middle',
    overflow: 'ellipsis',
  };
}

function draftWithSubtitle(): LabelSchema {
  return {
    ...emptySchema,
    elements: [
      {
        ...elementBase('accent-border', 0.7, 0.7, 88.6, 58.6, 1),
        type: 'rect',
        fill: '#ffffff',
        stroke: '#168bf2',
        strokeWidth: 1.4,
      },
      {
        ...elementBase('header-background', 0.7, 0.7, 88.6, 16, 2),
        type: 'rect',
        fill: '#168bf2',
      },
      {
        ...elementBase('title', 5, 1.5, 80, 13, 3),
        type: 'text',
        value: { type: 'static', value: '二维码标签测试' },
        style: { ...textStyle(5.82, '#ffffff'), fontWeight: 700 },
      },
      {
        ...elementBase('subtitle', 5, 18, 46, 6, 4),
        type: 'field',
        label: '测试',
        value: { type: 'field', fieldId: '' },
        separator: ': ',
        style: textStyle(3.7, '#1b2129'),
      },
      {
        ...elementBase('field-name', 5, 21, 46, 6.4, 10),
        type: 'field',
        label: '单行文本1',
        value: { type: 'field', fieldId: 'field_name' },
        separator: ': ',
        style: textStyle(3.7, '#1b2129'),
      },
      {
        ...elementBase('qrcode', 56, 23.5, 29, 29, 30),
        type: 'qrcode',
        value: { type: 'system', key: 'recordId' },
        options: {
          errorCorrection: 'M',
          quietZone: 1,
          foreground: '#000000',
          background: '#ffffff',
        },
      },
    ],
  };
}

describe('qrcode label settings page', () => {
  it('按表单懒创建模板，并以一次保存发布当前修订', async () => {
    api.listLabelTemplates.mockResolvedValue({ items: [], nextCursor: '' });
    api.createLabelTemplate.mockResolvedValue(templateDetail());
    api.saveLabelTemplateDraft.mockResolvedValue({ draftRevision: 2 });
    api.publishLabelTemplate.mockResolvedValue({ versionNo: 1 });

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
    const saveButton = wrapper.findAll('button').find((button) => button.text().trim() === '保存');
    expect(saveButton).toBeDefined();
    expect(wrapper.findAll('button').some((button) => button.text().trim() === '发布')).toBe(false);

    await saveButton!.trigger('click');
    await flushPromises();

    expect(api.createLabelTemplate).toHaveBeenCalledWith(
      expect.objectContaining({ formCode: 'form_assets' }),
    );
    expect(api.saveLabelTemplateDraft).toHaveBeenCalledWith(
      'label_test',
      expect.objectContaining({ draftRevision: 1 }),
    );
    expect(api.publishLabelTemplate).toHaveBeenCalledWith('label_test', 2);
  });

  it('将副标题排在标题栏内并为正文保留独立空间', async () => {
    api.listLabelTemplates.mockResolvedValue({
      items: [{ code: 'label_test' }],
      nextCursor: '',
    });
    api.getLabelTemplate.mockResolvedValue(templateDetail(draftWithSubtitle()));
    api.saveLabelTemplateDraft.mockResolvedValue({ draftRevision: 2 });
    api.publishLabelTemplate.mockResolvedValue({ versionNo: 1 });

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

    const saveButton = wrapper.findAll('button').find((button) => button.text().trim() === '保存');
    expect(saveButton).toBeDefined();
    await saveButton!.trigger('click');
    await flushPromises();

    const saveCall = api.saveLabelTemplateDraft.mock.calls.at(-1);
    expect(saveCall).toBeDefined();
    const schema = saveCall![1].schema as LabelSchema;
    const accentBorder = schema.elements.find((element) => element.id === 'accent-border');
    const header = schema.elements.find((element) => element.id === 'header-background');
    const title = schema.elements.find((element) => element.id === 'title');
    const subtitle = schema.elements.find((element) => element.id === 'subtitle');
    const firstField = schema.elements.find((element) => element.id.startsWith('field-'));
    const qrcode = schema.elements.find((element) => element.id === 'qrcode');
    if (
      accentBorder?.type !== 'rect' ||
      header?.type !== 'rect' ||
      title?.type !== 'text' ||
      subtitle?.type !== 'field' ||
      firstField?.type !== 'field' ||
      qrcode?.type !== 'qrcode'
    ) {
      throw new Error('标签布局元素缺失');
    }

    expect(schema.page.background).toBe('#168bf2');
    expect(accentBorder.stroke).toBe(schema.page.background);
    expect(header.fill).toBe(schema.page.background);
    expect(header.height).toBe(18);
    expect(title.y + title.height).toBeLessThanOrEqual(subtitle.y);
    expect(subtitle.y + subtitle.height).toBeLessThanOrEqual(header.y + header.height);
    expect(subtitle.x).toBe(title.x);
    expect(subtitle.width).toBe(title.width);
    expect(subtitle.style.color).toBe('#ffffff');
    expect(firstField.y).toBeGreaterThanOrEqual(header.y + header.height);
    expect(qrcode.y).toBeGreaterThanOrEqual(header.y + header.height);
  });
});
