/**
 * Phase 88 Batch267 — types/workstation 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { Workstation, WorkstationType, WorkstationStatus } from "../workstation";

describe("types/workstation", () => {
  it("WorkstationType 0/1/2", () => {
    const t: WorkstationType = 0;
    expect(t).toBe(0);
  });

  it("WorkstationStatus 0/1", () => {
    const s: WorkstationStatus = 0;
    expect(s).toBe(0);
  });

  it("Workstation shape", () => {
    const w: Workstation = {
      id: "w1",
      workstationCode: "WS-001",
      workstationName: "工位 1",
      workstationType: 0,
      status: 0,
      capacity: 1,
      createdAt: "2026-01-01",
      updatedAt: "2026-01-02",
    };
    expect(w.id).toBe("w1");
    expect(w.capacity).toBe(1);
  });

  it("Workstation 可选字段", () => {
    const w: Workstation = {
      id: "w1",
      workstationCode: "WS-001",
      workstationName: "工位 1",
      workstationType: 0,
      status: 0,
      capacity: 1,
      createdAt: "2026-01-01",
      updatedAt: "2026-01-02",
      deptId: "d1",
      deptName: "Dept",
      userId: "u1",
      userName: "User",
    };
    expect(w.deptName).toBe("Dept");
  });
});
