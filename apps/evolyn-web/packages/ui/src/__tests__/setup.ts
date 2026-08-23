import { vi } from 'vitest';

// Mock ResizeObserver。TanStack Virtual 会通过 new 调用该构造器。
class ResizeObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}

globalThis.ResizeObserver = ResizeObserverMock;

// happy-dom 不会根据样式计算布局尺寸，虚拟列表需要可见区域尺寸来生成初始行。
Object.defineProperties(HTMLElement.prototype, {
  offsetHeight: { configurable: true, get: () => 400 },
  offsetWidth: { configurable: true, get: () => 600 },
});

// Mock IntersectionObserver
globalThis.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}));

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // deprecated
    removeListener: vi.fn(), // deprecated
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});
