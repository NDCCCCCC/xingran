/**
 * Phase 88 Batch208 — services/operations/info-points 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: unknown) => ({ data: { url, payload: data } })),
  };
});

import * as api from "@/lib/api";
import { infoPointApi } from "../info-points";

describe("services/operations/info-points", () => {
  it("list", async () => {
    const r: unknown = await infoPointApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/info-points/list");
  });

  it("create", async () => {
    const r: unknown = await infoPointApi.create({
      code: "ip-1",
      name: "Info Point 1",
    } as unknown);
    expect(r.data.url).toBe("/ops/info-points");
  });

  it("update", async () => {
    const r: unknown = await infoPointApi.update("ip-1", { name: "X" });
    expect(r.data.url).toContain("/ip-1/update");
  });

  it("delete", async () => {
    const r: unknown = await infoPointApi.delete("ip-1");
    expect(r.data.url).toContain("/ip-1/delete");
  });

  it("batchDelete", async () => {
    const r: unknown = await infoPointApi.batchDelete(["a", "b"]);
    expect(r.data.url).toBe("/ops/info-points/batch");
    expect(r.data.payload.action).toBe("delete");
  });

  it("post spy", async () => {
    await infoPointApi.list({ current: 1, pageSize: 10 });
    expect(api.post).toHaveBeenCalled();
  });
});
