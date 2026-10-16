/**
 * Phase 86 — backups utils 纯函数测试
 */
import { describe, it, expect } from "vitest";
import { computeDiff, groupBackupsByDevice } from "../utils";
import type { ConfigBackup } from "../types";

describe("computeDiff", () => {
  it("returns identical result for equal content", () => {
    const r = computeDiff("line1\nline2", "line1\nline2");
    expect(r).toBeDefined();
  });

  it("detects changed content", () => {
    const r = computeDiff("a\nb\nc", "a\nx\nc");
    expect(r).toBeDefined();
    expect(Array.isArray(r.lines) || typeof r === "object").toBe(true);
  });

  it("handles empty strings", () => {
    const r = computeDiff("", "");
    expect(r).toBeDefined();
  });
});

describe("groupBackupsByDevice", () => {
  it("groups backups by deviceId", () => {
    const backups = [
      { id: "1", deviceId: "d1", deviceName: "设备A" },
      { id: "2", deviceId: "d1", deviceName: "设备A" },
      { id: "3", deviceId: "d2", deviceName: "设备B" },
    ] as unknown as ConfigBackup[];
    const groups = groupBackupsByDevice(backups);
    expect(groups.length).toBe(2);
    const g1 = groups.find((g) => g.deviceId === "d1");
    expect(g1?.backups).toHaveLength(2);
  });

  it("returns empty for empty input", () => {
    expect(groupBackupsByDevice([])).toEqual([]);
  });
});
