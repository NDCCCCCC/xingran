/**
 * Phase 88 Batch197 — pages/system/user/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { GENDER_OPTIONS, STATUS_OPTIONS, STATUS_TAG_CONFIG } from "../constants";

describe("system/user/constants", () => {
  it("GENDER_OPTIONS 3 项", () => {
    expect(GENDER_OPTIONS.length).toBe(3);
    expect(GENDER_OPTIONS.map((o) => o.value)).toEqual([0, 1, 2]);
    expect(GENDER_OPTIONS[0].label).toBe("男");
    expect(GENDER_OPTIONS[1].label).toBe("女");
    expect(GENDER_OPTIONS[2].label).toBe("保密");
  });

  it("STATUS_OPTIONS 共享 ENABLE_DISABLE_OPTIONS 2 项", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("STATUS_TAG_CONFIG 含 0/1", () => {
    expect(STATUS_TAG_CONFIG[0]).toBeDefined();
    expect(STATUS_TAG_CONFIG[1]).toBeDefined();
  });

  it("STATUS_TAG_CONFIG[0].text 是 启用 类", () => {
    expect(STATUS_TAG_CONFIG[0].text).toMatch(/启用|正常/);
  });
});
