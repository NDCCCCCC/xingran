/**
 * Phase 88 Batch219 — hooks/useSidebarDeptFilter 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { useSidebarDeptFilter } from "../useSidebarDeptFilter";

describe("hooks/useSidebarDeptFilter", () => {
  it("初始 selectedDeptId = ''", () => {
    const { result } = renderHook(() => useSidebarDeptFilter());
    expect(result.current.selectedDeptId).toBe("");
  });

  it("setSelectedDeptId 设置", () => {
    const { result } = renderHook(() => useSidebarDeptFilter());
    act(() => result.current.setSelectedDeptId("d1"));
    expect(result.current.selectedDeptId).toBe("d1");
  });

  it("handleDeptSelect 取第一个 key", () => {
    const onDeptChange = vi.fn();
    const { result } = renderHook(() => useSidebarDeptFilter({ onDeptChange }));
    act(() =>
      result.current.handleDeptSelect(["d2"], { selected: true, node: { key: "d2" } as any })
    );
    expect(result.current.selectedDeptId).toBe("d2");
    expect(onDeptChange).toHaveBeenCalledWith("d2");
  });

  it("handleDeptSelect 空数组 → ''", () => {
    const onDeptChange = vi.fn();
    const { result } = renderHook(() => useSidebarDeptFilter({ onDeptChange }));
    act(() => result.current.handleDeptSelect([], { selected: false, node: {} as any }));
    expect(result.current.selectedDeptId).toBe("");
    expect(onDeptChange).toHaveBeenCalledWith("");
  });
});
