/**
 * Phase 88 Batch189 — pages/monitor/logs/utils 测试
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
  processTimeRangeParams,
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

  it("formatLocalTime 空字符串 → '-'", () => {
    expect(formatLocalTime("")).toBe("-");
  });

  it("formatLocalTime 有效时间 → 含 '2026'", () => {
    const result = formatLocalTime("2026-08-30T10:00:00Z");
    expect(result).toContain("2026");
  });

  it("getBusinessTypeLabel 0 → 其它", () => {
    expect(getBusinessTypeLabel(0)).toBe("其它");
  });

  it("getBusinessTypeLabel 1 → 新增", () => {
    expect(getBusinessTypeLabel(1)).toBe("新增");
  });

  it("getBusinessTypeLabel 未知 → '-'", () => {
    expect(getBusinessTypeLabel(9999)).toBe("-");
  });

  it("renderRequestMethodTag GET → blue", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("GET")}</>, { wrapper });
    expect(baseElement.textContent).toContain("GET");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderRequestMethodTag POST → green", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("POST")}</>, { wrapper });
    expect(baseElement.textContent).toContain("POST");
  });

  it("renderRequestMethodTag PUT → orange", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("PUT")}</>, { wrapper });
    expect(baseElement.textContent).toContain("PUT");
  });

  it("renderRequestMethodTag DELETE → red", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("DELETE")}</>, { wrapper });
    expect(baseElement.textContent).toContain("DELETE");
  });

  it("renderRequestMethodTag 未知 → default", () => {
    const { baseElement } = render(<>{renderRequestMethodTag("PATCH")}</>, { wrapper });
    expect(baseElement.textContent).toContain("PATCH");
  });

  it("renderLogStatusTag oper Success(0) → 正常", () => {
    const { baseElement } = render(<>{renderLogStatusTag(0, "oper")}</>, { wrapper });
    expect(baseElement.textContent).toContain("正常");
  });

  it("renderLogStatusTag oper Failure(1) → 异常", () => {
    const { baseElement } = render(<>{renderLogStatusTag(1, "oper")}</>, { wrapper });
    expect(baseElement.textContent).toContain("异常");
  });

  it("renderLogStatusTag login Success(0) → 成功", () => {
    const { baseElement } = render(<>{renderLogStatusTag(0, "login")}</>, { wrapper });
    expect(baseElement.textContent).toContain("成功");
  });

  it("renderLogStatusTag login Failure(1) → 失败", () => {
    const { baseElement } = render(<>{renderLogStatusTag(1, "login")}</>, { wrapper });
    expect(baseElement.textContent).toContain("失败");
  });

  it("processTimeRangeParams 有效时间范围", () => {
    const params: any = {};
    const start = new Date("2026-08-30T00:00:00Z");
    const end = new Date("2026-08-30T23:59:59Z");
    processTimeRangeParams([start, end], params);
    expect(params.startTime).toBe(start.toISOString());
    expect(params.endTime).toBe(end.toISOString());
  });

  it("processTimeRangeParams 空 → 不动 params", () => {
    const params: any = { foo: 1 };
    processTimeRangeParams(null, params);
    expect(params.startTime).toBeUndefined();
    expect(params.foo).toBe(1);
  });
});
