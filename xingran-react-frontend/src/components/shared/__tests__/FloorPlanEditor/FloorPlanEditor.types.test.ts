/**
 * Phase 84 84-01a Task 2 — FloorPlanEditor types 与辅助函数断言
 */
import { describe, it, expect } from "vitest";
import type {
  WorkstationNode,
  FloorPlanEditorProps,
  ViewState,
  DragState,
  ContextMenuState,
} from "../../FloorPlanEditor.types";

describe("FloorPlanEditor.types", () => {
  it("WorkstationNode has required fields", () => {
    const node: WorkstationNode = {
      id: "ws-1",
      code: "A01",
      name: "工位1",
      x: 100,
      y: 200,
      width: 80,
      height: 60,
      status: 0,
      type: 1,
    };
    expect(node.id).toBe("ws-1");
    expect(node.x).toBe(100);
    expect(node.y).toBe(200);
    expect(node.status).toBe(0);
    expect(node.type).toBe(1);
  });

  it("WorkstationNode accepts optional rotation", () => {
    const node: WorkstationNode = {
      id: "ws-2",
      code: "A02",
      name: "工位2",
      x: 0,
      y: 0,
      width: 80,
      height: 60,
      status: 0,
      type: 1,
      rotation: 90,
    };
    expect(node.rotation).toBe(90);
  });

  it("FloorPlanEditorProps requires floorId and workstations", () => {
    const props: FloorPlanEditorProps = {
      floorId: "floor-1",
      workstations: [],
      onUpdatePosition: async () => {},
      onEdit: () => {},
    };
    expect(props.floorId).toBe("floor-1");
    expect(props.workstations).toHaveLength(0);
  });

  it("ViewState represents pan-zoom viewport", () => {
    const state: ViewState = {
      scale: 1.5,
      offsetX: 100,
      offsetY: -50,
      isDragging: false,
      dragStartX: 0,
      dragStartY: 0,
    };
    expect(state.scale).toBeGreaterThan(0);
  });

  it("DragState can be idle or active", () => {
    const idle: DragState = {
      isDragging: false,
      workstationId: null,
      startX: 0,
      startY: 0,
      originalX: 0,
      originalY: 0,
    };
    expect(idle.isDragging).toBe(false);
    expect(idle.workstationId).toBeNull();

    const active: DragState = {
      isDragging: true,
      workstationId: "ws-1",
      startX: 10,
      startY: 20,
      originalX: 100,
      originalY: 200,
    };
    expect(active.isDragging).toBe(true);
    expect(active.workstationId).toBe("ws-1");
  });

  it("ContextMenuState can be hidden or visible", () => {
    const hidden: ContextMenuState = { visible: false, x: 0, y: 0, workstation: null };
    expect(hidden.visible).toBe(false);
    const visible: ContextMenuState = {
      visible: true,
      x: 150,
      y: 250,
      workstation: {
        id: "ws-1",
        code: "A01",
        name: "工位1",
        x: 150,
        y: 250,
        width: 80,
        height: 60,
        status: 0,
        type: 1,
      },
    };
    expect(visible.visible).toBe(true);
    expect(visible.workstation?.id).toBe("ws-1");
  });
});
