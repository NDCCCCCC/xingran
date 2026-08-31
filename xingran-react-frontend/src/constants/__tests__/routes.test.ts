/**
 * Phase 88 Batch283 — constants/routes 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import * as routes from "../routes";

describe("constants/routes", () => {
  it("用户中心路由 3 项", () => {
    expect(routes.USER_PROFILE).toBe("/user/profile");
    expect(routes.USER_SETTINGS).toBe("/user/settings");
    expect(routes.USER_NOTICES).toBe("/user/my-notices");
  });

  it("主要页面 3 项", () => {
    expect(routes.DASHBOARD).toBe("/dashboard");
    expect(routes.MONITOR_DASHBOARD).toBe("/monitor/dashboard");
    expect(routes.LOGIN).toBe("/login");
  });

  it("系统管理 6 项", () => {
    expect(routes.SYSTEM_USER).toBe("/system/user");
    expect(routes.SYSTEM_ROLE).toBe("/system/role");
    expect(routes.SYSTEM_MENU).toBe("/system/menu");
    expect(routes.SYSTEM_DEPT).toBe("/system/dept");
    expect(routes.SYSTEM_DICT).toBe("/system/dict");
    expect(routes.SYSTEM_NOTICE).toBe("/system/notice");
  });

  it("网络设备 3 项", () => {
    expect(routes.NETWORK_DEVICES).toBe("/network/devices");
    expect(routes.NETWORK_PORTS).toBe("/network/ports");
    expect(routes.NETWORK_DISCOVERIES).toBe("/network/discoveries");
  });

  it("工单 3 项", () => {
    expect(routes.WORKORDER_ORDERS).toBe("/workorder/orders");
    expect(routes.WORKORDER_CATEGORIES).toBe("/workorder/categories");
    expect(routes.WORKORDER_STATISTICS).toBe("/workorder/statistics");
  });

  it("运维 + 值班 2 项", () => {
    expect(routes.OPS_BUILDINGS).toBe("/ops/operations/buildings");
    expect(routes.DUTY_MY_DUTY).toBe("/duty/my-duty");
  });
});
