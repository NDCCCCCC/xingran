/**
 * Phase 88 Batch376 — components/dashboard/widgets/configs/widgetRegistry 测试
 */
import { describe, it, expect } from "vitest";
import {
  widgetRegistry,
  getWidgetTypes,
  getWidgetConfig,
  getWidgetComponent,
  registerWidget,
  registerWidgets,
} from "../widgetRegistry";

describe("components/dashboard/widgets/configs/widgetRegistry", () => {
  it("widgetRegistry 含多种类型", () => {
    expect(Object.keys(widgetRegistry).length).toBeGreaterThan(0);
  });

  it("getWidgetTypes 返回所有类型", () => {
    const types = getWidgetTypes();
    expect(types.length).toBeGreaterThan(0);
    expect(types).toContain("stat-card");
  });

  it("getWidgetConfig 已知类型", () => {
    const config = getWidgetConfig("stat-card");
    expect(config).toBeDefined();
    expect(config?.displayName).toBeDefined();
  });

  it("getWidgetConfig 未知类型 → undefined", () => {
    expect(getWidgetConfig("unknown-type" as any)).toBeUndefined();
  });

  it("getWidgetComponent 已知类型 → 返回组件", () => {
    const comp = getWidgetComponent("stat-card");
    expect(comp).toBeDefined();
  });

  it("getWidgetComponent 未知类型 → undefined", () => {
    expect(getWidgetComponent("unknown-type" as any)).toBeUndefined();
  });

  it("registerWidget 添加新类型", () => {
    registerWidget({
      type: "custom-test" as any,
      displayName: "Custom Test",
      description: "test widget",
      icon: "🧪",
      defaultSize: { w: 4, h: 3 },
      component: null as any,
      supportsDataSource: false,
      supportedDataSources: [],
    });
    expect(getWidgetConfig("custom-test" as any)).toBeDefined();
    expect(getWidgetComponent("custom-test" as any)).toBeNull();
  });

  it("registerWidgets 批量添加", () => {
    registerWidgets([
      {
        type: "batch-1" as any,
        displayName: "Batch 1",
        description: "first",
        icon: "1",
        defaultSize: { w: 1, h: 1 },
        component: null as any,
        supportsDataSource: false,
        supportedDataSources: [],
      },
      {
        type: "batch-2" as any,
        displayName: "Batch 2",
        description: "second",
        icon: "2",
        defaultSize: { w: 1, h: 1 },
        component: null as any,
        supportsDataSource: false,
        supportedDataSources: [],
      },
    ]);
    expect(getWidgetConfig("batch-1" as any)).toBeDefined();
    expect(getWidgetConfig("batch-2" as any)).toBeDefined();
  });

  it("WidgetConfig displayName 字符串", () => {
    const config = getWidgetConfig("stat-card");
    expect(typeof config?.displayName).toBe("string");
  });

  it("WidgetConfig defaultSize 形状", () => {
    const config = getWidgetConfig("stat-card");
    expect(config?.defaultSize.w).toBeGreaterThan(0);
    expect(config?.defaultSize.h).toBeGreaterThan(0);
  });

  it("WidgetConfig supportsDataSource boolean", () => {
    const config = getWidgetConfig("stat-card");
    expect(typeof config?.supportsDataSource).toBe("boolean");
  });
});
