/**
 * 回归：TabBar hover 预取把「路由 URL」当「组件路径」传给 ComponentLoader.load()。
 *
 * 复现的用户现象（悬停顶部多标签栏 Tab）：
 *   [ComponentLoader] Path not in whitelist          (warn)
 *   [ComponentLoader] Invalid component path: network/discoveries.tsx (error)
 *   Uncaught (in promise) Error: Invalid component path: ...
 *
 * 根因：preloadComponents() 用 Promise.all 聚合且拒绝向外抛，调用方 TabBar.tsx
 * 没有 .catch → 变成未捕获的 promise rejection；同时 load() 对非法路径打 error 日志。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { componentLoader, preloadComponents } from "../componentLoader";

describe("预取：路由 URL 不是组件路径", () => {
  let errorSpy: ReturnType<typeof vi.spyOn>;
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    componentLoader.clearCache();
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
    componentLoader.clearCache();
  });

  it("传入路由 URL 时不应 reject（否则调用方得到未捕获的 promise rejection）", async () => {
    await expect(preloadComponents(["/network/discoveries"])).resolves.toBeUndefined();
  });

  it("传入路由 URL 时不应污染控制台", async () => {
    await preloadComponents([
      "/network/discoveries",
      "/system/list",
      "/system/notice",
      "/assets/dashboard",
    ]);

    expect(errorSpy).not.toHaveBeenCalled();
    expect(warnSpy).not.toHaveBeenCalled();
  });

  it("load() 拒绝非法组件路径时只抛错、不打 console.error", async () => {
    await expect(componentLoader.load("/network/discoveries")).rejects.toThrow(
      /Invalid component path/
    );

    expect(errorSpy).not.toHaveBeenCalled();
  });
});
