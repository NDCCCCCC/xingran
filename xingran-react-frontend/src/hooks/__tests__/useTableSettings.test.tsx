/**
 * Phase 88 Batch302 — hooks/useTableSettings 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/store/settingsStore", () => ({
  useSettingsStore: vi.fn(() => ({ preferences: {} })),
}));

import { useTableSettings } from "../useTableSettings";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <MemoryRouter>{children}</MemoryRouter>;
}

describe("hooks/useTableSettings", () => {
  it("默认返回 pagination + getTableProps", () => {
    const { result } = renderHook(() => useTableSettings(), { wrapper });
    expect(result.current.pagination).toBeDefined();
    expect(typeof result.current.getTableProps).toBe("function");
  });

  it("getTableProps 返回包含 pagination", () => {
    const { result } = renderHook(() => useTableSettings(), { wrapper });
    const props = result.current.getTableProps<{ id: string }>();
    expect(props.pagination).toBeDefined();
    expect(props.pagination?.current).toBe(1);
  });

  it("getTableProps 接受泛型参数不抛错", () => {
    const { result } = renderHook(() => useTableSettings(), { wrapper });
    expect(() => result.current.getTableProps<{ a: number; b: string }>()).not.toThrow();
  });

  it("options 显式为空对象也使用默认", () => {
    const { result } = renderHook(() => useTableSettings({}), { wrapper });
    expect(result.current.pagination).toBeDefined();
  });

  it("pageSize 选项不抛错", () => {
    const { result } = renderHook(() => useTableSettings({ pageSize: 25 }), { wrapper });
    expect(result.current.pagination).toBeDefined();
  });

  it("enableGlobalPagination=false 不抛错", () => {
    const { result } = renderHook(() => useTableSettings({ enableGlobalPagination: false }), {
      wrapper,
    });
    expect(result.current.pagination).toBeDefined();
  });
});
