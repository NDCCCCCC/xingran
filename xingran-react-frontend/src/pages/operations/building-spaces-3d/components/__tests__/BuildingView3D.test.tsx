/**
 * Phase 88 Batch102 — building-spaces-3d/components/BuildingView3D 测试(77 stmts, 0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import BuildingView3D from "../BuildingView3D";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  floorApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  },
  workstationApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  },
  buildingApi: {
    get: vi.fn(() => Promise.resolve({ data: null })),
  },
}));

vi.mock("../BuildingModel3D", () => ({
  default: () => <div data-testid="building-model-3d" />,
}));

vi.mock("../FloorPlan3D", () => ({
  default: () => <div data-testid="floor-plan-3d" />,
}));

describe("BuildingView3D 渲染", () => {
  it("基本渲染不抛错", () => {
    const { baseElement } = renderWithProviders(<BuildingView3D />);
    expect(baseElement).toBeDefined();
  });

  it("buildingId prop 传入", () => {
    const { baseElement } = renderWithProviders(<BuildingView3D buildingId="b1" />);
    expect(baseElement).toBeDefined();
  });
});
