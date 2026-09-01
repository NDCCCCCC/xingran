/**
 * Phase 88 Batch399 — components/shared/BatchDeleteButton 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { App } from "antd";
import BatchDeleteButton from "../BatchDeleteButton";
import type { ReactElement } from "react";

const wrapper = ({ children }: { children: ReactElement }) => (
  <App>{children}</App>
);

describe("components/shared/BatchDeleteButton", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../BatchDeleteButton");
    expect(typeof mod.default).toBe("function");
  });

  it("disabled 时不抛错", () => {
    expect(() => render(<BatchDeleteButton disabled selectedCount={0} />, { wrapper })).not.toThrow();
  });
});
