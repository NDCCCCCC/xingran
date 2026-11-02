/**
 * Phase 88 Batch362 — components/layout/shared/TabBar.constants 测试
 */
import { describe, it, expect } from "vitest";
import {
  SCROLL_STEP,
  INITIAL_DELAYS,
  DEFAULT_HEIGHT,
  DEFAULT_PADDING,
  MIN_WIDTH,
  SCROLL_TOLERANCE,
  DROPDOWN_MAX_ZINDEX,
} from "../TabBar.constants";

describe("components/layout/shared/TabBar.constants", () => {
  it("SCROLL_STEP = 200", () => {
    expect(SCROLL_STEP).toBe(200);
  });

  it("INITIAL_DELAYS = [0, 100, 300]", () => {
    expect(INITIAL_DELAYS).toEqual([0, 100, 300]);
    expect(INITIAL_DELAYS.length).toBe(3);
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
