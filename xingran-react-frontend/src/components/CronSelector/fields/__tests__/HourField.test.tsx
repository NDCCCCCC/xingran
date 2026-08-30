/**
 * Phase 88 Batch158 — components/CronSelector/fields/HourField 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import HourField from "../HourField";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("HourField", () => {
  it("periodType=every → 渲染 Radio.Group", () => {
    const { baseElement } = render(
      <HourField value={{ periodType: "every" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("每小时");
    expect(baseElement.textContent).toContain("指定");
    expect(baseElement.textContent).toContain("周期");
    expect(baseElement.textContent).toContain("范围");
  });

  it("Radio 切换 → onChange 调用", () => {
    const onChange = vi.fn();
    const { baseElement } = render(
      <HourField value={{ periodType: "every" }} onChange={onChange} />,
      { wrapper }
    );
    const radios = baseElement.querySelectorAll('input[type="radio"]');
    if (radios.length >= 2) fireEvent.click(radios[1]);
    expect(onChange).toHaveBeenCalled();
  });

  it("periodType=specific → 渲染 multiple Select", () => {
    const { baseElement } = render(
      <HourField value={{ periodType: "specific", specific: [9] }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("指定");
  });

  it("periodType=cycle → 渲染 cycleStart + cycleInterval", () => {
    const { baseElement } = render(
      <HourField
        value={{ periodType: "cycle", cycleStart: 0, cycleInterval: 2 }}
        onChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("从");
    expect(baseElement.textContent).toContain("小时");
  });

  it("periodType=range → 渲染 rangeStart + rangeEnd", () => {
    const { baseElement } = render(
      <HourField value={{ periodType: "range", rangeStart: 9, rangeEnd: 17 }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("从");
    expect(baseElement.textContent).toContain("到");
  });
});
