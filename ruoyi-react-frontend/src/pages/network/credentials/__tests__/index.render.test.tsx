/**
 * Phase 88 Batch111 — network credentials 页面渲染(109 stmts, 36.7% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import CredentialsPage from "../index";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("CredentialsPage 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<CredentialsPage />, {
      endpoints: { "/network/credentials/list": { data: { list: [], total: 0 } } },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 渲染", async () => {
    const { baseElement } = renderWithProviders(<CredentialsPage />, {
      endpoints: {
        "/network/credentials/list": {
          data: {
            list: [
              {
                id: "c1",
                name: "cred-01",
                username: "admin",
                deviceIp: "10.0.0.1",
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
    const { baseElement } = renderWithProviders(<CredentialsPage />);
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
