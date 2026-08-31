import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import EvolynRichTextEditor from '../EvolynRichTextEditor.vue';

describe('EvolynRichTextEditor', () => {
  it('renders the Remix Icon toolbar and accepts v-model content', async () => {
    const wrapper = mount(EvolynRichTextEditor, {
      props: { modelValue: '<p>初始内容</p>' },
    });

    await vi.waitFor(() => expect(wrapper.find('.tiptap').exists()).toBe(true));
    expect(wrapper.find('[aria-label="加粗"]').exists()).toBe(true);
    expect(wrapper.find('[aria-label="插入图片"]').attributes('disabled')).toBeDefined();
    expect(wrapper.text()).toContain('初始内容');
  });

  it('enables image insertion when an uploader is provided', async () => {
    const wrapper = mount(EvolynRichTextEditor, {
      props: { uploadImage: vi.fn().mockResolvedValue('https://cdn.lingyanyun.com/example.png') },
    });

    await vi.waitFor(() => expect(wrapper.find('.tiptap').exists()).toBe(true));
    expect(wrapper.find('[aria-label="插入图片"]').attributes('disabled')).toBeUndefined();
  });

  it('applies the requested toolbar size', async () => {
    const wrapper = mount(EvolynRichTextEditor, { props: { toolbarSize: 'small' } });

    await vi.waitFor(() => expect(wrapper.find('.tiptap').exists()).toBe(true));
    expect(wrapper.find('.evolyn-rich-text-toolbar').classes()).toContain(
      'evolyn-rich-text-toolbar--small',
    );
  });

  it('offers the supported font sizes', async () => {
    const wrapper = mount(EvolynRichTextEditor, { props: { modelValue: '<p>字号文本</p>' } });

    await vi.waitFor(() => expect(wrapper.find('.tiptap').exists()).toBe(true));
    const fontSize = wrapper.get<HTMLSelectElement>('[aria-label="字号"]');
    expect(fontSize.findAll('option').map((option) => option.text())).toEqual([
      '字号',
      '12',
      '14',
      '16',
      '18',
      '20',
      '22',
    ]);
  });

  it('opens a compact link editor that accepts every supported protocol', async () => {
    const wrapper = mount(EvolynRichTextEditor, { props: { modelValue: '<p>链接文本</p>' } });

    await vi.waitFor(() => expect(wrapper.find('.tiptap').exists()).toBe(true));
    await wrapper.get('[aria-label="编辑链接"]').trigger('click');

    await vi.waitFor(() =>
      expect(wrapper.find('.evolyn-rich-text-editor__link-input').exists()).toBe(true),
    );
    expect(wrapper.get('.evolyn-rich-text-editor__link-input').attributes('type')).toBe('text');
    expect(wrapper.find('.evolyn-rich-text-editor__link-actions').exists()).toBe(true);
  });
});
