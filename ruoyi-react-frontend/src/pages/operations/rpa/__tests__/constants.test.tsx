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
    const first = TRIGGER_TYPE_OPTIONS[0] as any;
    expect(getTriggerTypeText(first.value)).toBe(first.label);
  });

  it("returns color for known trigger type", () => {
    const first = TRIGGER_TYPE_OPTIONS[0] as any;
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
