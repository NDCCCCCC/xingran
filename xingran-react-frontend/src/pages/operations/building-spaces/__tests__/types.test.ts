/**
 * Phase 88 Batch243 — pages/operations/building-spaces/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { Building, Floor, WorkstationWithPosition, AnimationState, ModalView } from "../types";

describe("operations/building-spaces/types", () => {
  it("Building shape", () => {
    const b: Building = {
      id: "b1",
      name: "B1",
      code: "C1",
      totalFloors: 5,
      workstationCount: 100,
      status: 0,
    };
    expect(b.name).toBe("B1");
  });

  it("Floor shape", () => {
    const f: Floor = {
      id: "f1",
      buildingId: "b1",
      floorNo: "F1",
      name: "Floor 1",
      workstationCount: 20,
    };
    expect(f.floorNo).toBe("F1");
  });

  it("WorkstationWithPosition 含 position", () => {
    const w: WorkstationWithPosition = {
      id: "w1",
      name: "WS1",
      status: 0,
      type: 0,
      position: { x: 100, y: 200 },
    };
    expect(w.position?.x).toBe(100);
  });

  it("AnimationState 5 值", () => {
    const states: AnimationState[] = ["stacked", "expanding", "expanded", "flattening", "flat"];
    expect(states.length).toBe(5);
  });

  it("ModalView 3 值", () => {
    const views: ModalView[] = ["floors", "workstation", "transition"];
    expect(views.length).toBe(3);
  });
});
