/**
 * Phase 114 MAP3D-05 — HubeiMap / HubeiMapGL 地图事件监听 cleanup spy 测试
 *
 * 行为锁定（Plan 02 f748fa7/a6b3fdd 落地的监听器生命周期契约）：
 *  - HubeiMapGL init effect 挂载 zoomend + tiltend 各一次；unmount 后以同一函数引用
 *    removeEventListener（内联箭头形态下引用不一致 → remove 静默无效 → 本测试必红）；
 *  - HubeiMap init effect 挂载 zoomend 一次；unmount 后同引用移除；
 *  - AK 缺失守卫路径：不发起脚本加载、不挂任何监听、unmount 不抛错。
 *
 * mock 风格参照 BaiduMapScript.test.ts（操纵 window.BMap/BMapGL 全局 + vi.fn）；
 * AK 常量在模块加载时求值（HubeiMap.tsx 本地 const / constants.ts 导出 const），
 * 因此 vi.stubEnv 必须先于动态 import 执行（参照 index.render.test.tsx 的
 * beforeAll 动态导入模式）。
 */
import { describe, it, expect, vi, beforeAll, beforeEach, afterAll } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { waitFor } from "@testing-library/react";
import type { BuildingItem } from "../types";

// AK 常量在模块加载时求值——stub 必须先于 beforeAll 的动态 import
vi.stubEnv("VITE_BAIDU_MAP_AK", "test-ak");

// 脚本加载 mock：立即 resolve，不注入真实 <script>
vi.mock("../components/BaiduMapScript", () => ({
  loadBaiduMapScript: vi.fn(() => Promise.resolve()),
  loadBaiduMapGLScript: vi.fn(() => Promise.resolve()),
}));

/** fake 地图实例：事件挂载/移除为 spy，全部方法 vi.fn 兜底 */
const makeFakeMap = () => ({
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  getZoom: vi.fn(() => 10),
  getTilt: vi.fn(() => 0),
  centerAndZoom: vi.fn(),
  setMinZoom: vi.fn(),
  setMaxZoom: vi.fn(),
  enableScrollWheelZoom: vi.fn(),
  enableDoubleClickZoom: vi.fn(),
  addControl: vi.fn(),
  setMapStyleV2: vi.fn(),
  addOverlay: vi.fn(),
  removeOverlay: vi.fn(),
  openInfoWindow: vi.fn(),
  pointToOverlayPixel: vi.fn(() => ({ x: 0, y: 0 })),
  pixelToPoint: vi.fn(() => ({ lng: 114.3, lat: 30.6 })),
});

/**
 * fake BMapGL namespace：Boundary 空结果 → 组件走 addFallbackMask 分支，fake 全兜住。
 * 注意两点（缺一必炸）：
 *  - 组件以 `new BMapGL.Xxx(...)` 构造——实现必须是 function/class 形态（箭头函数
 *    不可 new，new 时抛 "is not a constructor"）；
 *  - 构造器实现必须写成独立 function 表达式再以标识符传入 vi.fn——内联
 *    `vi.fn(function () {...})` 会被 lint-staged 的 eslint --fix
 *    (prefer-arrow-callback) 静默改写回箭头形态，导致全量套件下必红。
 */
const makeFakeGLNamespace = (fakeMap: ReturnType<typeof makeFakeMap>) => {
  const MapCtor = function () {
    return fakeMap;
  };
  const PointCtor = function (lng: number, lat: number) {
    return { lng, lat };
  };
  const PixelCtor = function (x: number, y: number) {
    return { x, y };
  };
  const SizeCtor = function (w: number, h: number) {
    return { width: w, height: h };
  };
  const IconCtor = function () {
    return {};
  };
  const MarkerCtor = function () {
    return { addEventListener: vi.fn() };
  };
  const InfoWindowCtor = function () {
    return { setContent: vi.fn(), open: vi.fn(), close: vi.fn() };
  };
  const PolygonCtor = function () {
    return {};
  };
  const BoundaryCtor = function () {
    return {
      get: (_query: string, cb: (rs: { boundaries: string[] }) => void) => cb({ boundaries: [] }),
    };
  };
  const ZoomControlCtor = function () {
    return {};
  };
  const ScaleControlCtor = function () {
    return {};
  };
  const NavigationControlCtor = function () {
    return {};
  };
  return {
    Map: vi.fn(MapCtor),
    Point: vi.fn(PointCtor),
    Pixel: vi.fn(PixelCtor),
    Size: vi.fn(SizeCtor),
    Icon: vi.fn(IconCtor),
    Marker: vi.fn(MarkerCtor),
    InfoWindow: vi.fn(InfoWindowCtor),
    Polygon: vi.fn(PolygonCtor),
    Boundary: vi.fn(BoundaryCtor),
    ZoomControl: vi.fn(ZoomControlCtor),
    ScaleControl: vi.fn(ScaleControlCtor),
    NavigationControl: vi.fn(NavigationControlCtor),
  };
};

/** fake BMap namespace（与 GL 同构，无 ZoomControl） */
const makeFakeBMapNamespace = (fakeMap: ReturnType<typeof makeFakeMap>) => {
  const { ZoomControl: _zoomControl, ...rest } = makeFakeGLNamespace(fakeMap);
  return { ...rest, ANIMATION_BOUNCE: 1, ANIMATION_DROP: 2 };
};

let fakeMapGL: ReturnType<typeof makeFakeMap>;
let fakeMapBMap: ReturnType<typeof makeFakeMap>;

let HubeiMapGL: React.ComponentType<{ buildings: BuildingItem[] }>;
let HubeiMap: React.ComponentType<{ buildings: BuildingItem[] }>;

beforeAll(async () => {
  ({ default: HubeiMapGL } = await import("../components/HubeiMapGL"));
  ({ default: HubeiMap } = await import("../components/HubeiMap"));
});

beforeEach(() => {
  fakeMapGL = makeFakeMap();
  (window as any).BMapGL = makeFakeGLNamespace(fakeMapGL);
  fakeMapBMap = makeFakeMap();
  (window as any).BMap = makeFakeBMapNamespace(fakeMapBMap);
});

afterAll(() => {
  vi.unstubAllEnvs();
});

describe("HubeiMapGL 事件监听生命周期（MAP3D-05）", () => {
  it("渲染 → addEventListener 收到 zoomend/tiltend 各一次 → unmount 后以同一 handler 引用移除", async () => {
    const { unmount } = renderWithProviders(<HubeiMapGL buildings={[]} />);

    await waitFor(() => {
      expect(fakeMapGL.addEventListener).toHaveBeenCalledWith("zoomend", expect.any(Function));
      expect(fakeMapGL.addEventListener).toHaveBeenCalledWith("tiltend", expect.any(Function));
    });
    expect(fakeMapGL.addEventListener).toHaveBeenCalledTimes(2);

    const zoomCall = fakeMapGL.addEventListener.mock.calls.find((c) => c[0] === "zoomend");
    const tiltCall = fakeMapGL.addEventListener.mock.calls.find((c) => c[0] === "tiltend");
    expect(zoomCall).toBeDefined();
    expect(tiltCall).toBeDefined();

    unmount();

    // 引用一致性核心断言：removeEventListener 的第二参数与 addEventListener 捕获的
    // handler 引用全等（内联箭头形态下必红——remove 静默无效即本测试的回归信号）
    const removedZoom = fakeMapGL.removeEventListener.mock.calls.find((c) => c[0] === "zoomend");
    const removedTilt = fakeMapGL.removeEventListener.mock.calls.find((c) => c[0] === "tiltend");
    expect(removedZoom).toBeDefined();
    expect(removedZoom![1]).toBe(zoomCall![1]);
    expect(removedTilt).toBeDefined();
    expect(removedTilt![1]).toBe(tiltCall![1]);
  });
});

describe("HubeiMap 事件监听生命周期（MAP3D-05）", () => {
  it("渲染 → addEventListener 收到 zoomend 一次 → unmount 后以同一 handler 引用移除", async () => {
    const { unmount } = renderWithProviders(<HubeiMap buildings={[]} />);

    await waitFor(() => {
      expect(fakeMapBMap.addEventListener).toHaveBeenCalledWith("zoomend", expect.any(Function));
    });
    expect(fakeMapBMap.addEventListener).toHaveBeenCalledTimes(1);

    const zoomCall = fakeMapBMap.addEventListener.mock.calls.find((c) => c[0] === "zoomend");
    expect(zoomCall).toBeDefined();

    unmount();

    const removedZoom = fakeMapBMap.removeEventListener.mock.calls.find(
      (c) => c[0] === "zoomend" && c[1] === zoomCall![1]
    );
    expect(removedZoom).toBeDefined();
  });
});

describe("AK 缺失守卫路径（MAP3D-05 冒烟）", () => {
  it("VITE_BAIDU_MAP_AK 为空 → 不发起脚本加载、不挂任何监听、unmount 不抛错", async () => {
    // AK 常量已在两组件模块加载时求值——重置模块注册表后以空 AK 重新导入
    vi.stubEnv("VITE_BAIDU_MAP_AK", "");
    vi.resetModules();
    try {
      const { default: HubeiMapGLNoAK } = await import("../components/HubeiMapGL");
      const baiduScriptModule = await import("../components/BaiduMapScript");
      // vi.resetModules() 不重置 mock 注册表——被 mock 的 BaiduMapScript 模块
      // 跨 reset 存活且携带前序用例的调用记录，断言前先清空
      vi.mocked(baiduScriptModule.loadBaiduMapGLScript).mockClear();

      const { unmount } = renderWithProviders(<HubeiMapGLNoAK buildings={[]} />);

      // 让可能存在的异步 init 续体充分暴露（守卫生效时无任何异步加载）
      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(baiduScriptModule.loadBaiduMapGLScript).not.toHaveBeenCalled();
      expect(fakeMapGL.addEventListener).not.toHaveBeenCalled();
      expect(() => unmount()).not.toThrow();
    } finally {
      vi.stubEnv("VITE_BAIDU_MAP_AK", "test-ak");
    }
  });
});
