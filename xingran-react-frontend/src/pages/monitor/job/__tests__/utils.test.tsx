/**
 * Phase 88 Batch178 — pages/monitor/job/utils 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/utils/datetime", () => ({
  formatDateTime: vi.fn(() => "2026-08-30 10:00:00"),
}));

import {
  formatLocalTime,
  formatDuration,
  renderJobStatusTag,
  renderConcurrentTag,
  renderCronExpression,
  renderExceptionInfo,
} from "../utils";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("monitor/job/utils", () => {
  it("formatLocalTime → formatDateTime 调用", () => {
    expect(formatLocalTime("2026-01-01T00:00:00Z")).toBe("2026-08-30 10:00:00");
  });

  it("formatDuration 0 → '-'", () => {
    expect(formatDuration(0)).toBe("-");
  });

  it("formatDuration < 1000 → 'Nms'", () => {
    expect(formatDuration(500)).toBe("500ms");
  });

  it("formatDuration 1000 → '1.00s'", () => {
    expect(formatDuration(1000)).toBe("1.00s");
  });

  it("formatDuration 1500 → '1.50s'", () => {
    expect(formatDuration(1500)).toBe("1.50s");
  });

  it("renderJobStatusTag Normal (0) → 正常 green", () => {
    const { baseElement } = render(<>{renderJobStatusTag(0)}</>, { wrapper });
    expect(baseElement.textContent).toContain("正常");
  });

  it("renderJobStatusTag Paused (1) → 暂停", () => {
    const { baseElement } = render(<>{renderJobStatusTag(1)}</>, { wrapper });
    expect(baseElement.textContent).toContain("暂停");
  });

  it("renderConcurrentTag true → 允许", () => {
    const { baseElement } = render(<>{renderConcurrentTag(true)}</>, { wrapper });
    expect(baseElement.textContent).toContain("允许");
  });

  it("renderConcurrentTag false → 禁止", () => {
    const { baseElement } = render(<>{renderConcurrentTag(false)}</>, { wrapper });
    expect(baseElement.textContent).toContain("禁止");
  });

  it("renderCronExpression → 渲染 Tooltip + code", () => {
    const { baseElement } = render(<>{renderCronExpression("0/5 * * * *")}</>, { wrapper });
    expect(baseElement.textContent).toContain("0/5 * * * *");
    expect(baseElement.querySelector("code")).toBeTruthy();
  });

  it("renderExceptionInfo → 显示 查看错误", () => {
    const { baseElement } = render(<>{renderExceptionInfo("Error stack")}</>, { wrapper });
    expect(baseElement.textContent).toContain("查看错误");
  });

  it("renderExceptionInfo 空 → '-'", () => {
    const { baseElement } = render(<>{renderExceptionInfo("")}</>, { wrapper });
    expect(baseElement.textContent).toContain("-");
  });
});
