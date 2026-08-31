/**
 * Phase 88 Batch212 — services/operations/floors 测试
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
import { floorApi } from "../floors";

describe("services/operations/floors", () => {
  it("list", async () => {
    const r: any = await floorApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/floors/list");
  });

  it("getTree", async () => {
    const r: any = await floorApi.getTree();
    expect(r.data.url).toBe("/ops/floors/tree");
  });

  it("create", async () => {
    const r: any = await floorApi.create({ floorName: "F1" } as any);
    expect(r.data.url).toBe("/ops/floors");
  });

  it("update", async () => {
    const r: any = await floorApi.update("f1", { floorName: "F2" });
    expect(r.data.url).toContain("/f1/update");
  });

  it("delete", async () => {
    const r: any = await floorApi.delete("f1");
    expect(r.data.url).toContain("/f1/delete");
  });

  it("batchDelete", async () => {
    const r: any = await floorApi.batchDelete(["a", "b"]);
    expect(r.data.url).toBe("/ops/floors/batch");
    expect(r.data.payload.action).toBe("delete");
  });

  it("export", async () => {
    const r: any = await floorApi.export({ current: 1, pageSize: 100 });
    expect(r.data.url).toBe("/ops/floors/export");
  });

  it("import 传 FormData", async () => {
    const file = new File(["x"], "floors.xlsx");
    const r: any = await floorApi.import(file);
    expect(r.data.url).toBe("/ops/floors/import");
  });

  it("downloadTemplate", async () => {
    const r: any = await floorApi.downloadTemplate();
    expect(r.data.url).toBe("/ops/floors/template");
  });

  it("post spy", async () => {
    await floorApi.list({ current: 1, pageSize: 10 });
    expect(api.post).toHaveBeenCalled();
  });
});
