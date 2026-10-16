/**
 * Phase 88 Batch381 — hooks/useColumnConfig 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/columnConfigApi", () => ({
  columnConfigApi: {
    getByPageKey: vi.fn(async () => ({ data: [] })),
    save: vi.fn(async () => ({})),
    reset: vi.fn(async () => ({})),
  },
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

import { columnConfigApi } from "@/lib/columnConfigApi";
import { useColumnConfig } from "../useColumnConfig";

const defaultCols = [
  { key: "name", label: "Name", visible: true, order: 1 },
  { key: "age", label: "Age", visible: true, order: 2 },
  { key: "email", label: "Email", visible: false, order: 3 },
];

describe("hooks/useColumnConfig", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  it("返回初始 shape", async () => {
    const { result } = renderHook(
      () => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }),
      { wrapper }
    );
    expect(result.current.loading).toBeDefined();
    expect(result.current.saving).toBeDefined();
    expect(result.current.config).toBeDefined();
    expect(typeof result.current.loadConfig).toBe("function");
    expect(typeof result.current.saveConfig).toBe("function");
    expect(typeof result.current.resetConfig).toBe("function");
    expect(typeof result.current.toggleColumn).toBe("function");
    expect(typeof result.current.updateColumnOrder).toBe("function");
    expect(typeof result.current.updateColumnWidth).toBe("function");
  });

  it("挂载后调 columnConfigApi.getByPageKey", async () => {
    renderHook(() => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }), {
      wrapper,
    });
    await waitFor(() => {
      expect(columnConfigApi.getByPageKey).toHaveBeenCalled();
    });
  });

  it("loadConfig 失败 → 不抛错", async () => {
    vi.mocked(columnConfigApi.getByPageKey).mockRejectedValue(new Error("net"));
    const { result } = renderHook(
      () => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }),
      { wrapper }
    );
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
  });

  it("toggleColumn 函数存在", () => {
    const { result } = renderHook(
      () => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }),
      { wrapper }
    );
    expect(() => result.current.toggleColumn("name", false)).not.toThrow();
  });

  it("updateColumnOrder 函数存在", () => {
    const { result } = renderHook(
      () => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }),
      { wrapper }
    );
    expect(() => result.current.updateColumnOrder(defaultCols)).not.toThrow();
  });

  it("resetConfig 函数存在", () => {
    const { result } = renderHook(
      () => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }),
      { wrapper }
    );
    expect(() => result.current.resetConfig()).not.toThrow();
  });

  it("updateColumnWidth 函数存在", () => {
    const { result } = renderHook(
      () => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }),
      { wrapper }
    );
    expect(() => result.current.updateColumnWidth("name", 150)).not.toThrow();
  });

  it("saveConfig 函数存在", () => {
    const { result } = renderHook(
      () => useColumnConfig({ pageKey: "test", defaultColumns: defaultCols }),
      { wrapper }
    );
    expect(typeof result.current.saveConfig).toBe("function");
  });
});
