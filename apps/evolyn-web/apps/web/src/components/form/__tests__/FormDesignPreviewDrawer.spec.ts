import type { FormRuntimeAdapter } from '@evolyn.do/form/runtime';
import type { FormSchemaDocument } from '@evolyn.do/form/schema';
import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import FormDesignPreviewDrawer from '../FormDesignPreviewDrawer.vue';

const DrawerStub = defineComponent({
  name: 'ElDrawer',
  props: { modelValue: Boolean },
  emits: ['update:modelValue'],
  setup(_props, { emit, slots }) {
    return () =>
      h('div', { class: 'drawer-stub' }, [
        slots.header?.({ close: () => emit('update:modelValue', false) }),
        slots.default?.(),
      ]);
  },
});

const SurfaceStub = defineComponent({
  name: 'FormRuntimeSurface',
  props: {
    layout: String,
    actions: Array,
  },
  template: '<div class="surface-stub" :data-layout="layout" />',
});

const schema: FormSchemaDocument = {
  content: { type: 'form', layout: 'normal', items: [], layout_fields: [], field_layout: [] },
};
const adapter: FormRuntimeAdapter = { submit: async () => ({ accepted: true }) };

describe('formDesignPreviewDrawer', () => {
  it('设备切换只更新 Surface 布局，不重建填写会话', async () => {
    const wrapper = mount(FormDesignPreviewDrawer, {
      props: { modelValue: true, schema, formId: 'form_a', adapter },
      global: {
        stubs: { ElDrawer: DrawerStub, FormRuntimeSurface: SurfaceStub },
      },
    });
    const surface = wrapper.findComponent(SurfaceStub);
    const initialUid = surface.vm.$.uid;
    expect(surface.props('layout')).toBe('desktop');
    expect(surface.props('actions')).toHaveLength(2);

    await wrapper.findAll('.form-design-preview__viewport-button')[1].trigger('click');

    expect(wrapper.findComponent(SurfaceStub).vm.$.uid).toBe(initialUid);
    expect(wrapper.findComponent(SurfaceStub).props('layout')).toBe('mobile');
  });

  it('标准 Drawer header 关闭能力回传 v-model', async () => {
    const wrapper = mount(FormDesignPreviewDrawer, {
      props: { modelValue: true, schema, formId: 'form_a', adapter },
      global: {
        stubs: { ElDrawer: DrawerStub, FormRuntimeSurface: SurfaceStub },
      },
    });

    await wrapper.find('.form-design-preview__close').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false]);
  });
});
