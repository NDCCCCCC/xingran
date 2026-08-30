/**
 * Phase 88 Batch174 — pages/network/command/constants 测试
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
  STATUS_OPTIONS,
  STATUS_CONFIG,
  SIMPLE_STATUS_CONFIG,
  renderExecutionStatusTag,
  renderSimpleStatusTag,
} from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("network/command/constants", () => {
  it("STATUS_OPTIONS 4 项", () => {
    expect(STATUS_OPTIONS.length).toBe(4);
    expect(STATUS_OPTIONS[0].value).toBe("pending");
    expect(STATUS_OPTIONS[3].value).toBe("failed");
  });

  it("STATUS_CONFIG 4 个状态", () => {
    expect(Object.keys(STATUS_CONFIG)).toEqual(
      expect.arrayContaining(["pending", "running", "success", "failed"])
    );
  });

  it("SIMPLE_STATUS_CONFIG 4 个状态", () => {
    expect(Object.keys(SIMPLE_STATUS_CONFIG)).toEqual(
      expect.arrayContaining(["pending", "running", "success", "failed"])
    );
  });

  it("renderExecutionStatusTag → 渲染 Tag", () => {
    const { baseElement } = render(<>{renderExecutionStatusTag("success")}</>, { wrapper });
    expect(baseElement.textContent).toContain("成功");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderExecutionStatusTag 未知 status → 默认 pending", () => {
    const { baseElement } = render(<>{renderExecutionStatusTag("unknown")}</>, { wrapper });
    expect(baseElement.textContent).toContain("待执行");
  });

  it("renderExecutionStatusTag 各状态文案", () => {
    for (const [status, expected] of [
      ["pending", "待执行"],
      ["running", "执行中"],
      ["success", "成功"],
      ["failed", "失败"],
    ] as const) {
      const { baseElement } = render(<>{renderExecutionStatusTag(status)}</>, { wrapper });
      expect(baseElement.textContent).toContain(expected);
    }
  });

  it("renderSimpleStatusTag → 渲染 Tag", () => {
    const { baseElement } = render(<>{renderSimpleStatusTag("success")}</>, { wrapper });
    expect(baseElement.textContent).toContain("成功");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderSimpleStatusTag 未知 status → 默认 pending", () => {
    const { baseElement } = render(<>{renderSimpleStatusTag("unknown")}</>, { wrapper });
    expect(baseElement.textContent).toContain("待执行");
  });

  it("renderSimpleStatusTag 各状态文案", () => {
    for (const [status, expected] of [
      ["pending", "待执行"],
      ["running", "执行中"],
      ["success", "成功"],
      ["failed", "失败"],
    ] as const) {
      const { baseElement } = render(<>{renderSimpleStatusTag(status)}</>, { wrapper });
      expect(baseElement.textContent).toContain(expected);
    }
  });
});
