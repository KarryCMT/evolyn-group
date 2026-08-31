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
});
