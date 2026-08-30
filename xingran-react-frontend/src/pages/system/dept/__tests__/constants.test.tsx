/**
 * Phase 88 Batch185 — pages/system/dept/constants 测试
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
  EXTERNAL_ORG_OPTIONS,
  renderStatusTag,
  renderExternalOrgTag,
} from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("system/dept/constants", () => {
  it("STATUS_OPTIONS 共享 NORMAL_STOP_OPTIONS 2 项", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("EXTERNAL_ORG_OPTIONS 2 项", () => {
    expect(EXTERNAL_ORG_OPTIONS.length).toBe(2);
    expect(EXTERNAL_ORG_OPTIONS[0].value).toBe(0);
    expect(EXTERNAL_ORG_OPTIONS[1].value).toBe(1);
  });

  it("renderStatusTag 0 → 正常", () => {
    const { baseElement } = render(<>{renderStatusTag(0)}</>, { wrapper });
    expect(baseElement.textContent).toContain("正常");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderStatusTag 1 → 停用", () => {
    const { baseElement } = render(<>{renderStatusTag(1)}</>, { wrapper });
    expect(baseElement.textContent).toContain("停用");
  });

  it("renderExternalOrgTag 0 → 否", () => {
    const { baseElement } = render(<>{renderExternalOrgTag(0)}</>, { wrapper });
    expect(baseElement.textContent).toContain("否");
  });

  it("renderExternalOrgTag 1 → 是", () => {
    const { baseElement } = render(<>{renderExternalOrgTag(1)}</>, { wrapper });
    expect(baseElement.textContent).toContain("是");
  });
});
