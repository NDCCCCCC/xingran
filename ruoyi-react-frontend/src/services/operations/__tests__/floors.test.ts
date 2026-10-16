/**
 * Phase 88 Batch270 — services/operations/floors 测试
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

  it("post spy", async () => {
    await floorApi.list({ current: 1, pageSize: 10 });
    expect(api.post).toHaveBeenCalled();
  });
});
