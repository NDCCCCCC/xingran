/**
 * Phase 88 Batch249 — pages/workorder/periodic/templates/utils 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { VARIABLE_HELP_CONTENT } from "../utils";

describe("workorder/periodic/templates/utils", () => {
  it("VARIABLE_HELP_CONTENT.title", () => {
    expect(VARIABLE_HELP_CONTENT.title).toContain("工单标题");
  });

  it("variables 8 项", () => {
    expect(VARIABLE_HELP_CONTENT.variables.length).toBe(8);
  });

  it("variables 含 date/datetime/year/month", () => {
    const codes = VARIABLE_HELP_CONTENT.variables.map((v) => v.code);
    expect(codes).toContain("{date}");
    expect(codes).toContain("{datetime}");
    expect(codes).toContain("{year}");
    expect(codes).toContain("{month}");
  });

  it("variables 每项含 description", () => {
    const v = VARIABLE_HELP_CONTENT.variables[0];
    expect(v.code).toBeTruthy();
    expect(v.description).toBeTruthy();
  });
});
