/**
 * assetApi(资产对账)端点契约测试 (Phase 83-03)
 *
 * 锁定:reconciliationApi 统计/异常/规则/基线/刷新 + fixSuggestionApi 状态机
 * 各端点 URL、请求体形状与 data 解包回退。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));

import { fixSuggestionApi, reconciliationApi } from "./assetApi";

const SUMMARY = {
  totalAssets: 10,
  openExceptions: 2,
  criticalOpen: 1,
  last7dNew: 3,
  topConflictType: "A",
  topConflictCount: 2,
};

describe("reconciliationApi — 统计端点", () => {
  beforeEach(() => mockPost.mockReset());

  it("summary 默认 7d 并解包 data", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: SUMMARY });
    const result = await reconciliationApi.summary();
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/statistics/summary", { days: 7 });
    expect(result).toEqual(SUMMARY);
  });

  it("byConflictType / bySeverity 携带窗口参数,data 空时回退 {}", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { A: 2, B: 0 } });
    const grouped = await reconciliationApi.byConflictType(30);
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/statistics/by-conflict-type", {
      days: 30,
    });
    expect(grouped).toEqual({ A: 2, B: 0 });

    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    const empty = await reconciliationApi.bySeverity(30);
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/statistics/by-severity", {
      days: 30,
    });
    expect(empty).toEqual({});
  });

  it("healthTrend / topUnresolved / exceptionRuleStats data 空时回退 []", async () => {
    mockPost.mockResolvedValue({ code: 0, data: null });
    expect(await reconciliationApi.healthTrend(7)).toEqual([]);
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/statistics/health-trend", {
      days: 7,
    });

    expect(await reconciliationApi.topUnresolved(10)).toEqual([]);
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/statistics/top-unresolved", {
      limit: 10,
    });

    expect(await reconciliationApi.exceptionRuleStats()).toEqual([]);
    expect(mockPost).toHaveBeenCalledWith(
      "/asset/reconciliation/statistics/exception-rule-stats",
      {}
    );
  });
});

describe("reconciliationApi — 异常端点", () => {
  beforeEach(() => mockPost.mockReset());

  it("exceptionList 透传分页筛选参数,data 空时回退空页", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    const params = { current: 2, pageSize: 20, severity: "critical" };
    const result = await reconciliationApi.exceptionList(params);
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/exception/list", params);
    expect(result).toEqual({ list: [], total: 0, current: 1, pageSize: 20 });
  });

  it("exceptionGetByID / exceptionResolve 按 ID 拼接", async () => {
    mockPost.mockResolvedValue({ code: 0, data: { id: "e1" } });
    await reconciliationApi.exceptionGetByID("e1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/asset/reconciliation/exception/e1", {});
    await reconciliationApi.exceptionResolve("e1", { resolutionNote: "已人工修复" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/asset/reconciliation/exception/e1/resolve", {
      resolutionNote: "已人工修复",
    });
  });

  it("byWorkstation 透传 workstationId,window 默认 7d", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { visible: true, assets: [] } });
    await reconciliationApi.byWorkstation({ workstationId: "ws-1" });
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/by-workstation", {
      workstationId: "ws-1",
      window: "7d",
    });
  });
});

describe("reconciliationApi.exceptionRule — R3 规则 CRUD 与命中测试", () => {
  beforeEach(() => mockPost.mockReset());

  it("list / getById / create / update / delete", async () => {
    mockPost.mockResolvedValue({ code: 0, data: { id: "r1" } });
    await reconciliationApi.exceptionRule.list({ current: 1 });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/asset/reconciliation/exception-rule/list", {
      current: 1,
    });
    await reconciliationApi.exceptionRule.getById("r1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/asset/reconciliation/exception-rule/r1", {});
    const create = { ruleName: "核心交换机静默", ipCidr: "100.64.0.0/10" };
    await reconciliationApi.exceptionRule.create(create);
    expect(mockPost).toHaveBeenNthCalledWith(
      3,
      "/asset/reconciliation/exception-rule/create",
      create
    );
    await reconciliationApi.exceptionRule.update("r1", { ruleName: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(4, "/asset/reconciliation/exception-rule/r1/update", {
      ruleName: "改名",
    });
    await reconciliationApi.exceptionRule.delete("r1");
    expect(mockPost).toHaveBeenNthCalledWith(
      5,
      "/asset/reconciliation/exception-rule/r1/delete",
      {}
    );
  });

  it("test 命中测试透传 ip/userId/deptId,data 空时回退零值", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    const result = await reconciliationApi.exceptionRule.test({ ip: "ws-host", userId: "u1" });
    expect(mockPost).toHaveBeenCalledWith("/asset/reconciliation/exception-rule/test", {
      ip: "ws-host",
      userId: "u1",
    });
    expect(result).toEqual({
      matchedRules: [],
      mergedActions: [],
      finalSeverity: "",
      isSilence: false,
      needsUserDept: false,
    });
  });
});

describe("reconciliationApi.baseline / refresh", () => {
  beforeEach(() => mockPost.mockReset());

  it("snapshot / compare / refresh", async () => {
    mockPost.mockResolvedValue({ code: 0, data: null });
    await reconciliationApi.baseline.snapshot();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/asset/reconciliation/baseline/snapshot", {});
    const compare = await reconciliationApi.baseline.compare();
    expect(mockPost).toHaveBeenNthCalledWith(2, "/asset/reconciliation/baseline/compare", {});
    expect(compare.reductions).toEqual({});
    const refreshed = await reconciliationApi.refresh();
    expect(mockPost).toHaveBeenNthCalledWith(3, "/asset/reconciliation/refresh", {});
    expect(refreshed).toEqual({
      inserted: 0,
      skipped: 0,
      skippedSilence: 0,
      skippedThrottle: 0,
    });
  });
});

describe("fixSuggestionApi — R5 修复建议状态机", () => {
  beforeEach(() => mockPost.mockReset());

  it("list / getById / stats data 空时回退", async () => {
    mockPost.mockResolvedValue({ code: 0, data: null });
    const params = { current: 1, pageSize: 20, fixStatus: "pending" as const };
    expect(await fixSuggestionApi.list(params)).toEqual({
      list: [],
      total: 0,
      current: 1,
      pageSize: 20,
    });
    expect(mockPost).toHaveBeenNthCalledWith(
      1,
      "/asset/reconciliation/fix-suggestion/list",
      params
    );

    await fixSuggestionApi.getById("f1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/asset/reconciliation/fix-suggestion/f1", {});

    const stats = await fixSuggestionApi.stats(7);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/asset/reconciliation/fix-suggestion/stats", {
      windowDays: 7,
    });
    expect(stats.threshold).toBe(0.01);
    expect(stats.windowDays).toBe(7);
  });

  it("accept / reject / apply / rollback 状态机端点", async () => {
    mockPost.mockResolvedValue({ code: 0, data: { id: "f1" } });
    await fixSuggestionApi.accept("f1");
    expect(mockPost).toHaveBeenNthCalledWith(
      1,
      "/asset/reconciliation/fix-suggestion/f1/accept",
      {}
    );
    await fixSuggestionApi.reject("f1", "建议与实际不符,驳回");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/asset/reconciliation/fix-suggestion/f1/reject", {
      rejectionReason: "建议与实际不符,驳回",
    });
    await fixSuggestionApi.apply("f1");
    expect(mockPost).toHaveBeenNthCalledWith(
      3,
      "/asset/reconciliation/fix-suggestion/f1/apply",
      {}
    );
    await fixSuggestionApi.rollback("f1", "应用后业务异常,回滚");
    expect(mockPost).toHaveBeenNthCalledWith(
      4,
      "/asset/reconciliation/fix-suggestion/f1/rollback",
      {
        rollbackReason: "应用后业务异常,回滚",
      }
    );
  });
});
