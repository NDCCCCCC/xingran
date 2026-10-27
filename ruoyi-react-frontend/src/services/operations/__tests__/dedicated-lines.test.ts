/**
 * Phase 88 Batch214 — services/operations/dedicated-lines 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: unknown) => ({
      code: 0,
      data: { url, payload: data },
    })),
  };
});

import * as api from "@/lib/api";
import { dedicatedLineApi } from "../dedicated-lines";

describe("services/operations/dedicated-lines", () => {
  it("list", async () => {
    const r: unknown = await dedicatedLineApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/dedicated-lines/list");
  });

  it("create", async () => {
    const r: unknown = await dedicatedLineApi.create({ name: "L1" } as unknown);
    expect(r.data.url).toBe("/ops/dedicated-lines");
  });

  it("update", async () => {
    const r: unknown = await dedicatedLineApi.update("d1", { name: "L2" });
    expect(r.data.url).toContain("/d1/update");
  });

  it("delete", async () => {
    const r: unknown = await dedicatedLineApi.delete("d1");
    expect(r.data.url).toContain("/d1/delete");
  });

  it("batchDelete", async () => {
    const r: unknown = await dedicatedLineApi.batchDelete(["a"]);
    expect(r.data.url).toBe("/ops/dedicated-lines/batch");
  });

  it("export + import + downloadTemplate", async () => {
    const e: unknown = await dedicatedLineApi.export({ current: 1, pageSize: 10 });
    expect(e.data.url).toBe("/ops/dedicated-lines/export");

    const i: unknown = await dedicatedLineApi.import(new File(["x"], "dl.xlsx"));
    expect(i.data.url).toBe("/ops/dedicated-lines/import");

    const t: unknown = await dedicatedLineApi.downloadTemplate();
    expect(t.data.url).toBe("/ops/dedicated-lines/template");
  });

  it("post spy", async () => {
    await dedicatedLineApi.list({ current: 1, pageSize: 10 });
    expect(api.post).toHaveBeenCalled();
  });
});
