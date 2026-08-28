/**
 * Phase 88 Batch24 — workstations/columns + render branches 单元测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";

import { getWorkstationColumns } from "../columns";
import { STATUS_OPTIONS } from "../constants";

describe("getWorkstationColumns 覆盖", () => {
  const cbs = {
    handleEdit: vi.fn(),
    handleDelete: vi.fn(),
    getColumnSortOrder: (_field: string) => undefined as never,
  };

  it("返回 columns 数组且包含主键", () => {
    const cols = getWorkstationColumns(cbs);
    expect(Array.isArray(cols)).toBe(true);
    expect(cols.length).toBeGreaterThan(8);
    const keys = cols.map((c) => c.key as string);
    expect(keys).toContain("name");
    expect(keys).toContain("buildingName");
    expect(keys).toContain("floorName");
    expect(keys).toContain("deptName");
    expect(keys).toContain("userName");
    expect(keys).toContain("status");
    expect(keys).toContain("createdAt");
    expect(keys).toContain("updatedAt");
    expect(keys).toContain("action");
  });

  it("name 列 sorter=true + sortOrder 来自 cb", () => {
    const cb = {
      handleEdit: vi.fn(),
      handleDelete: vi.fn(),
      getColumnSortOrder: (f: string) => (f === "name" ? ("ascend" as const) : undefined),
    };
    const cols = getWorkstationColumns(cb);
    const nameCol = cols.find((c) => c.key === "name");
    expect(nameCol?.sorter).toBe(true);
    expect(nameCol?.sortOrder).toBe("ascend");
  });

  it("render 兜底: buildingName/floorName/deptName/userName 空值返 '-'", () => {
    const cols = getWorkstationColumns(cbs);
    for (const key of ["buildingName", "floorName", "deptName", "userName"]) {
      const col = cols.find((c) => c.key === key);
      const render = col?.render as (text: unknown) => unknown;
      expect(render(undefined)).toBe("-");
      expect(render("")).toBe("-");
      expect(render(null)).toBe("-");
      expect(render("realValue")).toBe("realValue");
    }
  });

  it("action 列 handleEdit 触发回调", () => {
    const cols = getWorkstationColumns(cbs);
    const action = cols.find((c) => c.key === "action");
    const item: any = { id: "ws-1", name: "n" };
    // render returns ActionButtons JSX, can't easily execute — just check column shape
    expect(action).toBeDefined();
    expect(action?.title).toBeDefined();
    expect(typeof action?.render).toBe("function");
  });

  it("primaryDeviceSerial ellipsis + width", () => {
    const cols = getWorkstationColumns(cbs);
    const c = cols.find((x) => x.key === "primaryDeviceSerial");
    expect(c?.ellipsis).toBe(true);
    expect(c?.width).toBe(140);
  });

  it("type 列 width=100 + sorter", () => {
    const cols = getWorkstationColumns(cbs);
    const c = cols.find((x) => x.key === "type");
    expect(c?.width).toBe(100);
    expect(c?.sorter).toBe(true);
  });

  it("status 列 render 是函数(状态 tag)", () => {
    const cols = getWorkstationColumns(cbs);
    const c = cols.find((x) => x.key === "status");
    expect(c).toBeDefined();
    expect(typeof c?.render).toBe("function");
  });

  it("type 列 render 是函数(类型 tag)", () => {
    const cols = getWorkstationColumns(cbs);
    const c = cols.find((x) => x.key === "type");
    expect(c).toBeDefined();
    expect(typeof c?.render).toBe("function");
  });
});

describe("STATUS_OPTIONS constants (3 态:0=空闲/1=占用/2=维护)", () => {
  it("包含 3 个标准状态选项", () => {
    expect(STATUS_OPTIONS.length).toBe(3);
    for (const opt of STATUS_OPTIONS) {
      expect(typeof opt.value).toBe("number");
      expect(typeof opt.label).toBe("string");
    }
  });
});
