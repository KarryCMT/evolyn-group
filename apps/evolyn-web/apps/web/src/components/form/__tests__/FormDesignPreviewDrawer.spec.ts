import type { FormRuntimeAdapter } from '@evolyn.do/form/runtime-web';
import type { FormSchemaDocument } from '@evolyn.do/form/schema';
import { mount } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { describe, expect, it } from 'vitest';
import { defineComponent, h, nextTick, reactive } from 'vue';
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
  name: 'FormWebRuntimeSurface',
  props: {
    layout: String,
    actions: Array,
  },
  template: '<div class="surface-stub" :data-layout="layout" />',
});

const schema: FormSchemaDocument = {
  content: {
    type: 'form',
    layout: 'normal',
    items: [],
    layout_fields: [],
    field_layout: [],
    fieldShowRules: [],
    submitRule: 2,
    widget_submit_rules: {},
  },
};
const adapter: FormRuntimeAdapter = { submit: async () => ({ accepted: true }) };

describe('formDesignPreviewDrawer', () => {
  it('设备切换只更新 Surface 布局，不重建填写会话', async () => {
    const wrapper = mount(FormDesignPreviewDrawer, {
      props: { modelValue: true, schema, formId: 'form_a', adapter },
      global: {
        // 组件 setup 读取 auth store（当前成员注入），挂载需提供 Pinia。
        plugins: [createPinia()],
        stubs: { ElDrawer: DrawerStub, FormWebRuntimeSurface: SurfaceStub },
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

  it('设计草稿原地更新时重建运行时会话，以同步最新字段属性', async () => {
    const liveSchema = reactive({
      content: {
        type: 'form' as const,
        layout: 'normal' as const,
        items: [],
        layout_fields: [],
        field_layout: [],
        fieldShowRules: [],
        submitRule: 2,
        widget_submit_rules: {},
      },
    }) as FormSchemaDocument;
    const wrapper = mount(FormDesignPreviewDrawer, {
      props: { modelValue: true, schema: liveSchema, formId: 'form_a', adapter },
      global: {
        // 组件 setup 读取 auth store（当前成员注入），挂载需提供 Pinia。
        plugins: [createPinia()],
        stubs: { ElDrawer: DrawerStub, FormWebRuntimeSurface: SurfaceStub },
      },
    });
    const initialUid = wrapper.findComponent(SurfaceStub).vm.$.uid;

    liveSchema.content.layout = 'grid-2';
    await nextTick();

    expect(wrapper.findComponent(SurfaceStub).vm.$.uid).not.toBe(initialUid);
  });

  it('标准 Drawer header 关闭能力回传 v-model', async () => {
    const wrapper = mount(FormDesignPreviewDrawer, {
      props: { modelValue: true, schema, formId: 'form_a', adapter },
      global: {
        // 组件 setup 读取 auth store（当前成员注入），挂载需提供 Pinia。
        plugins: [createPinia()],
        stubs: { ElDrawer: DrawerStub, FormWebRuntimeSurface: SurfaceStub },
      },
    });

    await wrapper.find('.form-design-preview__close').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false]);
  });
});
