/**
 * Phase 88 Batch246 — components/layout/sidebar.constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  DEFAULT_SIDEBAR_WIDTH,
  HEADER_HEIGHT,
  COLLAPSE_BUTTON_LEFT,
  MENU_FONT_SIZE,
  NAVIGATION_DELAY,
} from "../sidebar.constants";

describe("layout/sidebar.constants", () => {
  it("DEFAULT_SIDEBAR_WIDTH = 240", () => {
    expect(DEFAULT_SIDEBAR_WIDTH).toBe(240);
  });

  it("HEADER_HEIGHT = 64", () => {
    expect(HEADER_HEIGHT).toBe(64);
  });

  it("COLLAPSE_BUTTON_LEFT = 16", () => {
    expect(COLLAPSE_BUTTON_LEFT).toBe(16);
  });

  it("MENU_FONT_SIZE = 14", () => {
    expect(MENU_FONT_SIZE).toBe(14);
  });

  it("NAVIGATION_DELAY = 100", () => {
    expect(NAVIGATION_DELAY).toBe(100);
  });
});
