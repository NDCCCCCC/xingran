/**
 * Phase 88 Batch237 — pages/monitor/job/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { JobStatus, MisfirePolicy } from "../types";
import {
  STATUS_OPTIONS,
  MISFIRE_POLICY_OPTIONS,
  DEFAULT_FORM_VALUES,
  DEFAULT_SEARCH_FORM,
} from "../constants";

describe("monitor/job/constants", () => {
  it("STATUS_OPTIONS 2 项", () => {
    expect(STATUS_OPTIONS.length).toBe(2);
    expect(STATUS_OPTIONS[0]).toEqual({ label: "正常", value: JobStatus.Normal });
    expect(STATUS_OPTIONS[1]).toEqual({ label: "暂停", value: JobStatus.Paused });
  });

  it("MISFIRE_POLICY_OPTIONS 3 项", () => {
    expect(MISFIRE_POLICY_OPTIONS.length).toBe(3);
    expect(MISFIRE_POLICY_OPTIONS[0]).toEqual({
      label: "立即执行",
      value: MisfirePolicy.ExecuteImmediately,
    });
    expect(MISFIRE_POLICY_OPTIONS[2]).toEqual({ label: "放弃执行", value: MisfirePolicy.Discard });
  });

  it("DEFAULT_FORM_VALUES 默认", () => {
    expect(DEFAULT_FORM_VALUES.misfirePolicy).toBe(MisfirePolicy.ExecuteImmediately);
    expect(DEFAULT_FORM_VALUES.concurrent).toBe(false);
    expect(DEFAULT_FORM_VALUES.status).toBe(JobStatus.Normal);
  });

  it("DEFAULT_SEARCH_FORM 默认空", () => {
    expect(DEFAULT_SEARCH_FORM.jobName).toBe("");
    expect(DEFAULT_SEARCH_FORM.jobGroup).toBe("");
    expect(DEFAULT_SEARCH_FORM.status).toBeUndefined();
  });
});
