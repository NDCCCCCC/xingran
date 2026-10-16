/**
 * Phase 88 Batch292 — pages/monitor/logs/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { BusinessType, LogStatus } from "../types";
import type { OperLog, LoginLog, SearchFormState } from "../types";

describe("monitor/logs/types", () => {
  it("OperLog shape", () => {
    const l: OperLog = {
      id: "1",
      title: "用户登录",
      businessType: BusinessType.Other,
      method: "POST",
      requestMethod: "POST",
      operName: "admin",
      deptName: "IT",
      operUrl: "/login",
      operIp: "127.0.0.1",
      operLocation: "北京",
      operParam: "{}",
      operTime: "2026-01-01",
      status: LogStatus.Success,
      errorMessage: "",
    };
    expect(l.operName).toBe("admin");
  });

  it("LoginLog shape", () => {
    const l: LoginLog = {
      id: "1",
      userName: "admin",
      ipAddr: "127.0.0.1",
      loginLocation: "北京",
      browser: "Chrome",
      os: "Linux",
      status: LogStatus.Success,
      message: "ok",
      loginTime: "2026-01-01",
    };
    expect(l.userName).toBe("admin");
  });

  it("BusinessType 10 类别", () => {
    // 枚举: 10 个值, 但 Object.keys 因为反向映射会得到 20
    const numericValues = Object.values(BusinessType).filter((v) => typeof v === "number");
    expect(numericValues.length).toBe(10);
  });

  it("LogStatus 2 值", () => {
    expect(LogStatus.Success).toBe(0);
    expect(LogStatus.Failure).toBe(1);
  });

  it("SearchFormState shape", () => {
    const s: SearchFormState = {
      title: "登录",
      timeRange: null as any,
    };
    expect(s.title).toBe("登录");
  });
});
