import { describe, expect, it } from "vitest";
import { HUBEI_CITIES, getAllCities, getCityInfo } from "./baidu-map";

describe("baidu-map 湖北城市数据", () => {
  it("HUBEI_CITIES 覆盖 17 个市州", () => {
    expect(Object.keys(HUBEI_CITIES)).toHaveLength(17);
  });

  it("每个城市含中文名与 [lng, lat] 中心坐标", () => {
    for (const info of Object.values(HUBEI_CITIES)) {
      expect(typeof info.name).toBe("string");
      expect(info.name).not.toBe("");
      expect(info.center).toHaveLength(2);
      expect(info.center[0]).toBeGreaterThan(100);
      expect(info.center[0]).toBeLessThan(120);
      expect(info.center[1]).toBeGreaterThan(24);
      expect(info.center[1]).toBeLessThan(35);
    }
  });

  it("getCityInfo 按代码查询", () => {
    expect(getCityInfo("wuhan")).toEqual({ name: "武汉市", center: [114.305393, 30.593099] });
    expect(getCityInfo("nope")).toBeUndefined();
  });

  it("getAllCities 展开为 { code, name, center } 列表", () => {
    const cities = getAllCities();
    expect(cities).toHaveLength(17);
    expect(cities[0]).toMatchObject({
      code: "wuhan",
      name: "武汉市",
      center: [114.305393, 30.593099],
    });
    expect(cities.some((c) => c.code === "shennongjia")).toBe(true);
  });
});
