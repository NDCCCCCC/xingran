/**
 * Phase 88 Batch184 — pages/network/executions/constants 测试
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
  renderStatusTag,
  renderSimpleStatusTag,
} from "../constants";
import type { ExecutionStatus } from "../types";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("network/executions/constants", () => {
  it("STATUS_OPTIONS 4 项", () => {
    expect(STATUS_OPTIONS.length).toBe(4);
    expect(STATUS_OPTIONS.map((o) => o.value)).toEqual(["pending", "running", "success", "failed"]);
  });

  it("STATUS_CONFIG 4 个状态", () => {
    expect(Object.keys(STATUS_CONFIG)).toEqual(
      expect.arrayContaining(["pending", "running", "success", "failed"])
    );
  });

  it("renderStatusTag pending → 待执行", () => {
    const { baseElement } = render(<>{renderStatusTag("pending")}</>, { wrapper });
    expect(baseElement.textContent).toContain("待执行");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderStatusTag running → 执行中", () => {
    const { baseElement } = render(<>{renderStatusTag("running")}</>, { wrapper });
    expect(baseElement.textContent).toContain("执行中");
  });

  it("renderStatusTag success → 成功", () => {
    const { baseElement } = render(<>{renderStatusTag("success")}</>, { wrapper });
    expect(baseElement.textContent).toContain("成功");
  });

  it("renderStatusTag failed → 失败", () => {
    const { baseElement } = render(<>{renderStatusTag("failed")}</>, { wrapper });
    expect(baseElement.textContent).toContain("失败");
  });

  it("renderStatusTag 未知 → fallback pending", () => {
    const { baseElement } = render(<>{renderStatusTag("unknown" as ExecutionStatus)}</>, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("待执行");
  });

  it("renderSimpleStatusTag success → 成功 (无图标)", () => {
    const { baseElement } = render(<>{renderSimpleStatusTag("success")}</>, { wrapper });
    expect(baseElement.textContent).toContain("成功");
    // 简化版无 icon span
    expect(baseElement.querySelector(".anticon")).toBeFalsy();
  });

  it("renderSimpleStatusTag 未知 → fallback", () => {
    const { baseElement } = render(<>{renderSimpleStatusTag("unknown" as ExecutionStatus)}</>, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("待执行");
  });
});
