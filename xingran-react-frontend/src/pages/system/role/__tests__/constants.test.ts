/**
 * Phase 88 Batch197b — pages/system/role/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { DATA_SCOPE_OPTIONS, STATUS_OPTIONS, DEFAULT_FORM_VALUES } from "../constants";

describe("system/role/constants", () => {
  it("DATA_SCOPE_OPTIONS 5 项", () => {
    expect(DATA_SCOPE_OPTIONS.length).toBe(5);
    expect(DATA_SCOPE_OPTIONS[0].value).toBe(1);
    expect(DATA_SCOPE_OPTIONS[4].label).toBe("仅本人数据");
  });

  it("STATUS_OPTIONS 至少 2 项", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("DEFAULT_FORM_VALUES 默认", () => {
    expect(DEFAULT_FORM_VALUES.dataScope).toBe(1);
    expect(DEFAULT_FORM_VALUES.status).toBe(0);
    expect(DEFAULT_FORM_VALUES.roleSort).toBe(0);
    expect(DEFAULT_FORM_VALUES.menuCheckStrictly).toBe(true);
    expect(DEFAULT_FORM_VALUES.deptCheckStrictly).toBe(true);
  });
});
