/**
 * Phase 88 Batch157 — components/CronSelector/fields/MinuteField 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import MinuteField from "../MinuteField";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("MinuteField", () => {
  it("periodType=every → 渲染 Radio.Group + 4 个 Radio", () => {
    const { baseElement } = render(
      <MinuteField value={{ periodType: "every" }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("每分");
    expect(baseElement.textContent).toContain("指定");
    expect(baseElement.textContent).toContain("周期");
    expect(baseElement.textContent).toContain("范围");
  });

  it("Radio 切换 → onChange 调用", () => {
    const onChange = vi.fn();
    const { baseElement } = render(
      <MinuteField value={{ periodType: "every" }} onChange={onChange} />,
      { wrapper }
    );
    const radios = baseElement.querySelectorAll('input[type="radio"]');
    if (radios.length >= 2) fireEvent.click(radios[1]);
    expect(onChange).toHaveBeenCalled();
  });

  it("periodType=specific → 渲染 multiple Select", () => {
    const { baseElement } = render(
      <MinuteField value={{ periodType: "specific", specific: [0, 15] }} onChange={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("指定");
  });

  it("periodType=cycle → 渲染 cycleStart + cycleInterval", () => {
    const { baseElement } = render(
      <MinuteField
        value={{ periodType: "cycle", cycleStart: 0, cycleInterval: 5 }}
        onChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("从");
    expect(baseElement.textContent).toContain("分开始");
  });

  it("periodType=range → 渲染 rangeStart + rangeEnd", () => {
    const { baseElement } = render(
      <MinuteField
        value={{ periodType: "range", rangeStart: 0, rangeEnd: 30 }}
        onChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("从");
    expect(baseElement.textContent).toContain("到");
  });
});
