/**
 * Phase 53 W4 — networkApi port-write wrapper 行为单元测试 (UI-06)
 *
 * 锁定行为:
 * - 6 个 wrapper 各自调用 post() 时 URL 完全对齐 Phase 52 port_write_router.go kebab 路径
 * - request body shape 严格匹配后端契约 (portId/reason/description / deviceId+action+portIds+description?)
 * - wrapper 解包 BaseResponse envelope (result.data!) 直接返回业务数据
 * - wrapper 不 try/catch (透传 Promise.reject) — 不吞错误
 *
 * 通过 vi.mock 替换 ../api 模块的 post,断言 wrapper → post 的输入契约。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { PortResult, BatchResult } from "@/types/network";

// 捕获 post 调用的 mock — 必须以 networkApi.ts 的 import 解析路径为准。
// networkApi.ts 用 `import { post } from "../api"` (=> src/lib/api.ts),
// 我们从 src/lib/api/__tests__/ 视角也用 ../api => 同一个 src/lib/api.ts。
const mockPost = vi.fn();
vi.mock("../api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));

import {
  writeShutdown,
  writeUndoShutdown,
  writeDescription,
  writeDot1xEnable,
  writeDot1xDisable,
  batchWritePorts,
} from "../networkApi";

const PORT_RESULT_FIXTURE: PortResult = {
  portId: "port-1",
  action: "shutdown",
  status: "succeeded",
  noOp: false,
  currentState: "down",
  commandSent: "shutdown",
};

const BATCH_RESULT_FIXTURE: BatchResult = {
  succeeded: [PORT_RESULT_FIXTURE],
  failed: [],
  skipped: [],
};

describe("networkApi port-write wrappers (Phase 53 — UI-06)", () => {
  beforeEach(() => {
    mockPost.mockReset();
  });

  describe("writeShutdown", () => {
    it("calls POST /network/ports/write/shutdown with {portId, reason} and returns result.data", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: PORT_RESULT_FIXTURE });

      const result = await writeShutdown("port-1", "故障排查处理");

      expect(mockPost).toHaveBeenCalledTimes(1);
      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/shutdown", {
        portId: "port-1",
        reason: "故障排查处理",
      });
      expect(result).toEqual(PORT_RESULT_FIXTURE);
    });
  });

  describe("writeUndoShutdown", () => {
    it("calls POST /network/ports/write/undo-shutdown with {portId, reason}", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: PORT_RESULT_FIXTURE });

      const result = await writeUndoShutdown("port-1", "业务变更需要");

      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/undo-shutdown", {
        portId: "port-1",
        reason: "业务变更需要",
      });
      expect(result).toEqual(PORT_RESULT_FIXTURE);
    });
  });

  describe("writeDescription", () => {
    it("calls POST /network/ports/write/description with {portId, description, reason}", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: PORT_RESULT_FIXTURE });

      await writeDescription("port-1", "uplink-to-core", "安全合规要求");

      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/description", {
        portId: "port-1",
        description: "uplink-to-core",
        reason: "安全合规要求",
      });
    });

    it("forwards undefined reason when reason arg omitted (not null/empty)", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: PORT_RESULT_FIXTURE });

      await writeDescription("port-1", "new-desc");

      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/description", {
        portId: "port-1",
        description: "new-desc",
        reason: undefined,
      });
    });
  });

  describe("writeDot1xEnable", () => {
    it("calls POST /network/ports/write/dot1x-enable with {portId, reason}", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: PORT_RESULT_FIXTURE });

      await writeDot1xEnable("port-1", "临时测试验证");

      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/dot1x-enable", {
        portId: "port-1",
        reason: "临时测试验证",
      });
    });
  });

  describe("writeDot1xDisable", () => {
    it("calls POST /network/ports/write/dot1x-disable with {portId, reason}", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: PORT_RESULT_FIXTURE });

      await writeDot1xDisable("port-1", "故障排查处理");

      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/dot1x-disable", {
        portId: "port-1",
        reason: "故障排查处理",
      });
    });
  });

  describe("batchWritePorts", () => {
    it("calls POST /network/ports/write/batch with the full BatchWriteRequest and returns BatchResult", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: BATCH_RESULT_FIXTURE });

      const req = {
        deviceId: "dev-1",
        action: "shutdown" as const,
        portIds: ["port-1", "port-2"],
      };

      const result = await batchWritePorts(req);

      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/batch", req);
      expect(result).toEqual(BATCH_RESULT_FIXTURE);
    });

    it("forwards description field when provided (description action)", async () => {
      mockPost.mockResolvedValueOnce({ code: 0, data: BATCH_RESULT_FIXTURE });

      const req = {
        deviceId: "dev-1",
        action: "description" as const,
        portIds: ["port-1"],
        description: "new-port-desc",
      };

      await batchWritePorts(req);

      expect(mockPost).toHaveBeenCalledWith("/network/ports/write/batch", req);
    });
  });

  describe("URL contract — all 6 endpoints must be kebab-cased (Phase 52 router alignment)", () => {
    it("snapshot of every URL passed to post across all 6 wrappers", async () => {
      mockPost.mockResolvedValue({ code: 0, data: PORT_RESULT_FIXTURE });
      mockPost.mockResolvedValueOnce({ code: 0, data: BATCH_RESULT_FIXTURE });

      await writeShutdown("p", "r");
      await writeUndoShutdown("p", "r");
      await writeDescription("p", "d");
      await writeDot1xEnable("p", "r");
      await writeDot1xDisable("p", "r");
      await batchWritePorts({ deviceId: "d", action: "shutdown", portIds: ["p"] });

      const urls = mockPost.mock.calls.map((call) => call[0]);
      expect(urls).toEqual([
        "/network/ports/write/shutdown",
        "/network/ports/write/undo-shutdown",
        "/network/ports/write/description",
        "/network/ports/write/dot1x-enable",
        "/network/ports/write/dot1x-disable",
        "/network/ports/write/batch",
      ]);
    });
  });

  describe("envelope unwrapping & reject propagation (LANDMINE #5)", () => {
    it("returns result.data (not the full BaseResponse envelope)", async () => {
      mockPost.mockResolvedValueOnce({
        code: 0,
        data: PORT_RESULT_FIXTURE,
        message: "success",
      });

      const result = await writeShutdown("port-1", "reason");
      expect(result).toBe(PORT_RESULT_FIXTURE);
    });

    it("propagates Promise.reject from post() without swallowing (no try/catch)", async () => {
      const rejection = new Error("network down");
      mockPost.mockRejectedValueOnce(rejection);

      await expect(writeShutdown("port-1", "reason")).rejects.toBe(rejection);
    });
  });
});
