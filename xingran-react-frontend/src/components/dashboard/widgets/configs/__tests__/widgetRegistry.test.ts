/**
 * Phase 88 Batch33 — dashboard widgets registry 单元测试
 *
 * widgetRegistry 集中管理 7 种 widget 配置,验证 registry 完整性 + helper 行为。
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
import type { WidgetType, WidgetConfig } from "@/types/widgets/helpers";

const ALL_TYPES: WidgetType[] = [
  "stat-card",
  "chart",
  "table",
  "list",
  "progress",
  "metric",
  "Gauge" as any, // 兼容大小写历史 type
];

describe("widgetRegistry 完整性", () => {
  it("至少 6 种内置 widget", () => {
    expect(Object.keys(widgetRegistry).length).toBeGreaterThanOrEqual(6);
  });

  it("每个内置 widget 必备字段", () => {
    for (const cfg of Object.values(widgetRegistry)) {
      expect(cfg.type).toBeTruthy();
      expect(cfg.displayName).toBeTruthy();
      expect(cfg.description).toBeTruthy();
      expect(typeof cfg.icon).toBe("string");
      expect(cfg.defaultSize.w).toBeGreaterThan(0);
      expect(cfg.defaultSize.h).toBeGreaterThan(0);
      // component 是 lazy React 组件
      expect(cfg.component).toBeDefined();
    }
  });

  it("6 种核心 type 配置齐全", () => {
    const core: WidgetType[] = ["stat-card", "chart", "table", "list", "progress", "metric"];
    for (const t of core) {
      expect(widgetRegistry[t]).toBeDefined();
      expect(widgetRegistry[t].type).toBe(t);
    }
  });
});

describe("getWidgetTypes", () => {
  it("返回所有 type 字符串", () => {
    const types = getWidgetTypes();
    expect(types.length).toBeGreaterThanOrEqual(6);
    expect(types).toContain("stat-card");
    expect(types).toContain("chart");
  });
});

describe("getWidgetConfig", () => {
  it("返回已知 type 配置", () => {
    expect(getWidgetConfig("stat-card")?.displayName).toBe("统计卡片");
  });

  it("未知 type 返 undefined", () => {
    expect(getWidgetConfig("nonexistent" as any)).toBeUndefined();
  });
});

describe("getWidgetComponent", () => {
  it("已知 type 返 lazy 组件", () => {
    const c = getWidgetComponent("stat-card");
    expect(c).toBeDefined();
    // $$typeof 是 lazy 组件的标记
    expect((c as any).$$typeof).toBeDefined();
  });

  it("未知 type 返 undefined", () => {
    expect(getWidgetComponent("nonexistent" as any)).toBeUndefined();
  });
});

describe("registerWidget / registerWidgets", () => {
  it("注册新 widget 后 getWidgetConfig 命中", () => {
    const fakeType = "test-widget-33" as any;
    const fakeCfg: WidgetConfig = {
      type: fakeType,
      displayName: "测试",
      description: "测试 widget",
      icon: "🧪",
      defaultSize: { w: 1, h: 1 },
      component: null as any,
      supportsDataSource: false,
      supportedDataSources: [],
    };
    registerWidget(fakeCfg);
    expect(getWidgetConfig(fakeType)).toBe(fakeCfg);
  });

  it("registerWidgets 批量注册", () => {
    const cfg1: WidgetConfig = {
      type: "batch-widget-1" as any,
      displayName: "批1",
      description: "x",
      icon: "x",
      defaultSize: { w: 1, h: 1 },
      component: null as any,
      supportsDataSource: false,
      supportedDataSources: [],
    };
    const cfg2: WidgetConfig = { ...cfg1, type: "batch-widget-2" as any, displayName: "批2" };
    registerWidgets([cfg1, cfg2]);
    expect(getWidgetConfig("batch-widget-1" as any)?.displayName).toBe("批1");
    expect(getWidgetConfig("batch-widget-2" as any)?.displayName).toBe("批2");
  });
});
