/**
 * Phase 158 / REC-02 —— 定时调度页面的纯函数与列渲染契约测试
 *
 * 覆盖三件事：
 *   1. 简单选项 ↔ 定时规则（cron 表达式）换算正确且**互逆**；
 *   2. 面向员工的中文文案（运行方式）不含技术术语；
 *   3. 列表列定义把定时规则翻成中文运行方式，且开关/删除入口按权限码门控。
 */

import { describe, it, expect } from "vitest";
import {
  CUSTOM_LABEL,
  DEFAULT_INTERVAL_MINUTES,
  WEEKDAY_OPTIONS,
  cronToPreset,
  describeRunMode,
  formatClockTime,
  isValidCronShape,
  parseClockTime,
  presetToCron,
} from "../cronPresets";
import { getScheduleColumns } from "../columns";
import type { ScheduleRecord } from "../api";

// ---------------------------------------------------------------------------
// 1. 简单选项 → 定时规则
// ---------------------------------------------------------------------------

describe("简单选项 → 定时规则（presetToCron）", () => {
  it("每天默认 09:00 → 0 9 * * *", () => {
    expect(presetToCron("daily")).toBe("0 9 * * *");
    expect(presetToCron("daily", { hour: 9, minute: 0 })).toBe("0 9 * * *");
  });

  it("每天可指定其它时间", () => {
    expect(presetToCron("daily", { hour: 8, minute: 30 })).toBe("30 8 * * *");
    expect(presetToCron("daily", { hour: 0, minute: 0 })).toBe("0 0 * * *");
    expect(presetToCron("daily", { hour: 23, minute: 59 })).toBe("59 23 * * *");
  });

  it("每周默认周一 09:00 → 0 9 * * 1", () => {
    expect(presetToCron("weekly")).toBe("0 9 * * 1");
    expect(presetToCron("weekly", { weekday: 1, hour: 9, minute: 0 })).toBe("0 9 * * 1");
  });

  it("每周的星期几按业务口径（1=周一…7=周日）换算", () => {
    // 周一=1 → cron 1；周日=7 → cron 0（cron 的 0 是周日）
    expect(presetToCron("weekly", { weekday: 2, hour: 9, minute: 0 })).toBe("0 9 * * 2");
    expect(presetToCron("weekly", { weekday: 7, hour: 9, minute: 0 })).toBe("0 9 * * 0");
    expect(presetToCron("weekly", { weekday: 6, hour: 9, minute: 0 })).toBe("0 9 * * 6");
  });

  it("每隔 N 分钟默认 15 → */15 * * * *", () => {
    expect(presetToCron("interval")).toBe("*/15 * * * *");
    expect(presetToCron("interval", { minutes: DEFAULT_INTERVAL_MINUTES })).toBe("*/15 * * * *");
    expect(presetToCron("interval", { minutes: 5 })).toBe("*/5 * * * *");
    expect(presetToCron("interval", { minutes: 30 })).toBe("*/30 * * * *");
  });

  it("非法取值回落到默认值（不产出非法表达式）", () => {
    expect(presetToCron("daily", { hour: 99, minute: 99 })).toBe("0 9 * * *");
    expect(presetToCron("weekly", { weekday: 0 })).toBe("0 9 * * 1");
    expect(presetToCron("interval", { minutes: 0 })).toBe("*/15 * * * *");
  });
});

// ---------------------------------------------------------------------------
// 2. 定时规则 → 展示用（cronToPreset）
// ---------------------------------------------------------------------------

describe("定时规则 → 展示用（cronToPreset）", () => {
  it("每天给出中文文案「每天 09:00」", () => {
    const parsed = cronToPreset("0 9 * * *");
    expect(parsed.preset).toBe("daily");
    expect(parsed.label).toBe("每天 09:00");
  });

  it("每周给出中文文案「每周一 09:00」", () => {
    const parsed = cronToPreset("0 9 * * 1");
    expect(parsed.preset).toBe("weekly");
    expect(parsed.label).toBe("每周一 09:00");
    expect(cronToPreset("0 9 * * 0").label).toBe("每周日 09:00");
    expect(cronToPreset("30 8 * * 3").label).toBe("每周三 08:30");
  });

  it("每隔 N 分钟给出中文文案「每隔 15 分钟」", () => {
    const parsed = cronToPreset("*/15 * * * *");
    expect(parsed.preset).toBe("interval");
    expect(parsed.label).toBe("每隔 15 分钟");
  });

  it("识别不出的一律当「自定义」", () => {
    expect(cronToPreset("0 9 1 * *").preset).toBe("custom");
    expect(cronToPreset("0 9 * * * *").preset).toBe("custom"); // 6 位带秒
    expect(cronToPreset("").preset).toBe("custom");
    expect(cronToPreset("每天").preset).toBe("custom");
    expect(cronToPreset("0 9 * * 1").label).not.toBe(CUSTOM_LABEL);
    expect(cronToPreset("0 9 1 * *").label).toBe(CUSTOM_LABEL);
  });

  it("自定义时原样带回定时规则", () => {
    expect(cronToPreset("0 9 1 * *")).toEqual({
      preset: "custom",
      label: CUSTOM_LABEL,
      cron: "0 9 1 * *",
    });
  });
});

// ---------------------------------------------------------------------------
// 3. 互逆性（三个简单预设）
// ---------------------------------------------------------------------------

describe("简单选项与定时规则互逆", () => {
  const cases: Array<{
    preset: "daily" | "weekly" | "interval";
    opts: Record<string, number>;
    cron: string;
  }> = [
    { preset: "daily", opts: { hour: 9, minute: 0 }, cron: "0 9 * * *" },
    { preset: "daily", opts: { hour: 8, minute: 30 }, cron: "30 8 * * *" },
    { preset: "weekly", opts: { weekday: 1, hour: 9, minute: 0 }, cron: "0 9 * * 1" },
    { preset: "weekly", opts: { weekday: 5, hour: 18, minute: 15 }, cron: "15 18 * * 5" },
    { preset: "weekly", opts: { weekday: 7, hour: 9, minute: 0 }, cron: "0 9 * * 0" },
    { preset: "interval", opts: { minutes: 15 }, cron: "*/15 * * * *" },
    { preset: "interval", opts: { minutes: 5 }, cron: "*/5 * * * *" },
  ];

  it("定时规则 → 简单选项 → 定时规则（原文不变）", () => {
    cases.forEach(({ preset, opts, cron }) => {
      const parsed = cronToPreset(cron);
      expect(parsed.preset).toBe(preset);
      if (parsed.preset === "custom") {
        throw new Error(`定时规则 ${cron} 未被识别为 ${preset}`);
      }
      expect(presetToCron(parsed.preset, parsed.opts)).toBe(cron);
    });
  });

  it("简单选项 → 定时规则 → 简单选项（选项不变）", () => {
    cases.forEach(({ preset, opts, cron }) => {
      const parsed = cronToPreset(cron);
      expect(parsed.preset).toBe(preset);
      if (parsed.preset === "custom") {
        throw new Error(`定时规则 ${cron} 未被识别为 ${preset}`);
      }
      const roundTrip = presetToCron(preset, parsed.opts);
      expect(roundTrip).toBe(cron);
      const again = cronToPreset(roundTrip);
      expect(again.preset).toBe(preset);
    });
  });
});

// ---------------------------------------------------------------------------
// 4. 中文文案与形状校验
// ---------------------------------------------------------------------------

describe("中文文案与技术术语隔离", () => {
  it("运行方式文案是中文业务语义", () => {
    expect(describeRunMode("0 9 * * *")).toBe("每天 09:00");
    expect(describeRunMode("0 9 * * 1")).toBe("每周一 09:00");
    expect(describeRunMode("*/15 * * * *")).toBe("每隔 15 分钟");
    expect(describeRunMode("0 9 1 * *")).toBe(CUSTOM_LABEL);
  });

  it("中文文案里不出现技术术语", () => {
    const labels = ["0 9 * * *", "0 9 * * 3", "*/15 * * * *", "0 9 1 * *"].map(describeRunMode);
    labels.forEach((label) => {
      expect(label).not.toMatch(/cron/i);
      expect(label).not.toMatch(/taskId/i);
      expect(label).not.toMatch(/json/i);
    });
  });

  it("星期几选项覆盖周一到周日", () => {
    expect(WEEKDAY_OPTIONS.map((o) => o.label)).toEqual([
      "周一",
      "周二",
      "周三",
      "周四",
      "周五",
      "周六",
      "周日",
    ]);
    expect(WEEKDAY_OPTIONS.map((o) => o.value)).toEqual([1, 2, 3, 4, 5, 6, 7]);
  });

  it("时间文本与时分互转", () => {
    expect(formatClockTime(9, 0)).toBe("09:00");
    expect(formatClockTime(18, 30)).toBe("18:30");
    expect(parseClockTime("09:00")).toEqual({ hour: 9, minute: 0 });
    expect(parseClockTime("18:30")).toEqual({ hour: 18, minute: 30 });
    expect(parseClockTime("乱填")).toEqual({ hour: 9, minute: 0 });
  });
});

describe("定时规则形状校验", () => {
  it("接受 5 位与 6 位表达式", () => {
    expect(isValidCronShape("0 9 * * *")).toBe(true);
    expect(isValidCronShape("*/15 * * * *")).toBe(true);
    expect(isValidCronShape("0 0 9 * * *")).toBe(true);
  });

  it("拒绝字段数不对、含非法字符、超界的表达式", () => {
    expect(isValidCronShape("")).toBe(false);
    expect(isValidCronShape("0 9 * *")).toBe(false);
    expect(isValidCronShape("0 9 * * * * *")).toBe(false);
    expect(isValidCronShape("0 9 * * 每天")).toBe(false);
    expect(isValidCronShape("99 9 * * *")).toBe(false);
    expect(isValidCronShape("0 99 * * *")).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// 5. 列表列定义
// ---------------------------------------------------------------------------

describe("定时调度列表列定义", () => {
  const baseParams = {
    handleEdit: () => {},
    handleTrigger: () => {},
    handleDelete: () => {},
    handleStatusChange: () => {},
    canEdit: true,
    canDelete: true,
    taskNameMap: { "task-1": "每天的报表任务" },
  };

  const makeRecord = (overrides: Partial<ScheduleRecord> = {}): ScheduleRecord =>
    ({
      id: "sched-1",
      name: "早报调度",
      taskId: "task-1",
      cronExpr: "0 9 * * *",
      status: 0,
      ...overrides,
    }) as ScheduleRecord;

  it("列名是中文业务语义，不含技术术语", () => {
    const columns = getScheduleColumns(baseParams);
    const titles = columns.map((col) => String(col.title ?? ""));
    expect(titles).toEqual([
      "名称",
      "关联任务",
      "运行方式",
      "启用/停用",
      "最近触发时间",
      "下次触发时间",
      "操作",
    ]);
    titles.forEach((title) => {
      expect(title).not.toMatch(/cron/i);
      expect(title).not.toMatch(/taskId/i);
    });
  });

  it("「关联任务」列展示任务名称而不是内部标识", () => {
    const columns = getScheduleColumns(baseParams);
    const column = columns.find((col) => col.key === "taskName");
    const rendered = (column?.render as (v: unknown, r: ScheduleRecord) => unknown)(
      undefined,
      makeRecord()
    );
    expect(rendered).toBe("每天的报表任务");
  });

  it("「运行方式」列把定时规则翻成中文", () => {
    const columns = getScheduleColumns(baseParams);
    const column = columns.find((col) => col.key === "runMode");
    const render = column?.render as (v: unknown, r: ScheduleRecord) => unknown;
    expect(render(undefined, makeRecord({ cronExpr: "0 9 * * *" }))).toBe("每天 09:00");
    expect(render(undefined, makeRecord({ cronExpr: "0 9 * * 1" }))).toBe("每周一 09:00");
    expect(render(undefined, makeRecord({ cronExpr: "*/15 * * * *" }))).toBe("每隔 15 分钟");
    expect(render(undefined, makeRecord({ cronExpr: "0 9 1 * *" }))).toBe(CUSTOM_LABEL);
  });

  it("「最近触发时间」为空时显示「尚未运行」，「下次触发时间」为空时显示「—」", () => {
    const columns = getScheduleColumns(baseParams);
    const lastColumn = columns.find((col) => col.key === "lastRunTime");
    const nextColumn = columns.find((col) => col.key === "nextRunTime");
    const renderLast = lastColumn?.render as (v: unknown, r: ScheduleRecord) => unknown;
    const renderNext = nextColumn?.render as (v: unknown, r: ScheduleRecord) => unknown;

    expect(renderLast(undefined, makeRecord({ lastRunTime: null }))).toBe("尚未运行");
    expect(renderLast(undefined, makeRecord({ lastRunTime: "2026-11-25T01:02:03Z" }))).toMatch(
      /^\d{4}-\d{2}-\d{2} /
    );
    expect(renderNext(undefined, makeRecord({ nextRunTime: null }))).toBe("—");
    expect(renderNext(undefined, makeRecord({ nextRunTime: "2026-11-25T01:02:03Z" }))).toMatch(
      /^\d{4}-\d{2}-\d{2} /
    );
  });

  it("没有删除权限时不渲染删除入口", () => {
    const columns = getScheduleColumns({ ...baseParams, canDelete: false });
    const actionColumn = columns.find((col) => col.key === "action");
    expect(actionColumn).toBeTruthy();
    // 渲染元素里不应出现删除按钮文案
    const rendered = (actionColumn?.render as (v: unknown, r: ScheduleRecord) => { render?: () => unknown })(
      undefined,
      makeRecord()
    ) as { render?: () => unknown };
    expect(JSON.stringify(rendered)).not.toContain("删除");
  });
});
