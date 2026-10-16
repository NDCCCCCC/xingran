/**
 * Phase 88 Batch138 — components/reconciliation/HealthBadge 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useDict", () => ({
  useDict: vi.fn(() => ({
    data: [
      { dictValue: "B", listClass: "warning" },
      { dictValue: "C", listClass: "error" },
    ],
  })),
}));

let mockVisible = true;
vi.mock("../hooks/useReconciliationVisibility", () => ({
  useReconciliationVisibility: () => mockVisible,
}));

import { HealthBadge } from "../HealthBadge";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("HealthBadge", () => {
  beforeEach(() => {
    mockVisible = true;
  });

  it("conflictType=null → 健康无 Tooltip", () => {
    const onClick = vi.fn();
    const { baseElement } = render(
      <HealthBadge assetId="a1" conflictType={null} onClick={onClick} />,
      { wrapper }
    );
    const dot = baseElement.querySelector('[role="img"]');
    expect(dot).toBeTruthy();
    expect(baseElement.querySelector(".ant-tooltip")).toBeNull();
  });

  it("conflictType='B' → 有 Tooltip + 点击触发 onClick", () => {
    const onClick = vi.fn();
    const { baseElement } = render(
      <HealthBadge assetId="a1" conflictType="B" onClick={onClick} />,
      { wrapper }
    );
    const dot = baseElement.querySelector('[role="button"]');
    expect(dot).toBeTruthy();
    fireEvent.click(dot!);
    expect(onClick).toHaveBeenCalledWith("a1", "B");
  });

  it("Enter 键 → 触发 onClick", () => {
    const onClick = vi.fn();
    const { baseElement } = render(
      <HealthBadge assetId="a1" conflictType="C" onClick={onClick} />,
      { wrapper }
    );
    const dot = baseElement.querySelector('[role="button"]') as HTMLElement;
    fireEvent.keyDown(dot, { key: "Enter" });
    expect(onClick).toHaveBeenCalledWith("a1", "C");
  });

  it("visible=false → 渲染 - 占位", () => {
    mockVisible = false;
    const { baseElement } = render(
      <HealthBadge assetId="a1" conflictType="B" onClick={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toBe("-");
  });

  it("未知 conflictType → 默认 listClass=default 颜色", () => {
    const { baseElement } = render(
      <HealthBadge assetId="a1" conflictType="Z" onClick={vi.fn()} />,
      { wrapper }
    );
    const dot = baseElement.querySelector('[role="button"]') as HTMLElement;
    // Default color #d4d4d8
    expect(dot?.style.backgroundColor).toBeTruthy();
  });

  it("non-Enter 键 → 不触发 onClick", () => {
    const onClick = vi.fn();
    const { baseElement } = render(
      <HealthBadge assetId="a1" conflictType="B" onClick={onClick} />,
      { wrapper }
    );
    const dot = baseElement.querySelector('[role="button"]') as HTMLElement;
    fireEvent.keyDown(dot, { key: "Space" });
    expect(onClick).not.toHaveBeenCalled();
  });
});
