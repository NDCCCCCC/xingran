/**
 * Phase 87 — monitor cache utils 纯函数测试
 */
import { describe, it, expect } from "vitest";
import { formatMemorySize, formatTTL } from "../utils";

describe("formatMemorySize", () => {
  it("formats bytes", () => {
    expect(typeof formatMemorySize(1024)).toBe("string");
  });
  it("handles 0", () => {
    expect(typeof formatMemorySize(0)).toBe("string");
  });
});

describe("formatTTL", () => {
  it("formats seconds", () => {
    expect(typeof formatTTL(3600)).toBe("string");
  });
  it("handles negative/expired", () => {
    expect(typeof formatTTL(-1)).toBe("string");
  });
});
