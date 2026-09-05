/**
 * apiFactory 契约测试 (Phase 94-01 D-11)
 *
 * 锁定: createResourceApi<T> 八方法路径拼接 / dropdownPath 覆盖 / 泛型透传 /
 * statistics·searchOptions 解包空回退 / batch 参数合并。
 * mock 模式照抄 opsApi.test.ts (vi.hoisted + mockPost/mockGet + 双路径 vi.mock)。
 * CreatePayload 编译期排除行为由 npm run type-check gate 覆盖(生产代码侧) —
 * 此处不写永不生效的负向类型断言注释(tsconfig exclude .test.ts,写了也不报错)。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockPost, mockGet } = vi.hoisted(() => ({
  mockPost: vi.fn(),
  mockGet: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
}));
vi.mock("./api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
}));

import { createResourceApi } from "./apiFactory";
import type { Building } from "@/types";

beforeEach(() => {
  mockPost.mockReset();
  mockGet.mockReset();
});

describe("createResourceApi — 路径拼接八连(/ops/building)", () => {
  it("八方法依次 POST 到正确端点", async () => {
    const api = createResourceApi<Building>({ basePath: "/ops/building" });
    mockPost.mockResolvedValue({ code: 0, data: {} });

    await api.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/building/list", {
      current: 1,
      pageSize: 10,
    });

    await api.get("b1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/building/b1", {});

    await api.create({ name: "新楼" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/building", { name: "新楼" });

    await api.update("b1", { name: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ops/building/b1/update", { name: "改名" });

    await api.delete("b1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/ops/building/b1/delete", {});

    await api.batch("delete", { ids: ["b1"] });
    expect(mockPost).toHaveBeenNthCalledWith(6, "/ops/building/batch", {
      action: "delete",
      ids: ["b1"],
    });

    await api.statistics({ status: 0 });
    expect(mockPost).toHaveBeenNthCalledWith(7, "/ops/building/statistics", { status: 0 });

    await api.searchOptions({ name: "研发" });
    expect(mockPost).toHaveBeenNthCalledWith(8, "/ops/building/dropdown-options", {
      name: "研发",
    });
  });
});

describe("CrudApiConfig.dropdownPath 覆盖", () => {
  it("自定义 dropdownPath 时 searchOptions 请求 URL 使用自定义值", async () => {
    const api = createResourceApi<Building>({
      basePath: "/ops/workstation",
      dropdownPath: "/custom-dropdown",
    });
    mockPost.mockResolvedValueOnce({ code: 0, data: [] });

    await api.searchOptions({});

    expect(mockPost).toHaveBeenCalledWith("/ops/workstation/custom-dropdown", {});
  });
});

describe("泛型透传", () => {
  it("list 返回值 === mockPost resolve 的 BaseResponse 原对象(透传不解包)", async () => {
    const api = createResourceApi<Building>({ basePath: "/ops/building" });
    const base = {
      code: 0,
      message: "success",
      data: { list: [{ id: "b1", name: "研发楼" }], total: 1, current: 1, pageSize: 10 },
    };
    mockPost.mockResolvedValueOnce(base);

    const result = await api.list({ current: 1, pageSize: 10 });

    expect(result).toBe(base);
  });
});

describe("statistics / searchOptions 解包与空回退 (D-09)", () => {
  it("statistics 返回 data 对象,data 为 null/undefined 时回退 {}", async () => {
    const api = createResourceApi<Building>({ basePath: "/ops/building" });

    mockPost.mockResolvedValueOnce({ code: 0, data: { total: 5, enabled: 3 } });
    expect(await api.statistics()).toEqual({ total: 5, enabled: 3 });
    expect(mockPost).toHaveBeenCalledWith("/ops/building/statistics", {});

    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    expect(await api.statistics({ status: 0 })).toEqual({});

    mockPost.mockResolvedValueOnce({ code: 0 });
    expect(await api.statistics()).toEqual({});
  });

  it("searchOptions 返回 data 数组,data 为 null/undefined 时回退 []", async () => {
    const api = createResourceApi<Building>({ basePath: "/ops/building" });

    mockPost.mockResolvedValueOnce({ code: 0, data: [{ value: "b1", label: "研发楼" }] });
    expect(await api.searchOptions({ name: "研发" })).toEqual([{ value: "b1", label: "研发楼" }]);

    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    expect(await api.searchOptions()).toEqual([]);

    mockPost.mockResolvedValueOnce({ code: 0 });
    expect(await api.searchOptions()).toEqual([]);
  });
});

describe("batch 参数合并", () => {
  it("batch(action, data) → 请求体深度等于 { action, ...data }", async () => {
    const api = createResourceApi<Building>({ basePath: "/ops/building" });
    mockPost.mockResolvedValueOnce({ code: 0 });

    await api.batch("enable", { ids: ["b1", "b2"] });

    expect(mockPost).toHaveBeenCalledWith("/ops/building/batch", {
      action: "enable",
      ids: ["b1", "b2"],
    });
    expect(mockPost.mock.calls[0][1]).toEqual({ action: "enable", ids: ["b1", "b2"] });
  });
});
