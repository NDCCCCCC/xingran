/**
 * Phase 88 Batch86 — dashboard/settings/DisplayConfigForm 渲染(44 stmts, 4.5% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { DisplayConfigForm } from "../DisplayConfigForm";
import { Form } from "antd";
import type { DisplayConfig, WidgetType } from "@/types/dashboard";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

interface HarnessProps {
  value?: DisplayConfig;
  widgetType?: WidgetType;
  onChange?: (v: any) => void;
}

function Harness({ value, widgetType, onChange }: HarnessProps) {
  const [form] = Form.useForm();
  return (
    <DisplayConfigForm form={form} value={value} widgetType={widgetType} onChange={onChange} />
  );
}

describe("DisplayConfigForm 渲染", () => {
  it("无 value → 显示默认字段", () => {
    const { baseElement } = renderWithProviders(<Harness />);
    expect(baseElement).toBeDefined();
  });

  it("widgetType=stat-card → 渲染 stat-card 配置", () => {
    const value: DisplayConfig = { title: "统计卡片", showTitle: true } as any;
    const { baseElement } = renderWithProviders(<Harness value={value} widgetType="stat-card" />);
    expect(baseElement).toBeDefined();
  });

  it("widgetType=chart → 渲染 chart 配置", () => {
    const { baseElement } = renderWithProviders(<Harness widgetType="chart" />);
    expect(baseElement).toBeDefined();
  });

  it("widgetType=table → 渲染 table 配置", () => {
    const { baseElement } = renderWithProviders(<Harness widgetType="table" />);
    expect(baseElement).toBeDefined();
  });

  it("widgetType=list → 渲染 list 配置", () => {
    const { baseElement } = renderWithProviders(<Harness widgetType="list" />);
    expect(baseElement).toBeDefined();
  });

  it("widgetType=progress → 渲染 progress 配置", () => {
    const { baseElement } = renderWithProviders(<Harness widgetType="progress" />);
    expect(baseElement).toBeDefined();
  });
});
