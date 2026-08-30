/**
 * Phase 88 Batch168 — utils/typeGuards 测试
 */
import { describe, it, expect } from "vitest";
import { isError, getErrorMessage, hasProperty } from "../typeGuards";

describe("typeGuards", () => {
  it("isError → Error instance → true", () => {
    expect(isError(new Error("boom"))).toBe(true);
  });

  it("isError → object with message → true", () => {
    expect(isError({ message: "boom" })).toBe(true);
  });

  it("isError → string → false", () => {
    expect(isError("boom")).toBe(false);
  });

  it("isError → null → false", () => {
    expect(isError(null)).toBe(false);
  });

  it("isError → undefined → false", () => {
    expect(isError(undefined)).toBe(false);
  });

  it("getErrorMessage → Error → message", () => {
    expect(getErrorMessage(new Error("Boom"))).toBe("Boom");
  });

  it("getErrorMessage → string → 自身", () => {
    expect(getErrorMessage("Something wrong")).toBe("Something wrong");
  });

  it("getErrorMessage → null → '发生未知错误'", () => {
    expect(getErrorMessage(null)).toBe("发生未知错误");
  });

  it("getErrorMessage → undefined → '发生未知错误'", () => {
    expect(getErrorMessage(undefined)).toBe("发生未知错误");
  });

  it("hasProperty → 对象有 prop → true", () => {
    expect(hasProperty({ name: "alice" }, "name")).toBe(true);
  });

  it("hasProperty → 对象无 prop → false", () => {
    expect(hasProperty({ age: 30 }, "name")).toBe(false);
  });

  it("hasProperty → null → false", () => {
    expect(hasProperty(null, "name")).toBe(false);
  });
});
