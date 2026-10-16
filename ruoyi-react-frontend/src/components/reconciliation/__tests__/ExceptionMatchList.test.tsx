/**
 * Phase 88 Batch225 — components/reconciliation/ExceptionMatchList 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { ExceptionMatchList } from "../ExceptionMatchList";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const baseRule = {
  id: "r1",
  name: "Test Rule",
  scopeType: "global" as const,
  ipRange: "10.0.0.0/8",
  exceptionActions: ["silence", "no_alert"],
  reason: "For testing",
  expiresAt: "2026-12-31",
};

describe("reconciliation/ExceptionMatchList", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("loading 状态", () => {
    render(<ExceptionMatchList rules={[]} loading={true} />, { wrapper });
    expect(document.querySelector(".ant-spin")).toBeTruthy();
  });

  it("空 rules → Empty + 创建按钮", () => {
    render(<ExceptionMatchList rules={[]} loading={false} />, { wrapper });
    expect(screen.getByText("当前没有例外规则覆盖该资产所在 IP 段。")).toBeInTheDocument();
    expect(screen.getByText("去创建例外规则")).toBeInTheDocument();
  });

  it("渲染 rules + scopeTag + actions", () => {
    render(<ExceptionMatchList rules={[baseRule]} loading={false} />, { wrapper });
    expect(screen.getByText("Test Rule")).toBeInTheDocument();
    expect(screen.getByText("global")).toBeInTheDocument();
    expect(screen.getByText("10.0.0.0/8")).toBeInTheDocument();
    expect(screen.getByText("silence")).toBeInTheDocument();
    expect(screen.getByText("no_alert")).toBeInTheDocument();
  });

  it("reason + expiresAt 渲染", () => {
    render(<ExceptionMatchList rules={[baseRule]} loading={false} />, { wrapper });
    expect(screen.getByText("For testing")).toBeInTheDocument();
  });

  it("onCreateRule 优先于默认 navigate", () => {
    const onCreateRule = vi.fn();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    render(
      <ExceptionMatchList
        rules={[]}
        loading={false}
        onCreateRule={onCreateRule}
        assetIp="10.0.0.1"
        conflictType="ip_conflict"
      />,
      { wrapper }
    );
    fireEvent.click(screen.getByText("去创建例外规则"));
    expect(onCreateRule).toHaveBeenCalled();
    expect(openSpy).not.toHaveBeenCalled();
  });

  it("无 onCreateRule → window.open", () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    render(
      <ExceptionMatchList
        rules={[]}
        loading={false}
        assetIp="10.0.0.1"
        conflictType="ip_conflict"
      />,
      { wrapper }
    );
    fireEvent.click(screen.getByText("去创建例外规则"));
    expect(openSpy).toHaveBeenCalled();
    expect(openSpy.mock.calls[0][0]).toContain("/asset/reconciliation/exception-rules/new");
    expect(openSpy.mock.calls[0][0]).toContain("assetIp=10.0.0.1");
  });

  it("scopeType 不在 SCOPE_COLOR → default", () => {
    const rule = { ...baseRule, scopeType: "unknown" as any };
    render(<ExceptionMatchList rules={[rule]} loading={false} />, { wrapper });
    expect(screen.getByText("Test Rule")).toBeInTheDocument();
  });

  it("exceptionActions 包含 unknown → default tag", () => {
    const rule = { ...baseRule, exceptionActions: ["unknown"] };
    render(<ExceptionMatchList rules={[rule]} loading={false} />, { wrapper });
    expect(screen.getByText("unknown")).toBeInTheDocument();
  });
});
