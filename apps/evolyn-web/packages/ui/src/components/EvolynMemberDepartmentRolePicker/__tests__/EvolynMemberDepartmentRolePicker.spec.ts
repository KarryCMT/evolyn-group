import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import { nextTick, ref } from 'vue';
import EvolynMemberDepartmentRolePicker from '../EvolynMemberDepartmentRolePicker.vue';
import type { EvolynMemberDepartmentRolePickerSelection } from '../EvolynMemberDepartmentRolePicker.types';

const departments = [
  {
    id: 'engineering',
    label: '研发部',
    children: [{ id: 'web', label: '前端组' }],
  },
];

afterEach(() => {
  document.body.innerHTML = '';
});

function getTeleportedElement<T extends Element>(selector: string) {
  const element = document.body.querySelector<T>(selector);
  if (!element) throw new Error(`未找到弹窗元素：${selector}`);
  return element;
}

async function selectDepartment() {
  const checkbox = getTeleportedElement<HTMLInputElement>('input[aria-label="选择研发部"]');
  checkbox.checked = true;
  checkbox.dispatchEvent(new Event('change', { bubbles: true }));
  await nextTick();
}

describe('EvolynMemberDepartmentRolePicker', () => {
  it('only commits the draft selection after confirmation', async () => {
    const model = ref<EvolynMemberDepartmentRolePickerSelection[]>([]);
    const open = ref(true);
    const wrapper = mount(EvolynMemberDepartmentRolePicker, {
      props: {
        'onUpdate:modelValue': (value: EvolynMemberDepartmentRolePickerSelection[]) =>
          (model.value = value),
        'onUpdate:open': (value: boolean) => (open.value = value),
        departments,
        modelValue: model.value,
        open: open.value,
      },
      attachTo: document.body,
    });

    await nextTick();
    await selectDepartment();

    expect(model.value).toEqual([]);
    expect(document.body.textContent).toContain('已选择 1');

    getTeleportedElement<HTMLButtonElement>(
      '.evolyn-member-department-role-picker__actions button:last-child',
    ).click();

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([
      [{ id: 'engineering', label: '研发部', type: 'department' }],
    ]);
    expect(wrapper.emitted('confirm')?.[0]).toEqual([
      [{ id: 'engineering', label: '研发部', type: 'department' }],
    ]);
  });

  it('filters the visible department tree by space-separated search keywords', async () => {
    mount(EvolynMemberDepartmentRolePicker, {
      props: {
        open: true,
        departments: [...departments, { id: 'sales', label: '销售部', keywords: ['crm'] }],
      },
      attachTo: document.body,
    });

    const search = getTeleportedElement<HTMLInputElement>('input[type="search"]');
    search.value = '销售 crm';
    search.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();

    expect(document.body.textContent).toContain('销售部');
    expect(document.body.textContent).not.toContain('研发部');
  });

  it('resets unconfirmed changes after cancellation', async () => {
    const wrapper = mount(EvolynMemberDepartmentRolePicker, {
      props: { open: true, departments },
      attachTo: document.body,
    });

    await selectDepartment();
    getTeleportedElement<HTMLButtonElement>(
      '.evolyn-member-department-role-picker__actions button:first-child',
    ).click();

    expect(wrapper.emitted('update:modelValue')).toBeUndefined();
    expect(wrapper.emitted('cancel')).toHaveLength(1);
    expect(wrapper.emitted('close')?.[0]).toEqual(['cancel']);
  });
});
