/**
 * Phase 88 Batch328 — pages/duty/holidays/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  HOLIDAY_TYPE_OPTIONS,
  WEEKDAY_TEXTS,
  renderHolidayTypeTag,
  renderIsOffdayTag,
} from "../constants";

describe("pages/duty/holidays/constants", () => {
  it("HOLIDAY_TYPE_OPTIONS 3 项", () => {
    expect(HOLIDAY_TYPE_OPTIONS.length).toBe(3);
    expect(HOLIDAY_TYPE_OPTIONS[0].value).toBe("legal");
  });

  it("WEEKDAY_TEXTS 7 项", () => {
    expect(WEEKDAY_TEXTS.length).toBe(7);
    expect(WEEKDAY_TEXTS[1]).toBe("一");
  });

  it("renderHolidayTypeTag legal → red 法定节假日", () => {
    render(renderHolidayTypeTag("legal"));
    expect(screen.getByText("法定节假日")).toBeInTheDocument();
  });

  it("renderHolidayTypeTag workday → orange", () => {
    render(renderHolidayTypeTag("workday"));
    expect(screen.getByText("调休工作日")).toBeInTheDocument();
  });

  it("renderHolidayTypeTag custom → blue", () => {
    render(renderHolidayTypeTag("custom"));
    expect(screen.getByText("自定义")).toBeInTheDocument();
  });

  it("renderIsOffdayTag true → 休息日", () => {
    render(renderIsOffdayTag(true));
    expect(screen.getByText("休息日")).toBeInTheDocument();
  });

  it("renderIsOffdayTag false → 工作日", () => {
    render(renderIsOffdayTag(false));
    expect(screen.getByText("工作日")).toBeInTheDocument();
  });
});
