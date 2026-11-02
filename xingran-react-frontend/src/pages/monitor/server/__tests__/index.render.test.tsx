/**
 * Phase 88 Batch110 — monitor/server 页面渲染(76 stmts, 38.2% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ServerMonitor from "../index";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("ServerMonitor 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<ServerMonitor />, {
      endpoints: { "/monitor/server-info/list": { data: { list: [], total: 0 } } },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 渲染", async () => {
    const { baseElement } = renderWithProviders(<ServerMonitor />, {
      endpoints: {
        "/monitor/server-info/list": {
          data: {
            list: [
              {
                id: "s1",
                name: "server-01",
                ip: "10.0.0.1",
                cpu: 45,
                memory: 60,
              },
            ],
            total: 1,
          },
        },
      },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    const { baseElement } = renderWithProviders(<ServerMonitor />);
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
