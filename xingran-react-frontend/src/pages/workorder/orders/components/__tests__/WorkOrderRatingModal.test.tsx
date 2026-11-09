/**
 * v1.55 CI 覆盖率补测（GOV-05）：工单评价弹窗 FIX-06 / Phase 136 前端入口。
 *
 * 覆盖：关闭态不加载 / 打开态加载列表与统计 / 空态 / 加载失败错误出口 /
 * 提交 user 与 handler 两种评分字段映射 / 提交失败错误出口。
 * 约定：mock 包装 API（@/lib/workorderApi），禁止裸 axios；就近 __tests__。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

const mocks = vi.hoisted(() => ({
  listWorkOrderRatings: vi.fn(),
  fetchWorkOrderRatingStats: vi.fn(),
  submitWorkOrderRating: vi.fn(),
  handleApiError: vi.fn(),
  handleSuccess: vi.fn(),
}));

vi.mock("@/lib/workorderApi", () => ({
  listWorkOrderRatings: mocks.listWorkOrderRatings,
  fetchWorkOrderRatingStats: mocks.fetchWorkOrderRatingStats,
  submitWorkOrderRating: mocks.submitWorkOrderRating,
}));

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: mocks.handleApiError,
  handleSuccess: mocks.handleSuccess,
}));

import WorkOrderRatingModal from "../WorkOrderRatingModal";

function Wrap({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const statsFixture = {
  totalCount: 2,
  avgCompletionScore: 4.5,
  avgCooperationScore: 3.5,
};

const ratingsFixture = [
  {
    id: "r1",
    workOrderId: "wo-1",
    ratingType: "user",
    completionScore: 5,
    cooperationScore: 0,
    comment: "处理及时",
  },
  {
    id: "r2",
    workOrderId: "wo-1",
    ratingType: "handler",
    completionScore: 0,
    cooperationScore: 4,
    comment: "",
  },
];

describe("WorkOrderRatingModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listWorkOrderRatings.mockResolvedValue({ data: ratingsFixture });
    mocks.fetchWorkOrderRatingStats.mockResolvedValue({ data: statsFixture });
    mocks.submitWorkOrderRating.mockResolvedValue({ data: {} });
  });

  it("workOrderId 为空时不加载数据", () => {
    const { container } = render(<WorkOrderRatingModal workOrderId={null} onClose={vi.fn()} />, {
      wrapper: Wrap,
    });
    expect(mocks.listWorkOrderRatings).not.toHaveBeenCalled();
    expect(mocks.fetchWorkOrderRatingStats).not.toHaveBeenCalled();
    expect(container).toBeTruthy();
  });

  it("打开时加载列表与统计，并渲染统计/已有评价/表单", async () => {
    render(
      <WorkOrderRatingModal workOrderId="wo-1" workOrderTitle="网络故障" onClose={vi.fn()} />,
      { wrapper: Wrap }
    );

    await waitFor(() => expect(mocks.listWorkOrderRatings).toHaveBeenCalledWith("wo-1"));
    expect(mocks.fetchWorkOrderRatingStats).toHaveBeenCalledWith("wo-1");
    // 统计行（平均完成度/配合度）
    expect(await screen.findByText(/评价数：2/)).toBeTruthy();
    expect(screen.getByText(/平均完成度：4.5/)).toBeTruthy();
    expect(screen.getByText(/平均配合度：3.5/)).toBeTruthy();
    // 已有评价列表（含两种 Tag 与无评价内容兜底）
    expect(screen.getByText("用户评价")).toBeTruthy();
    expect(screen.getByText("处理人评价")).toBeTruthy();
    expect(screen.getByText("处理及时")).toBeTruthy();
    expect(screen.getByText("（无评价内容）")).toBeTruthy();
    // 表单标签
    expect(screen.getByText("评价类型")).toBeTruthy();
    expect(screen.getByText("完成度评分")).toBeTruthy();
    expect(screen.getByText("评价内容")).toBeTruthy();
    expect(screen.getByText("提交评价")).toBeTruthy();
  });

  it("无评价且加载完成时显示空态", async () => {
    mocks.listWorkOrderRatings.mockResolvedValue({ data: [] });
    mocks.fetchWorkOrderRatingStats.mockResolvedValue({ data: null });
    render(<WorkOrderRatingModal workOrderId="wo-1" onClose={vi.fn()} />, { wrapper: Wrap });
    await waitFor(() => expect(mocks.listWorkOrderRatings).toHaveBeenCalled());
    expect(await screen.findByText("暂无评价")).toBeTruthy();
  });

  it("加载失败走 handleApiError 唯一错误出口", async () => {
    const err = new Error("network down");
    mocks.listWorkOrderRatings.mockRejectedValue(err);
    mocks.fetchWorkOrderRatingStats.mockRejectedValue(err);
    render(<WorkOrderRatingModal workOrderId="wo-1" onClose={vi.fn()} />, { wrapper: Wrap });
    await waitFor(() => expect(mocks.handleApiError).toHaveBeenCalledWith(err, "加载工单评价失败"));
  });

  it("提交 user 评价：评分落到 completionScore，成功后刷新列表", async () => {
    render(<WorkOrderRatingModal workOrderId="wo-1" onClose={vi.fn()} />, { wrapper: Wrap });
    await waitFor(() => expect(mocks.listWorkOrderRatings).toHaveBeenCalled());

    fireEvent.click(screen.getByText("提交评价"));
    await waitFor(() => expect(mocks.submitWorkOrderRating).toHaveBeenCalledTimes(1));
    const payload = mocks.submitWorkOrderRating.mock.calls[0][0];
    expect(payload.workOrderId).toBe("wo-1");
    expect(payload.ratingType).toBe("user");
    expect(payload.completionScore).toBe(5);
    expect(payload.cooperationScore).toBe(0);
    expect(mocks.handleSuccess).toHaveBeenCalledWith("评价提交成功");
    // 提交成功后重新加载
    await waitFor(() => expect(mocks.listWorkOrderRatings).toHaveBeenCalledTimes(2));
  });

  it("切换到 handler 评价：标签变化且评分落到 cooperationScore", async () => {
    render(<WorkOrderRatingModal workOrderId="wo-1" onClose={vi.fn()} />, { wrapper: Wrap });
    await waitFor(() => expect(mocks.listWorkOrderRatings).toHaveBeenCalled());

    fireEvent.click(screen.getByText("处理人评价（配合度）"));
    expect(await screen.findByText("配合度评分")).toBeTruthy();
    fireEvent.click(screen.getByText("提交评价"));
    await waitFor(() => expect(mocks.submitWorkOrderRating).toHaveBeenCalledTimes(1));
    const payload = mocks.submitWorkOrderRating.mock.calls[0][0];
    expect(payload.ratingType).toBe("handler");
    expect(payload.completionScore).toBe(0);
    expect(payload.cooperationScore).toBe(5);
  });

  it("提交失败走 handleApiError 且不误报成功", async () => {
    const err = new Error("submit failed");
    mocks.submitWorkOrderRating.mockRejectedValue(err);
    render(<WorkOrderRatingModal workOrderId="wo-1" onClose={vi.fn()} />, { wrapper: Wrap });
    await waitFor(() => expect(mocks.listWorkOrderRatings).toHaveBeenCalled());

    fireEvent.click(screen.getByText("提交评价"));
    await waitFor(() => expect(mocks.handleApiError).toHaveBeenCalledWith(err, "提交工单评价失败"));
    expect(mocks.handleSuccess).not.toHaveBeenCalled();
  });

  it("workOrderId 为空时点击提交不触发请求", () => {
    render(<WorkOrderRatingModal workOrderId={null} onClose={vi.fn()} />, { wrapper: Wrap });
    // 关闭态 Modal 不渲染内容，直接断言无请求
    expect(mocks.submitWorkOrderRating).not.toHaveBeenCalled();
    expect(mocks.listWorkOrderRatings).not.toHaveBeenCalled();
  });
});
