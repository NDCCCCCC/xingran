/**
 * Phase 88 Batch195 — components/three/BuildingScene lazy wrapper 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/pages/operations/building-spaces-3d/components/BuildingModel3D", () => ({
  default: (props: any) => <div data-testid="model3d">Model3D {props.id || ""}</div>,
}));
vi.mock("@/pages/operations/building-spaces-3d/components/FloorPlan3D", () => ({
  default: (props: any) => <div data-testid="floorplan3d">FloorPlan3D {props.id || ""}</div>,
}));
vi.mock("@/pages/operations/building-spaces-3d/components/FloorView3D", () => ({
  default: () => <div data-testid="floorview3d">FloorView3D</div>,
}));
vi.mock("@/pages/operations/building-spaces-3d/components/BuildingView3D", () => ({
  default: () => <div data-testid="buildingview3d">BuildingView3D</div>,
}));

import {
  BuildingModel3DLazy,
  FloorPlan3DLazy,
  FloorView3DLazy,
  BuildingView3DLazy,
} from "../BuildingScene";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("three/BuildingScene lazy wrappers", () => {
  it("BuildingModel3DLazy → 渲染 Model3D", async () => {
    render(<BuildingModel3DLazy id="b1" />, { wrapper });
    await waitFor(() => {
      expect(screen.getByTestId("model3d")).toBeInTheDocument();
    });
  });

  it("FloorPlan3DLazy → 渲染 FloorPlan3D", async () => {
    render(<FloorPlan3DLazy id="f1" />, { wrapper });
    await waitFor(() => {
      expect(screen.getByTestId("floorplan3d")).toBeInTheDocument();
    });
  });

  it("FloorView3DLazy → 渲染 FloorView3D", async () => {
    render(<FloorView3DLazy />, { wrapper });
    await waitFor(() => {
      expect(screen.getByTestId("floorview3d")).toBeInTheDocument();
    });
  });

  it("BuildingView3DLazy → 渲染 BuildingView3D", async () => {
    render(<BuildingView3DLazy />, { wrapper });
    await waitFor(() => {
      expect(screen.getByTestId("buildingview3d")).toBeInTheDocument();
    });
  });
});
