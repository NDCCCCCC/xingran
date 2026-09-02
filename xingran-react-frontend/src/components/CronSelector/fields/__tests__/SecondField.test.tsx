/**
 * Phase 88 Batch404 — CronSelector/fields 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import SecondField from "../SecondField";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("components/CronSelector/fields/SecondField", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../SecondField");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", () => {
    expect(() =>
      render(<SecondField value={{ type: "every" } as any} onChange={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });
});

import MinuteField from "../MinuteField";
describe("components/CronSelector/fields/MinuteField", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../MinuteField");
    expect(typeof mod.default).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() =>
      render(<MinuteField value={{ type: "every" } as any} onChange={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });
});

import HourField from "../HourField";
describe("components/CronSelector/fields/HourField", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../HourField");
    expect(typeof mod.default).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() =>
      render(<HourField value={{ type: "every" } as any} onChange={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });
});

import DayField from "../DayField";
describe("components/CronSelector/fields/DayField", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../DayField");
    expect(typeof mod.default).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() =>
      render(<DayField value={{ type: "every" } as any} onChange={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });
});

import MonthField from "../MonthField";
describe("components/CronSelector/fields/MonthField", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../MonthField");
    expect(typeof mod.default).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() =>
      render(<MonthField value={{ type: "every" } as any} onChange={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });
});

import WeekField from "../WeekField";
describe("components/CronSelector/fields/WeekField", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../WeekField");
    expect(typeof mod.default).toBe("function");
  });
  it("基础渲染不抛错", () => {
    expect(() =>
      render(<WeekField value={{ type: "every" } as any} onChange={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });
});
