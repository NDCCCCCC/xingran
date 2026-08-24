import { afterEach, describe, expect, it, vi } from "vitest";
import { getAppMessage, setAppMessageInstance } from "./antdMessage";

describe("antdMessage 桥接", () => {
  afterEach(() => {
    // 复位为 no-op，避免污染其他测试文件
    setAppMessageInstance(null);
  });

  it("未注入实例时返回 no-op 实例：所有方法静默短路不抛错", () => {
    setAppMessageInstance(null);
    const message = getAppMessage();

    expect(() => {
      message.success("ok");
      message.error("bad");
      message.info("info");
      message.warning("warn");
      message.loading("loading");
      message.warn("deprecated-warn");
      message.open({ type: "success", content: "open" });
      message.destroy();
    }).not.toThrow();
  });

  it("setAppMessageInstance 注入后 getAppMessage 返回同一实例", () => {
    const instance = {
      success: vi.fn(),
      error: vi.fn(),
    } as unknown as ReturnType<typeof getAppMessage>;
    setAppMessageInstance(instance);

    expect(getAppMessage()).toBe(instance);
    getAppMessage().success("注入成功");
    expect(instance.success).toHaveBeenCalledWith("注入成功");
  });

  it("传 null 重置回 no-op（不再持有过期实例）", () => {
    const instance = {
      success: vi.fn(),
    } as unknown as ReturnType<typeof getAppMessage>;
    setAppMessageInstance(instance);
    setAppMessageInstance(null);

    const message = getAppMessage();
    expect(message).not.toBe(instance);
    expect(() => message.success("noop")).not.toThrow();
    expect(instance.success).not.toHaveBeenCalled();
  });
});
