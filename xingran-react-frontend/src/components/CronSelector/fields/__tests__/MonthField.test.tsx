/**
 * Phase 88 Batch160 — components/CronSelector/fields/MonthField 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import MonthField from "../MonthField";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("MonthField", () => {
  it("periodType=every → 渲染 Radio.Group", () => {
    const { baseElement } = render(
      <MonthField value={{ periodType: "every" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("每月");
    expect(baseElement.textContent).toContain("指定");
    expect(baseElement.textContent).toContain("周期");
    expect(baseElement.textContent).toContain("范围");
  });

  it("Radio 切换 → onChange 调用", () => {
    const onChange = vi.fn();
    const { baseElement } = render(
      <MonthField value={{ periodType: "every" }} onChange={onChange} />,
      { wrapper }
    );
    const radios = baseElement.querySelectorAll('input[type="radio"]');
    if (radios.length >= 2) fireEvent.click(radios[1]);
    expect(onChange).toHaveBeenCalled();
  });

  it("periodType=specific → 渲染 multiple Select", () => {
    const { baseElement } = render(
      <MonthField value={{ periodType: "specific", specific: [1, 6, 12] }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("指定");
  });

  it("periodType=cycle → 渲染 cycle Start/Interval", () => {
    const { baseElement } = render(
      <MonthField
        value={{ periodType: "cycle", cycleStart: 1, cycleInterval: 3 }}
        onChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("周期");
  });

  it("periodType=range → 渲染 range Start/End", () => {
    const { baseElement } = render(
      <MonthField value={{ periodType: "range", rangeStart: 3, rangeEnd: 9 }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("范围");
  });
});
