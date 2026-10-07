import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, shallowRef } from 'vue';
import { createMemoryHistory, createRouter, RouterView } from 'vue-router';
import { useUnsavedChangesGuard } from '../useUnsavedChangesGuard';

describe('useUnsavedChangesGuard', () => {
  it('registers browser protection only while dirty and cancels route leave', async () => {
    const dirty = shallowRef(false);
    const confirmLeave = vi.fn().mockResolvedValue(false);
    const listeners = new Map<string, EventListener>();
    const target = {
      addEventListener: vi.fn((name: string, listener: EventListener) =>
        listeners.set(name, listener),
      ),
      removeEventListener: vi.fn((name: string) => listeners.delete(name)),
    };
    const Page = defineComponent({
      setup() {
        useUnsavedChangesGuard({ dirty, confirmLeave, target });
        return () => h('div', 'design');
      },
    });
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/design', component: Page },
        { path: '/other', component: { template: '<div>other</div>' } },
      ],
    });
    await router.push('/design');
    await router.isReady();
    mount(RouterView, { global: { plugins: [router] } });
    await flushPromises();
    expect(target.addEventListener).not.toHaveBeenCalled();

    dirty.value = true;
    await nextTick();
    expect(target.addEventListener).toHaveBeenCalledWith('beforeunload', expect.any(Function));
    const event = new Event('beforeunload', { cancelable: true });
    listeners.get('beforeunload')?.(event);
    expect(event.defaultPrevented).toBe(true);

    await router.push('/other');
    expect(router.currentRoute.value.path).toBe('/design');
    expect(confirmLeave).toHaveBeenCalledOnce();

    dirty.value = false;
    await nextTick();
    expect(target.removeEventListener).toHaveBeenCalledWith('beforeunload', expect.any(Function));
    await router.push('/other');
    expect(router.currentRoute.value.path).toBe('/other');
  });
});
