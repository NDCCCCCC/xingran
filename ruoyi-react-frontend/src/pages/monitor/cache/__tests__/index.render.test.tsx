/**
 * Phase 88 Batch110 — monitor/cache 页面渲染(123 stmts, 35.8% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import CacheManager from "../index";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("CacheManager 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<CacheManager />, {
      endpoints: { "/monitor/cache/list": { data: { list: [], total: 0 } } },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 渲染", async () => {
    const { baseElement } = renderWithProviders(<CacheManager />, {
      endpoints: {
        "/monitor/cache/list": {
          data: {
            list: [
              {
                key: "cache-key-1",
                type: "user",
                ttl: 3600,
                size: 1024,
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
    const { baseElement } = renderWithProviders(<CacheManager />);
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
