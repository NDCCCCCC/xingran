/**
 * Phase 88 Batch261 — pages/monitor/cache/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { TYPE_OPTIONS, OPERATION_OPTIONS, LEVEL_OPTIONS, LEVEL_TAG_CONFIG } from "../constants";

describe("monitor/cache/constants", () => {
  it("TYPE_OPTIONS 6 项", () => {
    expect(TYPE_OPTIONS.length).toBe(6);
    expect(TYPE_OPTIONS[0]).toEqual({ label: "字符串", value: "string" });
    expect(TYPE_OPTIONS[1]).toEqual({ label: "列表", value: "list" });
  });

  it("OPERATION_OPTIONS 6 项", () => {
    expect(OPERATION_OPTIONS.length).toBe(6);
    expect(OPERATION_OPTIONS[0].value).toBe("get");
    expect(OPERATION_OPTIONS[5].value).toBe("ttl");
  });

  it("LEVEL_OPTIONS 3 项", () => {
    expect(LEVEL_OPTIONS.length).toBe(3);
    expect(LEVEL_OPTIONS[1].value).toBe("l1");
    expect(LEVEL_OPTIONS[2].value).toBe("l2");
  });

  it("LEVEL_TAG_CONFIG 3 标签", () => {
    expect(Object.keys(LEVEL_TAG_CONFIG).length).toBe(3);
    expect(LEVEL_TAG_CONFIG.l1.color).toBe("blue");
    expect(LEVEL_TAG_CONFIG.l2.color).toBe("green");
    expect(LEVEL_TAG_CONFIG.both.color).toBe("purple");
  });
});
