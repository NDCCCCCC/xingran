/**
 * Phase 88 Batch352 — pages/network/discoveries/utils 测试
 */
import { describe, it, expect } from "vitest";
import { parseIPRanges } from "../utils";

describe("pages/network/discoveries/utils", () => {
  it("空字符串 → 空数组", () => {
    expect(parseIPRanges("")).toEqual([]);
  });

  it("单 IP → 单 IP range", () => {
    const r = parseIPRanges("192.168.1.1");
    expect(r.length).toBe(1);
    expect(r[0]).toEqual({ startIP: "192.168.1.1", endIP: "192.168.1.1" });
  });

  it("IP range 格式 (start-end)", () => {
    const r = parseIPRanges("192.168.1.1-192.168.1.100");
    expect(r).toEqual([{ startIP: "192.168.1.1", endIP: "192.168.1.100" }]);
  });

  it("CIDR /24 → 解析为 /24 range", () => {
    const r = parseIPRanges("192.168.1.0/24");
    expect(r[0].startIP).toBe("192.168.1.1");
    expect(r[0].endIP).toBe("192.168.1.254");
  });

  it("CIDR /16 → 解析为 /16 range", () => {
    const r = parseIPRanges("172.16.0.0/16");
    expect(r[0].startIP).toBe("172.16.0.1");
    expect(r[0].endIP).toBe("172.16.255.254");
  });

  it("CIDR /8 → 单 IP 范围", () => {
    const r = parseIPRanges("10.0.0.0/8");
    expect(r[0].startIP).toBe("10.0.0.0");
    expect(r[0].endIP).toBe("10.0.0.0");
  });

  it("CIDR /25 仍 >= 24 → /24 处理", () => {
    const r = parseIPRanges("192.168.1.0/25");
    expect(r[0].startIP).toBe("192.168.1.1");
    expect(r[0].endIP).toBe("192.168.1.254");
  });

  it("CIDR /20 16 ≤ prefix < 24 → /16 处理", () => {
    const r = parseIPRanges("172.16.0.0/20");
    expect(r[0].startIP).toBe("172.16.0.1");
    expect(r[0].endIP).toBe("172.16.255.254");
  });

  it("多行混合 → 多个 ranges", () => {
    const r = parseIPRanges(`192.168.1.1
192.168.2.0/24
10.1.1.1-10.1.1.10`);
    expect(r.length).toBe(3);
  });

  it("跳过空行/空白行", () => {
    const r = parseIPRanges(`192.168.1.1

   `);
    expect(r.length).toBe(1);
  });
});
