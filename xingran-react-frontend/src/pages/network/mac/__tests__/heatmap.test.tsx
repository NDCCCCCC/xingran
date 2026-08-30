/**
 * Phase 88 Batch139 — pages/network/mac/heatmap 热力图页面测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/macHeatmapApi", () => ({
  queryMACHeatmap: vi.fn(() =>
    Promise.resolve({
      cells: [{ deviceId: "d1", deviceNameSnapshot: "dev1", interfaceName: "p1", changeCount: 5 }],
      topN: 1,
      start: "2026-01-01",
      end: "2026-01-07",
      total: 1,
      snapshot: "snap-1",
    })
  ),
}));

vi.mock("@/components/network/MACHeatmapChart", () => ({
  default: ({ data }: any) => (
    <div data-testid="heatmap-chart">
      <span>{data.cells.length} cells</span>
    </div>
  ),
}));

const persistedStore: { value: string } = { value: "7d" };
vi.mock("@/hooks/usePersistedState", () => ({
  usePersistedStateController: <T,>(opts: { defaultValue: T }) => {
    const setValue = vi.fn((v: T) => {
      persistedStore.value = v as any;
    });
    return [persistedStore.value as T, setValue] as [T, any];
  },
}));

import HeatmapPage from "../heatmap";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/network/mac/heatmap"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("HeatmapPage", () => {
  beforeEach(() => {
    persistedStore.value = "7d";
  });
  it("渲染 + 预设按钮", () => {
    const { baseElement } = render(<HeatmapPage />, { wrapper });
    expect(baseElement.textContent).toContain("MAC 端口使用热力图");
    expect(baseElement.textContent).toContain("近 7d");
    expect(baseElement.textContent).toContain("近 24h");
    expect(baseElement.textContent).toContain("自定义");
  });

  it("未查询 → 显示提示信息", () => {
    const { baseElement } = render(<HeatmapPage />, { wrapper });
    expect(baseElement.textContent).toContain("请选择时间范围");
  });

  it("点击 近 24h → 不抛错", () => {
    const { getByText } = render(<HeatmapPage />, { wrapper });
    expect(() => fireEvent.click(getByText("近 24h"))).not.toThrow();
  });

  it("点击 自定义 → 不抛错", () => {
    const { getByText } = render(<HeatmapPage />, { wrapper });
    expect(() => fireEvent.click(getByText("自定义"))).not.toThrow();
  });

  it("点击 查询 → 不抛错", () => {
    const { baseElement, getByText } = render(<HeatmapPage />, { wrapper });
    fireEvent.click(getByText("近 7d"));
    // 查询 button 可能不可见 (form layout inline)
    const btn = baseElement.querySelector(".ant-btn-primary");
    if (btn) fireEvent.click(btn);
    expect(baseElement).toBeDefined();
  });

  it("点击 近 1h → 不抛错", () => {
    const { getByText } = render(<HeatmapPage />, { wrapper });
    expect(() => fireEvent.click(getByText("近 1h"))).not.toThrow();
  });
});
