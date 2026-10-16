/**
 * Phase 88 Batch213 — services/operations/workstations 测试
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
import { workstationApi } from "../workstations";

describe("services/operations/workstations", () => {
  it("list", async () => {
    const r: any = await workstationApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/workstations/list");
  });

  it("create", async () => {
    const r: any = await workstationApi.create({ name: "WS1" } as any);
    expect(r.data.url).toBe("/ops/workstations");
  });

  it("update", async () => {
    const r: any = await workstationApi.update("w1", { name: "WS2" });
    expect(r.data.url).toContain("/w1/update");
  });

  it("delete", async () => {
    const r: any = await workstationApi.delete("w1");
    expect(r.data.url).toContain("/w1/delete");
  });

  it("batchDelete", async () => {
    const r: any = await workstationApi.batchDelete(["a", "b"]);
    expect(r.data.url).toBe("/ops/workstations/batch");
    expect(r.data.payload.action).toBe("delete");
  });

  it("export", async () => {
    const r: any = await workstationApi.export({ current: 1, pageSize: 100 });
    expect(r.data.url).toBe("/ops/workstations/export");
  });

  it("import FormData", async () => {
    const file = new File(["x"], "ws.xlsx");
    const r: any = await workstationApi.import(file);
    expect(r.data.url).toBe("/ops/workstations/import");
  });

  it("downloadTemplate", async () => {
    const r: any = await workstationApi.downloadTemplate();
    expect(r.data.url).toBe("/ops/workstations/template");
  });

  it("post spy", async () => {
    await workstationApi.list({ current: 1, pageSize: 10 });
    expect(api.post).toHaveBeenCalled();
  });
});
