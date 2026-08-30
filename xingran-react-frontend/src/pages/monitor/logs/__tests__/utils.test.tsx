/**
 * Phase 88 Batch177 — pages/monitor/logs/utils 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  formatLocalTime,
  getBusinessTypeLabel,
  renderRequestMethodTag,
  renderLogStatusTag,
} from "../utils";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("monitor/logs/utils", () => {
  it("formatLocalTime null → '-'", () => {
    expect(formatLocalTime(null)).toBe("-");
  });

  it("formatLocalTime undefined → '-'", () => {
    expect(formatLocalTime(undefined)).toBe("-");
  });

  it("formatLocalTime 有效 → 本地字符串", () => {
    const result = formatLocalTime("2026-01-01T00:00:00Z");
    expect(result).toBeTruthy();
    expect(result).not.toBe("-");
  });

  it("getBusinessTypeLabel 1 → '新增'", () => {
    expect(getBusinessTypeLabel(1)).toBe("新增");
  });

  it("getBusinessTypeLabel 3 → '删除'", () => {
    expect(getBusinessTypeLabel(3)).toBe("删除");
  });

  it("getBusinessTypeLabel 99 → '-'", () => {
    expect(getBusinessTypeLabel(99)).toBe("-");
  });

  it("renderRequestMethodTag GET → blue", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("GET")}</>, { wrapper });
    const tag = baseElement.querySelector(".ant-tag");
    expect(tag?.className ?? "").toContain("ant-tag-blue");
  });

  it("renderRequestMethodTag POST → green", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("POST")}</>, { wrapper });
    const tag = baseElement.querySelector(".ant-tag");
    expect(tag?.className ?? "").toContain("ant-tag-green");
  });

  it("renderRequestMethodTag 未知 → default", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("UNKNOWN")}</>, { wrapper });
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderLogStatusTag oper success → 正常 (LogStatus.Success=0)", () => {
    const { baseElement } = render(<>{renderLogStatusTag(0, "oper")}</>, { wrapper });
    expect(baseElement.textContent).toContain("正常");
  });

  it("renderLogStatusTag oper failed → 异常 (LogStatus.Failure=1)", () => {
    const { baseElement } = render(<>{renderLogStatusTag(1, "oper")}</>, { wrapper });
    expect(baseElement.textContent).toContain("异常");
  });

  it("renderLogStatusTag login success → 成功", () => {
    const { baseElement } = render(<>{renderLogStatusTag(0, "login")}</>, { wrapper });
    expect(baseElement.textContent).toContain("成功");
  });

  it("renderLogStatusTag login failed → 失败", () => {
    const { baseElement } = render(<>{renderLogStatusTag(1, "login")}</>, { wrapper });
    expect(baseElement.textContent).toContain("失败");
  });
});
