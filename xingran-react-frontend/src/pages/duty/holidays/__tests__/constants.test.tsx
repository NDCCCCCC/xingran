/**
 * Phase 88 Batch175 — pages/duty/holidays/constants 测试
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
  HOLIDAY_TYPE_OPTIONS,
  WEEKDAY_TEXTS,
  renderHolidayTypeTag,
  renderIsOffdayTag,
} from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("duty/holidays/constants", () => {
  it("HOLIDAY_TYPE_OPTIONS 3 项", () => {
    expect(HOLIDAY_TYPE_OPTIONS.length).toBe(3);
    expect(HOLIDAY_TYPE_OPTIONS.map((o) => o.value)).toEqual(["legal", "workday", "custom"]);
  });

  it("WEEKDAY_TEXTS 7 项", () => {
    expect(WEEKDAY_TEXTS.length).toBe(7);
    expect(WEEKDAY_TEXTS[0]).toBe("日");
    expect(WEEKDAY_TEXTS[6]).toBe("六");
  });

  it("renderHolidayTypeTag legal → 法定节假日", () => {
    const { baseElement } = render(<>{renderHolidayTypeTag("legal")}</>, { wrapper });
    expect(baseElement.textContent).toContain("法定节假日");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderHolidayTypeTag workday → 调休工作日", () => {
    const { baseElement } = render(<>{renderHolidayTypeTag("workday")}</>, { wrapper });
    expect(baseElement.textContent).toContain("调休工作日");
  });

  it("renderHolidayTypeTag custom → 自定义", () => {
    const { baseElement } = render(<>{renderHolidayTypeTag("custom")}</>, { wrapper });
    expect(baseElement.textContent).toContain("自定义");
  });

  it("renderHolidayTypeTag 未知类型 → 显示原始字符串", () => {
    const { baseElement } = render(<>{renderHolidayTypeTag("unknown" as any)}</>, { wrapper });
    expect(baseElement.textContent).toContain("unknown");
  });

  it("renderIsOffdayTag true → 休息日", () => {
    const { baseElement } = render(<>{renderIsOffdayTag(true)}</>, { wrapper });
    expect(baseElement.textContent).toContain("休息日");
  });

  it("renderIsOffdayTag false → 工作日", () => {
    const { baseElement } = render(<>{renderIsOffdayTag(false)}</>, { wrapper });
    expect(baseElement.textContent).toContain("工作日");
  });
});
