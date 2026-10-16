/**
 * Phase 88 Batch247 — components/operations/WorkstationDeviceTable/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { DEVICE_SOURCE_LABELS } from "../types";
import type { WorkstationDeviceTableProps } from "../types";

describe("operations/WorkstationDeviceTable/types", () => {
  it("DEVICE_SOURCE_LABELS 4 源", () => {
    expect(Object.keys(DEVICE_SOURCE_LABELS).length).toBeGreaterThanOrEqual(4);
    expect(DEVICE_SOURCE_LABELS.ad).toBe("域控");
    expect(DEVICE_SOURCE_LABELS.asset).toBe("资产");
    expect(DEVICE_SOURCE_LABELS.manual).toBe("手动");
  });

  it("WorkstationDeviceTableProps shape", () => {
    const p: WorkstationDeviceTableProps = {
      workstationId: "ws1",
      expandable: true,
      onDeviceChange: () => {},
    };
    expect(p.workstationId).toBe("ws1");
  });

  it("WorkstationDeviceTableProps 必填 only", () => {
    const p: WorkstationDeviceTableProps = { workstationId: "ws1" };
    expect(p.expandable).toBeUndefined();
    expect(p.onDeviceChange).toBeUndefined();
  });
});
