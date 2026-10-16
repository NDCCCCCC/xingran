/**
 * Phase 84 84-01a Task 1 — 批量操作组件测试
 * BatchDeleteButton(Popconfirm 二次确认) / BatchExportModal(9 实体类型多选导出)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import BatchDeleteButton from "../BatchDeleteButton";
import BatchExportModal from "../BatchExportModal";

describe("BatchDeleteButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing when selectedCount=0", () => {
    renderWithProviders(<BatchDeleteButton selectedCount={0} onConfirm={vi.fn()} />);
    expect(screen.queryByText(/批量删除/)).toBeNull();
  });

  it("renders count in button label (D-12 props 组合)", () => {
    renderWithProviders(<BatchDeleteButton selectedCount={3} onConfirm={vi.fn()} />);
    expect(screen.getByText("批量删除 (3)")).not.toBeNull();
    renderWithProviders(<BatchDeleteButton selectedCount={7} onConfirm={vi.fn()} />);
    expect(screen.getByText("批量删除 (7)")).not.toBeNull();
  });

  it("shows popconfirm and calls onConfirm after OK click", async () => {
    const onConfirm = vi.fn(() => Promise.resolve());
    renderWithProviders(<BatchDeleteButton selectedCount={2} onConfirm={onConfirm} />);
    fireEvent.click(screen.getByText("批量删除 (2)"));
    const okBtn = await screen.findByRole("button", { name: "确 定" });
    fireEvent.click(okBtn);
    await waitFor(() => {
      expect(onConfirm).toHaveBeenCalledTimes(1);
    });
  });

  it("respects custom confirmTitle", async () => {
    renderWithProviders(
      <BatchDeleteButton selectedCount={5} onConfirm={vi.fn()} confirmTitle="确认删除设备?" />
    );
    fireEvent.click(screen.getByText("批量删除 (5)"));
    expect(await screen.findByText("确认删除设备?")).not.toBeNull();
  });
});

describe("BatchExportModal", () => {
  it("renders all default entity types when visible", async () => {
    renderWithProviders(<BatchExportModal visible onConfirm={vi.fn()} onCancel={vi.fn()} />);
    expect(await screen.findByText("网络设备")).not.toBeNull();
    expect(screen.getByText("端口采集")).not.toBeNull();
    expect(screen.getByText("MAC地址")).not.toBeNull();
  });

  it("closes when X button clicked (onCancel triggered)", async () => {
    const onCancel = vi.fn();
    renderWithProviders(<BatchExportModal visible onConfirm={vi.fn()} onCancel={onCancel} />);
    // 等待 modal 渲染完毕
    await screen.findByText("网络设备");
    // 关闭按钮:ant-modal-close class 的 button
    const closeBtn = document.querySelector(".ant-modal-close") as HTMLButtonElement;
    expect(closeBtn).not.toBeNull();
    fireEvent.click(closeBtn);
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("全选 and 清空 shortcut buttons present (D-11)", async () => {
    renderWithProviders(<BatchExportModal visible onConfirm={vi.fn()} onCancel={vi.fn()} />);
    await screen.findByText("网络设备");
    // 全选/清空快捷按钮可见
    expect(screen.getByRole("button", { name: "全 选" })).not.toBeNull();
    expect(screen.getByRole("button", { name: "清 空" })).not.toBeNull();
  });

  it("取消勾选一个实体后 okText count updates", async () => {
    renderWithProviders(<BatchExportModal visible onConfirm={vi.fn()} onCancel={vi.fn()} />);
    await screen.findByText("配置模板");
    // 取消勾选"配置模板"
    const label = screen.getByText("配置模板").closest("label")!;
    const input = label.querySelector('input[type="checkbox"]') as HTMLInputElement;
    fireEvent.click(input);
    // 默认全选 → 取消 1 → 8 个
    expect(await screen.findByText(/确认导出 \(8\)/)).not.toBeNull();
  });
});
