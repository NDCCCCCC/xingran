/**
 * Phase 88 Batch251 — design-system/components/LayoutSwitcher 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let currentLayout = "classic";
const mockSetLayout = vi.fn();

vi.mock("@/store/layoutStore", () => ({
  useLayout: () => ({ layout: currentLayout, setLayout: mockSetLayout }),
}));

import LayoutSwitcher from "../LayoutSwitcher";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("design-system/components/LayoutSwitcher", () => {
  it("渲染 3 选项", () => {
    const { container } = render(<LayoutSwitcher />, { wrapper });
    expect(container.querySelectorAll(".ant-segmented-item").length).toBe(3);
  });

  it("经典 tooltip 文本存在 (data attr)", () => {
    const { container } = render(<LayoutSwitcher />, { wrapper });
    // Tooltip 包装的 span 文本 "经典" 可见
    expect(container.textContent).toContain("经典");
  });

  it("混合 + 创新 标签", () => {
    const { container } = render(<LayoutSwitcher />, { wrapper });
    expect(container.textContent).toContain("混合");
    expect(container.textContent).toContain("创新");
  });

  it("点击切换 → setLayout", () => {
    mockSetLayout.mockClear();
    render(<LayoutSwitcher />, { wrapper });
    const items = screen.getAllByRole("radio");
    fireEvent.click(items[1]); // 混合
    expect(mockSetLayout).toHaveBeenCalledWith("hybrid");
  });
});
