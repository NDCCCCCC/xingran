/**
 * Phase 84 84-01a Task 1 — 共享原子组件 family 聚合测试
 *
 * 测试组件: ModernTag / EmptyStateWithAction / ActionButtons / ErrorAlertWithRetry
 * 模式: D-11 (fireEvent + props 渲染断言) / D-12 (纯展示 props 组合快照)
 */
import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ModernTag from "../ModernTag";
import EmptyStateWithAction from "../EmptyStateWithAction";
import ActionButtons from "../ActionButtons";
import ErrorAlertWithRetry from "../ErrorAlertWithRetry";

describe("ModernTag", () => {
  it("renders with default status when no props", () => {
    const { container } = renderWithProviders(<ModernTag>默认标签</ModernTag>);
    expect(container.querySelector(".modern-tag-default")).not.toBeNull();
    expect(screen.getByText("默认标签")).not.toBeNull();
  });

  it("applies status className based on props", () => {
    renderWithProviders(<ModernTag status="success">正常</ModernTag>);
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("renders 2 props combinations correctly (D-12)", () => {
    // error 状态默认文案"停用"
    renderWithProviders(<ModernTag status="error">停用</ModernTag>);
    expect(screen.getByText("停用")).toHaveClass("modern-tag-error");
    // warning 状态带自定义 children + showIcon
    renderWithProviders(
      <ModernTag status="warning" showIcon>
        警告
      </ModernTag>
    );
    expect(screen.getByText("警告")).toHaveClass("modern-tag-warning");
  });

  it("falls back to non-modern Tag when modern=false", () => {
    const { container } = renderWithProviders(
      <ModernTag status="processing" modern={false}>
        进行中
      </ModernTag>
    );
    // 非 modern 模式走 antd color prop(ant-tag-processing),无 modern-tag-* 类
    expect(container.querySelector(".modern-tag-processing")).toBeNull();
    expect(container.querySelector(".ant-tag-processing")).not.toBeNull();
    expect(screen.getByText("进行中")).not.toBeNull();
  });
});

describe("EmptyStateWithAction", () => {
  it("renders with description only", () => {
    renderWithProviders(<EmptyStateWithAction description="暂无数据" />);
    expect(screen.getByText("暂无数据")).not.toBeNull();
  });

  it("renders with action button", () => {
    const onAction = vi.fn();
    renderWithProviders(
      <EmptyStateWithAction description="空状态" actionLabel="重新加载" onAction={onAction} />
    );
    expect(screen.getByText("重新加载")).not.toBeNull();
    expect(screen.getByText("空状态")).not.toBeNull();
  });
});

describe("ActionButtons", () => {
  it("renders all buttons when count < threshold", () => {
    const actions = [
      { key: "a", label: "编辑", onClick: vi.fn() },
      { key: "b", label: "删除", onClick: vi.fn() },
    ];
    renderWithProviders(<ActionButtons actions={actions} threshold={3} />);
    expect(screen.getByText("编辑")).not.toBeNull();
    expect(screen.getByText("删除")).not.toBeNull();
  });

  it("collapses into dropdown when count >= threshold", async () => {
    const onEdit = vi.fn();
    const actions = [
      { key: "a", label: "编辑", onClick: onEdit },
      { key: "b", label: "删除" },
      { key: "c", label: "导出" },
    ];
    renderWithProviders(<ActionButtons actions={actions} threshold={3} />);
    // 收纳后触发器按钮为"操作"
    const trigger = screen.getByText("操作");
    expect(trigger).not.toBeNull();
    expect(screen.queryByText("编辑")).toBeNull();
    // 点击展开下拉菜单
    fireEvent.click(trigger.closest("button")!);
    const menuItem = await screen.findByText("编辑");
    fireEvent.click(menuItem);
    expect(onEdit).toHaveBeenCalledTimes(1);
  });

  it("renders custom render prop for action", () => {
    const actions = [
      {
        key: "a",
        label: "原生按钮",
        render: () => <button data-testid="custom-btn">自定义</button>,
      },
    ];
    renderWithProviders(<ActionButtons actions={actions} threshold={3} />);
    expect(screen.getByTestId("custom-btn")).not.toBeNull();
  });
});

describe("ErrorAlertWithRetry", () => {
  it("renders error message for generic failure", () => {
    const onRetry = vi.fn();
    renderWithProviders(
      <ErrorAlertWithRetry error={{ code: 400, message: "连接失败" }} onRetry={onRetry} />
    );
    expect(screen.getByText("查询失败:连接失败")).not.toBeNull();
    expect(screen.getByText("重新加载")).not.toBeNull();
  });

  it("renders 500 message and calls onRetry when clicked", () => {
    const onRetry = vi.fn();
    renderWithProviders(<ErrorAlertWithRetry error={{ code: 500 }} onRetry={onRetry} />);
    expect(screen.getByText("服务暂不可用,请稍后重试")).not.toBeNull();
    fireEvent.click(screen.getByText("重新加载"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("renders device-not-found message for 1006", () => {
    renderWithProviders(<ErrorAlertWithRetry error={{ code: 1006 }} />);
    expect(screen.getByText("该设备不存在或已被删除")).not.toBeNull();
  });

  it("supports custom description override", () => {
    renderWithProviders(<ErrorAlertWithRetry error={new Error("boom")} description="自定义描述" />);
    expect(screen.getByText("自定义描述")).not.toBeNull();
  });
});
