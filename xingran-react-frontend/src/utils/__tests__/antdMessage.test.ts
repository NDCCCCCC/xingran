/**
 * Phase 88 Batch169 — utils/antdMessage 测试
 */
import { describe, it, expect } from "vitest";
import { getAppMessage, setAppMessageInstance } from "../antdMessage";

describe("antdMessage", () => {
  it("getAppMessage 在 <App> 未挂载时 → 返回 no-op 实例", () => {
    setAppMessageInstance(null);
    const msg = getAppMessage();
    expect(typeof msg.success).toBe("function");
    expect(typeof msg.error).toBe("function");
    expect(typeof msg.info).toBe("function");
    expect(typeof msg.warning).toBe("function");
  });

  it("setAppMessageInstance(null) → getAppMessage no-op", () => {
    setAppMessageInstance(null);
    const msg = getAppMessage();
    expect(msg).toBeDefined();
  });

  it("setAppMessageInstance(instance) → getAppMessage 返回该实例", () => {
    const successFn = () => "success-result";
    const customInstance = {
      success: successFn,
      error: () => "error",
      info: () => "info",
      warning: () => "warning",
      loading: () => "loading",
      warn: () => "warn",
      open: () => "open",
      destroy: () => "destroy",
    } as any;
    setAppMessageInstance(customInstance);
    expect(getAppMessage().success()).toBe("success-result");
  });

  it("no-op 实例调用不抛错", () => {
    setAppMessageInstance(null);
    const msg = getAppMessage();
    expect(() => msg.success("test")).not.toThrow();
    expect(() => msg.error("test")).not.toThrow();
    expect(() => msg.info("test")).not.toThrow();
    expect(() => msg.warning("test")).not.toThrow();
  });

  it("setAppMessageInstance 多次设置覆盖", () => {
    const inst1 = { success: () => "1" } as any;
    const inst2 = { success: () => "2" } as any;
    setAppMessageInstance(inst1);
    expect(getAppMessage().success()).toBe("1");
    setAppMessageInstance(inst2);
    expect(getAppMessage().success()).toBe("2");
  });
});
