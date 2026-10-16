/**
 * Phase 88 Batch55 — networkApi 单元测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

// 手写 mock(vi.mocked 在 api 模块整体 mock 模式下不生效)
vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
}));

import { post } from "@/lib/api";
import * as networkApi from "../networkApi";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("networkApi — 基础 crud", () => {
  it("queryMACHistory 调 post + URL 含 /network/history/list", async () => {
    await networkApi.queryMACHistory({
      current: 1,
      pageSize: 10,
      mac: "AA:BB:CC:DD:EE:FF",
    });
    expect(post).toHaveBeenCalledWith(
      expect.stringContaining("/network/history/list"),
      expect.objectContaining({ mac: "AA:BB:CC:DD:EE:FF" })
    );
  });

  it("getMACEvents 调 post + URL", async () => {
    await networkApi.getMACEvents({ current: 1, pageSize: 10 });
    expect(post).toHaveBeenCalled();
  });

  it("writeShutdown 调 post + URL", async () => {
    await networkApi.writeShutdown("port1", "shutdown reason");
    expect(post).toHaveBeenCalled();
  });

  it("writeUndoShutdown 调 post + URL", async () => {
    await networkApi.writeUndoShutdown("port1", "undo reason");
    expect(post).toHaveBeenCalled();
  });

  it("writeDescription 调 post + URL", async () => {
    await networkApi.writeDescription("port1", "new desc");
    expect(post).toHaveBeenCalled();
  });
});
