/**
 * Phase 88 Batch428 — FloorPlanEditor.useWorkstationDrag hook 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWorkstationDrag } from "../FloorPlanEditor.hooks";
import { GRID_SIZE } from "../FloorPlanEditor.constants";
import type { WorkstationNode } from "../FloorPlanEditor.types";

const baseNode: WorkstationNode = {
  id: "ws-1",
  positionX: 0,
  positionY: 0,
  rotation: 0,
} as unknown as WorkstationNode;

describe("useWorkstationDrag", () => {
  it("初始 dragState 未拖拽", () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    expect(result.current.dragState.isDragging).toBe(false);
    expect(result.current.dragState.workstationId).toBeNull();
    expect(result.current.draggedNodePos).toBeNull();
  });

  it("handleStartDrag 设置状态", () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    act(() => result.current.handleStartDrag("ws-1", 100, 200, 0, 0));
    expect(result.current.dragState.isDragging).toBe(true);
    expect(result.current.dragState.workstationId).toBe("ws-1");
    expect(result.current.dragState.startX).toBe(100);
    expect(result.current.dragState.startY).toBe(200);
    expect(result.current.dragState.originalX).toBe(0);
    expect(result.current.dragState.originalY).toBe(0);
  });

  it("handleDragMove 未拖拽时 no-op", () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    act(() => result.current.handleDragMove(50, 50, 1));
    expect(result.current.draggedNodePos).toBeNull();
  });

  it("handleDragMove 拖拽中按 scale 计算偏移", () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    act(() => result.current.handleStartDrag("ws-1", 100, 200, 0, 0));
    act(() => result.current.handleDragMove(110, 220, 1));
    expect(result.current.draggedNodePos).toEqual({ x: 10, y: 20 });
  });

  it("handleDragMove scale=2 时偏移减半", () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    act(() => result.current.handleStartDrag("ws-1", 100, 200, 0, 0));
    act(() => result.current.handleDragMove(120, 220, 2));
    expect(result.current.draggedNodePos).toEqual({ x: 10, y: 10 });
  });

  it("handleEndDrag 未拖拽时重置不调 onUpdate", async () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    await act(async () => {
      await result.current.handleEndDrag();
    });
    expect(onUpdate).not.toHaveBeenCalled();
  });

  it("handleEndDrag 拖拽后调用 onUpdate 并吸附到网格", async () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    act(() => result.current.handleStartDrag("ws-1", 0, 0, 0, 0));
    act(() => result.current.handleDragMove(GRID_SIZE * 3 + 5, GRID_SIZE * 2 + 8, 1));
    await act(async () => {
      await result.current.handleEndDrag();
    });
    expect(onUpdate).toHaveBeenCalledTimes(1);
    const arg = onUpdate.mock.calls[0][0][0];
    expect(arg.id).toBe("ws-1");
    expect(arg.positionX).toBe(GRID_SIZE * 3);
    expect(arg.positionY).toBe(GRID_SIZE * 2);
    expect(result.current.dragState.isDragging).toBe(false);
    expect(result.current.draggedNodePos).toBeNull();
  });

  it("handleEndDrag workstation 不在列表中只重置", async () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [], onUpdatePosition: onUpdate })
    );
    act(() => result.current.handleStartDrag("ghost", 0, 0, 0, 0));
    act(() => result.current.handleDragMove(50, 50, 1));
    await act(async () => {
      await result.current.handleEndDrag();
    });
    expect(onUpdate).not.toHaveBeenCalled();
    expect(result.current.dragState.isDragging).toBe(false);
  });

  it("clearDraggedPos 清空 draggedNodePos", () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() =>
      useWorkstationDrag({ workstations: [baseNode], onUpdatePosition: onUpdate })
    );
    act(() => result.current.handleStartDrag("ws-1", 0, 0, 0, 0));
    act(() => result.current.handleDragMove(50, 50, 1));
    expect(result.current.draggedNodePos).not.toBeNull();
    act(() => result.current.clearDraggedPos());
    expect(result.current.draggedNodePos).toBeNull();
  });
});
