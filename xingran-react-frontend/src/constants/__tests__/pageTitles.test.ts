/**
 * Phase 88 Batch290 — constants/pageTitles 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  PAGE_TITLES,
  SPECIAL_PATH_TITLES,
  DYNAMIC_ROUTE_PATTERNS,
  matchDynamicRouteTitle,
  getSpecialPathTitle,
} from "../pageTitles";

describe("constants/pageTitles", () => {
  it("PAGE_TITLES 4 项", () => {
    expect(PAGE_TITLES.DASHBOARD).toBe("仪表盘");
    expect(PAGE_TITLES.MONITOR_DASHBOARD).toBe("监控仪表盘");
    expect(PAGE_TITLES.HOME).toBe("首页");
    expect(PAGE_TITLES.NOTICE_DETAIL).toBe("通知详情");
  });

  it("SPECIAL_PATH_TITLES 3 路径", () => {
    expect(SPECIAL_PATH_TITLES["/"]).toBe("首页");
    expect(SPECIAL_PATH_TITLES["/dashboard"]).toBe("仪表盘");
    expect(SPECIAL_PATH_TITLES["/monitor/dashboard"]).toBe("监控仪表盘");
  });

  it("DYNAMIC_ROUTE_PATTERNS 1 项", () => {
    expect(DYNAMIC_ROUTE_PATTERNS.length).toBe(1);
  });

  it("matchDynamicRouteTitle 匹配 my-notices UUID", () => {
    expect(matchDynamicRouteTitle("/my-notices/abc-123")).toBe("通知详情");
  });

  it("matchDynamicRouteTitle 不匹配", () => {
    expect(matchDynamicRouteTitle("/other")).toBeNull();
  });

  it("getSpecialPathTitle 匹配", () => {
    expect(getSpecialPathTitle("/")).toBe("首页");
    expect(getSpecialPathTitle("/dashboard")).toBe("仪表盘");
  });

  it("getSpecialPathTitle 不匹配", () => {
    expect(getSpecialPathTitle("/other")).toBeNull();
  });
});
