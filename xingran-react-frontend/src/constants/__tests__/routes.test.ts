/**
 * Phase 88 Batch360 — constants/routes 测试
 */
import { describe, it, expect } from "vitest";
import * as routes from "../routes";

describe("constants/routes", () => {
  describe("用户中心路由", () => {
    it("USER_PROFILE", () => {
      expect(routes.USER_PROFILE).toBe("/user/profile");
    });

    it("USER_SETTINGS", () => {
      expect(routes.USER_SETTINGS).toBe("/user/settings");
    });

    it("USER_NOTICES", () => {
      expect(routes.USER_NOTICES).toBe("/user/my-notices");
    });
  });

  describe("主要页面路由", () => {
    it("DASHBOARD", () => {
      expect(routes.DASHBOARD).toBe("/dashboard");
    });

    it("MONITOR_DASHBOARD", () => {
      expect(routes.MONITOR_DASHBOARD).toBe("/monitor/dashboard");
    });

    it("LOGIN", () => {
      expect(routes.LOGIN).toBe("/login");
    });
  });

  describe("系统管理路由", () => {
    it("SYSTEM_USER", () => {
      expect(routes.SYSTEM_USER).toBe("/system/user");
    });

    it("SYSTEM_ROLE", () => {
      expect(routes.SYSTEM_ROLE).toBe("/system/role");
    });

    it("SYSTEM_MENU", () => {
      expect(routes.SYSTEM_MENU).toBe("/system/menu");
    });

    it("SYSTEM_DEPT", () => {
      expect(routes.SYSTEM_DEPT).toBe("/system/dept");
    });

    it("SYSTEM_DICT", () => {
      expect(routes.SYSTEM_DICT).toBe("/system/dict");
    });

    it("SYSTEM_NOTICE", () => {
      expect(routes.SYSTEM_NOTICE).toBe("/system/notice");
    });
  });

  describe("网络设备路由", () => {
    it("NETWORK_DEVICES", () => {
      expect(routes.NETWORK_DEVICES).toBe("/network/devices");
    });

    it("NETWORK_PORTS", () => {
      expect(routes.NETWORK_PORTS).toBe("/network/ports");
    });

    it("NETWORK_DISCOVERIES", () => {
      expect(routes.NETWORK_DISCOVERIES).toBe("/network/discoveries");
    });
  });

  describe("工单/运维/值班路由", () => {
    it("WORKORDER_ORDERS", () => {
      expect(routes.WORKORDER_ORDERS).toBe("/workorder/orders");
    });

    it("WORKORDER_CATEGORIES", () => {
      expect(routes.WORKORDER_CATEGORIES).toBe("/workorder/categories");
    });

    it("WORKORDER_STATISTICS", () => {
      expect(routes.WORKORDER_STATISTICS).toBe("/workorder/statistics");
    });

    it("OPS_BUILDINGS", () => {
      expect(routes.OPS_BUILDINGS).toBe("/ops/operations/buildings");
    });

    it("DUTY_MY_DUTY", () => {
      expect(routes.DUTY_MY_DUTY).toBe("/duty/my-duty");
    });
  });

  it("所有路由以 / 开头", () => {
    for (const key of Object.keys(routes)) {
      expect(routes[key as keyof typeof routes]).toMatch(/^\//);
    }
  });
});
