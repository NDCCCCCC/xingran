/**
 * Phase 88 Batch227 — components/dashboard/widgets/WidgetDataFilter 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/store/authStore", () => ({
  useAuthStore: {
    getState: vi.fn(() => ({ user: { id: "u1", deptId: "d1" } })),
  },
}));

import { applyWidgetDataFilter, getAvailableDataFilters } from "../WidgetDataFilter";

describe("dashboard/widgets/WidgetDataFilter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("getAvailableDataFilters 4 项", () => {
    const filters = getAvailableDataFilters();
    expect(filters.length).toBe(4);
    expect(filters[0].value).toBe("none");
    expect(filters[1].value).toBe("dept");
    expect(filters[2].value).toBe("user");
    expect(filters[3].value).toBe("custom");
  });

  it("applyWidgetDataFilter 未启用 → 返回原 params", () => {
    const widget: any = { dataFilter: { enabled: false, filterType: "dept", filterConfig: {} } };
    const result = applyWidgetDataFilter(widget, { x: 1 });
    expect(result).toEqual({ x: 1 });
  });

  it("applyWidgetDataFilter 无 dataFilter → 原 params", () => {
    const widget: any = {};
    const result = applyWidgetDataFilter(widget, { x: 1 });
    expect(result).toEqual({ x: 1 });
  });

  it("applyWidgetDataFilter dept 过滤", () => {
    const widget: any = {
      dataFilter: {
        enabled: true,
        filterType: "dept",
        filterConfig: { field: "orgId" },
      },
    };
    const result = applyWidgetDataFilter(widget, { x: 1 });
    expect(result).toEqual({ x: 1, orgId: "d1" });
  });

  it("applyWidgetDataFilter dept 默认 field", () => {
    const widget: any = {
      dataFilter: {
        enabled: true,
        filterType: "dept",
        filterConfig: {},
      },
    };
    const result = applyWidgetDataFilter(widget, {});
    expect(result).toEqual({ deptId: "d1" });
  });

  it("applyWidgetDataFilter user 过滤", () => {
    const widget: any = {
      dataFilter: {
        enabled: true,
        filterType: "user",
        filterConfig: { field: "ownerId" },
      },
    };
    const result = applyWidgetDataFilter(widget, {});
    expect(result).toEqual({ ownerId: "u1" });
  });

  it("applyWidgetDataFilter custom 合并 filterConfig", () => {
    const widget: any = {
      dataFilter: {
        enabled: true,
        filterType: "custom",
        filterConfig: { extra: "x" },
      },
    };
    const result = applyWidgetDataFilter(widget, { base: 1 });
    expect(result).toEqual({ base: 1, extra: "x" });
  });

  it("applyWidgetDataFilter 未知 filterType → 原 params", () => {
    const widget: any = {
      dataFilter: {
        enabled: true,
        filterType: "unknown",
        filterConfig: {},
      },
    };
    const result = applyWidgetDataFilter(widget, { x: 1 });
    expect(result).toEqual({ x: 1 });
  });

  it("user 为 null → 原 params", async () => {
    const authStore = await import("@/store/authStore");
    vi.mocked(authStore.useAuthStore.getState).mockReturnValueOnce({ user: null } as any);
    const widget: any = {
      dataFilter: {
        enabled: true,
        filterType: "dept",
        filterConfig: { field: "deptId" },
      },
    };
    const result = applyWidgetDataFilter(widget, { x: 1 });
    expect(result).toEqual({ x: 1 });
  });
});
