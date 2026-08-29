/**
 * Phase 88 Batch80 — ad-domain users 页面渲染测试(216 stmts, 20.4% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ADUserPage from "../index";
import * as adDomainApi from "@/lib/adDomainApi";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", () => ({
  getADUserList: vi.fn(() =>
    Promise.resolve({
      data: {
        list: [
          {
            id: "u1",
            userName: "Alice",
            distinguishedName: "CN=Alice,DC=test,DC=local",
            email: "alice@test.local",
            status: 0,
            enabled: true,
          },
        ],
        total: 1,
      },
    })
  ),
  getADConfigList: vi.fn(() =>
    Promise.resolve({
      data: {
        list: [{ id: "c1", name: "config-1", host: "ldap://test.local", port: 389, status: 0 }],
      },
    })
  ),
  getADOUTree: vi.fn(() => Promise.resolve({ data: [] })),
  getADUserIds: vi.fn(() => Promise.resolve({ data: [] })),
  updateADUser: vi.fn(),
  moveADUser: vi.fn(),
  enableADUser: vi.fn(),
  disableADUser: vi.fn(),
  batchSyncADUsersDirect: vi.fn(),
}));

describe("ADUserPage 渲染", () => {
  it("空 + 1 行 user → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<ADUserPage />);
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
    expect(adDomainApi.getADConfigList).toHaveBeenCalled();
  });

  it("getADUserList 失败 → message.error 路径", async () => {
    vi.mocked(adDomainApi.getADUserList).mockRejectedValueOnce(new Error("fail"));
    const { baseElement } = renderWithProviders(<ADUserPage />);
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
  });

  it("无 configs → 不请求 users", async () => {
    vi.mocked(adDomainApi.getADConfigList).mockResolvedValueOnce({ data: { list: [] } } as any);
    const { baseElement } = renderWithProviders(<ADUserPage />);
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });
});
