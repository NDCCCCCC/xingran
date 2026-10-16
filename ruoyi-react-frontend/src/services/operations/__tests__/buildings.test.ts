/**
 * Phase 88 Batch209 — services/operations/buildings 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: any) => ({
      code: 0,
      data: { url, payload: data },
    })),
  };
});

import * as api from "@/lib/api";
import { buildingApi } from "../buildings";

describe("services/operations/buildings", () => {
  it("list", async () => {
    const r: any = await buildingApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/buildings/list");
  });

  it("create", async () => {
    const r: any = await buildingApi.create({ name: "B1" } as any);
    expect(r.data.url).toBe("/ops/buildings");
  });

  it("update", async () => {
    const r: any = await buildingApi.update("b1", { name: "B2" });
    expect(r.data.url).toContain("/b1/update");
  });

  it("delete", async () => {
    const r: any = await buildingApi.delete("b1");
    expect(r.data.url).toContain("/b1/delete");
  });

  it("batchDelete", async () => {
    const r: any = await buildingApi.batchDelete(["a", "b"]);
    expect(r.data.url).toBe("/ops/buildings/batch");
    expect(r.data.payload.action).toBe("delete");
  });

  it("export", async () => {
    const r: any = await buildingApi.export({ current: 1, pageSize: 100 });
    expect(r.data.url).toBe("/ops/buildings/export");
  });

  it("import 传 FormData", async () => {
    const file = new File(["x"], "buildings.xlsx");
    const r: any = await buildingApi.import(file);
    expect(r.data.url).toBe("/ops/buildings/import");
  });

  it("downloadTemplate", async () => {
    const r: any = await buildingApi.downloadTemplate();
    expect(r.data.url).toBe("/ops/buildings/template");
  });

  it("post spy", async () => {
    await buildingApi.list({ current: 1, pageSize: 10 });
    expect(api.post).toHaveBeenCalled();
  });
});
