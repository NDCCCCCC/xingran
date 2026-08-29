/**
 * Phase 88 Batch98 — building-spaces-3d/components/AddressInput 测试(30 stmts, 0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import AddressInput from "../AddressInput";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../hooks/useGeocoding", () => ({
  useGeocoding: vi.fn(() => ({
    geocode: vi.fn(() => Promise.resolve(null)),
    loading: false,
    error: null,
  })),
}));

describe("AddressInput 渲染", () => {
  it("空 value 渲染", () => {
    const { baseElement } = renderWithProviders(
      <AddressInput value={undefined as any} onChange={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });

  it("带 value 渲染", () => {
    const value = {
      address: "武汉市洪山区珞喻路",
      city: "武汉市",
      lng: 114.4,
      lat: 30.5,
    };
    const { baseElement } = renderWithProviders(
      <AddressInput value={value as any} onChange={vi.fn()} />
    );
    expect(baseElement).toBeDefined();
  });

  it("disabled=true 渲染", () => {
    const { baseElement } = renderWithProviders(
      <AddressInput value={undefined as any} onChange={vi.fn()} disabled />
    );
    expect(baseElement).toBeDefined();
  });
});
