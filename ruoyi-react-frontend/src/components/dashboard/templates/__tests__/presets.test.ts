/**
 * Phase 88 Batch388 — dashboard/templates/presets.ts 测试
 */
import { describe, it, expect } from "vitest";
import {
  presetDashboardTemplates,
  getPresetTemplate,
  operationsOverviewTemplate,
  workorderManagementTemplate,
  dutyManagementTemplate,
  systemMonitorTemplate,
} from "../presets";

describe("dashboard/templates/presets", () => {
  it("presetDashboardTemplates 导出4个模板", () => {
    expect(presetDashboardTemplates).toHaveLength(4);
  });

  it("每个模板有 displayName / description / dashboard / type", () => {
    presetDashboardTemplates.forEach((t) => {
      expect(typeof t.displayName).toBe("string");
      expect(typeof t.description).toBe("string");
      expect(typeof t.dashboard).toBe("object");
      expect(typeof t.type).toBe("string");
    });
  });

  it("operationsOverviewTemplate 类型正确", () => {
    expect(operationsOverviewTemplate.type).toBe("operations-overview");
    expect(operationsOverviewTemplate.dashboard.layout.widgets.length).toBeGreaterThan(0);
  });

  it("workorderManagementTemplate 类型正确", () => {
    expect(workorderManagementTemplate.type).toBe("workorder-management");
    expect(workorderManagementTemplate.dashboard.layout.widgets.length).toBeGreaterThan(0);
  });

  it("dutyManagementTemplate 类型正确", () => {
    expect(dutyManagementTemplate.type).toBe("duty-management");
    expect(dutyManagementTemplate.dashboard.layout.widgets.length).toBeGreaterThan(0);
  });

  it("systemMonitorTemplate 类型正确", () => {
    expect(systemMonitorTemplate.type).toBe("system-monitor");
    expect(systemMonitorTemplate.dashboard.layout.widgets.length).toBeGreaterThan(0);
  });

  it("getPresetTemplate 返回匹配模板", () => {
    const t = getPresetTemplate("operations-overview");
    expect(t).toBeDefined();
    expect(t?.type).toBe("operations-overview");
  });

  it("getPresetTemplate 不存在类型返回 undefined", () => {
    const t = getPresetTemplate("non-existent-type" as any);
    expect(t).toBeUndefined();
  });

  it("所有 dashboard.name 非空", () => {
    presetDashboardTemplates.forEach((t) => {
      expect(t.dashboard.name.length).toBeGreaterThan(0);
    });
  });

  it("所有 dashboard.widgets 包含必要的字段", () => {
    presetDashboardTemplates.forEach((t) => {
      t.dashboard.layout.widgets.forEach((w: any) => {
        expect(typeof w.id).toBe("string");
        expect(typeof w.type).toBe("string");
        expect(typeof w.title).toBe("string");
        expect(typeof w.position).toBe("object");
        expect(typeof w.enabled).toBe("boolean");
      });
    });
  });

  it("所有模板 layout.widgets 数量与模板一致", () => {
    const templates = [
      { t: operationsOverviewTemplate, minWidgets: 7 },
      { t: workorderManagementTemplate, minWidgets: 8 },
      { t: dutyManagementTemplate, minWidgets: 6 },
      { t: systemMonitorTemplate, minWidgets: 6 },
    ];
    templates.forEach(({ t, minWidgets }) => {
      expect(t.dashboard.layout.widgets.length).toBeGreaterThanOrEqual(minWidgets);
    });
  });
});
