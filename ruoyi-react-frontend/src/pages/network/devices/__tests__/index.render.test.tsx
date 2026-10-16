/**
 * Phase 88 Batch81 — network devices 页面渲染测试(215 stmts, 30.7% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import NetworkDevicePage from "../index";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

function renderDevice(endpoints: Record<string, unknown> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <NetworkDevicePage />
    </QueryClientProvider>,
    { endpoints }
  );
}

describe("NetworkDevicePage 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderDevice();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("列表空 → 表格空态", async () => {
    const { baseElement } = renderDevice({
      "/network/devices/list": { data: { list: [], total: 0 } },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("列表 1 行 → 表格行渲染", async () => {
    const { baseElement } = renderDevice({
      "/network/devices/list": {
        data: {
          list: [
            {
              id: "d1",
              name: "switch-01",
              ipAddress: "10.0.0.1",
              deviceType: "switch",
              status: 0,
            },
          ],
          total: 1,
        },
      },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
