/**
 * Phase 88 Batch310 — pages/monitor/job/constants 测试
 */
import { describe, it, expect } from "vitest";
import {
  STATUS_OPTIONS,
  MISFIRE_POLICY_OPTIONS,
  DEFAULT_FORM_VALUES,
  DEFAULT_SEARCH_FORM,
} from "../constants";
import { JobStatus, MisfirePolicy } from "../types";

describe("pages/monitor/job/constants", () => {
  it("STATUS_OPTIONS 2 项", () => {
    expect(STATUS_OPTIONS.length).toBe(2);
  });

  it("STATUS_OPTIONS 含 Normal + Paused", () => {
    expect(STATUS_OPTIONS[0].value).toBe(JobStatus.Normal);
    expect(STATUS_OPTIONS[1].value).toBe(JobStatus.Paused);
    expect(STATUS_OPTIONS[0].label).toBe("正常");
    expect(STATUS_OPTIONS[1].label).toBe("暂停");
  });

  it("MISFIRE_POLICY_OPTIONS 3 项", () => {
    expect(MISFIRE_POLICY_OPTIONS.length).toBe(3);
  });

  it("MISFIRE_POLICY_OPTIONS 顺序正确", () => {
    expect(MISFIRE_POLICY_OPTIONS[0].value).toBe(MisfirePolicy.ExecuteImmediately);
    expect(MISFIRE_POLICY_OPTIONS[1].value).toBe(MisfirePolicy.ExecuteOnce);
    expect(MISFIRE_POLICY_OPTIONS[2].value).toBe(MisfirePolicy.Discard);
  });

  it("MISFIRE_POLICY_OPTIONS 标签", () => {
    expect(MISFIRE_POLICY_OPTIONS[0].label).toBe("立即执行");
    expect(MISFIRE_POLICY_OPTIONS[1].label).toBe("执行一次");
    expect(MISFIRE_POLICY_OPTIONS[2].label).toBe("放弃执行");
  });

  it("DEFAULT_FORM_VALUES", () => {
    expect(DEFAULT_FORM_VALUES.misfirePolicy).toBe(MisfirePolicy.ExecuteImmediately);
    expect(DEFAULT_FORM_VALUES.concurrent).toBe(false);
    expect(DEFAULT_FORM_VALUES.status).toBe(JobStatus.Normal);
  });

  it("DEFAULT_SEARCH_FORM 默认值", () => {
    expect(DEFAULT_SEARCH_FORM.jobName).toBe("");
    expect(DEFAULT_SEARCH_FORM.jobGroup).toBe("");
    expect(DEFAULT_SEARCH_FORM.status).toBeUndefined();
  });

  it("JobStatus 枚举值", () => {
    expect(JobStatus.Normal).toBe(0);
    expect(JobStatus.Paused).toBe(1);
  });

  it("MisfirePolicy 枚举值", () => {
    expect(MisfirePolicy.ExecuteImmediately).toBe(1);
    expect(MisfirePolicy.ExecuteOnce).toBe(2);
    expect(MisfirePolicy.Discard).toBe(3);
  });
});
