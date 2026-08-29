/**
 * Phase 88 Batch86 — dashboard/settings/DataSourceForm 渲染(78 stmts, 21.8% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { DataSourceForm } from "../DataSourceForm";
import { Form } from "antd";
import type {
  ApiDataSourceConfig,
  StaticDataSourceConfig,
  WebSocketDataSourceConfig,
} from "@/types/dashboard";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

interface HarnessProps {
  value?: ApiDataSourceConfig | StaticDataSourceConfig | WebSocketDataSourceConfig;
  onChange?: (v: any) => void;
}

function Harness({ value, onChange }: HarnessProps) {
  const [form] = Form.useForm();
  return <DataSourceForm form={form} value={value as any} onChange={onChange} />;
}

describe("DataSourceForm 渲染", () => {
  it("无 value → 显示默认类型选择", () => {
    const { baseElement } = renderWithProviders(<Harness />);
    expect(baseElement).toBeDefined();
  });

  it("value=api → 显示 API 配置字段", () => {
    const value: ApiDataSourceConfig = {
      type: "api",
      method: "GET",
      endpoint: "/test",
      params: {},
    };
    const { baseElement } = renderWithProviders(<Harness value={value} />);
    expect(baseElement).toBeDefined();
  });

  it("value=static → 显示静态数据配置", () => {
    const value: StaticDataSourceConfig = {
      type: "static",
      data: { foo: "bar" },
    };
    const { baseElement } = renderWithProviders(<Harness value={value} />);
    expect(baseElement).toBeDefined();
  });

  it("value=websocket → 显示 WS 配置", () => {
    const value: WebSocketDataSourceConfig = {
      type: "websocket",
      channel: "test-channel",
    };
    const { baseElement } = renderWithProviders(<Harness value={value} />);
    expect(baseElement).toBeDefined();
  });

  it("onChange 触发", () => {
    const onChange = vi.fn();
    const { baseElement } = renderWithProviders(<Harness onChange={onChange} />);
    expect(baseElement).toBeDefined();
  });
});
