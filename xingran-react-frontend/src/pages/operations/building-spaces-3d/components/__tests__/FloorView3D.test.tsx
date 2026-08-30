/**
 * Phase 88 Batch101 — building-spaces-3d/components/FloorView3D 测试(46 stmts, 0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import FloorView3D from "../FloorView3D";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  workstationApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [], total: 0 } })),
  },
}));

vi.mock("../FloorPlan3D", () => ({
  default: () => <div data-testid="floor-plan-3d" />,
}));

describe("FloorView3D 渲染", () => {
  it("基本渲染不抛错", () => {
    const { baseElement } = renderWithProviders(<FloorView3D />);
    expect(baseElement).toBeDefined();
  });

  it("buildingId prop 传入", () => {
    const { baseElement } = renderWithProviders(<FloorView3D buildingId="b1" />);
    expect(baseElement).toBeDefined();
  });
});
