/**
 * Phase 88 Batch258 — pages/operations/building-spaces-3d/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { CityGroup, BuildingItem, MarkerPosition, MarkerData } from "../types";

describe("operations/building-spaces-3d/types", () => {
  it("CityGroup shape", () => {
    const c: CityGroup = {
      code: "BJS",
      name: "北京",
      center: [116.4, 39.9],
      buildings: [],
      buildingCount: 0,
    };
    expect(c.code).toBe("BJS");
    expect(c.center.length).toBe(2);
  });

  it("BuildingItem shape", () => {
    const b: BuildingItem = {
      id: "b1",
      name: "B1",
      code: "C1",
      cityCode: "BJS",
      cityName: "北京",
      address: "xxx",
      level: 1,
      status: 0,
    };
    expect(b.level).toBe(1);
  });

  it("BuildingItem level 2", () => {
    const b: BuildingItem = {
      id: "b2",
      name: "B2",
      code: "C2",
      cityCode: "BJS",
      cityName: "北京",
      address: "yyy",
      level: 2,
      status: 1,
    };
    expect(b.level).toBe(2);
  });

  it("MarkerPosition shape", () => {
    const p: MarkerPosition = { lng: 116.4, lat: 39.9 };
    expect(p.lng).toBe(116.4);
  });

  it("MarkerData shape", () => {
    const m: MarkerData = {
      id: "m1",
      type: "building",
      position: { lng: 116.4, lat: 39.9 },
      title: "B1",
      data: {
        id: "b1",
        name: "B1",
        code: "C1",
        cityCode: "BJS",
        cityName: "北京",
        address: "xxx",
        level: 1,
        status: 0,
      },
    };
    expect(m.type).toBe("building");
  });
});
