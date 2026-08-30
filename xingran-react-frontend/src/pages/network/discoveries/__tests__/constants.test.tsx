/**
 * Phase 88 Batch182 — pages/network/discoveries/constants 测试
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
  DISCOVERY_TYPE_OPTIONS,
  STATUS_OPTIONS,
  STATUS_CONFIG,
  renderStatusTag,
} from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("network/discoveries/constants", () => {
  it("DISCOVERY_TYPE_OPTIONS 2 项", () => {
    expect(DISCOVERY_TYPE_OPTIONS.length).toBe(2);
    expect(DISCOVERY_TYPE_OPTIONS[0].value).toBe("snmp");
  });

  it("STATUS_OPTIONS 4 项", () => {
    expect(STATUS_OPTIONS.length).toBe(4);
    expect(STATUS_OPTIONS.map((o) => o.value)).toEqual([
      "pending",
      "running",
      "completed",
      "failed",
    ]);
  });

  it("STATUS_CONFIG 4 个状态", () => {
    expect(Object.keys(STATUS_CONFIG)).toEqual(
      expect.arrayContaining(["pending", "running", "completed", "failed"])
    );
  });

  it("renderStatusTag pending → 待执行", () => {
    const { baseElement } = render(<>{renderStatusTag("pending")}</>, { wrapper });
    expect(baseElement.textContent).toContain("待执行");
  });

  it("renderStatusTag running → 扫描中", () => {
    const { baseElement } = render(<>{renderStatusTag("running")}</>, { wrapper });
    expect(baseElement.textContent).toContain("扫描中");
  });

  it("renderStatusTag completed → 已完成", () => {
    const { baseElement } = render(<>{renderStatusTag("completed")}</>, { wrapper });
    expect(baseElement.textContent).toContain("已完成");
  });

  it("renderStatusTag failed → 失败", () => {
    const { baseElement } = render(<>{renderStatusTag("failed")}</>, { wrapper });
    expect(baseElement.textContent).toContain("失败");
  });

  it("renderStatusTag 未知 → fallback pending", () => {
    const { baseElement } = render(<>{renderStatusTag("unknown" as any)}</>, { wrapper });
    expect(baseElement.textContent).toContain("待执行");
  });
});
