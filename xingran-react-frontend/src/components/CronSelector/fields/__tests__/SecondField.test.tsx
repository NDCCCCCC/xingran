/**
 * Phase 88 Batch424 — CronSelector/fields 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import SecondField from "../SecondField";
import MinuteField from "../MinuteField";
import HourField from "../HourField";
import DayField from "../DayField";
import MonthField from "../MonthField";
import WeekField from "../WeekField";
import type { ReactElement, ReactNode } from "react";
import type { CronValue } from "../../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const everyValue = { type: "every" } as CronValue;
const cb = (): void => {};

describe("components/CronSelector/fields/SecondField", () => {
  it("导出为函数组件", () => {
    expect(typeof SecondField).toBe("function");
  });

  it("基础渲染不抛错", () => {
    expect(() => render(<SecondField value={everyValue} onChange={cb} />, { wrapper })).not.toThrow();
  });
});

describe("components/CronSelector/fields/MinuteField", () => {
  it("导出为函数组件", () => {
    expect(typeof MinuteField).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() => render(<MinuteField value={everyValue} onChange={cb} />, { wrapper })).not.toThrow();
  });
});

describe("components/CronSelector/fields/HourField", () => {
  it("导出为函数组件", () => {
    expect(typeof HourField).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() => render(<HourField value={everyValue} onChange={cb} />, { wrapper })).not.toThrow();
  });
});

describe("components/CronSelector/fields/DayField", () => {
  it("导出为函数组件", () => {
    expect(typeof DayField).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() => render(<DayField value={everyValue} onChange={cb} />, { wrapper })).not.toThrow();
  });
});

describe("components/CronSelector/fields/MonthField", () => {
  it("导出为函数组件", () => {
    expect(typeof MonthField).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() => render(<MonthField value={everyValue} onChange={cb} />, { wrapper })).not.toThrow();
  });
});

describe("components/CronSelector/fields/WeekField", () => {
  it("导出为函数组件", () => {
    expect(typeof WeekField).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() => render(<WeekField value={everyValue} onChange={cb} />, { wrapper })).not.toThrow();
  });
});