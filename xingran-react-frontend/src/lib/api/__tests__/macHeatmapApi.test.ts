/**
 * Phase 88 Batch306 — lib/api/macHeatmapApi 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: any) => ({
      data: {
        cells: [{ deviceId: "d1", interfaceName: "eth0", changeCount: 5, date: "2026-08-01" }],
        topN: 100,
        start: "2026-08-01",
        end: "2026-08-07",
        total: 1,
        snapshot: "snap1",
        url,
        payload: data,
      },
    })),
  };
});

import { post } from "@/lib/api";
import { queryMACHeatmap } from "../macHeatmapApi";

describe("lib/api/macHeatmapApi", () => {
  it("queryMACHeatmap 调用 post /network/history/heatmap", async () => {
    await queryMACHeatmap({ topN: 50 });
    expect(post).toHaveBeenCalledWith("/network/history/heatmap", { topN: 50 });
  });

  it("返回 result.data", async () => {
    const r = await queryMACHeatmap({ topN: 10 });
    expect(r.cells.length).toBe(1);
    expect(r.cells[0].deviceId).toBe("d1");
    expect(r.topN).toBe(100);
    expect(r.total).toBe(1);
    expect(r.snapshot).toBe("snap1");
  });

  it("params 为空对象", async () => {
    await queryMACHeatmap({});
    expect(post).toHaveBeenCalledWith("/network/history/heatmap", {});
  });

  it("params 含 startTime/endTime", async () => {
    await queryMACHeatmap({
      startTime: "2026-08-01",
      endTime: "2026-08-07",
      topN: 100,
    });
    expect(post).toHaveBeenCalled();
    const lastCall = vi.mocked(post).mock.calls[vi.mocked(post).mock.calls.length - 1];
    expect(lastCall[1]).toMatchObject({
      topN: 100,
      startTime: "2026-08-01",
      endTime: "2026-08-07",
    });
  });
});
