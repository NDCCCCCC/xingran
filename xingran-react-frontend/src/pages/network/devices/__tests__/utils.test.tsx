/**
 * Phase 86 — devices utils 测试
 */
import { describe, it, expect } from "vitest";
import { getOptionLabel, getStatusColor } from "../utils";

describe("getOptionLabel", () => {
  it("finds label for matching value", () => {
    const opts = [
      { value: 1, label: "启用" },
      { value: 0, label: "停用" },
    ];
    expect(getOptionLabel(opts, 1)).toBe("启用");
    expect(getOptionLabel(opts, 0)).toBe("停用");
  });

  it("returns undefined for missing value", () => {
    expect(getOptionLabel([{ value: 1, label: "a" }], 99)).toBeUndefined();
  });

  it("handles string values", () => {
    expect(getOptionLabel([{ value: "k", label: "标签" }], "k")).toBe("标签");
  });
});

describe("getStatusColor", () => {
  it("returns color for status 0/1", () => {
    expect(getStatusColor(0)).toBeTruthy();
    expect(getStatusColor(1)).toBeTruthy();
  });
});
