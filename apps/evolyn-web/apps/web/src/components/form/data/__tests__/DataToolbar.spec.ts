import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import DataToolbar from '../../../../../../../packages/data-workspace/src/components/DataToolbar.vue';

const stubs = {
  ElDropdown: { name: 'ElDropdown', template: '<div class="dropdown"><slot/><slot name="dropdown"/></div>' },
  ElDropdownMenu: { template: '<div><slot/></div>' },
  ElDropdownItem: { props: ['command', 'disabled'], template: '<button :disabled="disabled"><slot/></button>' },
  ElInput: { template: '<input>' },
};

describe('dataToolbar 层级动作', () => {
  it('兼容扁平动作并派发下拉子动作', async () => {
    const wrapper = mount(DataToolbar, {
      props: {
        search: '',
        actions: [
          { key: 'create', label: '新增', tone: 'primary' },
          { key: 'more', label: '更多', children: [
            { key: 'download-labels', label: '下载二维码标签' },
            { key: 'delete', label: '批量删除', disabled: true },
          ] },
        ],
        'onUpdate:search': () => undefined,
      },
      global: { stubs },
    });

    await wrapper.findAll('.data-toolbar__action')[0].trigger('click');
    expect(wrapper.emitted('action')?.[0]).toEqual(['create']);
    wrapper.getComponent({ name: 'ElDropdown' }).vm.$emit('command', 'download-labels');
    expect(wrapper.emitted('action')?.[1]).toEqual(['download-labels']);
    expect(wrapper.findAll('.dropdown button')[2].attributes('disabled')).toBeDefined();
  });
});
