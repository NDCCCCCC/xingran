/**
 * Phase 88 Batch285 — utils/antdMessage 测试
 */
import { describe, it, expect, beforeEach, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { setAppMessageInstance, getAppMessage } from "../antdMessage";

describe("utils/antdMessage", () => {
  beforeEach(() => {
    setAppMessageInstance(null);
  });

  it("getAppMessage 默认 → no-op 实例", () => {
    const m = getAppMessage();
    expect(m).toBeDefined();
    // no-op 不抛错
    m.success("test");
    m.error("err");
    m.info("info");
    m.warning("warn");
    m.loading("load");
    m.warn("warn");
    m.open("open");
    m.destroy();
  });

  it("setAppMessageInstance(null) → no-op", () => {
    setAppMessageInstance(null);
    const m = getAppMessage();
    expect(m).toBeDefined();
  });

  it("setAppMessageInstance 设置 mock 实例", () => {
    const mockInstance: any = {
      success: vi.fn(),
      error: vi.fn(),
    };
    setAppMessageInstance(mockInstance);
    const m = getAppMessage();
    expect(m).toBe(mockInstance);
  });
});
