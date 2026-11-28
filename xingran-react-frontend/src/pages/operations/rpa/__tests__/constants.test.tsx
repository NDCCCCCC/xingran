/**
 * Phase 85 — rpa constants 纯函数测试
 */
import { describe, it, expect } from "vitest";
import {
  TRIGGER_TYPE_OPTIONS,
  TASK_STATUS_OPTIONS,
  EXECUTION_STATUS_OPTIONS,
  WORKER_STATUS_OPTIONS,
  getTriggerTypeText,
  getTriggerTypeColor,
  getTaskStatusText,
  getTaskStatusColor,
} from "../constants";

describe("rpa constants (D-12)", () => {
  it("TRIGGER_TYPE_OPTIONS non-empty", () => {
    expect(TRIGGER_TYPE_OPTIONS.length).toBeGreaterThan(0);
  });

  it("TASK_STATUS_OPTIONS non-empty", () => {
    expect(TASK_STATUS_OPTIONS.length).toBeGreaterThan(0);
  });

  it("EXECUTION_STATUS_OPTIONS non-empty", () => {
    expect(EXECUTION_STATUS_OPTIONS.length).toBeGreaterThan(0);
  });

  it("WORKER_STATUS_OPTIONS non-empty", () => {
    expect(WORKER_STATUS_OPTIONS.length).toBeGreaterThan(0);
  });
});

describe("getTriggerTypeText/Color", () => {
  it("returns text for first known trigger type", () => {
    const first = TRIGGER_TYPE_OPTIONS[0] as unknown;
    expect(getTriggerTypeText(first.value)).toBe(first.label);
  });

  it("returns color for known trigger type", () => {
    const first = TRIGGER_TYPE_OPTIONS[0] as unknown;
    expect(getTriggerTypeColor(first.value)).toBeTruthy();
  });
});

describe("getTaskStatusText/Color", () => {
  it("returns text for status 0/1", () => {
    expect(getTaskStatusText(0)).toBeTruthy();
    expect(getTaskStatusText(1)).toBeTruthy();
  });

  it("returns color for status 0/1", () => {
    expect(getTaskStatusColor(0)).toBeTruthy();
    expect(getTaskStatusColor(1)).toBeTruthy();
  });
});

// CONTRACT-02（D2）：三张列表状态筛选前后端同构的前端侧断言。
//
// 这两组断言是 CONTRACT-02 的「前端可观察证据」：
//   1. 任务页筛选值必须是**任务态数字 0/1**（启用/停用），不得再是执行态字符串
//      —— 旧实现传 "pending"/"running" 会被后端 TaskStatus(*int) 直接 400。
//   2. 执行态枚举必须覆盖后端 RPAExecutionStatus 全部 6 个取值（含 timeout）
//      —— 旧实现缺 timeout，超时记录无从筛选、标签回落成 "-"。
describe("CONTRACT-02 前后端状态枚举同构", () => {
  it("TASK_STATUS_OPTIONS 是任务态数字 0/1，且不含后端不存在的 2", () => {
    expect(TASK_STATUS_OPTIONS.map((o) => o.value)).toEqual([0, 1]);
    for (const opt of TASK_STATUS_OPTIONS) {
      expect(typeof opt.value).toBe("number");
    }
  });

  it("TASK_STATUS_OPTIONS 语义为启用/停用（不是草稿/禁用）", () => {
    expect(getTaskStatusText(0)).toBe("启用");
    expect(getTaskStatusText(1)).toBe("停用");
  });

  it("EXECUTION_STATUS_OPTIONS 覆盖后端全部 6 个执行态（含 timeout）", () => {
    expect(EXECUTION_STATUS_OPTIONS.map((o) => o.value).sort()).toEqual(
      ["cancelled", "failed", "pending", "running", "success", "timeout"].sort()
    );
  });

  it("执行态枚举里不得出现后端没有的 completed", () => {
    expect(EXECUTION_STATUS_OPTIONS.map((o) => o.value)).not.toContain("completed");
  });
});
