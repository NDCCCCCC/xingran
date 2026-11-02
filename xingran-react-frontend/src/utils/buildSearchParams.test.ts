import { describe, expect, it } from "vitest";
import type { FormInstance } from "antd";
import { buildSearchParams } from "./buildSearchParams";

function fakeForm(values: Record<string, unknown>): FormInstance<unknown> {
  return { getFieldsValue: () => values } as unknown as FormInstance<unknown>;
}

describe("buildSearchParams（表单 + 部门 + 分页合并）", () => {
  it("空入参返回空对象", () => {
    expect(buildSearchParams({})).toEqual({});
  });

  it("过滤 undefined / null / 空字符串字段，保留 0 与 false", () => {
    const params = buildSearchParams({
      searchForm: fakeForm({
        name: "工位",
        remark: "",
        status: 0,
        enabled: false,
        orgId2: null,
        extra: undefined,
      }),
    });
    expect(params).toEqual({ name: "工位", status: 0, enabled: false });
  });

  it("deptId 注入为 orgId，空 deptId 不注入", () => {
    expect(buildSearchParams({ deptId: "dept-uuid" })).toEqual({ orgId: "dept-uuid" });
    expect(buildSearchParams({ deptId: "" })).toEqual({});
    expect(buildSearchParams({ deptId: undefined })).toEqual({});
  });

  it("page 注入 current / pageSize（部分提供部分省略）", () => {
    expect(buildSearchParams({ page: { current: 2, pageSize: 20 } })).toEqual({
      current: 2,
      pageSize: 20,
    });
    expect(buildSearchParams({ page: { current: 3 } })).toEqual({ current: 3 });
    expect(buildSearchParams({ page: {} })).toEqual({});
  });

  it("三者合并时表单值与注入字段共存", () => {
    const params = buildSearchParams({
      searchForm: fakeForm({ name: "A" }),
      deptId: "d1",
      page: { current: 1, pageSize: 10 },
    });
    expect(params).toEqual({ name: "A", orgId: "d1", current: 1, pageSize: 10 });
  });
});
