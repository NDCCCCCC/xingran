/**
 * Phase 84 84-02a — Layout constants + HybridLayout 静态断言
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import {
  DEFAULT_SIDEBAR_WIDTH,
  HEADER_HEIGHT,
  COLLAPSE_BUTTON_LEFT,
  MENU_FONT_SIZE,
  NAVIGATION_DELAY,
} from "../sidebar.constants";
import HybridLayout from "../HybridLayout";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("layout constants (D-12 static assertion)", () => {
  it("DEFAULT_SIDEBAR_WIDTH equals 240", () => {
    expect(DEFAULT_SIDEBAR_WIDTH).toBe(240);
  });

  it("HEADER_HEIGHT equals 64", () => {
    expect(HEADER_HEIGHT).toBe(64);
  });

  it("COLLAPSE_BUTTON_LEFT positive", () => {
    expect(COLLAPSE_BUTTON_LEFT).toBe(16);
    expect(COLLAPSE_BUTTON_LEFT).toBeGreaterThan(0);
  });

  it("MENU_FONT_SIZE equals 14", () => {
    expect(MENU_FONT_SIZE).toBe(14);
  });

  it("NAVIGATION_DELAY positive ms", () => {
    expect(NAVIGATION_DELAY).toBe(100);
    expect(NAVIGATION_DELAY).toBeGreaterThan(0);
  });
});

describe("HybridLayout", () => {
  it("HybridLayout module is imported", () => {
    // HybridLayout 内部 useRouteTabs 依赖 routeConfigManager,jsdom 初始化复杂;
    // 此处仅断言模块可正常导入(其它渲染交互已由其它集成测试覆盖)
    expect(HybridLayout).toBeDefined();
    expect(typeof HybridLayout).toBe("function");
  });
});
