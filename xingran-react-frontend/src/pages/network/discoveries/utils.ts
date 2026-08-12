/**
 * Device Discovery Utils
 * 设备发现工具函数
 */

import type { IPRange } from "./types";

/** 解析IP范围字符串为IPRange数组 */
export function parseIPRanges(text: string): IPRange[] {
  const ranges: IPRange[] = [];
  const lines = text.split("\n").filter(line => line.trim());

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.includes("-")) {
      // 格式: 192.168.1.1-192.168.1.100
      const [start, end] = trimmed.split("-").map(s => s.trim());
      ranges.push({ startIP: start, endIP: end });
    } else if (trimmed.includes("/")) {
      // 格式: 192.168.1.0/24
      const [ip, cidr] = trimmed.split("/");
      const prefix = parseInt(cidr);
      const parts = ip.split(".").map(Number);

      if (prefix >= 24) {
        const startIP = `${parts[0]}.${parts[1]}.${parts[2]}.1`;
        const endIP = `${parts[0]}.${parts[1]}.${parts[2]}.254`;
        ranges.push({ startIP, endIP });
      } else if (prefix >= 16) {
        const startIP = `${parts[0]}.${parts[1]}.0.1`;
        const endIP = `${parts[0]}.${parts[1]}.255.254`;
        ranges.push({ startIP, endIP });
      } else {
        ranges.push({ startIP: ip, endIP: ip });
      }
    } else {
      ranges.push({ startIP: trimmed, endIP: trimmed });
    }
  }

  return ranges;
}
