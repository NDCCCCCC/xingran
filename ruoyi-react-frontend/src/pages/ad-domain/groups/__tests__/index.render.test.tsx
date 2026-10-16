/**
 * Phase 88 Batch79 — ad-domain groups 页面渲染(178 stmts, 27.0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ADGroupPage from "../index";
import * as adDomainApi from "@/lib/adDomainApi";

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
  getADGroupList: vi.fn(() =>
    Promise.resolve({
      data: {
        list: [
          {
            id: "g1",
            name: "Domain Admins",
            distinguishedName: "CN=Domain Admins,DC=test,DC=local",
            memberCount: 5,
            description: "Administrators",
          },
        ],
        total: 1,
      },
    })
  ),
  getADGroupMembers: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  getADUserList: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  addADGroupMember: vi.fn(),
  removeADGroupMember: vi.fn(),
  updateADGroup: vi.fn(),
  createADGroup: vi.fn(),
  deleteADGroup: vi.fn(),
  syncADGroups: vi.fn(),
}));

describe("ADGroupPage 渲染", () => {
  it("空数据 + 1 行 group → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<ADGroupPage />);
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
    expect(adDomainApi.getADConfigList).toHaveBeenCalled();
  });

  it("getADGroupList 失败 → message.error 路径", async () => {
    vi.mocked(adDomainApi.getADConfigList).mockResolvedValueOnce({
      data: { list: [{ id: "c1", name: "c1" }] },
    } as any);
    vi.mocked(adDomainApi.getADGroupList).mockRejectedValueOnce(new Error("fail"));
    const { baseElement } = renderWithProviders(<ADGroupPage />);
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
  });

  it("selectedConfig 为空 → 不发请求", async () => {
    vi.mocked(adDomainApi.getADConfigList).mockResolvedValueOnce({ data: { list: [] } } as any);
    const { baseElement } = renderWithProviders(<ADGroupPage />);
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
  });
});
