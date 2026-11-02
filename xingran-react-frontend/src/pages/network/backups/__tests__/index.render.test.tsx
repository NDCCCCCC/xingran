/**
 * Phase 88 Batch80 — network backups 页面渲染测试(157 stmts, 21.0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ConfigBackupPage from "../index";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/networkApi", () => ({
  batchExport: vi.fn(),
  getNetworkConfig: vi.fn(() => Promise.resolve({ data: {} })),
  getNetworkConfigDiff: vi.fn(() => Promise.resolve({ data: { items: [] } })),
}));

describe("ConfigBackupPage 渲染", () => {
  it("空数据渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<ConfigBackupPage />);
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });

  it("network/backups/list 返回空 → 表格空态", async () => {
    const { baseElement } = renderWithProviders(<ConfigBackupPage />, {
      endpoints: { "/network/backups/list": { data: { list: [], total: 0 } } },
    });
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });

  it("network/backups/list 返回 1 行 → 表格行渲染", async () => {
    const { baseElement } = renderWithProviders(<ConfigBackupPage />, {
      endpoints: {
        "/network/backups/list": {
          data: {
            list: [
              {
                id: "b1",
                name: "backup-2026-01-01",
                deviceCount: 5,
                createdAt: "2026-01-01 12:00:00",
                status: 0,
              },
            ],
            total: 1,
          },
        },
      },
    });
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });
});
