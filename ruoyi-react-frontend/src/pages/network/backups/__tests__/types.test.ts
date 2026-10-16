/**
 * Phase 88 Batch275 — pages/network/backups/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { DiffLine, DiffResult, DeviceBackupGroup, BackupStatistics } from "../types";

describe("network/backups/types", () => {
  it("DiffLine 4 类型", () => {
    const types: DiffLine["type"][] = ["same", "removed", "added", "empty"];
    expect(types.length).toBe(4);
  });

  it("DiffLine shape", () => {
    const l: DiffLine = { type: "added", content: "new line", lineNum: 1 };
    expect(l.type).toBe("added");
  });

  it("DiffResult shape", () => {
    const r: DiffResult = {
      leftContent: "a",
      rightContent: "b",
      leftLines: [{ type: "removed", content: "a" }],
      rightLines: [{ type: "added", content: "b" }],
      oldVersion: "v1",
      newVersion: "v2",
    };
    expect(r.oldVersion).toBe("v1");
  });

  it("DeviceBackupGroup shape", () => {
    const g: DeviceBackupGroup = {
      deviceId: "d1",
      deviceName: "D1",
      ipAddress: "10.0.0.1",
      backups: [],
      latestBackup: {} as any,
      backupCount: 5,
      autoCount: 3,
      manualCount: 2,
    };
    expect(g.backupCount).toBe(5);
  });

  it("BackupStatistics shape", () => {
    const s: BackupStatistics = {
      total: 100,
      auto: 70,
      manual: 30,
      devices: 10,
    };
    expect(s.total).toBe(100);
  });
});
