/**
 * Phase 88 Batch264 — pages/network/discoveries/utils 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { parseIPRanges } from "../utils";

describe("network/discoveries/utils", () => {
  it("空字符串 → []", () => {
    expect(parseIPRanges("")).toEqual([]);
  });

  it("单 IP", () => {
    const r = parseIPRanges("10.0.0.1");
    expect(r).toEqual([{ startIP: "10.0.0.1", endIP: "10.0.0.1" }]);
  });

  it("IP 范围 (start-end)", () => {
    const r = parseIPRanges("192.168.1.1-192.168.1.100");
    expect(r).toEqual([{ startIP: "192.168.1.1", endIP: "192.168.1.100" }]);
  });

  it("CIDR /24 → 1-254", () => {
    const r = parseIPRanges("192.168.1.0/24");
    expect(r).toEqual([{ startIP: "192.168.1.1", endIP: "192.168.1.254" }]);
  });

  it("CIDR /16 → x.x.0.1 - x.x.255.254", () => {
    const r = parseIPRanges("10.0.0.0/16");
    expect(r).toEqual([{ startIP: "10.0.0.1", endIP: "10.0.255.254" }]);
  });

  it("CIDR /8 → same start/end IP", () => {
    const r = parseIPRanges("10.0.0.0/8");
    expect(r).toEqual([{ startIP: "10.0.0.0", endIP: "10.0.0.0" }]);
  });

  it("多行混合", () => {
    const r = parseIPRanges("10.0.0.1\n192.168.1.0/24\n172.16.0.1-172.16.0.10");
    expect(r.length).toBe(3);
  });

  it("空行 过滤", () => {
    const r = parseIPRanges("10.0.0.1\n\n\n10.0.0.2");
    expect(r.length).toBe(2);
  });
});
