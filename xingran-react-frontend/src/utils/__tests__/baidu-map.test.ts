/**
 * Phase 88 Batch323 — utils/baidu-map 测试
 */
import { describe, it, expect } from "vitest";
import { HUBEI_CITIES, getCityInfo, getAllCities } from "../baidu-map";

describe("utils/baidu-map", () => {
  it("HUBEI_CITIES 含 17 个城市", () => {
    expect(Object.keys(HUBEI_CITIES).length).toBe(17);
  });

  it("HUBEI_CITIES.wuhan 含 name + center", () => {
    expect(HUBEI_CITIES.wuhan.name).toBe("武汉市");
    expect(HUBEI_CITIES.wuhan.center).toEqual([114.305393, 30.593099]);
  });

  it("getCityInfo 已知 code", () => {
    const info = getCityInfo("wuhan");
    expect(info?.name).toBe("武汉市");
  });

  it("getCityInfo 未知 code → undefined", () => {
    expect(getCityInfo("not-a-city")).toBeUndefined();
  });

  it("getAllCities 返回数组", () => {
    const all = getAllCities();
    expect(Array.isArray(all)).toBe(true);
    expect(all.length).toBe(17);
  });

  it("getAllCities 含 code 字段", () => {
    const all = getAllCities();
    expect(all[0]).toHaveProperty("code");
    expect(all[0]).toHaveProperty("name");
    expect(all[0]).toHaveProperty("center");
  });

  it("getAllCities 含 wuhan", () => {
    const all = getAllCities();
    const wuhan = all.find((c) => c.code === "wuhan");
    expect(wuhan).toBeDefined();
    expect(wuhan?.name).toBe("武汉市");
  });
});
