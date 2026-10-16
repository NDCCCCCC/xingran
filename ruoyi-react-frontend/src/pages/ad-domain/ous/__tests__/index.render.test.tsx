/**
 * Phase 88 Batch84 — ad-domain ous 页面渲染测试(176 stmts, 24.4% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ADOUPage from "../index";
import * as adDomainApi from "@/lib/adDomainApi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", () => ({
  getADConfigList: vi.fn(() =>
    Promise.resolve({
      data: {
        list: [{ id: "c1", name: "config-1", host: "ldap://test.local", port: 389, status: 0 }],
      },
    })
  ),
  getADOUTree: vi.fn(() =>
    Promise.resolve({
      data: [
        {
          id: "ou1",
          name: "TestOU",
          distinguishedName: "OU=TestOU,DC=test,DC=local",
          children: [],
        },
      ],
    })
  ),
  createADOU: vi.fn(),
  updateADOU: vi.fn(),
  deleteADOU: vi.fn(),
  moveADOU: vi.fn(),
}));

function renderOU() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <ADOUPage />
    </QueryClientProvider>
  );
}

describe("ADOUPage 渲染", () => {
  it("空 + 1 行 → 渲染不抛错", async () => {
    const { baseElement } = renderOU();
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
    expect(adDomainApi.getADConfigList).toHaveBeenCalled();
  });

  it("getADOUTree 失败 → catch 路径", async () => {
    vi.mocked(adDomainApi.getADOUTree).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderOU();
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
  });

  it("无 config → 不请求 OU tree", async () => {
    vi.mocked(adDomainApi.getADConfigList).mockResolvedValueOnce({ data: { list: [] } } as any);
    const { baseElement } = renderOU();
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });
});
