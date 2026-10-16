/**
 * Phase 88 Batch240 — pages/network/discoveries/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { IPRange, DiscoveryStatus, DiscoveryStatistics, ModalState } from "../types";

describe("network/discoveries/types", () => {
  it("IPRange shape", () => {
    const r: IPRange = { startIP: "10.0.0.1", endIP: "10.0.0.255" };
    expect(r.startIP).toBe("10.0.0.1");
  });

  it("DiscoveryStatus 4 值", () => {
    const s: DiscoveryStatus[] = ["pending", "running", "completed", "failed"];
    expect(s.length).toBe(4);
  });

  it("DiscoveryStatistics shape", () => {
    const s: DiscoveryStatistics = {
      total: 100,
      pending: 5,
      running: 2,
      completed: 90,
      failed: 3,
      totalDevices: 50,
    };
    expect(s.totalDevices).toBe(50);
  });

  it("ModalState shape", () => {
    const s: ModalState = { modalVisible: true, resultModalVisible: false };
    expect(s.modalVisible).toBe(true);
    expect(s.resultModalVisible).toBe(false);
  });
});
