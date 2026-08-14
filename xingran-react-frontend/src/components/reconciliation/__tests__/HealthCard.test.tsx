/**
 * HealthCard 单元测试
 *
 * 锁定行为(UI-SPEC D-A1-01):
 *   - useReconciliationVisibility() === false → render null
 *   - isLoading=true → Skeleton 渲染
 *   - isError=true → Result status="error" + 重试按钮
 *   - data.healthScore.total === 0 → 空态 "该工位暂无关联资产。"
 *   - data 正常 → 5 KPI + score + 趋势图
 *   - onApplyException 按钮被点击时调用 prop
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

// mock 子 hooks(useWorkstationHealth + useReconciliationVisibility)
const mockUseReconciliationVisibility = vi.fn();
const mockUseWorkstationHealth = vi.fn();

vi.mock("../hooks/useReconciliationVisibility", () => ({
  useReconciliationVisibility: () => mockUseReconciliationVisibility() as boolean,
}));

vi.mock("../hooks/useWorkstationHealth", () => ({
  useWorkstationHealth: (_workstationId: string) =>
    mockUseWorkstationHealth() as ReturnType<typeof vi.fn>,
}));

// mock ReactECharts 避免引入 zrender/jsdom 复杂依赖
vi.mock("echarts-for-react", () => ({
  default: () => <div data-testid="echarts-mock" />,
}));

import { HealthCard } from "../HealthCard";

const SAMPLE_HEALTH_DATA = {
  visible: true,
  workstation: { id: "ws-1", name: "Test WS", code: "TW01" },
  healthScore: {
    total: 10,
    normal: 5,
    drift: 2,
    conflict: 1,
    noData: 1,
    exceptionHit: 1,
    score: 75,
    trend: [
      { date: "2026-06-22", openCount: 3, criticalCount: 1, newCount: 5 },
      { date: "2026-06-23", openCount: 2, criticalCount: 0, newCount: 3 },
    ],
  },
  assets: [
    { assetId: "a-1", assetCode: "A001", conflictType: "A", severity: "low" },
    { assetId: "a-2", assetCode: "A002", conflictType: "C", severity: "high" },
  ],
};

describe("HealthCard", () => {
  beforeEach(() => {
    mockUseReconciliationVisibility.mockReset();
    mockUseWorkstationHealth.mockReset();
  });

  it("renders null when useReconciliationVisibility returns false", () => {
    mockUseReconciliationVisibility.mockReturnValue(false);
    mockUseWorkstationHealth.mockReturnValue({
      data: SAMPLE_HEALTH_DATA,
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    const { container } = render(<HealthCard workstationId="ws-1" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders Skeleton when isLoading=true", () => {
    mockUseReconciliationVisibility.mockReturnValue(true);
    mockUseWorkstationHealth.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      refetch: vi.fn(),
    });

    const { container } = render(<HealthCard workstationId="ws-1" />);
    // antd Skeleton uses "ant-skeleton" class
    expect(container.querySelector(".ant-skeleton")).toBeTruthy();
  });

  it("renders 5 KPIs + score + trend when data is loaded", () => {
    mockUseReconciliationVisibility.mockReturnValue(true);
    mockUseWorkstationHealth.mockReturnValue({
      data: SAMPLE_HEALTH_DATA,
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    render(<HealthCard workstationId="ws-1" />);

    // Card title
    expect(screen.getByText("对账健康度")).toBeInTheDocument();
    // Score (75) — antd Statistic renders it as text
    expect(screen.getByText("75")).toBeInTheDocument();
    // 5 KPI titles
    expect(screen.getByText("正常")).toBeInTheDocument();
    expect(screen.getByText("漂移")).toBeInTheDocument();
    expect(screen.getByText("冲突")).toBeInTheDocument();
    expect(screen.getByText("无数据")).toBeInTheDocument();
    expect(screen.getByText("例外命中")).toBeInTheDocument();
    // echarts 趋势图
    expect(screen.getByTestId("echarts-mock")).toBeInTheDocument();
  });

  it("renders empty state when total=0", () => {
    mockUseReconciliationVisibility.mockReturnValue(true);
    mockUseWorkstationHealth.mockReturnValue({
      data: {
        ...SAMPLE_HEALTH_DATA,
        healthScore: { ...SAMPLE_HEALTH_DATA.healthScore, total: 0 },
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    render(<HealthCard workstationId="ws-1" />);
    // 55-01 HealthCard-test-fix: HealthCard 把"对账健康度:"前缀与消息渲染在同一文本节点
    // (整个 <div> 内容是一个 text node), 精确 getByText 找不到。改 regex 容忍前缀。
    expect(screen.getByText(/该工位暂无关联资产/)).toBeInTheDocument();
  });

  it("calls onApplyException when 申请例外 button is clicked", () => {
    mockUseReconciliationVisibility.mockReturnValue(true);
    mockUseWorkstationHealth.mockReturnValue({
      data: SAMPLE_HEALTH_DATA,
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });
    const onApplyException = vi.fn();

    render(<HealthCard workstationId="ws-1" onApplyException={onApplyException} />);

    // 申请例外 button appears in HealthCard when onApplyException prop is provided
    const button = screen.getByRole("button", { name: "申请例外" });
    fireEvent.click(button);
    expect(onApplyException).toHaveBeenCalledTimes(1);
  });
});
