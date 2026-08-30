/**
 * Phase 88 Batch180 — pages/system/dict/constants 测试
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
  DEFAULT_TYPE_FORM_VALUES,
  DEFAULT_DATA_FORM_VALUES,
  renderStatusTag,
} from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("system/dict/constants", () => {
  it("STATUS_OPTIONS 至少 2 项", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("STATUS_CONFIG 至少 2 个状态", () => {
    expect(Object.keys(STATUS_CONFIG).length).toBeGreaterThanOrEqual(2);
  });

  it("DEFAULT_TYPE_FORM_VALUES status=0", () => {
    expect(DEFAULT_TYPE_FORM_VALUES.status).toBe(0);
  });

  it("DEFAULT_DATA_FORM_VALUES 字段", () => {
    expect(DEFAULT_DATA_FORM_VALUES.status).toBe(0);
    expect(DEFAULT_DATA_FORM_VALUES.isDefault).toBe(false);
  });

  it("renderStatusTag 0 → 启用 Tag", () => {
    const { baseElement } = render(<>{renderStatusTag(0)}</>, { wrapper });
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderStatusTag 1 → 禁用 Tag", () => {
    const { baseElement } = render(<>{renderStatusTag(1)}</>, { wrapper });
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });
});
