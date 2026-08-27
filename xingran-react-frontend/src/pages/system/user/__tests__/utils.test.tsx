/**
 * Phase 86 — user utils/constants 测试
 */
import { describe, it, expect } from "vitest";
import { formatGender, formatStatus } from "../utils";
import { GENDER_OPTIONS, STATUS_OPTIONS, STATUS_TAG_CONFIG } from "../constants";

describe("formatGender", () => {
  it("formats gender 0/1/2 with dict", () => {
    const dict: any[] = [
      { dictValue: "0", dictLabel: "男" },
      { dictValue: "1", dictLabel: "女" },
      { dictValue: "2", dictLabel: "未知" },
    ];
    expect(formatGender(0, dict)).toBe("男");
    expect(formatGender(1, dict)).toBe("女");
  });

  it("falls back without dict", () => {
    expect(typeof formatGender(0)).toBe("string");
  });
});

describe("formatStatus", () => {
  it("returns text and color for status 0/1", () => {
    const s0 = formatStatus(0);
    const s1 = formatStatus(1);
    expect(s0.text).toBeTruthy();
    expect(s0.color).toBeTruthy();
    expect(s1.text).toBeTruthy();
  });
});

describe("user constants (D-12)", () => {
  it("GENDER_OPTIONS non-empty", () => {
    expect(GENDER_OPTIONS.length).toBeGreaterThan(0);
  });

  it("STATUS_OPTIONS is 启停 2 项", () => {
    expect(STATUS_OPTIONS.length).toBe(2);
  });

  it("STATUS_TAG_CONFIG covers 0/1", () => {
    expect(STATUS_TAG_CONFIG[0]).toBeDefined();
    expect(STATUS_TAG_CONFIG[1]).toBeDefined();
  });
});
