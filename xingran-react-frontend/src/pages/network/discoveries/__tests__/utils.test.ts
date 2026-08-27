/* eslint-disable no-restricted-syntax -- 测试向量 IP 字面量,非运行时配置 */
/**
 * Phase 86 — discoveries utils 纯函数测试
 */
import { describe, it, expect } from "vitest";
import { parseIPRanges } from "../utils";
import { DISCOVERY_TYPE_OPTIONS, STATUS_OPTIONS, STATUS_CONFIG } from "../constants";

describe("parseIPRanges", () => {
  it("parses explicit range with dash", () => {
    const r = parseIPRanges("192.168.1.1-192.168.1.100");
    expect(r).toEqual([{ startIP: "192.168.1.1", endIP: "192.168.1.100" }]);
  });

  it("parses /24 CIDR into .1-.254", () => {
    const r = parseIPRanges("192.168.1.0/24");
    expect(r[0].startIP).toBe("192.168.1.1");
    expect(r[0].endIP).toBe("192.168.1.254");
  });

  it("parses /16 CIDR into x.x.0.1-x.x.255.254", () => {
    const r = parseIPRanges("10.0.0.0/16");
    expect(r[0].startIP).toBe("10.0.0.1");
    expect(r[0].endIP).toBe("10.0.255.254");
  });

  it("parses /8 CIDR as single IP fallback", () => {
    const r = parseIPRanges("10.0.0.0/8");
    expect(r[0].startIP).toBe("10.0.0.0");
  });

  it("parses single IP", () => {
    const r = parseIPRanges("192.168.1.5");
    expect(r).toEqual([{ startIP: "192.168.1.5", endIP: "192.168.1.5" }]);
  });

  it("handles multi-line input with blank lines", () => {
    const r = parseIPRanges("192.168.1.1\n\n10.0.0.1-10.0.0.5\n");
    expect(r).toHaveLength(2);
  });

  it("returns empty for empty string", () => {
    expect(parseIPRanges("")).toEqual([]);
  });
});

describe("discoveries constants (D-12)", () => {
  it("DISCOVERY_TYPE_OPTIONS non-empty", () => {
    expect(DISCOVERY_TYPE_OPTIONS.length).toBeGreaterThan(0);
  });

  it("STATUS_OPTIONS non-empty with STATUS_CONFIG mapping", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThan(0);
    const keys = Object.keys(STATUS_CONFIG);
    expect(keys.length).toBeGreaterThan(0);
  });
});
