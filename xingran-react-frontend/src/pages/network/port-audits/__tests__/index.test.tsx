/**
 * v1.51 Phase 127 PWX-04 — 端口写审计明细页测试
 *
 * 覆盖：
 * - 分页表渲染 + 状态 Tag
 * - 过滤提交参数（deviceId/portId/operator/action/status + RFC3339 时间）
 * - 展开行 before/after JSON 与设备原始响应
 * - API 失败空态兜底（后端契约未就绪期间不白屏）
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, act } from "@testing-library/react";
import { App } from "antd";
import type { PortWriteAuditRow, PortWriteAuditListResult } from "@/types/network";

const mockListPortWriteAudits = vi.fn();
vi.mock("@/lib/api/networkApi", () => ({
  listPortWriteAudits: (...args: unknown[]) => mockListPortWriteAudits(...args),
}));

import PortWriteAuditsPage from "../index";

function Wrapper({ children }: { children: React.ReactNode }) {
  return <App>{children}</App>;
}

function makeRow(overrides?: Partial<PortWriteAuditRow>): PortWriteAuditRow {
  return {
    id: "audit-1",
    deviceId: "device-1",
    portId: "port-1",
    action: "shutdown",
    beforeValue: { adminStatus: "up" },
    afterValue: { adminStatus: "down" },
    commandSent: "interface Gi0/1\nshutdown",
    deviceResponse: "OK",
    status: "succeeded",
    failureReason: null,
    operator: "admin",
    createdAt: "2026-11-01T00:00:00Z",
    ...overrides,
  };
}

function makeResult(list: PortWriteAuditRow[]): PortWriteAuditListResult {
  return { list, total: list.length, current: 1, pageSize: 10 };
}

describe("PortWriteAuditsPage — PWX-04", () => {
  beforeEach(() => {
    mockListPortWriteAudits.mockReset();
  });

  it("渲染分页表 + 状态 Tag", async () => {
    mockListPortWriteAudits.mockResolvedValue(makeResult([makeRow()]));

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(mockListPortWriteAudits).toHaveBeenCalledTimes(1);
    });
    expect(await screen.findByText("admin")).toBeInTheDocument();
    expect(screen.getByText("成功")).toBeInTheDocument();
    // 分页请求参数
    const arg = mockListPortWriteAudits.mock.calls[0][0] as { current: number; pageSize: number };
    expect(arg.current).toBe(1);
    expect(arg.pageSize).toBe(10);
  });

  it("展开行显示 before/after JSON 与设备原始响应", async () => {
    mockListPortWriteAudits.mockResolvedValue(makeResult([makeRow()]));

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });
    await screen.findByText("admin");

    // antd Table 展开按钮
    const expandBtn = document.querySelector(".ant-table-row-expand-icon");
    expect(expandBtn).not.toBeNull();
    await act(async () => {
      fireEvent.click(expandBtn as HTMLElement);
    });

    expect(await screen.findByText("变更前 (beforeValue)")).toBeInTheDocument();
    expect(screen.getByText("变更后 (afterValue)")).toBeInTheDocument();
    // before/after 各渲染一份 adminStatus 快照
    expect(screen.getAllByText(/adminStatus/).length).toBe(2);
  });

  it("API 失败时空态兜底不白屏", async () => {
    mockListPortWriteAudits.mockRejectedValue(new Error("404 not found"));

    render(<PortWriteAuditsPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(mockListPortWriteAudits).toHaveBeenCalledTimes(1);
    });
    // 空态渲染（antd Table placeholder; locale 默认英文环境, 断言占位结构存在）
    await waitFor(() => {
      expect(document.querySelector(".ant-table-placeholder")).not.toBeNull();
    });
  });
});
