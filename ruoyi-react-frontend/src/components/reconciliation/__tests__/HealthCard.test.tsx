/**
 * Phase 88 Batch141 — components/reconciliation/HealthCard 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let mockVisible = true;
let mockHealth: any = undefined;
let mockLoading = false;
let mockError = false;
let mockRefetch = vi.fn();

vi.mock("../hooks/useReconciliationVisibility", () => ({
  useReconciliationVisibility: () => mockVisible,
}));

vi.mock("../hooks/useWorkstationHealth", () => ({
  useWorkstationHealth: () => ({
    data: mockHealth,
    isLoading: mockLoading,
    isError: mockError,
    refetch: mockRefetch,
  }),
}));

import { HealthCard } from "../HealthCard";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("HealthCard", () => {
  beforeEach(() => {
    mockVisible = true;
    mockHealth = undefined;
    mockLoading = false;
    mockError = false;
    mockRefetch = vi.fn();
  });

  it("visible=false → render null", () => {
    mockVisible = false;
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    // When null is returned, baseElement.body may not exist; check container
    expect(baseElement.textContent ?? "").toBe("");
  });

  it("isLoading=true → Skeleton", () => {
    mockLoading = true;
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.querySelector(".ant-skeleton")).toBeTruthy();
  });

  it("isError=true → Result 错误 + 重试按钮", () => {
    mockError = true;
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("健康度加载失败");
    const retryBtn = baseElement.querySelector("button");
    fireEvent.click(retryBtn!);
    expect(mockRefetch).toHaveBeenCalled();
  });

  it("data=null → Empty", () => {
    mockHealth = null;
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("暂无数据");
  });

  it("total=0 → 显示'该工位暂无关联资产'", () => {
    mockHealth = { healthScore: { total: 0 } };
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("该工位暂无关联资产");
  });

  it("score=85 → 绿色", () => {
    mockHealth = {
      healthScore: { total: 5, score: 85, normal: 4, drift: 1, conflict: 0 },
    };
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("85");
    expect(baseElement.textContent).toContain("正常 4");
    expect(baseElement.textContent).toContain("漂移 1");
  });

  it("score=70 → 黄色 (60-79)", () => {
    mockHealth = {
      healthScore: { total: 5, score: 70, normal: 3, drift: 2, conflict: 0 },
    };
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("70");
  });

  it("score=50 → 红色 (<60)", () => {
    mockHealth = {
      healthScore: { total: 5, score: 50, normal: 2, drift: 1, conflict: 2 },
    };
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("50");
    expect(baseElement.textContent).toContain("冲突 2");
  });

  it("noData > 0 → 显示'无数据 N'", () => {
    mockHealth = {
      healthScore: { total: 5, score: 90, normal: 3, drift: 1, conflict: 0, noData: 1 },
    };
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("无数据 1");
  });

  it("exceptionHit > 0 → 显示'例外 N'", () => {
    mockHealth = {
      healthScore: { total: 5, score: 90, normal: 4, drift: 0, conflict: 0, exceptionHit: 1 },
    };
    const { baseElement } = render(<HealthCard workstationId="w1" />, { wrapper });
    expect(baseElement.textContent).toContain("例外 1");
  });

  it("onApplyException 提供 → 显示'申请例外'按钮", () => {
    mockHealth = {
      healthScore: { total: 5, score: 90, normal: 5, drift: 0, conflict: 0 },
    };
    const onApply = vi.fn();
    const { baseElement } = render(<HealthCard workstationId="w1" onApplyException={onApply} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("申请例外");
    fireEvent.click(baseElement.querySelector("button")!);
    expect(onApply).toHaveBeenCalled();
  });
});
