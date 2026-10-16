/**
 * Phase 88 Batch399 — components/shared/ActionButtons 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { App } from "antd";
import ActionButtons from "../ActionButtons";
import type { ReactElement } from "react";

const wrapper = ({ children }: { children: ReactElement }) => <App>{children}</App>;

describe("components/shared/ActionButtons", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../ActionButtons");
    expect(typeof mod.default).toBe("function");
  });

  it("无 actions 时不抛错", () => {
    expect(() => render(<ActionButtons actions={[]} />, { wrapper })).not.toThrow();
  });
});
