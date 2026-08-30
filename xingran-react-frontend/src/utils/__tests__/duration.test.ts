/**
 * Phase 88 Batch172 — utils/duration 测试
 */
import { describe, it, expect } from "vitest";
import { formatDurationSeconds } from "../duration";

describe("formatDurationSeconds", () => {
  it("0 → '0s'", () => {
    expect(formatDurationSeconds(0)).toBe("0s");
  });

  it("负数 → '0s'", () => {
    expect(formatDurationSeconds(-10)).toBe("0s");
  });

  it("null → '0s'", () => {
    expect(formatDurationSeconds(null)).toBe("0s");
  });

  it("undefined → '0s'", () => {
    expect(formatDurationSeconds(undefined)).toBe("0s");
  });

  it("NaN → '0s'", () => {
    expect(formatDurationSeconds(NaN)).toBe("0s");
  });

  it("Infinity → '0s'", () => {
    expect(formatDurationSeconds(Infinity)).toBe("0s");
  });

  it("45 < 60s → '45s'", () => {
    expect(formatDurationSeconds(45)).toBe("45s");
  });

  it("60 → '1m'", () => {
    expect(formatDurationSeconds(60)).toBe("1m");
  });

  it("900 (15m) → '15m'", () => {
    expect(formatDurationSeconds(900)).toBe("15m");
  });

  it("3600 (1h) → '1h'", () => {
    expect(formatDurationSeconds(3600)).toBe("1h");
  });

  it("9000 → '2h' (9000/3600=2)", () => {
    expect(formatDurationSeconds(9000)).toBe("2h");
  });

  it("86400 (1d) → '1d 0h'", () => {
    expect(formatDurationSeconds(86400)).toBe("1d 0h");
  });

  it("151200 (1d 18h) → '1d 18h'", () => {
    expect(formatDurationSeconds(151200)).toBe("1d 18h");
  });
});
