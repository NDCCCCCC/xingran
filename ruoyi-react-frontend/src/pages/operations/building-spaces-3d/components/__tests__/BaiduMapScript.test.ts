/**
 * Phase 88 Batch98 — BaiduMapScript 测试(36 stmts, 0% → 高)
 */
import { describe, it, expect, beforeEach } from "vitest";
import {
  loadBaiduMapGLScript,
  loadBaiduMapScript,
  isBaiduMapLoaded,
  isBaiduMapGLLoaded,
} from "../BaiduMapScript";

describe("BaiduMapScript", () => {
  beforeEach(() => {
    // 重置 globalThis.BMap / BMapGL
    delete (window as unknown).BMapGL;
    delete (window as unknown).BMap;
  });

  it("loadBaiduMapGLScript: 已加载 BMapGL → 立即 resolve", async () => {
    (window as unknown).BMapGL = {};
    await expect(loadBaiduMapGLScript("test-ak")).resolves.toBeUndefined();
  });

  it("loadBaiduMapScript: 已加载 BMap → 立即 resolve", async () => {
    (window as unknown).BMap = {};
    await expect(loadBaiduMapScript("test-ak")).resolves.toBeUndefined();
  });

  it("isBaiduMapLoaded: BMap 存在 → true", () => {
    (window as unknown).BMap = {};
    expect(isBaiduMapLoaded()).toBe(true);
  });

  it("isBaiduMapLoaded: BMap 不存在 → false", () => {
    expect(isBaiduMapLoaded()).toBe(false);
  });

  it("isBaiduMapGLLoaded: BMapGL 存在 → true", () => {
    (window as unknown).BMapGL = {};
    expect(isBaiduMapGLLoaded()).toBe(true);
  });

  it("isBaiduMapGLLoaded: BMapGL 不存在 → false", () => {
    expect(isBaiduMapGLLoaded()).toBe(false);
  });
});
