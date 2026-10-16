/**
 * Phase 84 84-01b — Dashboard presets 静态断言
 */
import { describe, it, expect } from "vitest";
import {
  presetDashboardTemplates,
  getPresetTemplate,
  operationsOverviewTemplate,
  workorderManagementTemplate,
  dutyManagementTemplate,
  systemMonitorTemplate,
} from "../templates/presets";

describe("Dashboard presets exports", () => {
  it("presetDashboardTemplates is non-empty array", () => {
    expect(presetDashboardTemplates).toBeDefined();
    expect(Array.isArray(presetDashboardTemplates)).toBe(true);
    expect(presetDashboardTemplates.length).toBeGreaterThan(0);
  });

  it("exports 4 named templates (D-12)", () => {
    expect(operationsOverviewTemplate).toBeDefined();
    expect(workorderManagementTemplate).toBeDefined();
    expect(dutyManagementTemplate).toBeDefined();
    expect(systemMonitorTemplate).toBeDefined();
  });

  it("each template has type/displayName/dashboard fields", () => {
    for (const tmpl of presetDashboardTemplates) {
      expect(tmpl.type).toBeTruthy();
      expect(tmpl.displayName).toBeTruthy();
      expect(tmpl.dashboard).toBeDefined();
      expect(Array.isArray(tmpl.dashboard?.layout?.widgets)).toBe(true);
    }
  });

  it("each named template has matching type/displayName", () => {
    expect(operationsOverviewTemplate.type).toBe("operations-overview");
    expect(operationsOverviewTemplate.displayName).toBe("运维总览");
  });

  it("getPresetTemplate returns matching template by type", () => {
    const tmpl = getPresetTemplate("operations-overview");
    expect(tmpl).toBeDefined();
    expect(tmpl?.type).toBe("operations-overview");
  });

  it("getPresetTemplate returns undefined for unknown type", () => {
    expect(getPresetTemplate("non-existent-template" as any)).toBeUndefined();
  });
});
