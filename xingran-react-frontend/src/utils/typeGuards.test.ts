import { describe, expect, it } from "vitest";
import { getErrorMessage, hasProperty, isError } from "./typeGuards";

describe("isError", () => {
  it("Error 实例为 true", () => {
    expect(isError(new Error("x"))).toBe(true);
  });

  it("鸭子类型 { message: string } 为 true（跨 realm/序列化对象）", () => {
    expect(isError({ message: "duck-typed" })).toBe(true);
  });

  it("message 非字符串的对象为 false", () => {
    expect(isError({ message: 123 })).toBe(false);
  });

  it("字符串 / null / undefined / number 为 false", () => {
    expect(isError("plain")).toBe(false);
    expect(isError(null)).toBe(false);
    expect(isError(undefined)).toBe(false);
    expect(isError(42)).toBe(false);
  });
});

describe("getErrorMessage", () => {
  it("Error 对象取 message", () => {
    expect(getErrorMessage(new Error("real-error"))).toBe("real-error");
  });

  it("鸭子对象取 message", () => {
    expect(getErrorMessage({ message: "duck" })).toBe("duck");
  });

  it("字符串原样返回", () => {
    expect(getErrorMessage("raw-string")).toBe("raw-string");
  });

  it("其余类型返回兜底文案", () => {
    expect(getErrorMessage(42)).toBe("发生未知错误");
    expect(getErrorMessage(null)).toBe("发生未知错误");
    expect(getErrorMessage(undefined)).toBe("发生未知错误");
  });
});

describe("hasProperty", () => {
  it("存在指定属性为 true", () => {
    expect(hasProperty({ code: 0 }, "code")).toBe(true);
    expect(hasProperty({ a: undefined }, "a")).toBe(true); // in 语义含 undefined 值
  });

  it("属性不存在或非对象为 false", () => {
    expect(hasProperty({}, "code")).toBe(false);
    expect(hasProperty(null, "code")).toBe(false);
    expect(hasProperty("string", "length")).toBe(false);
    expect(hasProperty(undefined, "x")).toBe(false);
  });
});
