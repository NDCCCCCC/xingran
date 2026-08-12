/**
 * Vitest 测试 setup — jsdom + 必要 polyfill
 */
import "@testing-library/jest-dom/vitest";

// jsdom 不提供 matchMedia,部分 antd 组件需要
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});
