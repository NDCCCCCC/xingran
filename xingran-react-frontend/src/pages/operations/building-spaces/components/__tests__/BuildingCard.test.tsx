/**
 * Phase 88 Batch326 — operations/building-spaces/components/BuildingCard 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import BuildingCard from "../BuildingCard";

vi.mock("../styles.module.css", () => ({
  default: {
    buildingCard: "buildingCard",
    buildingCardHeader: "buildingCardHeader",
    buildingIcon: "buildingIcon",
    buildingTitle: "buildingTitle",
    buildingCode: "buildingCode",
    buildingAddress: "buildingAddress",
    buildingStats: "buildingStats",
    buildingStat: "buildingStat",
    buildingStatValue: "buildingStatValue",
    buildingStatLabel: "buildingStatLabel",
  },
}));

describe("operations/building-spaces/components/BuildingCard", () => {
  const baseBuilding: any = {
    id: "b1",
    name: "A 座",
    code: "BLD-A",
    totalFloors: 12,
    workstationCount: 240,
    address: "北京海淀",
  };

  it("渲染 name + code", () => {
    render(<BuildingCard building={baseBuilding} onClick={vi.fn()} />);
    expect(screen.getByText("A 座")).toBeInTheDocument();
    expect(screen.getByText("BLD-A")).toBeInTheDocument();
  });

  it("渲染 address 含 emoji", () => {
    render(<BuildingCard building={baseBuilding} onClick={vi.fn()} />);
    expect(screen.getByText(/北京海淀/)).toBeInTheDocument();
  });

  it("无 address → 不渲染 address 行", () => {
    const { address, ...rest } = baseBuilding;
    render(<BuildingCard building={rest} onClick={vi.fn()} />);
    expect(screen.queryByText(/北京海淀/)).toBeNull();
  });

  it("渲染 totalFloors + workstationCount 默认值", () => {
    render(
      <BuildingCard
        building={{ ...baseBuilding, totalFloors: undefined, workstationCount: undefined }}
        onClick={vi.fn()}
      />
    );
    expect(screen.getAllByText("0").length).toBe(2);
  });

  it("点击触发 onClick", () => {
    const onClick = vi.fn();
    const { container } = render(<BuildingCard building={baseBuilding} onClick={onClick} />);
    const card = container.querySelector(".buildingCard") as HTMLElement;
    fireEvent.click(card);
    expect(onClick).toHaveBeenCalled();
  });

  it("楼层数/工位数 显示正确", () => {
    render(<BuildingCard building={baseBuilding} onClick={vi.fn()} />);
    expect(screen.getByText("12")).toBeInTheDocument();
    expect(screen.getByText("240")).toBeInTheDocument();
    expect(screen.getByText("楼层数")).toBeInTheDocument();
    expect(screen.getByText("工位数")).toBeInTheDocument();
  });
});
