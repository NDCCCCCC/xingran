/**
 * dutyApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:值班池 / 排班 / 节假日 / 值班配置 / 用户部门下拉 各端点 URL 与参数形状。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));
vi.mock("./api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));

import {
  batchCreateHolidays,
  batchDeleteDutySchedules,
  createDutyPool,
  createHoliday,
  deleteDutyPool,
  deleteDutySchedule,
  deleteHoliday,
  generateSchedule,
  getDeptList,
  getDeptTree,
  getDutyConfig,
  getDutyPool,
  getDutyPoolList,
  getDutyPoolStatistics,
  getDutyScheduleList,
  getHolidayList,
  getHolidayYears,
  getMonthlyDutySchedule,
  getMyDutyStats,
  getTodayDuty,
  getUserList,
  manualDuty,
  swapDuty,
  updateDutyConfig,
  updateDutyPool,
  updateHoliday,
} from "./dutyApi";

const OK = { code: 0 };

describe("dutyApi — 值班池", () => {
  beforeEach(() => mockPost.mockReset());

  it("getDutyPoolList POST /duty/pools/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { current: 1, pageSize: 10, poolName: "池" };
    await getDutyPoolList(params);
    expect(mockPost).toHaveBeenCalledWith("/duty/pools/list", params);
  });

  it("getDutyPoolStatistics POST /duty/pools/statistics", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { total: 0 } });
    await getDutyPoolStatistics();
    expect(mockPost).toHaveBeenCalledWith("/duty/pools/statistics", {});
  });

  it("createDutyPool POST /duty/pools", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const data = { poolName: "一线池", dailyCount: 1, memberIds: ["u1"] };
    await createDutyPool(data);
    expect(mockPost).toHaveBeenCalledWith("/duty/pools", data);
  });

  it("getDutyPool / updateDutyPool / deleteDutyPool 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    await getDutyPool("p1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/duty/pools/p1", {});
    await updateDutyPool("p1", { poolName: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/duty/pools/p1/update", { poolName: "改名" });
    await deleteDutyPool("p1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/duty/pools/p1/delete");
  });
});

describe("dutyApi — 排班", () => {
  beforeEach(() => mockPost.mockReset());

  it("getDutyScheduleList POST /duty/schedules/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { current: 1, pageSize: 20, startDate: "2026-08-01" };
    await getDutyScheduleList(params);
    expect(mockPost).toHaveBeenCalledWith("/duty/schedules/list", params);
  });

  it("generateSchedule POST /duty/schedules/generate", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const data = {
      poolId: "p1",
      startDate: "2026-09-01",
      endDate: "2026-09-30",
      dutyType: "weekday" as const,
    };
    await generateSchedule(data);
    expect(mockPost).toHaveBeenCalledWith("/duty/schedules/generate", data);
  });

  it("getTodayDuty / swapDuty / manualDuty / deleteDutySchedule", async () => {
    mockPost.mockResolvedValue(OK);
    await getTodayDuty();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/duty/schedules/today", {});
    await swapDuty({ fromScheduleId: "s1", toScheduleId: "s2" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/duty/schedules/swap", {
      fromScheduleId: "s1",
      toScheduleId: "s2",
    });
    await manualDuty({
      poolId: "p1",
      dutyDate: "2026-09-01",
      dutyType: "weekend",
      userIds: ["u1"],
    });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/duty/schedules/manual", {
      poolId: "p1",
      dutyDate: "2026-09-01",
      dutyType: "weekend",
      userIds: ["u1"],
    });
    await deleteDutySchedule("s1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/duty/schedules/s1/delete");
  });

  it("batchDeleteDutySchedules POST /duty/schedules/batch-delete {ids}", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await batchDeleteDutySchedules(["s1", "s2"]);
    expect(mockPost).toHaveBeenCalledWith("/duty/schedules/batch-delete", { ids: ["s1", "s2"] });
  });

  it("getMonthlyDutySchedule POST /duty/schedules/monthly {year,month}", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await getMonthlyDutySchedule(2026, 9);
    expect(mockPost).toHaveBeenCalledWith("/duty/schedules/monthly", { year: 2026, month: 9 });
  });
});

describe("dutyApi — 节假日与配置", () => {
  beforeEach(() => mockPost.mockReset());

  it("getHolidayList / createHoliday / updateHoliday / deleteHoliday / batch / years", async () => {
    mockPost.mockResolvedValue(OK);
    await getHolidayList(2026);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/duty/holidays/list", { year: 2026 });
    const holiday = {
      holidayDate: "2026-10-01",
      holidayName: "国庆",
      isOffday: true,
      holidayType: "legal" as const,
      year: 2026,
    };
    await createHoliday(holiday);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/duty/holidays", holiday);
    await updateHoliday("h1", { holidayName: "国庆节" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/duty/holidays/h1/update", {
      holidayName: "国庆节",
    });
    await deleteHoliday("h1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/duty/holidays/h1/delete");
    await batchCreateHolidays([holiday]);
    expect(mockPost).toHaveBeenNthCalledWith(5, "/duty/holidays/batch", { holidays: [holiday] });
    await getHolidayYears();
    expect(mockPost).toHaveBeenNthCalledWith(6, "/duty/holidays/years", {});
  });

  it("getDutyConfig / updateDutyConfig", async () => {
    mockPost.mockResolvedValue(OK);
    await getDutyConfig();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/duty/config", {});
    await updateDutyConfig({ reminderEnabled: true });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/duty/config/update", { reminderEnabled: true });
  });
});

describe("dutyApi — 用户与部门下拉", () => {
  beforeEach(() => mockPost.mockReset());

  it("getUserList 默认 current=1/pageSize=1000 且允许覆盖", async () => {
    mockPost.mockResolvedValue(OK);
    await getUserList();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/system/users/list", {
      current: 1,
      pageSize: 1000,
    });
    await getUserList({ recursiveDeptId: "d1", pageSize: 50 });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/system/users/list", {
      current: 1,
      pageSize: 50,
      recursiveDeptId: "d1",
    });
  });

  it("getDeptList / getDeptTree", async () => {
    mockPost.mockResolvedValue(OK);
    await getDeptList();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/system/departments/list");
    await getDeptTree();
    expect(mockPost).toHaveBeenNthCalledWith(2, "/system/departments/tree");
  });
});

describe("dutyApi — 我的值班", () => {
  it("getMyDutyStats POST /duty/my-duty/stats", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce(OK);
    await getMyDutyStats();
    expect(mockPost).toHaveBeenCalledWith("/duty/my-duty/stats", {});
  });
});
