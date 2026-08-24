/**
 * macHeatmapApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:POST /network/history/heatmap 参数透传 + result.data 解包。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
vi.mock("../api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));

import { queryMACHeatmap } from "../macHeatmapApi";

describe("macHeatmapApi", () => {
  beforeEach(() => {
    mockPost.mockReset();
  });

  it("queryMACHeatmap 调用 /network/history/heatmap 并解包 data", async () => {
    const heatmapResult = {
      cells: [{ deviceId: "d1", interfaceName: "GE0/0/1", date: "2026-08-24", changeCount: 3 }],
      topN: 100,
      start: "2026-08-18",
      end: "2026-08-24",
      total: 1,
      snapshot: "mv_mac_port_daily_count",
    };
    mockPost.mockResolvedValueOnce({ code: 0, data: heatmapResult });

    const params = { startTime: "2026-08-18", endTime: "2026-08-24", topN: 100 };
    const result = await queryMACHeatmap(params);

    expect(mockPost).toHaveBeenCalledWith("/network/history/heatmap", params);
    expect(result).toBe(heatmapResult);
  });

  it("空参数调用透传 undefined 字段", async () => {
    mockPost.mockResolvedValueOnce({
      code: 0,
      data: { cells: [], topN: 100, start: "", end: "", total: 0, snapshot: "" },
    });

    await queryMACHeatmap({});

    expect(mockPost).toHaveBeenCalledWith("/network/history/heatmap", {});
  });
});
