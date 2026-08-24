/**
 * queryKeys 集中查询键工厂测试 (Phase 83-03)
 *
 * 锁定各 query key 形状,防止组件间 key 漂移:
 * - 根段标识资源族,判别参数依次追加
 * - as const 元组(可断言 readonly 数组结构)
 */
import { describe, expect, it } from "vitest";
import { queryKeys } from "./queryKeys";

describe("queryKeys", () => {
  it("dict 族:all / list(type)", () => {
    expect(queryKeys.dict.all).toEqual(["dict"]);
    expect(queryKeys.dict.list("sys_user_sex")).toEqual(["dict", "sys_user_sex"]);
  });

  it("list 族:all(resource) / page(resource, params)", () => {
    expect(queryKeys.list.all("workstation")).toEqual(["list", "workstation"]);
    const params = { current: 1, pageSize: 10 };
    expect(queryKeys.list.page("workstation", params)).toEqual(["list", "workstation", params]);
  });

  it("dept / duty 族", () => {
    expect(queryKeys.dept.all).toEqual(["dept"]);
    expect(queryKeys.dept.tree()).toEqual(["dept", "tree"]);
    expect(queryKeys.duty.all).toEqual(["duty"]);
    expect(queryKeys.duty.poolMembers("dept-1")).toEqual(["duty", "pool-members", "dept-1"]);
  });

  it("role 族:list 无参时以空对象占位", () => {
    expect(queryKeys.role.all).toEqual(["role"]);
    expect(queryKeys.role.list()).toEqual(["role", "list", {}]);
    const params = { status: 0 };
    expect(queryKeys.role.list(params)).toEqual(["role", "list", params]);
  });

  it("locationAlias 族(Phase 39)", () => {
    expect(queryKeys.locationAlias.all).toEqual(["location-alias"]);
    expect(queryKeys.locationAlias.byLocation("loc-1")).toEqual([
      "location-alias",
      "by-location",
      "loc-1",
    ]);
    expect(queryKeys.locationAlias.list()).toEqual(["location-alias", "list", {}]);
  });

  it("reconciliation 族 — 统计类 key 携带窗口/limit 判别参数", () => {
    expect(queryKeys.reconciliation.all).toEqual(["reconciliation"]);
    expect(queryKeys.reconciliation.summary(7)).toEqual(["reconciliation", "summary", 7]);
    expect(queryKeys.reconciliation.byConflictType(30)).toEqual([
      "reconciliation",
      "by-conflict-type",
      30,
    ]);
    expect(queryKeys.reconciliation.bySeverity(90)).toEqual(["reconciliation", "by-severity", 90]);
    expect(queryKeys.reconciliation.healthTrend(7)).toEqual(["reconciliation", "health-trend", 7]);
    expect(queryKeys.reconciliation.topUnresolved(10)).toEqual([
      "reconciliation",
      "top-unresolved",
      10,
    ]);
    expect(queryKeys.reconciliation.ruleStats()).toEqual(["reconciliation", "rule-stats"]);
    expect(queryKeys.reconciliation.baselineCompare()).toEqual([
      "reconciliation",
      "baseline-compare",
    ]);
  });

  it("reconciliation 族 — 异常/规则/修复建议 key", () => {
    const listParams = { current: 1, pageSize: 20 };
    expect(queryKeys.reconciliation.exceptionList(listParams)).toEqual([
      "reconciliation",
      "exception-list",
      listParams,
    ]);
    expect(queryKeys.reconciliation.exceptionDetail("exc-1")).toEqual([
      "reconciliation",
      "exception-detail",
      "exc-1",
    ]);
    expect(queryKeys.reconciliation.ruleList()).toEqual(["reconciliation", "rule-list", {}]);
    expect(queryKeys.reconciliation.ruleDetail("rule-1")).toEqual([
      "reconciliation",
      "rule-detail",
      "rule-1",
    ]);
    const matchParams = { ip: "ws-host.example.internal", userId: "u1" };
    expect(queryKeys.reconciliation.matchTest(matchParams)).toEqual([
      "reconciliation",
      "match-test",
      matchParams,
    ]);
    expect(queryKeys.reconciliation.workstationHealth("ws-1")).toEqual([
      "reconciliation",
      "workstation-health",
      "ws-1",
    ]);
    expect(queryKeys.reconciliation.assetHealth("asset-1")).toEqual([
      "reconciliation",
      "asset-health",
      "asset-1",
    ]);
    expect(queryKeys.reconciliation.fixSuggestionList(listParams)).toEqual([
      "reconciliation",
      "fix-suggestion-list",
      listParams,
    ]);
    expect(queryKeys.reconciliation.fixSuggestionDetail("fix-1")).toEqual([
      "reconciliation",
      "fix-suggestion-detail",
      "fix-1",
    ]);
    expect(queryKeys.reconciliation.fixSuggestionStats(7)).toEqual([
      "reconciliation",
      "fix-suggestion-stats",
      7,
    ]);
  });

  it("全部 key 为数组元组(as const,长度与判别参数一致)", () => {
    expect(Array.isArray(queryKeys.dict.all)).toBe(true);
    expect(queryKeys.dict.all).toHaveLength(1);
    expect(queryKeys.reconciliation.summary(7)).toHaveLength(3);
    expect(queryKeys.reconciliation.exceptionDetail("x")).toHaveLength(3);
  });
});
