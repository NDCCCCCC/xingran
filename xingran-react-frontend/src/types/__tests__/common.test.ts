/**
 * Phase 88 Batch272 — types/common 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  FormFieldError,
  FormData,
  UnknownError,
  isFormValidationError,
  VoidCallback,
  AsyncVoidCallback,
  SuccessCallback,
  ErrorCallback,
} from "../common";

describe("types/common", () => {
  it("FormFieldError shape", () => {
    const e: FormFieldError = { name: ["field"], errors: ["required"] };
    expect(e.name[0]).toBe("field");
  });

  it("FormData 含字段", () => {
    const d: FormData = { name: "x", age: 10, active: true };
    expect(d.name).toBe("x");
  });

  it("UnknownError shape", () => {
    const e: UnknownError = {
      message: "err",
      code: 500,
      response: { data: { message: "internal" }, status: 500 },
    };
    expect(e.code).toBe(500);
  });

  it("isFormValidationError true", () => {
    const e = { errorFields: [{ name: ["f"], errors: ["err"] }] };
    expect(isFormValidationError(e)).toBe(true);
  });

  it("isFormValidationError false", () => {
    expect(isFormValidationError(new Error("x"))).toBe(false);
    expect(isFormValidationError(null)).toBe(false);
    expect(isFormValidationError({ errorFields: "not array" })).toBe(false);
  });

  it("VoidCallback 编译", () => {
    const cb: VoidCallback = () => {};
    expect(typeof cb).toBe("function");
  });

  it("AsyncVoidCallback 编译", () => {
    const cb: AsyncVoidCallback = async () => {};
    expect(typeof cb).toBe("function");
  });

  it("SuccessCallback 编译", () => {
    const cb: SuccessCallback<string> = (data) => {};
    expect(typeof cb).toBe("function");
  });

  it("ErrorCallback 编译", () => {
    const cb: ErrorCallback = (error) => {};
    expect(typeof cb).toBe("function");
  });
});
