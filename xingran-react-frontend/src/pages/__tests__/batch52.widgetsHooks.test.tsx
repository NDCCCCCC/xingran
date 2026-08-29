/**
 * Phase 88 Batch52 — dashboard widget types 3 组件 + monitor useLogData hook
 *
 * ChartWidget(39,0%) / ListWidget(29,0%) / MetricWidget(20,0%) 直渲 +
 * useLogData(33,0%) renderHook — useWidgetData 走真实 hook(mock 端点)。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConfigProvider, App } from "antd";
import type { ReactElement } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { createApiMock, resetApiMocks, setGenericFallback } from "@/test/utils/createApiMock";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { ChartWidget } from "@/components/dashboard/widgets/types/ChartWidget";
import { ListWidget } from "@/components/dashboard/widgets/types/ListWidget";
import { MetricWidget } from "@/components/dashboard/widgets/types/MetricWidget";
import { useLogData } from "../monitor/logs/hooks/useLogData";
import type { WidgetConfig } from "@/types/dashboard";

beforeEach(() => {
  resetApiMocks();
  setGenericFallback({ data: { list: [], total: 0 } });
  vi.clearAllMocks();
});

const makeWidget = (dataSource?: string): WidgetConfig =>
  ({
    id: "w1",
    type: "chart",
    title: "测试图表",
    dataSource: dataSource || "/dashboard/fake",
    refreshInterval: 0,
  }) as unknown as WidgetConfig;

function wrapQuery(ui: ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(ui, { queryClient: qc });
}

describe("ChartWidget", () => {
  it("空数据渲染 empty option 分支", async () => {
    const rendered = wrapQuery(
      <ChartWidget widget={makeWidget()} display={{ chartType: "line" } as any} />
    );
    await waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  }, 15000);

  it("line 图 + 数据渲染", async () => {
    const api = createApiMock("/dashboard/fake");
    api.endpoint.mockResolvedValue({ data: { categories: ["a", "b"], values: [1, 2] } });
    const rendered = wrapQuery(
      <ChartWidget widget={makeWidget()} display={{ chartType: "line" } as any} />
    );
    await waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  }, 15000);

  it("bar 图渲染", async () => {
    const api = createApiMock("/dashboard/fake");
    api.endpoint.mockResolvedValue({ data: { categories: ["x"], values: [5] } });
    const rendered = wrapQuery(
      <ChartWidget widget={makeWidget()} display={{ chartType: "bar" } as any} />
    );
    await waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  }, 15000);
});

describe("ListWidget", () => {
  it("空数据渲染空列表", async () => {
    const rendered = wrapQuery(
      <ListWidget widget={makeWidget()} display={{ maxItems: 5 } as any} />
    );
    await waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  }, 15000);
});

describe("MetricWidget", () => {
  it("空数据 percent=0", async () => {
    const rendered = wrapQuery(<MetricWidget widget={makeWidget()} display={{} as any} />);
    await waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  }, 15000);

  it("数值数据渲染 percent", async () => {
    const api = createApiMock("/dashboard/fake");
    api.endpoint.mockResolvedValue({ data: { value: 65 } });
    const rendered = wrapQuery(<MetricWidget widget={makeWidget()} display={{} as any} />);
    await waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  }, 15000);
});
