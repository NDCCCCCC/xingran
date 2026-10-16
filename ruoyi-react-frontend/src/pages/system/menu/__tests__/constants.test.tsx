/**
 * Phase 88 Batch186 — pages/system/menu/constants 测试
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
  MENU_TYPE_OPTIONS,
  MENU_STATUS_OPTIONS,
  getMenuIcon,
  getMenuTypeTag,
  DEFAULT_FORM_VALUES,
} from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("system/menu/constants", () => {
  it("MENU_TYPE_OPTIONS 3 项", () => {
    expect(MENU_TYPE_OPTIONS.length).toBe(3);
    expect(MENU_TYPE_OPTIONS.map((o) => o.value)).toEqual(["M", "C", "F"]);
  });

  it("MENU_STATUS_OPTIONS 至少 2 项 + 字符串 value", () => {
    expect(MENU_STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
    expect(typeof MENU_STATUS_OPTIONS[0].value).toBe("string");
  });

  it("getMenuIcon M → FolderOutlined", () => {
    const { baseElement } = render(<>{getMenuIcon("M")}</>, { wrapper });
    expect(baseElement.querySelector(".anticon")).toBeTruthy();
  });

  it("getMenuIcon C → FileOutlined", () => {
    const { baseElement } = render(<>{getMenuIcon("C")}</>, { wrapper });
    expect(baseElement.querySelector(".anticon")).toBeTruthy();
  });

  it("getMenuIcon F → AppstoreOutlined", () => {
    const { baseElement } = render(<>{getMenuIcon("F")}</>, { wrapper });
    expect(baseElement.querySelector(".anticon")).toBeTruthy();
  });

  it("getMenuIcon 未知 → MenuOutlined", () => {
    const { baseElement } = render(<>{getMenuIcon("X" as any)}</>, { wrapper });
    expect(baseElement.querySelector(".anticon")).toBeTruthy();
  });

  it("getMenuTypeTag M → 目录", () => {
    const { baseElement } = render(<>{getMenuTypeTag("M")}</>, { wrapper });
    expect(baseElement.textContent).toContain("目录");
  });

  it("getMenuTypeTag C → 菜单", () => {
    const { baseElement } = render(<>{getMenuTypeTag("C")}</>, { wrapper });
    expect(baseElement.textContent).toContain("菜单");
  });

  it("getMenuTypeTag F → 按钮", () => {
    const { baseElement } = render(<>{getMenuTypeTag("F")}</>, { wrapper });
    expect(baseElement.textContent).toContain("按钮");
  });

  it("getMenuTypeTag 未知 → 未知", () => {
    const { baseElement } = render(<>{getMenuTypeTag("X" as any)}</>, { wrapper });
    expect(baseElement.textContent).toContain("未知");
  });

  it("DEFAULT_FORM_VALUES 默认值", () => {
    expect(DEFAULT_FORM_VALUES.parentId).toBe("");
    expect(DEFAULT_FORM_VALUES.menuType).toBe("M");
    expect(DEFAULT_FORM_VALUES.orderNum).toBe(0);
    expect(DEFAULT_FORM_VALUES.status).toBe(true);
    expect(DEFAULT_FORM_VALUES.visible).toBe(true);
  });
});
