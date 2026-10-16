/**
 * Phase 88 Batch26 — monitor logs loginColumns/operColumns 单元测试
 */
import { describe, it, expect } from "vitest";

import { getLoginLogColumns } from "../logs/columns/loginColumns";
import { getOperLogColumns } from "../logs/columns/operColumns";

const cbs = {
  handleViewDetail: () => {},
  getColumnSortOrder: (_: string) => undefined as never,
};

describe("getLoginLogColumns", () => {
  it("返回列数组含关键 key", () => {
    const cols = getLoginLogColumns(cbs);
    expect(cols.length).toBeGreaterThan(5);
    const keys = cols.map((c) => c.key as string);
    expect(keys).toEqual(
      expect.arrayContaining(["id", "userName", "ipAddr", "loginLocation", "status", "action"])
    );
  });

  it("userName render: nickname 不同于 userName 时合并展示", () => {
    const cols = getLoginLogColumns(cbs);
    const col = cols.find((c) => c.key === "userName");
    const render = col?.render as (_: unknown, r: any) => any;
    const el = render(undefined, { userName: "alice", nickname: "Alice Zhang" });
    // Space 组件包裹 children 数组
    const text = JSON.stringify(el.props.children);
    expect(text).toContain("Alice Zhang（alice）");
  });

  it("userName render: nickname 相同/缺失回退 userName", () => {
    const cols = getLoginLogColumns(cbs);
    const col = cols.find((c) => c.key === "userName");
    const render = col?.render as (_: unknown, r: any) => any;
    const el1 = render(undefined, { userName: "bob", nickname: "bob" });
    expect(JSON.stringify(el1.props.children)).toContain("bob");
    const el2 = render(undefined, { userName: "" });
    expect(JSON.stringify(el2.props.children)).toContain("-");
  });
});

describe("getOperLogColumns", () => {
  it("返回列数组含关键 key", () => {
    const cols = getOperLogColumns(cbs);
    expect(cols.length).toBeGreaterThan(6);
    const keys = cols.map((c) => c.key as string);
    expect(keys).toEqual(
      expect.arrayContaining([
        "id",
        "title",
        "businessType",
        "requestMethod",
        "operName",
        "status",
        "action",
      ])
    );
  });

  it("businessType render 委托 getBusinessTypeLabel", () => {
    const cols = getOperLogColumns(cbs);
    const col = cols.find((c) => c.key === "businessType");
    const render = col?.render as (t: number) => unknown;
    expect(render(9999)).toBe("-");
  });

  it("requestMethod render 委托 renderRequestMethodTag", () => {
    const cols = getOperLogColumns(cbs);
    const col = cols.find((c) => c.key === "requestMethod");
    const render = col?.render as (m: string) => any;
    expect(render("GET").props.color).toBe("blue");
  });

  it("sorter 传参生效", () => {
    const cb2 = {
      ...cbs,
      getColumnSortOrder: (f: string) => (f === "title" ? ("ascend" as const) : undefined),
    };
    const cols = getOperLogColumns(cb2);
    const col = cols.find((c) => c.key === "title");
    expect(col?.sortOrder).toBe("ascend");
  });
});
