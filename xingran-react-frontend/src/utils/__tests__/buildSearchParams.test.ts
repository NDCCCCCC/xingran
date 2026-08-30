/**
 * Phase 88 Batch171 — utils/buildSearchParams 测试
 */
import { describe, it, expect } from "vitest";
import { buildSearchParams } from "../buildSearchParams";

describe("buildSearchParams", () => {
  it("无 options → 返回空对象", () => {
    expect(buildSearchParams({})).toEqual({});
  });

  it("只有 deptId → 注入 orgId", () => {
    expect(buildSearchParams({ deptId: "d1" })).toEqual({ orgId: "d1" });
  });

  it("只有 page → 注入 current + pageSize", () => {
    expect(buildSearchParams({ page: { current: 2, pageSize: 20 } })).toEqual({
      current: 2,
      pageSize: 20,
    });
  });

  it("只有 current → 只设 current", () => {
    expect(buildSearchParams({ page: { current: 1 } })).toEqual({ current: 1 });
  });

  it("deptId='' → 不注入 orgId", () => {
    expect(buildSearchParams({ deptId: "" })).toEqual({});
  });

  it("searchForm 提供 → 提取非空字段", () => {
    const form = {
      getFieldsValue: () => ({
        name: "alice",
        status: 0,
        keyword: "",
        empty: null,
        undef: undefined,
      }),
    } as any;
    const result = buildSearchParams({ searchForm: form });
    expect(result.name).toBe("alice");
    expect(result.status).toBe(0);
    expect("keyword" in result).toBe(false);
    expect("empty" in result).toBe(false);
    expect("undef" in result).toBe(false);
  });

  it("searchForm + deptId + page 全部提供", () => {
    const form = {
      getFieldsValue: () => ({ name: "test" }),
    } as any;
    const result = buildSearchParams({
      searchForm: form,
      deptId: "d1",
      page: { current: 1, pageSize: 10 },
    });
    expect(result).toEqual({ name: "test", orgId: "d1", current: 1, pageSize: 10 });
  });

  it("searchForm 无值 → 只设分页", () => {
    const form = { getFieldsValue: () => ({}) } as any;
    const result = buildSearchParams({
      searchForm: form,
      page: { current: 1 },
    });
    expect(result).toEqual({ current: 1 });
  });
});
