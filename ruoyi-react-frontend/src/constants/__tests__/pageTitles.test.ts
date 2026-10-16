/**
 * Phase 88 Batch305 — constants/pageTitles 测试
 */
import { describe, it, expect } from "vitest";
import {
  PAGE_TITLES,
  SPECIAL_PATH_TITLES,
  DYNAMIC_ROUTE_PATTERNS,
  matchDynamicRouteTitle,
  getSpecialPathTitle,
} from "../pageTitles";

describe("constants/pageTitles", () => {
  it("PAGE_TITLES 包含 4 项", () => {
    expect(Object.keys(PAGE_TITLES).length).toBe(4);
  });

  it("PAGE_TITLES 值正确", () => {
    expect(PAGE_TITLES.DASHBOARD).toBe("仪表盘");
    expect(PAGE_TITLES.MONITOR_DASHBOARD).toBe("监控仪表盘");
    expect(PAGE_TITLES.HOME).toBe("首页");
    expect(PAGE_TITLES.NOTICE_DETAIL).toBe("通知详情");
  });

  it("SPECIAL_PATH_TITLES 三个特殊路径", () => {
    expect(SPECIAL_PATH_TITLES["/"]).toBe("首页");
    expect(SPECIAL_PATH_TITLES["/dashboard"]).toBe("仪表盘");
    expect(SPECIAL_PATH_TITLES["/monitor/dashboard"]).toBe("监控仪表盘");
  });

  it("DYNAMIC_ROUTE_PATTERNS 含 notice 正则", () => {
    expect(DYNAMIC_ROUTE_PATTERNS.length).toBe(1);
    expect(DYNAMIC_ROUTE_PATTERNS[0].title).toBe("通知详情");
  });

  it("matchDynamicRouteTitle 命中 UUID 通知详情", () => {
    expect(matchDynamicRouteTitle("/my-notices/abc-123-def")).toBe("通知详情");
  });

  it("matchDynamicRouteTitle 命中纯 hex UUID", () => {
    expect(matchDynamicRouteTitle("/my-notices/deadbeef-1234-5678-9012-abcdef012345")).toBe(
      "通知详情"
    );
  });

  it("matchDynamicRouteTitle 不命中 → null", () => {
    expect(matchDynamicRouteTitle("/dashboard")).toBeNull();
  });

  it("matchDynamicRouteTitle 空字符串 → null", () => {
    expect(matchDynamicRouteTitle("")).toBeNull();
  });

  it("matchDynamicRouteTitle 不带 UUID 段 → null", () => {
    expect(matchDynamicRouteTitle("/my-notices/")).toBeNull();
  });

  it("matchDynamicRouteTitle 大写 UUID 仍命中", () => {
    expect(matchDynamicRouteTitle("/my-notices/ABCDEF12-3456-7890-ABCD-EF1234567890")).toBe(
      "通知详情"
    );
  });

  it("getSpecialPathTitle 命中", () => {
    expect(getSpecialPathTitle("/")).toBe("首页");
    expect(getSpecialPathTitle("/dashboard")).toBe("仪表盘");
  });

  it("getSpecialPathTitle 不命中 → null", () => {
    expect(getSpecialPathTitle("/users")).toBeNull();
    expect(getSpecialPathTitle("")).toBeNull();
  });

  it("getSpecialPathTitle undefined-key → null", () => {
    expect(getSpecialPathTitle("/not-a-special-path")).toBeNull();
  });
});
