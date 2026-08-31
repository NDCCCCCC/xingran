/**
 * Phase 88 Batch254 — pages/operations/building-spaces/components/FloorStack 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../FloorCard3D", () => ({
  default: ({ floor, onClick, isSelected }: any) => (
    <div data-testid={`floor-card-${floor.id}`} data-selected={isSelected} onClick={onClick}>
      {floor.name}
    </div>
  ),
}));

vi.mock("../styles.module.css", () => ({ default: {} }));

import FloorStack from "../FloorStack";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const sampleFloors: any[] = [
  { id: "f1", name: "Floor 1", buildingId: "b1", floorNo: "F1", workstationCount: 10 },
  { id: "f2", name: "Floor 2", buildingId: "b1", floorNo: "F2", workstationCount: 20 },
];

describe("operations/building-spaces/FloorStack", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  it("空 floors → empty state", () => {
    render(<FloorStack floors={[]} onFloorClick={vi.fn()} />, { wrapper });
    expect(screen.getByText("暂无楼层数据")).toBeInTheDocument();
  });

  it("渲染 floors", () => {
    render(<FloorStack floors={sampleFloors} onFloorClick={vi.fn()} />, { wrapper });
    expect(screen.getByTestId("floor-card-f1")).toBeInTheDocument();
    expect(screen.getByTestId("floor-card-f2")).toBeInTheDocument();
  });

  it("点击 floor → 延迟后 onFloorClick", () => {
    const onFloorClick = vi.fn();
    render(<FloorStack floors={sampleFloors} onFloorClick={onFloorClick} />, { wrapper });
    fireEvent.click(screen.getByTestId("floor-card-f1"));
    // selectedFloorId 立即设置
    expect(screen.getByTestId("floor-card-f1").getAttribute("data-selected")).toBe("true");
    // 300ms 后调用 onFloorClick
    act(() => {
      vi.advanceTimersByTime(350);
    });
    expect(onFloorClick).toHaveBeenCalledWith(sampleFloors[0]);
    vi.useRealTimers();
  });

  it("onFloorClick 后 selectedFloorId 重置为 null", () => {
    const onFloorClick = vi.fn();
    render(<FloorStack floors={sampleFloors} onFloorClick={onFloorClick} />, { wrapper });
    fireEvent.click(screen.getByTestId("floor-card-f2"));
    act(() => {
      vi.advanceTimersByTime(350);
    });
    expect(screen.getByTestId("floor-card-f2").getAttribute("data-selected")).toBe("false");
    vi.useRealTimers();
  });
});
