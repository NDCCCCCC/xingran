/**
 * Phase 88 Batch222 — types/duty 值班类型
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type { DutyType, TodayDutyRecord, MyDutyStats } from "../duty";

describe("types/duty", () => {
  it("DutyType 3 类别", () => {
    const t: DutyType[] = ["weekday", "weekend", "holiday"];
    expect(t.length).toBe(3);
  });

  it("TodayDutyRecord shape", () => {
    const r: TodayDutyRecord = { poolName: "P1", dutyType: "weekday" };
    expect(r.poolName).toBe("P1");
    expect(r.dutyType).toBe("weekday");
  });

  it("MyDutyStats shape 含 nextDutyDate/todayDutyRecords", () => {
    const s: MyDutyStats = {
      isOnDutyToday: true,
      thisMonthCount: 5,
      totalCount: 100,
      nextDutyDate: "2026-09-01",
      nextDutyPoolName: "Main",
      todayDutyRecords: [{ poolName: "Main", dutyType: "weekday" }],
    };
    expect(s.isOnDutyToday).toBe(true);
    expect(s.todayDutyRecords?.length).toBe(1);
  });

  it("MyDutyStats 可选字段缺失", () => {
    const s: MyDutyStats = {
      isOnDutyToday: false,
      thisMonthCount: 0,
      totalCount: 0,
    };
    expect(s.nextDutyDate).toBeUndefined();
    expect(s.todayDutyRecords).toBeUndefined();
  });
});
