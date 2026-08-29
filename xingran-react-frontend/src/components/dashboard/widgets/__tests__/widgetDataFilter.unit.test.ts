/**
 * Phase 88 Batch48 — dashboard widgets WidgetDataFilter 单元测试
 *
 * applyWidgetDataFilter 4 分支(dept/user/custom/未启用/无 user) +
 * getAvailableDataFilters 4 项配置断言。
 */
import { describe, it, expect, beforeEach } from "vitest";

import { useAuthStore } from "@/store/authStore";
import { applyWidgetDataFilter, getAvailableDataFilters } from "../WidgetDataFilter";
import type { WidgetConfig } from "@/types/dashboard";

const baseWidget = (dataFilter?: WidgetConfig["dataFilter"]): WidgetConfig =>
  ({
    id: "w1",
    type: "stat",
    title: "测试",
    dataFilter,
  }) as unknown as WidgetConfig;

beforeEach(() => {
  useAuthStore.setState({
    user: { id: "u1", deptId: "d1", username: "tester" } as any,
  });
});

describe("applyWidgetDataFilter", () => {
  it("dataFilter 未启用 → 原样返回 baseParams", () => {
    const result = applyWidgetDataFilter(
      baseWidget({ enabled: false, filterType: "dept", filterConfig: {} }),
      { page: 1 }
    );
    expect(result).toEqual({ page: 1 });
  });

  it("dataFilter undefined → 原样返回", () => {
    const result = applyWidgetDataFilter(baseWidget(undefined), { page: 1 });
    expect(result).toEqual({ page: 1 });
  });

  it("user 为 null → 原样返回", () => {
    useAuthStore.setState({ user: null as any });
    const result = applyWidgetDataFilter(
      baseWidget({ enabled: true, filterType: "dept", filterConfig: {} }),
      { page: 1 }
    );
    expect(result).toEqual({ page: 1 });
  });

  it("filterType=dept → 注入 user.deptId(默认 field)", () => {
    const result = applyWidgetDataFilter(
      baseWidget({ enabled: true, filterType: "dept", filterConfig: {} }),
      { page: 1 }
    );
    expect(result).toEqual({ page: 1, deptId: "d1" });
  });

  it("filterType=dept + 自定义 field → 注入指定 field", () => {
    const result = applyWidgetDataFilter(
      baseWidget({ enabled: true, filterType: "dept", filterConfig: { field: "orgId" } }),
      { page: 1 }
    );
    expect(result).toEqual({ page: 1, orgId: "d1" });
  });

  it("filterType=user → 注入 user.id(默认 field)", () => {
    const result = applyWidgetDataFilter(
      baseWidget({ enabled: true, filterType: "user", filterConfig: {} }),
      { page: 1 }
    );
    expect(result).toEqual({ page: 1, userId: "u1" });
  });

  it("filterType=user + 自定义 field", () => {
    const result = applyWidgetDataFilter(
      baseWidget({ enabled: true, filterType: "user", filterConfig: { field: "ownerId" } }),
      {}
    );
    expect(result).toEqual({ ownerId: "u1" });
  });

  it("filterType=custom → 展开 filterConfig", () => {
    const result = applyWidgetDataFilter(
      baseWidget({
        enabled: true,
        filterType: "custom",
        filterConfig: { status: 0, keyword: "x" },
      }),
      { page: 1 }
    );
    expect(result).toEqual({ page: 1, status: 0, keyword: "x" });
  });

  it("filterType 未知 → 原样返回(default 分支)", () => {
    const result = applyWidgetDataFilter(
      baseWidget({ enabled: true, filterType: "unknown" as any, filterConfig: {} }),
      { page: 1 }
    );
    expect(result).toEqual({ page: 1 });
  });
});

describe("getAvailableDataFilters", () => {
  it("返回 4 项过滤配置", () => {
    const filters = getAvailableDataFilters();
    expect(filters.length).toBe(4);
    expect(filters.map((f) => f.value)).toEqual(["none", "dept", "user", "custom"]);
  });

  it("每项含 label + defaultConfig", () => {
    for (const f of getAvailableDataFilters()) {
      expect(typeof f.label).toBe("string");
      expect(f.label.length).toBeGreaterThan(0);
      expect(f.defaultConfig).toBeDefined();
      expect(typeof f.defaultConfig.filterType).toBe("string");
    }
  });

  it("none 项 enabled=false,其余 enabled=true", () => {
    const filters = getAvailableDataFilters();
    expect(filters[0].defaultConfig.enabled).toBe(false);
    for (let i = 1; i < filters.length; i++) {
      expect(filters[i].defaultConfig.enabled).toBe(true);
    }
  });
});
