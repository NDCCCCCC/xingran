/**
 * Phase 88 Batch252 — design-system/components/DensitySwitcher 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let currentDensity: any = "comfortable";
const mockSetDensity = vi.fn();

vi.mock("@/store/layoutStore", () => ({
  useLayout: () => ({ density: currentDensity, setDensity: mockSetDensity }),
}));

import DensitySwitcher from "../DensitySwitcher";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("design-system/components/DensitySwitcher", () => {
  beforeEach(() => {
    currentDensity = "comfortable";
    mockSetDensity.mockClear();
  });

  it("渲染 3 选项", () => {
    const { container } = render(<DensitySwitcher />, { wrapper });
    expect(container.querySelectorAll(".ant-segmented-item").length).toBe(3);
  });

  it("3 标签:紧凑/舒适/宽松", () => {
    render(<DensitySwitcher />, { wrapper });
    expect(screen.getByText("紧凑")).toBeInTheDocument();
    expect(screen.getByText("舒适")).toBeInTheDocument();
    expect(screen.getByText("宽松")).toBeInTheDocument();
  });

  it("点击切换 → setDensity", () => {
    render(<DensitySwitcher />, { wrapper });
    fireEvent.click(screen.getByText("紧凑"));
    expect(mockSetDensity).toHaveBeenCalledWith("compact");
  });

  it("点击宽松 → setDensity", () => {
    render(<DensitySwitcher />, { wrapper });
    fireEvent.click(screen.getByText("宽松"));
    expect(mockSetDensity).toHaveBeenCalledWith("spacious");
  });

  it("size=small", () => {
    const { container } = render(<DensitySwitcher />, { wrapper });
    expect(container.querySelector(".ant-segmented-sm")).toBeTruthy();
  });
});
