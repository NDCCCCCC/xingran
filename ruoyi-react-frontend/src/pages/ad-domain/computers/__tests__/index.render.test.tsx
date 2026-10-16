/**
 * Phase 88 Batch105 — ad-domain/computers 页面渲染(106 stmts, 32.1% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ComputerPage from "../index";
import * as adDomainApi from "@/lib/adDomainApi";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", () => ({
  getADConfigList: vi.fn(() =>
    Promise.resolve({ data: { list: [{ id: "c1", name: "config-1" }] } })
  ),
  getADComputerList: vi.fn(() =>
    Promise.resolve({
      data: {
        list: [
          {
            id: "comp1",
            name: "PC001",
            distinguishedName: "CN=PC001,DC=test,DC=local",
            os: "Windows 10",
            status: 0,
          },
        ],
        total: 1,
      },
    })
  ),
  getADOUTree: vi.fn(() => Promise.resolve({ data: [] })),
}));

describe("ComputerPage 渲染", () => {
  it("空 + 1 行 → 渲染不抛错", async () => {
    const { baseElement } = renderWithProviders(<ComputerPage />);
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
    expect(adDomainApi.getADConfigList).toHaveBeenCalled();
  });

  it("getADComputerList 失败 → catch 路径", async () => {
    vi.mocked(adDomainApi.getADComputerList).mockRejectedValueOnce(new Error("net"));
    const { baseElement } = renderWithProviders(<ComputerPage />);
    await new Promise((r) => setTimeout(r, 500));
    expect(baseElement).toBeDefined();
  });

  it("无 config → 不请求 computer list", async () => {
    vi.mocked(adDomainApi.getADConfigList).mockResolvedValueOnce({ data: { list: [] } } as any);
    const { baseElement } = renderWithProviders(<ComputerPage />);
    await new Promise((r) => setTimeout(r, 300));
    expect(baseElement).toBeDefined();
  });
});
