/**
 * Phase 84 84-01a Task 2 — FloorPlanEditor hooks 测试
 * useWorkstationDrag(工位拖拽状态管理)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWorkstationDrag } from "../../FloorPlanEditor.hooks";

describe("useWorkstationDrag", () => {
  const makeOptions = () => ({
    workstations: [],
    onUpdatePosition: vi.fn(() => Promise.resolve()),
  });

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("initializes dragState as idle", () => {
    const { result } = renderHook(() => useWorkstationDrag(makeOptions()));
    expect(result.current.dragState.isDragging).toBe(false);
    expect(result.current.dragState.workstationId).toBeNull();
    expect(result.current.draggedNodePos).toBeNull();
  });

  it("handleStartDrag activates drag and captures workstationId", () => {
    const { result } = renderHook(() => useWorkstationDrag(makeOptions()));
    act(() => {
      result.current.handleStartDrag("ws-1", 100, 200, 300, 400);
    });
    expect(result.current.dragState.isDragging).toBe(true);
    expect(result.current.dragState.workstationId).toBe("ws-1");
    // draggedNodePos set by handleDragMove, not handleStartDrag
    expect(result.current.draggedNodePos).toBeNull();
  });

  it("clearDraggedPos resets drag state", () => {
    const { result } = renderHook(() => useWorkstationDrag(makeOptions()));
    act(() => {
      result.current.handleStartDrag("ws-1", 10, 20, 100, 200);
    });
    act(() => {
      result.current.clearDraggedPos();
    });
    expect(result.current.draggedNodePos).toBeNull();
  });

  it("handleStartDrag preserves original coordinates", () => {
    const { result } = renderHook(() => useWorkstationDrag(makeOptions()));
    act(() => {
      result.current.handleStartDrag("ws-2", 50, 60, 500, 600);
    });
    expect(result.current.dragState.originalX).toBe(500);
    expect(result.current.dragState.originalY).toBe(600);
  });
});
