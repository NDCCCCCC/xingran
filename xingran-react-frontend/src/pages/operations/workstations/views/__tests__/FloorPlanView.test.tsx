/**
 * Phase 88 Batch412 — pages/operations/workstations/views/FloorPlanView 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { WorkstationFloorPlanView } from "../FloorPlanView";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/shared/FloorPlanEditor", () => ({
  default: () => <div data-testid="floor-plan-editor" />,
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("WorkstationFloorPlanView", () => {
  it("基础渲染不抛错", () => {
    expect(() =>
      render(
        <WorkstationFloorPlanView
          selectedFloorForPlan="floor1"
          floorOptions={[{ value: "floor1", label: "F1" }]}
          floorPlanWorkstations={[]}
          allWorkstations={[]}
          onFloorChange={vi.fn()}
          onPositionUpdate={vi.fn()}
          onEdit={vi.fn()}
          onCloseFloorPlan={vi.fn()}
        />,
        { wrapper }
      )
    ).not.toThrow();
  });
});
