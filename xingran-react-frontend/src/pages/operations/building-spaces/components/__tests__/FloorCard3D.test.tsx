/**
 * Phase 88 Batch289 — pages/operations/building-spaces/components/FloorCard3D 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("../styles.module.css", () => ({
  default: {
    floorCard: "fc",
    selected: "sel",
    floorCardNumber: "n",
    floorCardName: "nm",
    floorCardStats: "st",
  },
}));

import FloorCard3D from "../FloorCard3D";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const sampleFloor: any = {
  id: "f1",
  buildingId: "b1",
  floorNo: "F1",
  name: "Floor 1",
  workstationCount: 10,
};

describe("operations/building-spaces/FloorCard3D", () => {
  it("渲染 floor 名称 + 编号 + 工位数", () => {
    render(<FloorCard3D floor={sampleFloor} index={0} isSelected={false} onClick={vi.fn()} />, {
      wrapper,
    });
    expect(screen.getByText("F1")).toBeInTheDocument();
    expect(screen.getByText("Floor 1")).toBeInTheDocument();
    expect(screen.getByText("10 个工位")).toBeInTheDocument();
  });

  it("workstationCount undefined → 0 工位", () => {
    const f = { ...sampleFloor, workstationCount: undefined };
    render(<FloorCard3D floor={f} index={0} isSelected={false} onClick={vi.fn()} />, { wrapper });
    expect(screen.getByText("0 个工位")).toBeInTheDocument();
  });

  it("data-index + data-floor 属性", () => {
    const { container } = render(
      <FloorCard3D floor={sampleFloor} index={2} isSelected={false} onClick={vi.fn()} />,
      { wrapper }
    );
    const div = container.querySelector("[data-index='2']");
    expect(div).toBeTruthy();
    expect(div?.getAttribute("data-floor")).toBe("F1");
  });

  it("isSelected → selected class", () => {
    const { container } = render(
      <FloorCard3D floor={sampleFloor} index={0} isSelected={true} onClick={vi.fn()} />,
      { wrapper }
    );
    expect(container.innerHTML).toContain("sel");
  });

  it("点击 → onClick 调用", () => {
    const onClick = vi.fn();
    const { container } = render(
      <FloorCard3D floor={sampleFloor} index={0} isSelected={false} onClick={onClick} />,
      { wrapper }
    );
    const div = container.querySelector("[data-index='0']")!;
    fireEvent.click(div);
    expect(onClick).toHaveBeenCalled();
  });
});
