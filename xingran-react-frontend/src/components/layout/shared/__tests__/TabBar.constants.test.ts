/**
 * Phase 88 Batch224 — components/layout/shared/TabBar.constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  SCROLL_STEP,
  INITIAL_DELAYS,
  DEFAULT_HEIGHT,
  DEFAULT_PADDING,
  MIN_WIDTH,
  SCROLL_TOLERANCE,
  DROPDOWN_MAX_ZINDEX,
} from "../TabBar.constants";

describe("layout/shared/TabBar.constants", () => {
  it("SCROLL_STEP = 200", () => {
    expect(SCROLL_STEP).toBe(200);
  });

  it("INITIAL_DELAYS 3 阶段", () => {
    expect(INITIAL_DELAYS.length).toBe(3);
    expect(INITIAL_DELAYS[0]).toBe(0);
    expect(INITIAL_DELAYS[1]).toBe(100);
    expect(INITIAL_DELAYS[2]).toBe(300);
  });

  it("DEFAULT_HEIGHT = 40", () => {
    expect(DEFAULT_HEIGHT).toBe(40);
  });

  it("DEFAULT_PADDING = 16", () => {
    expect(DEFAULT_PADDING).toBe(16);
  });

  it("MIN_WIDTH = 32", () => {
    expect(MIN_WIDTH).toBe(32);
  });

  it("SCROLL_TOLERANCE = 1", () => {
    expect(SCROLL_TOLERANCE).toBe(1);
  });

  it("DROPDOWN_MAX_ZINDEX = 1000", () => {
    expect(DROPDOWN_MAX_ZINDEX).toBe(1000);
  });
});
