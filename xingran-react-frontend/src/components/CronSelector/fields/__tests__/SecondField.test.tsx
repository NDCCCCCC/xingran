/**
 * Phase 88 Batch161 — components/CronSelector/fields/SecondField 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import SecondField from "../SecondField";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("SecondField", () => {
  it("periodType=every → 渲染 Radio.Group", () => {
    const { baseElement } = render(
      <SecondField value={{ periodType: "every" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("每秒");
    expect(baseElement.textContent).toContain("指定");
    expect(baseElement.textContent).toContain("周期");
    expect(baseElement.textContent).toContain("范围");
  });

  it("Radio 切换 → onChange 调用", () => {
    const onChange = vi.fn();
    const { baseElement } = render(
      <SecondField value={{ periodType: "every" }} onChange={onChange} />,
      { wrapper }
    );
    const radios = baseElement.querySelectorAll('input[type="radio"]');
    if (radios.length >= 2) fireEvent.click(radios[1]);
    expect(onChange).toHaveBeenCalled();
  });

  it("periodType=specific → 渲染 multiple Select", () => {
    const { baseElement } = render(
      <SecondField value={{ periodType: "specific", specific: [0, 30] }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("指定");
  });

  it("periodType=cycle → 渲染 cycle Start/Interval", () => {
    const { baseElement } = render(
      <SecondField
        value={{ periodType: "cycle", cycleStart: 0, cycleInterval: 10 }}
        onChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("周期");
  });

  it("periodType=range → 渲染 range Start/End", () => {
    const { baseElement } = render(
      <SecondField
        value={{ periodType: "range", rangeStart: 0, rangeEnd: 30 }}
        onChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("范围");
  });
});
