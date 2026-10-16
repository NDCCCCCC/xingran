/**
 * Phase 88 Batch293 — pages/monitor/job/types 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { JobStatus, MisfirePolicy } from "../types";
import type { JobInfo, JobLog, PageData, SearchFormState } from "../types";

describe("monitor/job/types", () => {
  it("JobInfo shape", () => {
    const j: JobInfo = {
      id: "1",
      jobName: "test",
      jobGroup: "DEFAULT",
      invokeTarget: "test()",
      cronExpression: "0/5 * * * *",
      misfirePolicy: MisfirePolicy.ExecuteImmediately,
      concurrent: false,
      status: JobStatus.Normal,
      createdAt: "2026-01-01",
      updatedAt: "2026-01-01",
      createdBy: "admin",
      updatedBy: "admin",
    };
    expect(j.jobName).toBe("test");
  });

  it("JobLog shape", () => {
    const l: JobLog = {
      id: "1",
      jobName: "test",
      jobGroup: "DEFAULT",
      invokeTarget: "test()",
      jobMessage: "ok",
      status: JobStatus.Normal,
      duration: 100,
      createdAt: "2026-01-01",
    };
    expect(l.jobMessage).toBe("ok");
  });

  it("PageData shape", () => {
    const p: PageData = {
      list: [],
      total: 0,
      current: 1,
      pageSize: 10,
    };
    expect(p.list.length).toBe(0);
  });

  it("SearchFormState shape", () => {
    const s: SearchFormState = { jobName: "", jobGroup: "", status: undefined };
    expect(s.status).toBeUndefined();
  });

  it("JobStatus 2 值", () => {
    expect(JobStatus.Normal).toBe(0);
    expect(JobStatus.Paused).toBe(1);
  });

  it("MisfirePolicy 3 值", () => {
    expect(MisfirePolicy.ExecuteImmediately).toBe(1);
    expect(MisfirePolicy.ExecuteOnce).toBe(2);
    expect(MisfirePolicy.Discard).toBe(3);
  });
});
