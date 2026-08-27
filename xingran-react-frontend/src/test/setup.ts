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

// Phase 84 D-13 — ResizeObserver 集中 polyfill(对齐 BulkWriteDrawer.test.tsx
// L27-36 inline 形态):antd v6 Drawer / Modal / Select 等浮层组件挂载时依赖
// ResizeObserver,jsdom 缺失会在渲染期抛错。集中沉淀后新测试文件无需再 inline
// 重写;既有文件(如 BulkWriteDrawer)的 inline stub 由 wave 1 起顺手迁移。
// 注意:本文件只在 vitest setupFiles 中加载,production build 不经过此处(T-84-00-02);
// 按 D-13「按需沉淀」纪律,IntersectionObserver / PointerEvent / canvas getContext
// 等 polyfill 留待对应 wave 实际渲染失败时再补。
class ResizeObserverStub {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
if (typeof globalThis.ResizeObserver === "undefined") {
  (globalThis as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver =
    ResizeObserverStub;
}
