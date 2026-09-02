/**
 * Phase 88 Batch399 — components/shared/ErrorAlertWithRetry 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { App } from "antd";
import ErrorAlertWithRetry from "../ErrorAlertWithRetry";
import type { ReactElement } from "react";

const wrapper = ({ children }: { children: ReactElement }) => <App>{children}</App>;

describe("components/shared/ErrorAlertWithRetry", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../ErrorAlertWithRetry");
    expect(typeof mod.default).toBe("function");
  });

  it("默认渲染不抛错", () => {
    expect(() => render(<ErrorAlertWithRetry />, { wrapper })).not.toThrow();
  });
});
