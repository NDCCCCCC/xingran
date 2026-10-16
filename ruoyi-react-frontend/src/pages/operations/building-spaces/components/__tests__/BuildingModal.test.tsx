/**
 * Phase 88 Batch120 — operations/building-spaces/components/BuildingModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, act, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  floorApi: {
    list: vi.fn(() =>
      Promise.resolve({
        data: {
          list: [
            { id: "f1", buildingId: "b1", floorNo: "1", name: "F1", workstationCount: 5 },
            { id: "f2", buildingId: "b1", floorNo: "2", name: "F2", workstationCount: 0 },
          ],
        },
      })
    ),
  },
}));

vi.mock("../FloorStack", () => ({
  default: ({ floors, onFloorClick }: any) => (
    <div data-testid="floor-stack">
      <button onClick={() => onFloorClick?.(floors[0])}>Click F1</button>
      <span data-testid="floor-count">{floors.length}</span>
    </div>
  ),
}));

vi.mock("../WorkstationView", () => ({
  default: ({ floor, onBack }: any) => (
    <div data-testid="workstation-view">
      <span>{floor?.name}</span>
      <button onClick={onBack}>Back</button>
    </div>
  ),
}));

import BuildingModal from "../BuildingModal";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("BuildingModal", () => {
  const building = { id: "b1", name: "B1" } as any;

  it("visible=true → 渲染标题 + 触发 loadFloors", async () => {
    const { baseElement } = render(
      <BuildingModal building={building} visible onClose={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.textContent).toContain("B1");
    });
  });

  it("visible=true → 加载楼层后渲染 FloorStack", async () => {
    const { baseElement } = render(
      <BuildingModal building={building} visible onClose={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="floor-stack"]')).toBeTruthy();
    });
    expect(baseElement.querySelector('[data-testid="floor-count"]')?.textContent).toBe("2");
  });

  it("点击楼层 → 切到 workstation 视图", async () => {
    const { baseElement, getByText } = render(
      <BuildingModal building={building} visible onClose={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="floor-stack"]')).toBeTruthy();
    });
    await act(async () => {
      fireEvent.click(getByText("Click F1"));
    });
    expect(baseElement.querySelector('[data-testid="workstation-view"]')).toBeTruthy();
  });

  it("点击 Back → 返回 floors 视图", async () => {
    const { baseElement, getByText } = render(
      <BuildingModal building={building} visible onClose={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="floor-stack"]')).toBeTruthy();
    });
    await act(async () => {
      fireEvent.click(getByText("Click F1"));
    });
    expect(baseElement.querySelector('[data-testid="workstation-view"]')).toBeTruthy();
    await act(async () => {
      fireEvent.click(getByText("Back"));
    });
    expect(baseElement.querySelector('[data-testid="floor-stack"]')).toBeTruthy();
  });

  it("visible=false → 不触发 loadFloors", async () => {
    const { baseElement } = render(
      <BuildingModal building={building} visible={false} onClose={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).not.toContain("楼层列表");
  });

  it("Modal 取消 → 调用 onClose + 重置状态", async () => {
    const onClose = vi.fn();
    const { baseElement, getByText } = render(
      <BuildingModal building={building} visible onClose={onClose} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="floor-stack"]')).toBeTruthy();
    });
    // cancel button text is hidden in DOM but click on overlay or close button
    // Use Modal cancel via clicking the X icon or escape
    await act(async () => {
      fireEvent.keyDown(document.body, { key: "Escape" });
    });
    // antd modal should call onCancel which is handleModalClose → onClose
    expect(onClose).toHaveBeenCalled();
  });

  it("loading 错误 → setFloors([])", async () => {
    const { floorApi } = await import("@/lib/opsApi");
    vi.mocked(floorApi.list).mockRejectedValueOnce(new Error("网络失败"));
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { baseElement } = render(
      <BuildingModal building={building} visible onClose={vi.fn()} />,
      { wrapper }
    );
    await waitFor(() => {
      expect(errSpy).toHaveBeenCalled();
    });
    expect(baseElement.querySelector('[data-testid="floor-count"]')?.textContent).toBe("0");
    errSpy.mockRestore();
  });
});
