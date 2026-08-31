/**
 * Phase 88 Batch307 — lib/api/networkApi 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, params?: any) => ({
      data: {
        list: [{ id: "r1", macAddress: "00:11:22" }],
        total: 1,
        current: params?.current ?? 1,
        pageSize: params?.pageSize ?? 10,
        url,
        payload: params,
      },
    })),
  };
});

import { post } from "@/lib/api";
import {
  queryMACHistory,
  getMACEvents,
  writeShutdown,
  writeUndoShutdown,
  writeDescription,
  writeDot1xEnable,
  writeDot1xDisable,
  writeSetAccessVlan,
  writePortBinding,
  batchWritePorts,
  getPortMACBundle,
} from "../networkApi";

describe("lib/api/networkApi", () => {
  it("queryMACHistory 调用 post /network/history/list", async () => {
    const r = await queryMACHistory({ current: 1, pageSize: 20 });
    expect(r.list.length).toBe(1);
    expect(post).toHaveBeenCalledWith("/network/history/list", { current: 1, pageSize: 20 });
  });

  it("getMACEvents 默认 current=1 + pageSize=100", async () => {
    const r = await getMACEvents("00:11:22", "2026-08-01", "2026-08-07");
    expect(r.current).toBe(1);
    expect(r.pageSize).toBe(100);
    expect(r.hasMore).toBe(false);
  });

  it("getMACEvents 自定义 pageSize → hasMore 计算", async () => {
    const r = await getMACEvents("00:11:22", "2026-08-01", "2026-08-07", {
      current: 2,
      pageSize: 10,
    });
    expect(r.current).toBe(2);
    expect(r.pageSize).toBe(10);
  });

  it("writeShutdown 调用 shutdown endpoint", async () => {
    await writeShutdown("port-1", "test reason");
    expect(post).toHaveBeenCalledWith("/network/ports/write/shutdown", {
      portId: "port-1",
      reason: "test reason",
    });
  });

  it("writeUndoShutdown 调用 endpoint", async () => {
    await writeUndoShutdown("port-2", "undo");
    expect(post).toHaveBeenCalledWith("/network/ports/write/undo-shutdown", {
      portId: "port-2",
      reason: "undo",
    });
  });

  it("writeDescription 含 description 可选 reason", async () => {
    await writeDescription("port-3", "desc-x", "audit");
    expect(post).toHaveBeenCalledWith("/network/ports/write/description", {
      portId: "port-3",
      description: "desc-x",
      reason: "audit",
    });
  });

  it("writeDescription 无 reason", async () => {
    await writeDescription("port-3", "desc-x");
    expect(post).toHaveBeenCalledWith("/network/ports/write/description", {
      portId: "port-3",
      description: "desc-x",
      reason: undefined,
    });
  });

  it("writeDot1xEnable/Disable 调用 endpoint", async () => {
    await writeDot1xEnable("port-4", "auth");
    await writeDot1xDisable("port-4", "noauth");
    expect(post).toHaveBeenCalledWith("/network/ports/write/dot1x-enable", expect.any(Object));
    expect(post).toHaveBeenCalledWith("/network/ports/write/dot1x-disable", expect.any(Object));
  });

  it("writeSetAccessVlan 含 vlanId", async () => {
    await writeSetAccessVlan("port-5", 100, "vlan");
    expect(post).toHaveBeenCalledWith("/network/ports/write/set-access-vlan", {
      portId: "port-5",
      vlanId: 100,
      reason: "vlan",
    });
  });

  it("writePortBinding add + mac", async () => {
    await writePortBinding("port-6", "add", "10.0.0.1", "00:AA:BB", "binding");
    expect(post).toHaveBeenCalledWith("/network/ports/write/port-binding", {
      portId: "port-6",
      op: "add",
      ipAddress: "10.0.0.1",
      macAddress: "00:AA:BB",
      reason: "binding",
    });
  });

  it("writePortBinding remove 不含 mac", async () => {
    await writePortBinding("port-7", "remove", "10.0.0.2", undefined, "unbinding");
    expect(post).toHaveBeenCalledWith("/network/ports/write/port-binding", {
      portId: "port-7",
      op: "remove",
      ipAddress: "10.0.0.2",
      macAddress: undefined,
      reason: "unbinding",
    });
  });

  it("batchWritePorts 调用 batch endpoint", async () => {
    await batchWritePorts({
      deviceId: "d1",
      action: "shutdown",
      portIds: ["p1", "p2"],
    } as any);
    expect(post).toHaveBeenCalledWith("/network/ports/write/batch", {
      deviceId: "d1",
      action: "shutdown",
      portIds: ["p1", "p2"],
    });
  });

  it("getPortMACBundle 返回 bundle", async () => {
    const r = await getPortMACBundle("d1", "eth0");
    expect(r.error).toBeNull();
    expect(Array.isArray(r.current)).toBe(true);
  });

  it("getPortMACBundle error 收集", async () => {
    vi.mocked(post).mockRejectedValueOnce(new Error("net"));
    const r = await getPortMACBundle("d1", "eth0");
    expect(r.error).toBeInstanceOf(Error);
  });
});
