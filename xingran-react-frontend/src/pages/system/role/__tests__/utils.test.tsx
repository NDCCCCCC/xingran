/**
 * Phase 86 — role utils/constants 测试
 */
import { describe, it, expect } from "vitest";
import { processTreeData, formatLocalTime } from "../utils";
import { DATA_SCOPE_OPTIONS, STATUS_OPTIONS, DEFAULT_FORM_VALUES } from "../constants";

const menuTree: any[] = [
  {
    id: 1,
    menuName: "系统管理",
    menuType: "M",
    children: [{ id: 11, menuName: "用户", menuType: "C" }],
  },
  { id: 2, menuName: "监控", menuType: "M" },
];

describe("processTreeData", () => {
  it("processes menu tree without error", () => {
    const result = processTreeData(menuTree);
    expect(Array.isArray(result)).toBe(true);
  });

  it("handles empty input", () => {
    const result = processTreeData([]);
    expect(result).toEqual([]);
  });
});

describe("formatLocalTime", () => {
  it("formats ISO time string", () => {
    const r = formatLocalTime("2026-08-27T10:00:00Z");
    expect(typeof r).toBe("string");
    expect(r).toBeTruthy();
  });
});

describe("role constants (D-12)", () => {
  it("DATA_SCOPE_OPTIONS non-empty", () => {
    expect(DATA_SCOPE_OPTIONS.length).toBeGreaterThan(0);
  });

  it("STATUS_OPTIONS is 0/1 启停", () => {
    expect(STATUS_OPTIONS.length).toBe(2);
  });

  it("DEFAULT_FORM_VALUES defined", () => {
    expect(DEFAULT_FORM_VALUES).toBeDefined();
  });
});
