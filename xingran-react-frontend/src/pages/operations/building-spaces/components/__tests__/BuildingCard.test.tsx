/**
 * Phase 88 Batch154 — operations/building-spaces/components/BuildingCard 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import BuildingCard from "../BuildingCard";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("BuildingCard", () => {
  it("渲染 楼宇 name + code", () => {
    const { baseElement } = render(
      <BuildingCard building={{ name: "主楼", code: "B001" } as any} onClick={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("主楼");
    expect(baseElement.textContent).toContain("B001");
  });

  it("address 存在 → 显示", () => {
    const { baseElement } = render(
      <BuildingCard
        building={{ name: "B", code: "B001", address: "北京市朝阳区" } as any}
        onClick={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("北京市朝阳区");
  });

  it("address 不存在 → 不显示地址", () => {
    const { baseElement } = render(
      <BuildingCard building={{ name: "B", code: "B001" } as any} onClick={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).not.toContain("📍");
  });

  it("显示楼层数 + 工位数", () => {
    const { baseElement } = render(
      <BuildingCard
        building={
          {
            name: "B",
            code: "B001",
            totalFloors: 10,
            workstationCount: 50,
          } as any
        }
        onClick={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("10");
    expect(baseElement.textContent).toContain("50");
    expect(baseElement.textContent).toContain("楼层数");
    expect(baseElement.textContent).toContain("工位数");
  });

  it("totalFloors undefined → 显示 0", () => {
    const { baseElement } = render(
      <BuildingCard building={{ name: "B", code: "B001" } as any} onClick={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("0");
  });

  it("点击 card → onClick 调用", () => {
    const onClick = vi.fn();
    const { baseElement } = render(
      <BuildingCard building={{ name: "B", code: "B001" } as any} onClick={onClick} />,
      { wrapper }
    );
    const card = baseElement.querySelector('[class*="buildingCard"]');
    fireEvent.click(card!);
    expect(onClick).toHaveBeenCalled();
  });
});
