/**
 * Phase 84 84-01b — Dashboard widget/layout/registry 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { widgetRegistry } from "../widgets/configs/widgetRegistry";
import { DashboardGridPlaceholder } from "../layout/DashboardGrid";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("DashboardGridPlaceholder", () => {
  it("renders without crash (D-12 props)", () => {
    const { container } = renderWithProviders(<DashboardGridPlaceholder />);
    expect(container).not.toBeNull();
  });
});

describe("widgetRegistry", () => {
  it("registers widget types", () => {
    expect(typeof widgetRegistry).toBe("object");
    expect(widgetRegistry).not.toBeNull();
    const types = Object.keys(widgetRegistry);
    expect(types.length).toBeGreaterThan(0);
  });

  it("each registered widget has required fields (D-12 static assertion)", () => {
    for (const type of Object.keys(widgetRegistry)) {
      const cfg = widgetRegistry[type];
      expect(cfg.type).toBeDefined();
      expect(cfg.displayName).toBeTruthy();
      expect(cfg.component).toBeDefined();
      expect(cfg.defaultSize).toBeDefined();
      expect(cfg.defaultSize.w).toBeGreaterThan(0);
      expect(cfg.defaultSize.h).toBeGreaterThan(0);
    }
  });

  it("widget types are non-empty strings", () => {
    for (const type of Object.keys(widgetRegistry)) {
      expect(type).toBeTruthy();
      expect(typeof type).toBe("string");
    }
  });
});
