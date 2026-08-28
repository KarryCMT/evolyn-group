import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import Button from '../Button.vue';
import type { ButtonProps } from '../Button.types';

describe('Button', () => {
  it('renders correctly', () => {
    const wrapper = mount(Button, {
      slots: {
        default: 'Test Button',
      },
    });

    expect(wrapper.text()).toBe('Test Button');
    expect(wrapper.classes()).toContain('evolyn-button');
  });

  it('applies correct type class', () => {
    const wrapper = mount(Button, {
      props: {
        type: 'primary',
      } as ButtonProps,
    });

    expect(wrapper.classes()).toContain('evolyn-button--primary');
  });

  it('applies correct size class', () => {
    const wrapper = mount(Button, {
      props: {
        size: 'large',
      } as ButtonProps,
    });

    expect(wrapper.classes()).toContain('evolyn-button--large');
  });

  it('applies disabled state correctly', () => {
    const wrapper = mount(Button, {
      props: {
        disabled: true,
      } as ButtonProps,
    });

    expect(wrapper.classes()).toContain('is-disabled');
    expect(wrapper.attributes('disabled')).toBeDefined();
  });

  it('applies round style correctly', () => {
    const wrapper = mount(Button, {
      props: {
        round: true,
      } as ButtonProps,
    });

    expect(wrapper.classes()).toContain('is-round');
  });

  it('uses submit as the native form type when requested', () => {
    const wrapper = mount(Button, { props: { nativeType: 'submit' } });

    expect(wrapper.attributes('type')).toBe('submit');
  });

  it('renders a loading indicator and prevents clicks while loading', async () => {
    const wrapper = mount(Button, { props: { loading: true } });

    expect(wrapper.classes()).toContain('is-loading');
    expect(wrapper.find('.evolyn-button__icon svg').exists()).toBe(true);
    expect(wrapper.attributes('disabled')).toBeDefined();
    await wrapper.trigger('click');
    expect(wrapper.emitted('click')).toBeFalsy();
  });

  it('emits click event when clicked', async () => {
    const wrapper = mount(Button);

    await wrapper.trigger('click');

    expect(wrapper.emitted('click')).toBeTruthy();
    expect(wrapper.emitted('click')).toHaveLength(1);
  });

  it('does not emit click event when disabled', async () => {
    const wrapper = mount(Button, {
      props: {
        disabled: true,
      } as ButtonProps,
    });

    await wrapper.trigger('click');

    expect(wrapper.emitted('click')).toBeFalsy();
  });

  it('passes click event object correctly', async () => {
    const wrapper = mount(Button);

    await wrapper.trigger('click');

    const clickEvents = wrapper.emitted('click') as MouseEvent[][];
    // noUncheckedIndexedAccess 下首元素可能为 undefined，可选链保证类型安全且断言语义不变
    expect(clickEvents[0]?.[0]).toBeInstanceOf(MouseEvent);
  });
});
