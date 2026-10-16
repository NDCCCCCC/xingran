/**
 * Phase 88 Batch162 — components/CronSelector/fields/WeekField 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import WeekField from "../WeekField";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("WeekField", () => {
  it("periodType=every → 渲染 Radio.Group", () => {
    const { baseElement } = render(
      <WeekField value={{ periodType: "every" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("每周");
    expect(baseElement.textContent).toContain("指定");
  });

  it("Radio 切换 → onChange 调用", () => {
    const onChange = vi.fn();
    const { baseElement } = render(
      <WeekField value={{ periodType: "every" }} onChange={onChange} />,
      { wrapper }
    );
    const radios = baseElement.querySelectorAll('input[type="radio"]');
    if (radios.length >= 2) fireEvent.click(radios[1]);
    expect(onChange).toHaveBeenCalled();
  });

  it("periodType=specific → 渲染 Checkbox.Group 周一-周日", () => {
    const { baseElement } = render(
      <WeekField
        value={{ periodType: "specific", specific: [2, 3, 4, 5, 6] }}
        onChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("周一");
    expect(baseElement.textContent).toContain("周日");
  });
});
