/**
 * Phase 88 Batch269 — components/shared/FloorPlanEditor.types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  WorkstationNode,
  FloorPlanEditorProps,
  ViewState,
  DragState,
  ContextMenuState,
} from "../FloorPlanEditor.types";

describe("shared/FloorPlanEditor.types", () => {
  it("WorkstationNode shape", () => {
    const n: WorkstationNode = {
      id: "w1",
      code: "WS1",
      name: "工位 1",
      x: 100,
      y: 200,
      width: 80,
      height: 60,
      status: 0,
      type: 0,
    };
    expect(n.x).toBe(100);
  });

  it("WorkstationNode rotation 可选", () => {
    const n: WorkstationNode = {
      id: "w1",
      code: "WS1",
      name: "工位 1",
      x: 0,
      y: 0,
      width: 80,
      height: 60,
      status: 0,
      type: 0,
      rotation: 90,
    };
    expect(n.rotation).toBe(90);
  });

  it("FloorPlanEditorProps shape", () => {
    const p: FloorPlanEditorProps = {
      floorId: "f1",
      workstations: [],
      onUpdatePosition: async () => {},
      onEdit: () => {},
    };
    expect(p.floorId).toBe("f1");
  });

  it("ViewState shape", () => {
    const v: ViewState = {
      scale: 1,
      offsetX: 0,
      offsetY: 0,
      isDragging: false,
      dragStartX: 0,
      dragStartY: 0,
    };
    expect(v.scale).toBe(1);
  });

  it("DragState shape", () => {
    const d: DragState = {
      isDragging: true,
      workstationId: "w1",
      startX: 10,
      startY: 20,
      originalX: 5,
      originalY: 5,
    };
    expect(d.workstationId).toBe("w1");
  });

  it("ContextMenuState shape", () => {
    const c: ContextMenuState = {
      visible: true,
      x: 100,
      y: 200,
      workstation: null,
    };
    expect(c.visible).toBe(true);
  });
});
