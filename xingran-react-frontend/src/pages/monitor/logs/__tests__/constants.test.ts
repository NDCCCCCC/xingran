/**
 * Phase 88 Batch262 — pages/monitor/logs/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { BusinessType } from "../types";
import {
  BUSINESS_TYPE_OPTIONS,
  LOG_STATUS_OPTIONS,
  LOGIN_STATUS_OPTIONS,
  DEFAULT_OPER_SEARCH_FORM,
  DEFAULT_LOGIN_SEARCH_FORM,
} from "../constants";

describe("monitor/logs/constants", () => {
  it("BUSINESS_TYPE_OPTIONS 10 项", () => {
    expect(BUSINESS_TYPE_OPTIONS.length).toBe(10);
    expect(BUSINESS_TYPE_OPTIONS[0]).toEqual({ label: "其它", value: BusinessType.Other });
    expect(BUSINESS_TYPE_OPTIONS[1]).toEqual({ label: "新增", value: BusinessType.Create });
  });

  it("LOG_STATUS_OPTIONS 2 项", () => {
    expect(LOG_STATUS_OPTIONS.length).toBe(2);
  });

  it("LOGIN_STATUS_OPTIONS 2 项", () => {
    expect(LOGIN_STATUS_OPTIONS.length).toBe(2);
  });

  it("DEFAULT_OPER_SEARCH_FORM 默认空", () => {
    expect(DEFAULT_OPER_SEARCH_FORM.title).toBe("");
    expect(DEFAULT_OPER_SEARCH_FORM.businessType).toBeUndefined();
    expect(DEFAULT_OPER_SEARCH_FORM.status).toBeUndefined();
    expect(DEFAULT_OPER_SEARCH_FORM.operName).toBe("");
  });

  it("DEFAULT_LOGIN_SEARCH_FORM 默认空", () => {
    expect(DEFAULT_LOGIN_SEARCH_FORM.userName).toBe("");
    expect(DEFAULT_LOGIN_SEARCH_FORM.ipAddr).toBe("");
    expect(DEFAULT_LOGIN_SEARCH_FORM.status).toBeUndefined();
  });
});
