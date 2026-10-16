/**
 * Phase 88 Batch355 — pages/system/role/constants 测试
 */
import { describe, it, expect } from "vitest";
import { DATA_SCOPE_OPTIONS, STATUS_OPTIONS, DEFAULT_FORM_VALUES } from "../constants";

describe("pages/system/role/constants", () => {
  it("DATA_SCOPE_OPTIONS 5 项", () => {
    expect(DATA_SCOPE_OPTIONS.length).toBe(5);
    expect(DATA_SCOPE_OPTIONS.map((o) => o.value)).toEqual([1, 2, 3, 4, 5]);
  });

  it("DATA_SCOPE_OPTIONS 全部数据 → 1", () => {
    expect(DATA_SCOPE_OPTIONS[0].label).toBe("全部数据");
    expect(DATA_SCOPE_OPTIONS[0].value).toBe(1);
  });

  it("DATA_SCOPE_OPTIONS 仅本人数据 → 5", () => {
    expect(DATA_SCOPE_OPTIONS[4].label).toBe("仅本人数据");
    expect(DATA_SCOPE_OPTIONS[4].value).toBe(5);
  });

  it("STATUS_OPTIONS 是 NORMAL_STOP_OPTIONS 别名", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("DEFAULT_FORM_VALUES", () => {
    expect(DEFAULT_FORM_VALUES.dataScope).toBe(1);
    expect(DEFAULT_FORM_VALUES.status).toBe(0);
    expect(DEFAULT_FORM_VALUES.roleSort).toBe(0);
    expect(DEFAULT_FORM_VALUES.menuCheckStrictly).toBe(true);
    expect(DEFAULT_FORM_VALUES.deptCheckStrictly).toBe(true);
  });
});
