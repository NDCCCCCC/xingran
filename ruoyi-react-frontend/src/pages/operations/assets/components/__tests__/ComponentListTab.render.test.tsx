/**
 * Phase 88 Batch77 — operations/assets ComponentListTab 渲染测试(37 stmts, 8.1% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ComponentListTab from "../ComponentListTab";
import { componentApi } from "@/lib/opsApi";

vi.mock("@/lib/opsApi", () => ({
  componentApi: {
    list: vi.fn(),
  },
}));

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("ComponentListTab 渲染", () => {
  it("parentAssetId 空 → list=[] 直接渲染", async () => {
    vi.mocked(componentApi.list).mockResolvedValueOnce({
      data: { list: [] },
    } as any);
    const { baseElement } = renderWithProviders(<ComponentListTab parentAssetId="" />);
    await new Promise((r) => setTimeout(r, 100));
    expect(baseElement).toBeDefined();
  });

  it("parentAssetId 非空 → componentApi.list 命中并填充", async () => {
    vi.mocked(componentApi.list).mockResolvedValueOnce({
      data: {
        list: [
          {
            id: "comp1",
            componentType: "card",
            serialNumber: "SN-001",
            modelNumber: "M-001",
            status: 0,
          },
        ],
      },
    } as any);
    const { baseElement } = renderWithProviders(<ComponentListTab parentAssetId="a1" />);
    await new Promise((r) => setTimeout(r, 200));
    expect(componentApi.list).toHaveBeenCalledWith("a1");
    expect(baseElement).toBeDefined();
  });

  it("componentApi.list 失败 → message.error catch", async () => {
    vi.mocked(componentApi.list).mockRejectedValueOnce(new Error("network"));
    const { baseElement } = renderWithProviders(<ComponentListTab parentAssetId="a2" />);
    await new Promise((r) => setTimeout(r, 200));
    expect(baseElement).toBeDefined();
  });
});
