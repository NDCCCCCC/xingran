/**
 * managerApi 契约测试（v1.56 PR #114 CI 覆盖率补齐）
 *
 * 锁定：机构 4 端点 + 自动生成 + 规则 4 端点 + 经理预览的路径/方法/降级语义。
 * 此前该文件 0 覆盖（30 条语句），是 lib/ 目录拉到 floor 以下的主因之一。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));

import {
  MANAGER_ORG_TARGET_PARENT,
  createManagerOrg,
  createManagerRule,
  deleteManagerOrg,
  deleteManagerRule,
  generateManagerOrgsFromDept,
  getManagerOrgList,
  getManagerRuleList,
  isManagerEnabled,
  previewManagers,
  updateManagerOrg,
  updateManagerRule,
} from "./managerApi";

describe("managerApi 机构端点", () => {
  beforeEach(() => mockPost.mockReset());

  it("getManagerOrgList POST /system/manager-orgs/list 并在 data 缺失时回落空页", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    const page = await getManagerOrgList({ current: 2, pageSize: 20 });
    expect(mockPost).toHaveBeenCalledWith("/system/manager-orgs/list", {
      current: 2,
      pageSize: 20,
    });
    expect(page).toEqual({ list: [], total: 0, current: 2, pageSize: 20 });
  });

  it("createManagerOrg POST /system/manager-orgs", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await createManagerOrg({ name: "某机构" } as never);
    expect(mockPost).toHaveBeenCalledWith("/system/manager-orgs", { name: "某机构" });
  });

  it("updateManagerOrg 路径带 id", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await updateManagerOrg("o-1", { name: "改" } as never);
    expect(mockPost).toHaveBeenCalledWith("/system/manager-orgs/o-1/update", { name: "改" });
  });

  it("deleteManagerOrg 路径带 id 且体为空对象", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await deleteManagerOrg("o-2");
    expect(mockPost).toHaveBeenCalledWith("/system/manager-orgs/o-2/delete", {});
  });

  it("generateManagerOrgsFromDept 在 data 缺失时回落零值结果", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    const r = await generateManagerOrgsFromDept();
    expect(mockPost).toHaveBeenCalledWith("/system/manager-orgs/generate-from-dept", {});
    expect(r).toEqual({ totalCandidates: 0, created: 0, skippedExisting: 0, orphanCount: 0 });
  });
});

describe("managerApi 规则端点", () => {
  beforeEach(() => mockPost.mockReset());

  it("getManagerRuleList POST /system/manager-rules/list", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { list: [], total: 5 } });
    const page = await getManagerRuleList({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenCalledWith("/system/manager-rules/list", {
      current: 1,
      pageSize: 10,
    });
    expect(page.total).toBe(5);
  });

  it("createManagerRule / updateManagerRule / deleteManagerRule 三个写端点", async () => {
    mockPost.mockResolvedValue({ code: 0 });
    await createManagerRule({} as never);
    expect(mockPost).toHaveBeenCalledWith("/system/manager-rules", {});
    await updateManagerRule("r-1", {} as never);
    expect(mockPost).toHaveBeenCalledWith("/system/manager-rules/r-1/update", {});
    await deleteManagerRule("r-2");
    expect(mockPost).toHaveBeenCalledWith("/system/manager-rules/r-2/delete", {});
  });
});

describe("managerApi 经理预览", () => {
  beforeEach(() => mockPost.mockReset());

  it("空 userIds 直接返回空数组，不发请求", async () => {
    const r = await previewManagers([]);
    expect(r).toEqual([]);
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("非空 userIds POST manager-preview 并解包 data", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: [{ userId: "u-1", managerName: "张三" }] });
    const r = await previewManagers(["u-1"]);
    expect(mockPost).toHaveBeenCalledWith("/system/manager-rules/manager-preview", {
      userIds: ["u-1"],
    });
    expect(r[0].managerName).toBe("张三");
  });

  it("请求失败静默降级为空数组（预览是增强展示）", async () => {
    mockPost.mockRejectedValueOnce(new Error("boom"));
    const r = await previewManagers(["u-1"]);
    expect(r).toEqual([]);
  });
});

describe("isManagerEnabled 类型宽容", () => {
  it.each([
    [undefined, false],
    [null, false],
    [0, true],
    [1, false],
    [true, true],
    [false, false],
  ])("%s -> %s", (input, expected) => {
    expect(isManagerEnabled(input as never)).toBe(expected);
  });
});

describe("常量", () => {
  it("MANAGER_ORG_TARGET_PARENT 是稳定哨兵值", () => {
    expect(MANAGER_ORG_TARGET_PARENT).toBe("parent");
  });
});
