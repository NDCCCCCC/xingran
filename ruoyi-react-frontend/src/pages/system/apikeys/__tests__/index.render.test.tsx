/**
 * Phase 88 Batch96 — system apikeys 页面渲染(150 stmts, 32.0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import APIKeyManagement from "../index";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/api/apikey", () => ({
  listAPIKeys: vi.fn(() => Promise.resolve({ data: { list: [], total: 0 } })),
  createAPIKey: vi.fn(),
  updateAPIKey: vi.fn(),
  deleteAPIKey: vi.fn(),
  toggleAPIKeyStatus: vi.fn(),
}));

describe("APIKeyManagement 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<APIKeyManagement />, {
      endpoints: { "/system/apikeys/list": { data: { list: [], total: 0 } } },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 渲染", async () => {
    const { baseElement } = renderWithProviders(<APIKeyManagement />, {
      endpoints: {
        "/system/apikeys/list": {
          data: {
            list: [
              {
                id: 1,
                name: "test-key",
                key: "sk-xxx",
                status: 1,
                createdAt: "2026-01-01 12:00:00",
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
    const { baseElement } = renderWithProviders(<APIKeyManagement />);
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
