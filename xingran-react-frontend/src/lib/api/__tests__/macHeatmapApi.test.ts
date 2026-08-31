/**
 * Phase 88 Batch244 — lib/api/macHeatmapApi 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    post: vi.fn(async (url: string, data?: any) => ({
      code: 0,
      data: {
        url,
        payload: data,
        cells: [],
        topN: 100,
        start: "s",
        end: "e",
        total: 0,
        snapshot: "v1",
      },
    })),
  };
});

import * as api from "@/lib/api";
import { queryMACHeatmap } from "../macHeatmapApi";

describe("lib/api/macHeatmapApi", () => {
  it("queryMACHeatmap 调用", async () => {
    const r = await queryMACHeatmap({ startTime: "2026-01-01", endTime: "2026-01-07" });
    expect(r.cells).toEqual([]);
    expect(r.topN).toBe(100);
  });

  it("post 被调用", async () => {
    vi.mocked(api.post).mockClear();
    await queryMACHeatmap({});
    expect(api.post).toHaveBeenCalled();
    expect(api.post).toHaveBeenCalledWith("/network/history/heatmap", {});
  });

  it("HeatmapQuery 形状", () => {
    const q: import("../macHeatmapApi").HeatmapQuery = {
      startTime: "2026-01-01",
      endTime: "2026-01-07",
      topN: 50,
    };
    expect(q.topN).toBe(50);
  });
});
